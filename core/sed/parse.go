package sed

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// addrKind is the form of one address.
type addrKind uint8

const (
	addrNone     addrKind = iota
	addrLine              // N
	addrStep              // first~step
	addrLast              // $
	addrRegex             // /regex/ or \cregexc
	addrPlus              // addr1,+N (second address only)
	addrMultiple          // addr1,~N (second address only)
)

// address selects lines for a command.
type address struct {
	kind addrKind
	n    int64          // the line, the first line of a step, or N of +N and ~N
	step int64          // addrStep
	re   *regexp.Regexp // addrRegex; nil is the empty regex: the last one used
}

// command is one compiled command. A script is a flat list of them: a block
// is its '{' command, which jumps past the block when its address does not
// select the line.
type command struct {
	name   byte
	negate bool
	a1, a2 address

	// The state of a two-address range while the script runs.
	active  bool  // inside the range
	end     int64 // its last line, when the second address fixes one
	started bool  // a 0,/regex/ range has been entered

	jump  int    // '{': the command after the block; b t T: the target
	label string // b t T
	text  []byte // a i c
	file  string // r R w W, and the w flag of s
	out   *output
	num   int // q Q: the exit status; l: the line length, or -1

	re     *regexp.Regexp // s; nil is the empty regex
	repl   replacement
	global bool // s///g
	print  bool // s///p
	nth    int  // s///N: the first match to replace, 1 by default
	tr     *translit

	pos int // where in the script the command is, for a diagnostic
}

// segment is one script source inside the parser's buffer: a -e expression
// or a script file.
type segment struct {
	start, end int    // src[start:end]
	file       string // the file's name, for a script file
	expr       int    // the number of the -e expression otherwise
}

// scriptError is a mistake in the script, already worded the way GNU sed
// words it, location included.
type scriptError struct{ msg string }

func (e *scriptError) Error() string { return e.msg }

// The diagnostics of GNU sed's compiler.
const (
	msgMultipleBang    = "multiple `!'s"
	msgUnexpectedComma = "unexpected `,'"
	msgBadFirstAddr    = "invalid usage of +N or ~N as first address"
	msgUnmatchedOpen   = "unmatched `{'"
	msgUnexpectedClose = "unexpected `}'"
	msgExtraChars      = "extra characters after command"
	msgExpectedSlash   = "expected \\ after `a', `c' or `i'"
	msgCloseNoAddr     = "`}' doesn't want any addresses"
	msgColonNoAddr     = ": doesn't want any addresses"
	msgCommentNoAddr   = "comments don't accept any addresses"
	msgMissingCommand  = "missing command"
	msgOneAddr         = "command only uses one address"
	msgUntermAddr      = "unterminated address regex"
	msgUntermS         = "unterminated `s' command"
	msgUntermY         = "unterminated `y' command"
	msgUnknownSOpt     = "unknown option to `s'"
	msgMultipleP       = "multiple `p' options to `s' command"
	msgMultipleG       = "multiple `g' options to `s' command"
	msgMultipleNumber  = "multiple number options to `s' command"
	msgZeroNumber      = "number option to `s' command may not be zero"
	msgYLength         = "strings for `y' command are different lengths"
	msgDelimiter       = "delimiter character is not a single-byte character"
	msgNewerVersion    = "expected newer version of sed"
	msgLineZero        = "invalid usage of line address 0"
	msgColonLabel      = "\":\" lacks a label"
	msgSandbox         = "e/r/w commands disabled in sandbox mode"
	msgMissingFile     = "missing filename in r/R/w/W commands"
	msgEmptyModifiers  = "cannot specify modifiers on empty regexp"
	msgNoExecute       = "the `e' command is not supported: sedx does not run commands"
	msgNoExecuteFlag   = "the `e' option to `s' is not supported: sedx does not run commands"
)

// eof is what the parser reads at the end of the script.
const eof = -1

// parser compiles a script. It reports a mistake by panicking with a
// *scriptError, which parse recovers: the grammar is read by many small
// functions and none of them can go on after one.
type parser struct {
	src      string
	pos      int // the next character to read
	segs     []segment
	extended bool
	sandbox  bool

	prog   []command
	blocks []int          // the '{' commands still open
	labels map[string]int // a label's name to the command after it
	quiet  bool           // the script starts with #n
}

// program is a compiled script.
type program struct {
	commands []command
	quiet    bool // the script's first line is #n
}

// parse compiles the script sources, joined as GNU sed joins them: each ends
// where a newline would. It returns a *scriptError for the first mistake.
func parse(sources []source, extended, sandbox bool) (prog program, err error) {
	p := &parser{extended: extended, sandbox: sandbox, labels: map[string]int{}}
	var src strings.Builder
	exprs := 0
	for _, s := range sources {
		seg := segment{start: src.Len(), file: s.file}
		if s.file == "" {
			exprs++
			seg.expr = exprs
		}
		src.WriteString(s.text)
		seg.end = src.Len()
		src.WriteByte('\n')
		p.segs = append(p.segs, seg)
	}
	p.src = src.String()
	defer func() {
		if r := recover(); r != nil {
			se, ok := r.(*scriptError)
			if !ok {
				panic(r)
			}
			err = se
		}
	}()
	p.run()
	return program{commands: p.prog, quiet: p.quiet}, nil
}

// fail reports msg at the parser's position.
func (p *parser) fail(msg string) {
	panic(&scriptError{p.where(p.pos, false) + msg})
}

// failf reports a formatted message at the parser's position.
func (p *parser) failf(format string, a ...any) {
	p.fail(fmt.Sprintf(format, a...))
}

// where renders the location prefix of a diagnostic about the character
// before pos: "-e expression #N, char M: " or "file F line L: ". GNU sed
// counts the characters read so far in the expression, and reading its end
// does not count. whole asks for the location of an entire source, which
// GNU reports as character 0.
func (p *parser) where(pos int, whole bool) string {
	seg := p.segs[len(p.segs)-1]
	for _, s := range p.segs {
		// The newline after a source reads as that source's end.
		if pos <= s.end+1 {
			seg = s
			break
		}
	}
	pos = max(seg.start, min(pos, seg.end))
	if seg.file != "" {
		line := 1 + strings.Count(p.src[seg.start:pos], "\n")
		return "file " + seg.file + " line " + strconv.Itoa(line) + ": "
	}
	char := pos - seg.start
	if whole {
		char = 0
	}
	return "-e expression #" + strconv.Itoa(seg.expr) + ", char " + strconv.Itoa(char) + ": "
}

// atSourceEnd reports whether the character just read is the newline that
// stands for the end of a source rather than one written in the script.
func (p *parser) atSourceEnd() bool {
	for _, s := range p.segs {
		if s.end == p.pos-1 {
			return true
		}
	}
	return false
}

// next reads one character, or eof.
func (p *parser) next() int {
	if p.pos >= len(p.src) {
		return eof
	}
	c := p.src[p.pos]
	p.pos++
	return int(c)
}

// unread puts back the character just read.
func (p *parser) unread(c int) {
	if c != eof {
		p.pos--
	}
}

// nonblank reads the next character that is not a blank.
func (p *parser) nonblank() int {
	c := p.next()
	for c == ' ' || c == '\t' {
		c = p.next()
	}
	return c
}

func isSpace(c int) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

func isDigit(c int) bool { return c >= '0' && c <= '9' }

// integer reads a decimal number whose first character is c. A character
// that is not a digit reads as zero, as in GNU sed.
func (p *parser) integer(c int) int64 {
	var n int64
	for isDigit(c) {
		if n < 1<<53 {
			n = n*10 + int64(c-'0')
		}
		c = p.next()
	}
	p.unread(c)
	return n
}

func (p *parser) run() {
	for {
		c := p.next()
		for c == ';' || isSpace(c) {
			c = p.next()
		}
		if c == eof {
			break
		}
		cmd := command{num: -1, pos: p.pos}
		if p.address(c, &cmd.a1) {
			if cmd.a1.kind == addrPlus || cmd.a1.kind == addrMultiple {
				p.fail(msgBadFirstAddr)
			}
			c = p.nonblank()
			if c == ',' {
				if !p.address(p.nonblank(), &cmd.a2) {
					p.fail(msgUnexpectedComma)
				}
				c = p.nonblank()
			}
			if cmd.a1.kind == addrLine && cmd.a1.n == 0 && cmd.a2.kind != addrRegex {
				p.fail(msgLineZero)
			}
		}
		if c == '!' {
			cmd.negate = true
			c = p.nonblank()
			if c == '!' {
				p.fail(msgMultipleBang)
			}
		}
		if c == eof || p.atSourceEnd() {
			p.fail(msgMissingCommand)
		}
		p.command(byte(c), &cmd)
	}
	if len(p.blocks) > 0 {
		open := p.prog[p.blocks[len(p.blocks)-1]]
		panic(&scriptError{p.where(open.pos, true) + msgUnmatchedOpen})
	}
	for i := range p.prog {
		cmd := &p.prog[i]
		if cmd.name != 'b' && cmd.name != 't' && cmd.name != 'T' {
			continue
		}
		cmd.jump = len(p.prog)
		if cmd.label != "" {
			target, ok := p.labels[cmd.label]
			if !ok {
				panic(&scriptError{"can't find label for jump to `" + cmd.label + "'"})
			}
			cmd.jump = target
		}
	}
}

// address reads an address whose first character is c. It reports false,
// leaving c for the caller, when c does not start one.
func (p *parser) address(c int, a *address) bool {
	switch {
	case c == '/' || c == '\\':
		if c == '\\' {
			c = p.next()
		}
		pattern, ok := p.delimited(c, true)
		if !ok {
			p.fail(msgUntermAddr)
		}
		icase, multiline := false, false
		for {
			c = p.nonblank()
			if c == 'I' {
				icase = true
				continue
			}
			if c == 'M' {
				multiline = true
				continue
			}
			p.unread(c)
			break
		}
		a.kind = addrRegex
		a.re = p.regex(pattern, icase, multiline)
	case isDigit(c):
		a.kind = addrLine
		a.n = p.integer(c)
		c = p.nonblank()
		if c != '~' {
			p.unread(c)
			break
		}
		// first~0 is the one line first.
		if step := p.integer(p.nonblank()); step > 0 {
			a.kind = addrStep
			a.step = step
		}
	case c == '+' || c == '~':
		a.kind = addrPlus
		if c == '~' {
			a.kind = addrMultiple
		}
		a.n = p.integer(p.nonblank())
	case c == '$':
		a.kind = addrLast
	default:
		return false
	}
	return true
}

// regex compiles a pattern read from the script; the empty pattern stands
// for the last regular expression used when the script runs.
func (p *parser) regex(pattern string, icase, multiline bool) *regexp.Regexp {
	if pattern == "" {
		if icase || multiline {
			p.fail(msgEmptyModifiers)
		}
		return nil
	}
	re, err := compileRegex(pattern, p.extended, icase, multiline)
	if err != nil {
		p.fail(err.Error())
	}
	return re
}

// delimited reads up to the next unescaped delim, as the regular expression
// of an address or an s command when regex is set, and as a replacement or
// a y string otherwise. A backslash before the delimiter is dropped (the
// delimiter is then an ordinary character), as is one before a newline. It
// reports false when the delimiter never comes.
func (p *parser) delimited(delim int, regex bool) (string, bool) {
	if delim == eof || delim == '\n' || delim == '\\' {
		p.unread(delim)
		return "", false
	}
	if delim >= 0x80 {
		p.fail(msgDelimiter)
	}
	var b []byte
	for {
		c := p.next()
		if c == eof {
			return "", false
		}
		if c == '\n' {
			// Left unread so that the diagnostic names this line.
			p.unread(c)
			return "", false
		}
		if c == delim {
			return string(b), true
		}
		if c == '\\' {
			c = p.next()
			switch {
			case c == eof:
				return "", false
			case c == 'n' && regex:
				c = '\n'
			case c != '\n' && (c != delim || (!regex && c == '&')):
				b = append(b, '\\')
			}
		}
		b = append(b, byte(c))
	}
}

// endOfCommand checks that nothing but a separator follows a command.
func (p *parser) endOfCommand() {
	c := p.nonblank()
	switch c {
	case '}', '#':
		p.unread(c)
	case eof, '\n', ';':
	default:
		p.fail(msgExtraChars)
	}
}

// word reads a label: up to white space, a semicolon or a closing brace.
func (p *parser) word() string {
	var b []byte
	c := p.nonblank()
	for c != eof && !isSpace(c) && c != ';' && c != '}' {
		b = append(b, byte(c))
		c = p.next()
	}
	p.unread(c)
	return string(b)
}

// filename reads a file name: the rest of the line.
func (p *parser) filename() string {
	var b []byte
	for c := p.nonblank(); c != eof && c != '\n'; c = p.next() {
		b = append(b, byte(c))
	}
	if len(b) == 0 {
		p.fail(msgMissingFile)
	}
	return string(b)
}

// command reads the command named name, whose addresses are already in cmd.
func (p *parser) command(name byte, cmd *command) {
	cmd.name = name
	addressed := cmd.a1.kind != addrNone
	switch name {
	case '#':
		if addressed {
			p.fail(msgCommentNoAddr)
		}
		// #n on the script's first line, and nothing else on it, is -n.
		if p.pos == 1 && strings.HasPrefix(p.src, "#n\n") {
			p.quiet = true
		}
		for c := p.next(); c != eof && c != '\n'; {
			c = p.next()
		}
		return
	case '{':
		p.blocks = append(p.blocks, len(p.prog))
	case '}':
		if len(p.blocks) == 0 {
			p.fail(msgUnexpectedClose)
		}
		if addressed {
			p.fail(msgCloseNoAddr)
		}
		p.prog[p.blocks[len(p.blocks)-1]].jump = len(p.prog)
		p.blocks = p.blocks[:len(p.blocks)-1]
		p.endOfCommand()
		return
	case '=', 'd', 'D', 'g', 'G', 'h', 'H', 'n', 'N', 'p', 'P', 'x', 'z', 'F':
		p.endOfCommand()
	case 'e':
		p.fail(msgNoExecute)
	case 'a', 'i', 'c':
		cmd.text = p.text()
	case ':':
		if addressed {
			p.fail(msgColonNoAddr)
		}
		label := p.word()
		if label == "" {
			p.fail(msgColonLabel)
		}
		if _, dup := p.labels[label]; !dup {
			p.labels[label] = len(p.prog)
		}
		return
	case 'b', 't', 'T':
		cmd.label = p.word()
	case 'q', 'Q', 'l':
		if name != 'l' {
			if cmd.a2.kind != addrNone {
				p.fail(msgOneAddr)
			}
			cmd.num = 0
		}
		c := p.nonblank()
		if isDigit(c) {
			cmd.num = int(min(p.integer(c), 1<<30))
		} else {
			p.unread(c)
		}
		p.endOfCommand()
	case 'r', 'R', 'w', 'W':
		if p.sandbox {
			p.fail(msgSandbox)
		}
		cmd.file = p.filename()
	case 's':
		p.substitution(cmd)
	case 'y':
		delim := p.next()
		from, ok := p.delimited(delim, false)
		var to string
		if ok {
			to, ok = p.delimited(delim, false)
		}
		if !ok {
			p.fail(msgUntermY)
		}
		if cmd.tr, ok = newTranslit(from, to); !ok {
			p.fail(msgYLength)
		}
		p.endOfCommand()
	case 'v':
		if newerThanGNU(p.word()) {
			p.fail(msgNewerVersion)
		}
		p.endOfCommand()
		return
	default:
		p.failf("unknown command: `%c'", name)
	}
	p.prog = append(p.prog, *cmd)
}

// text reads the text of an a, i or c command, in either form: the GNU one
// ("a text", where the blanks before the text are dropped) or the POSIX one
// ("a\" and the text on the following lines, kept as written). In both, a
// backslash before a newline continues the text on the next line and one
// before any other character is removed; the text ends with a newline.
func (p *parser) text() []byte {
	c := p.nonblank()
	if c == eof || p.atSourceEnd() {
		p.fail(msgExpectedSlash)
	}
	text := []byte{}
	if c == '\\' {
		c = p.next()
		if c == eof || (p.atSourceEnd() && p.pos == len(p.src)) {
			// "a\" at the end of the script: no text at all. '$a\' is
			// the idiom that ends a file's last line with a newline.
			return text
		}
		if c != '\n' {
			// "a\text": the text starts here, leading blanks included.
			p.unread(c)
		}
	} else {
		p.unread(c)
	}
	for {
		c = p.next()
		if c == eof || c == '\n' {
			break
		}
		if c == '\\' {
			c = p.next()
			if c == eof {
				break
			}
			if b, end, ok := charEscape(p.src, p.pos-1); ok {
				text = append(text, b)
				p.pos = end + 1
				continue
			}
		}
		text = append(text, byte(c))
	}
	return append(text, '\n')
}

// substitution reads the rest of an s command: the regular expression, the
// replacement and the flags.
func (p *parser) substitution(cmd *command) {
	delim := p.next()
	pattern, ok := p.delimited(delim, true)
	var rhs string
	if ok {
		rhs, ok = p.delimited(delim, false)
	}
	if !ok {
		p.fail(msgUntermS)
	}
	icase, multiline := false, false
flags:
	for {
		c := p.next()
		switch {
		case c == 'i' || c == 'I':
			icase = true
		case c == 'm' || c == 'M':
			multiline = true
		case c == 'e':
			p.fail(msgNoExecuteFlag)
		case c == 'p':
			if cmd.print {
				p.fail(msgMultipleP)
			}
			cmd.print = true
		case c == 'g':
			if cmd.global {
				p.fail(msgMultipleG)
			}
			cmd.global = true
		case c == 'w':
			if p.sandbox {
				p.fail(msgSandbox)
			}
			cmd.file = p.filename()
			break flags
		case isDigit(c):
			if cmd.nth != 0 {
				p.fail(msgMultipleNumber)
			}
			n := p.integer(c)
			if n == 0 {
				p.fail(msgZeroNumber)
			}
			cmd.nth = int(min(n, 1<<30))
		case c == '}' || c == '#':
			p.unread(c)
			break flags
		case c == eof || c == '\n' || c == ';':
			break flags
		case c == ' ' || c == '\t' || c == '\r':
			// Blanks between and after the flags are harmless.
		default:
			p.fail(msgUnknownSOpt)
		}
	}
	if cmd.nth == 0 {
		cmd.nth = 1
	}
	cmd.re = p.regex(pattern, icase, multiline)
	groups := -1
	if cmd.re != nil {
		groups = cmd.re.NumSubexp()
	}
	var bad int
	if cmd.repl, bad = parseReplacement(rhs, groups); bad >= 0 {
		p.failf("invalid reference \\%d on `s' command's RHS", bad)
	}
}

// gnuVersion is the GNU sed release whose behavior sedx follows, and so the
// newest one a v command may ask for.
var gnuVersion = [3]int{4, 9, 0}

// newerThanGNU reports whether version, the argument of a v command, asks
// for a sed newer than gnuVersion.
func newerThanGNU(version string) bool {
	if version == "" {
		return false
	}
	fields := strings.SplitN(version, ".", 3)
	for i, want := range gnuVersion {
		got := 0
		if i < len(fields) {
			n, err := strconv.Atoi(fields[i])
			if err != nil {
				return true
			}
			got = n
		}
		if got != want {
			return got > want
		}
	}
	return false
}
