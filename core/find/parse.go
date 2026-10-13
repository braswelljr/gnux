package find

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
)

// pred is one node of the expression: a test, an action or an operator.
type pred func(e *entry) bool

// parser turns the expression words into a tree of preds.
type parser struct {
	f    *Find
	args []string
	pos  int

	// hasAction records an action other than -prune, which suppresses the
	// implicit -print.
	hasAction bool
	// firstTest is the first test or action seen; a global option after it
	// earns GNU's warning.
	firstTest string
}

// parseExpression builds f.root from the expression words.
func (f *Find) parseExpression(args []string) error {
	p := &parser{f: f, args: args}
	printer := f.printPred(f.stdout, '\n')
	if len(args) == 0 {
		f.root = printer
		return nil
	}

	root, err := p.comma("")
	if err != nil {
		return err
	}
	if p.pos < len(p.args) {
		// Only a ')' with no '(' before it stops the parse early.
		return core.Usagef("invalid expression; you have too many ')'")
	}
	if !p.hasAction {
		root = f.and(root, printer)
	}
	f.root = root
	return nil
}

func (p *parser) peek() (string, bool) {
	if p.pos < len(p.args) {
		return p.args[p.pos], true
	}
	return "", false
}

// arg takes the argument of predicate name.
func (p *parser) arg(name string) (string, error) {
	if p.pos >= len(p.args) {
		return "", core.Usagef("missing argument to `%s'", name)
	}
	p.pos++
	return p.args[p.pos-1], nil
}

func isOr(s string) bool  { return s == "-o" || s == "-or" }
func isAnd(s string) bool { return s == "-a" || s == "-and" }
func isNot(s string) bool { return s == "!" || s == "-not" }

// comma parses "expr , expr": both sides are evaluated and the value is the
// right one's. after is the operator the expression follows, for messages.
func (p *parser) comma(after string) (pred, error) {
	left, err := p.or(after)
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || tok != "," {
			return left, nil
		}
		p.pos++
		right, err := p.or(tok)
		if err != nil {
			return nil, err
		}
		left = p.f.comma(left, right)
	}
}

func (p *parser) or(after string) (pred, error) {
	left, err := p.and(after)
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || !isOr(tok) {
			return left, nil
		}
		p.pos++
		right, err := p.and(tok)
		if err != nil {
			return nil, err
		}
		left = p.f.or(left, right)
	}
}

func (p *parser) and(after string) (pred, error) {
	left, err := p.not(after)
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := p.peek()
		if !ok || tok == ")" || tok == "," || isOr(tok) {
			return left, nil
		}
		op := ""
		if isAnd(tok) {
			p.pos++
			op = tok
		}
		right, err := p.not(op)
		if err != nil {
			return nil, err
		}
		left = p.f.and(left, right)
	}
}

func (p *parser) not(after string) (pred, error) {
	tok, ok := p.peek()
	if ok && isNot(tok) {
		p.pos++
		operand, err := p.not(tok)
		if err != nil {
			return nil, err
		}
		return func(e *entry) bool { return !operand(e) }, nil
	}
	return p.primary(after)
}

// primary parses a parenthesized expression or one predicate with its
// arguments.
func (p *parser) primary(after string) (pred, error) {
	tok, ok := p.peek()
	switch {
	case !ok:
		return nil, core.Usagef("expected an expression after '%s'", after)
	case tok == ")" && after != "":
		return nil, core.Usagef("expected an expression between '%s' and ')'", after)
	case tok == ")":
		return nil, core.Usagef("invalid expression; you have too many ')'")
	case tok == "," || isOr(tok) || isAnd(tok):
		return nil, core.Usagef("invalid expression; you have used a binary operator '%s' with nothing before it.", tok)
	case tok == "(":
		p.pos++
		if next, ok := p.peek(); ok && next == ")" {
			return nil, core.Usagef("invalid expression; empty parentheses are not allowed.")
		}
		if _, ok := p.peek(); !ok {
			return nil, core.Usagef("invalid expression; I was expecting to find a ')' somewhere but did not see one.")
		}
		inner, err := p.comma("")
		if err != nil {
			return nil, err
		}
		if next, ok := p.peek(); !ok || next != ")" {
			return nil, core.Usagef("invalid expression; I was expecting to find a ')' somewhere but did not see one.")
		}
		p.pos++
		return inner, nil
	case len(tok) > 1 && tok[0] == '-':
		p.pos++
		return p.predicate(tok)
	}
	return nil, core.Usagef("paths must precede expression: `%s'", tok)
}

// --- operators ---

// The operators stop evaluating once -quit has run: GNU exits on the spot.

func (f *Find) and(l, r pred) pred {
	return func(e *entry) bool { return l(e) && !f.quit && r(e) }
}

func (f *Find) or(l, r pred) pred {
	return func(e *entry) bool { return l(e) || (!f.quit && r(e)) }
}

func (f *Find) comma(l, r pred) pred {
	return func(e *entry) bool {
		l(e)
		return !f.quit && r(e)
	}
}

func truePred(*entry) bool { return true }

// --- predicates ---

// global notes a global option: it is not positional, so GNU warns when one
// follows a test or an action.
func (p *parser) global(name string) {
	if p.firstTest != "" && p.f.warn {
		p.f.diag.Warnf("warning: you have specified the global option %s after the argument %s, but global options are not positional, i.e., %s affects tests specified before it as well as those specified after it.  Please specify global options before other arguments.", name, p.firstTest, name)
	}
}

// predicate parses the predicate called name, taking its arguments.
func (p *parser) predicate(name string) (pred, error) {
	f := p.f

	// Options: always true, and they neither count as a test for the
	// global-option warning nor as an action.
	switch name {
	case "-maxdepth", "-mindepth":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		n, perr := strconv.ParseUint(arg, 10, 31)
		if perr != nil {
			return nil, core.Usagef("Expected a positive decimal integer argument to %s, but got '%s'", name, arg)
		}
		p.global(name)
		if name == "-maxdepth" {
			f.maxDepth = int(n)
		} else {
			f.minDepth = int(n)
		}
		return truePred, nil
	case "-depth", "-d":
		if name == "-d" && f.warn {
			f.diag.Warnf("warning: the -d option is deprecated; please use -depth instead, because the latter is a POSIX-compliant feature.")
		}
		p.global(name)
		f.depthFirst = true
		return truePred, nil
	case "-xdev", "-mount":
		p.global(name)
		f.xdev = true
		return truePred, nil
	case "-noleaf":
		p.global(name)
		return truePred, nil
	case "-ignore_readdir_race", "-noignore_readdir_race":
		p.global(name)
		f.ignoreRace = name == "-ignore_readdir_race"
		return truePred, nil
	case "-daystart":
		// The origin of the day tests becomes the start of today.
		t := f.now.In(f.Location)
		f.dayStart = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, f.Location)
		return truePred, nil
	case "-follow":
		// Deprecated spelling of -L, taking effect from here on.
		f.follow = followAll
		return truePred, nil
	case "-warn", "-nowarn":
		f.warn = name == "-warn"
		return truePred, nil
	case "-regextype":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		if !knownRegexType(arg) {
			return nil, core.Usagef("Unknown regular expression type '%s'; valid types are %s.", arg, regexTypeList())
		}
		f.regexType = arg
		return truePred, nil
	}

	if p.firstTest == "" {
		p.firstTest = name
	}

	switch name {
	case "-true":
		return truePred, nil
	case "-false":
		return func(*entry) bool { return false }, nil

	case "-name", "-iname":
		pat, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		if f.warn && strings.Contains(pat, "/") && pat != "/" {
			f.diag.Warnf("warning: '%s' matches against basenames only, but the given pattern contains a directory separator ('/'), thus the expression will evaluate to false all the time.  Did you mean '%s'?",
				name, map[string]string{"-name": "-wholename", "-iname": "-iwholename"}[name])
		}
		fold := name == "-iname"
		return func(e *entry) bool {
			if e.de != nil {
				return fnmatch(pat, e.de.Name(), fold)
			}
			return fnmatch(pat, baseName(e.path), fold)
		}, nil
	case "-path", "-wholename", "-ipath", "-iwholename":
		pat, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		fold := name[1] == 'i'
		return func(e *entry) bool { return fnmatch(pat, e.path, fold) }, nil
	case "-lname", "-ilname":
		pat, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		fold := name == "-ilname"
		return func(e *entry) bool {
			target, ok := f.linkTarget(e)
			return ok && fnmatch(pat, target, fold)
		}, nil
	case "-regex", "-iregex":
		pat, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		re, err := compileRegex(f.regexType, pat, name == "-iregex")
		if err != nil {
			return nil, core.Usagef("%s", err.Error())
		}
		return func(e *entry) bool { return re.MatchString(e.path) }, nil

	case "-type", "-xtype":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		set, err := parseTypes(name, arg)
		if err != nil {
			return nil, err
		}
		if name == "-type" {
			return func(e *entry) bool { return set[typeLetter(f.typeOf(e))] }, nil
		}
		return func(e *entry) bool { return set[f.xtype(e)] }, nil

	case "-size":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		cmp, unit, err := parseSize(arg)
		if err != nil {
			return nil, err
		}
		return func(e *entry) bool {
			info, ok := f.stat(e)
			if !ok {
				return false
			}
			size := uint64(max(info.Size(), 0))
			// GNU rounds the size up to whole units before comparing.
			return cmp.match((size + unit - 1) / unit)
		}, nil
	case "-empty":
		return f.empty, nil

	case "-newer", "-anewer", "-cnewer":
		x := map[string]byte{"-newer": 'm', "-anewer": 'a', "-cnewer": 'c'}[name]
		return p.newer(name, x, 'm')
	case "-mtime", "-atime", "-ctime":
		return p.age(name, name[1], 24*time.Hour)
	case "-mmin", "-amin", "-cmin":
		return p.age(name, name[1], time.Minute)

	case "-perm":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		kind, bits, ok := parsePerm(arg)
		if !ok {
			return nil, core.Usagef("invalid mode '%s'", arg)
		}
		return func(e *entry) bool {
			info, ok := f.stat(e)
			if !ok {
				return false
			}
			have := unixPerm(info.Mode())
			switch kind {
			case '-':
				return have&bits == bits
			case '/':
				// GNU: a mode with no bits set matches every file.
				return bits == 0 || have&bits != 0
			}
			return have == bits
		}, nil

	case "-user", "-group":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		id, ok := lookupID(arg, name == "-user")
		if !ok {
			if name == "-user" {
				return nil, core.Usagef("'%s' is not the name of a known user", arg)
			}
			return nil, core.Usagef("'%s' is not the name of an existing group", arg)
		}
		return f.idPred(name == "-user", numCmp{n: uint64(id)}), nil
	case "-uid", "-gid":
		cmp, err := p.number(name)
		if err != nil {
			return nil, err
		}
		return f.idPred(name == "-uid", cmp), nil
	case "-nouser", "-nogroup":
		user := name == "-nouser"
		return func(e *entry) bool {
			sys, ok := f.sysOf(e)
			if !ok {
				return false
			}
			id := sys.GID
			if user {
				id = sys.UID
			}
			// A platform with no numeric owners has no unnamed ones either.
			return id >= 0 && !f.names.known(id, user)
		}, nil

	case "-links", "-inum":
		cmp, err := p.number(name)
		if err != nil {
			return nil, err
		}
		links := name == "-links"
		return func(e *entry) bool {
			sys, ok := f.sysOf(e)
			if !ok {
				return false
			}
			if links {
				return cmp.match(sys.Nlink)
			}
			return cmp.match(sys.Ino)
		}, nil
	case "-samefile":
		arg, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		ref, err := f.refStat(arg)
		if err != nil {
			return nil, core.Usagef("'%s': %s", arg, errText(err))
		}
		return func(e *entry) bool {
			info, ok := f.stat(e)
			return ok && os.SameFile(ref, info)
		}, nil

	case "-readable":
		return func(e *entry) bool { return access(e.path, accessRead) }, nil
	case "-writable":
		return func(e *entry) bool { return access(e.path, accessWrite) }, nil
	case "-executable":
		return func(e *entry) bool { return access(e.path, accessExec) }, nil

	case "-prune":
		return func(e *entry) bool {
			e.pruned = true
			return true
		}, nil
	case "-quit":
		p.hasAction = true
		return func(*entry) bool {
			f.quit = true
			return true
		}, nil

	case "-print":
		p.hasAction = true
		return f.printPred(f.stdout, '\n'), nil
	case "-print0":
		p.hasAction = true
		return f.printPred(f.stdout, 0), nil
	case "-fprint", "-fprint0":
		o, err := p.file(name)
		if err != nil {
			return nil, err
		}
		if name == "-fprint0" {
			return f.printPred(o, 0), nil
		}
		return f.printPred(o, '\n'), nil
	case "-printf":
		p.hasAction = true
		format, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		return f.printfPred(f.stdout, format)
	case "-fprintf":
		o, err := p.file(name)
		if err != nil {
			return nil, err
		}
		format, err := p.arg(name)
		if err != nil {
			return nil, err
		}
		return f.printfPred(o, format)
	case "-ls":
		p.hasAction = true
		return f.lsPred(f.stdout), nil
	case "-fls":
		o, err := p.file(name)
		if err != nil {
			return nil, err
		}
		return f.lsPred(o), nil

	case "-delete":
		p.hasAction = true
		// Deleting needs a directory's contents gone before the directory.
		f.depthFirst = true
		return f.delete, nil

	case "-exec", "-execdir", "-ok", "-okdir":
		p.hasAction = true
		return p.exec(name)

	case "-fstype", "-context", "-used", "-files0-from":
		return nil, core.Usagef("the %s predicate is not supported", name)
	}

	if len(name) == 8 && strings.HasPrefix(name, "-newer") {
		if x, y := name[6], name[7]; strings.IndexByte("aBcm", x) >= 0 && strings.IndexByte("aBcmt", y) >= 0 {
			return p.newer(name, x, y)
		}
	}
	return nil, core.Usagef("unknown predicate `%s'", name)
}

// file takes the FILE argument of an -f* action and opens it.
func (p *parser) file(name string) (*output, error) {
	p.hasAction = true
	path, err := p.arg(name)
	if err != nil {
		return nil, err
	}
	o, err := p.f.outputFor(path)
	if err != nil {
		return nil, core.Usagef("'%s': %s", path, errText(err))
	}
	return o, nil
}

// --- numbers ---

// numCmp is a numeric argument: +N (more than), -N (less than) or N (exactly).
type numCmp struct {
	kind byte // 0, '+' or '-'
	n    uint64
}

func (c numCmp) match(v uint64) bool {
	switch c.kind {
	case '+':
		return v > c.n
	case '-':
		return v < c.n
	}
	return v == c.n
}

// splitSign removes the +/- of a numeric argument.
func splitSign(s string) (byte, string) {
	if s != "" && (s[0] == '+' || s[0] == '-') {
		return s[0], s[1:]
	}
	return 0, s
}

func parseNum(s string) (numCmp, bool) {
	kind, digits := splitSign(s)
	n, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return numCmp{}, false
	}
	return numCmp{kind: kind, n: n}, true
}

// number takes the numeric argument of predicate name.
func (p *parser) number(name string) (numCmp, error) {
	arg, err := p.arg(name)
	if err != nil {
		return numCmp{}, err
	}
	cmp, ok := parseNum(arg)
	if !ok {
		return numCmp{}, core.Usagef("invalid argument `%s' to `%s'", arg, name)
	}
	return cmp, nil
}

// parseSize parses -size's N[cwbkMG].
func parseSize(arg string) (numCmp, uint64, error) {
	if arg == "" {
		return numCmp{}, 0, core.Usagef("invalid null argument to -size")
	}
	unit := uint64(512)
	digits := arg
	switch c := arg[len(arg)-1]; {
	case c >= '0' && c <= '9':
	case strings.IndexByte("bcwkMG", c) >= 0:
		unit = map[byte]uint64{'b': 512, 'c': 1, 'w': 2, 'k': 1 << 10, 'M': 1 << 20, 'G': 1 << 30}[c]
		digits = arg[:len(arg)-1]
	default:
		return numCmp{}, 0, core.Usagef("invalid -size type '%c'", c)
	}
	cmp, ok := parseNum(digits)
	if !ok {
		return numCmp{}, 0, core.Usagef("Invalid argument '%s' to -size", arg)
	}
	return cmp, unit, nil
}

// --- types ---

// typeLetter maps a file type to find's letter for it.
func typeLetter(mode fs.FileMode) byte {
	switch {
	case mode.IsRegular():
		return 'f'
	case mode&fs.ModeDir != 0:
		return 'd'
	case mode&fs.ModeSymlink != 0:
		return 'l'
	case mode&fs.ModeNamedPipe != 0:
		return 'p'
	case mode&fs.ModeSocket != 0:
		return 's'
	case mode&fs.ModeCharDevice != 0:
		return 'c'
	case mode&fs.ModeDevice != 0:
		return 'b'
	}
	return 'U'
}

// parseTypes parses the comma-separated letters of -type and -xtype.
func parseTypes(name, arg string) (*[256]bool, error) {
	if arg == "" {
		return nil, core.Usagef("Arguments to %s should contain at least one letter", name)
	}
	var set [256]bool
	for i := 0; i < len(arg); i += 2 {
		c := arg[i]
		if c == ',' || strings.IndexByte("bcdpfls", c) < 0 {
			return nil, core.Usagef("Unknown argument to %s: %c", name, c)
		}
		if set[c] {
			return nil, core.Usagef("Duplicate file type '%c' in the argument list to %s.", c, name)
		}
		set[c] = true
		switch {
		case i+1 == len(arg):
		case arg[i+1] != ',':
			return nil, core.Usagef("Must separate multiple arguments to %s using: ','", name)
		case i+2 == len(arg):
			return nil, core.Usagef("Last file type in list argument to %s is missing, i.e., list is ending on: ','", name)
		}
	}
	return &set, nil
}

// xtype answers -xtype: the type -type does not look at. Without -L that is
// the type of a symbolic link's target (or 'l' for a broken link); for a
// link that was followed it is 'l'.
func (f *Find) xtype(e *entry) byte {
	if e.follow {
		return 'l'
	}
	t := f.typeOf(e)
	if t&fs.ModeSymlink == 0 || f.follow == followAll {
		return typeLetter(t)
	}
	info, err := os.Stat(e.path)
	if err != nil {
		return 'l'
	}
	return typeLetter(info.Mode().Type())
}

// linkTarget returns what a symbolic link points to; ok is false for any
// other file, and for a link that was followed.
func (f *Find) linkTarget(e *entry) (string, bool) {
	if e.follow || f.typeOf(e)&fs.ModeSymlink == 0 {
		return "", false
	}
	target, err := os.Readlink(e.path)
	if err != nil {
		return "", false
	}
	return target, true
}

// empty answers -empty: a regular file of size 0, or a directory with no
// entries.
func (f *Find) empty(e *entry) bool {
	t := f.typeOf(e)
	switch {
	case t.IsRegular():
		info, ok := f.stat(e)
		return ok && info.Size() == 0
	case t&fs.ModeDir != 0:
		isEmpty, err := dirEmpty(e.path)
		if err != nil {
			f.diag.Errorf("'%s': %s", e.path, errText(err))
			return false
		}
		return isEmpty
	}
	return false
}

// dirEmpty reports whether the directory at path has no entries.
func dirEmpty(path string) (bool, error) {
	d, err := os.Open(path)
	if err != nil {
		return false, err
	}
	names, err := d.Readdirnames(1)
	_ = d.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	return len(names) == 0, nil
}

// --- times ---

// timeOf returns the access, change, modification or birth time of e. Where
// the platform records no change time the modification time stands in.
func (f *Find) timeOf(e *entry, which byte) (time.Time, bool) {
	info, ok := f.stat(e)
	if !ok {
		return time.Time{}, false
	}
	return fileTime(info, e.sysstat().Atime, e.sysstat().Ctime, e.sysstat().Btime, which)
}

func fileTime(info os.FileInfo, atime, ctime, btime time.Time, which byte) (time.Time, bool) {
	var t time.Time
	switch which {
	case 'a':
		t = atime
	case 'c':
		t = ctime
	case 'B':
		return btime, !btime.IsZero()
	default:
		return info.ModTime(), true
	}
	if t.IsZero() {
		t = info.ModTime()
	}
	return t, true
}

// age parses -mtime N and its relatives. GNU turns N into a reference time
// once, here, and compares timestamps against it:
//
//	+N  older than the reference
//	-N  newer than the reference
//	N   within one unit after the reference
//
// For the day tests the origin is 24 hours ago (so fractions of a day are
// dropped: -mtime +1 is "at least two days old"), or the start of today with
// -daystart. For the minute tests it is now, or the end of today.
func (p *parser) age(name string, which byte, unit time.Duration) (pred, error) {
	f := p.f
	arg, err := p.arg(name)
	if err != nil {
		return nil, err
	}
	kind, digits := splitSign(arg)
	n, perr := strconv.ParseFloat(digits, 64)
	if perr != nil || n < 0 || digits == "" || strings.ContainsAny(digits, "+-xXpPiInN") {
		return nil, core.Usagef("invalid argument `%s' to `%s'", arg, name)
	}

	origin := f.dayStart
	if unit == time.Minute {
		origin = origin.Add(24 * time.Hour)
	} else if kind == '-' {
		origin = origin.Add(24*time.Hour - time.Second)
	}
	ref := origin.Add(-time.Duration(n * float64(unit)))

	return func(e *entry) bool {
		ts, ok := f.timeOf(e, which)
		if !ok {
			return false
		}
		switch kind {
		case '+':
			return ts.Before(ref)
		case '-':
			return ts.After(ref)
		}
		delta := ts.Sub(ref)
		return delta > 0 && delta <= unit
	}, nil
}

// newer parses -newer FILE, -anewer, -cnewer and -newerXY REFERENCE: true
// when the file's X time is later than the reference's Y time, or than the
// time REFERENCE spells when Y is t.
func (p *parser) newer(name string, x, y byte) (pred, error) {
	f := p.f
	arg, err := p.arg(name)
	if err != nil {
		return nil, err
	}

	var ref time.Time
	if y == 't' {
		t, ok := parseTimestamp(arg, f.Location)
		if !ok {
			return nil, core.Usagef("I cannot figure out how to interpret '%s' as a date or time", arg)
		}
		ref = t
	} else {
		info, err := f.refStat(arg)
		if err != nil {
			return nil, core.Usagef("'%s': %s", arg, errText(err))
		}
		re := &entry{path: arg, info: info, infoDone: true}
		t, ok := f.timeOf(re, y)
		if !ok {
			return nil, core.Usagef("This system does not provide a way to find the birth time of a file.")
		}
		ref = t
	}

	return func(e *entry) bool {
		ts, ok := f.timeOf(e, x)
		return ok && ts.After(ref)
	}, nil
}

// --- owners ---

func (f *Find) idPred(user bool, cmp numCmp) pred {
	return func(e *entry) bool {
		sys, ok := f.sysOf(e)
		if !ok {
			return false
		}
		id := sys.GID
		if user {
			id = sys.UID
		}
		return id >= 0 && cmp.match(uint64(id))
	}
}

// --- simple actions ---

// printPred prints the path followed by end (a newline, or NUL for -print0).
func (f *Find) printPred(o *output, end byte) pred {
	return func(e *entry) bool {
		f.buf = append(append(f.buf[:0], e.path...), end)
		f.write(o, f.buf)
		return true
	}
}

// delete answers -delete: remove the file, or the (by now empty) directory.
// "." is left alone, as in GNU, and a failure is diagnosed and false.
func (f *Find) delete(e *entry) bool {
	if baseName(e.path) == "." {
		return true
	}
	err := os.Remove(e.path)
	if err == nil || (f.ignoreRace && os.IsNotExist(err)) {
		return true
	}
	text := errText(err)
	if isEmpty, derr := dirEmpty(e.path); derr == nil && !isEmpty && f.isDir(e) {
		// The error for a directory that still has entries differs by
		// platform; GNU's words for it do not.
		text = "Directory not empty"
	}
	f.diag.Errorf("cannot delete '%s': %s", e.path, text)
	return false
}
