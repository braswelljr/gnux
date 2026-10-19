package shell

import (
	"bytes"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// expander expands the words of one command. It carries the streams a command
// substitution runs against and what went wrong along the way.
type expander struct {
	sh *Shell
	s  core.Streams

	// failed is set when an expansion was refused (an unset variable under
	// -u); the diagnostic has been written and the command must not run.
	failed bool

	// substituted is set once a command substitution has run, and status is
	// the exit status of the last one: what `X=$(cmd)` reports.
	substituted bool
	status      int
}

// fields turns a word into its final arguments: variables are substituted, a
// leading ~ becomes the home directory, and an unquoted glob pattern is
// replaced by the files it matches. A variable's value is never re-split or
// re-globbed, so a path with spaces stays one argument without quoting. Only
// two things make more than one argument out of a word: $@, one per
// positional parameter, and an unquoted $(...), one per line of output.
func (ex *expander) fields(w word) []string {
	fields, _ := ex.expand(w, true)
	return fields
}

// single expands a word where exactly one string is wanted (the value of an
// assignment, the word a case tests): nothing is split or globbed.
func (ex *expander) single(w word) string {
	fields, _ := ex.expand(w, false)
	return strings.Join(fields, "")
}

// pattern expands a case pattern: what was quoted, or came from a variable,
// matches only itself.
func (ex *expander) pattern(w word) string {
	_, pattern := ex.expand(w, false)
	return pattern
}

// expand is what fields, single and pattern share. With split false the word
// yields at most one field, and pattern is that field as a glob pattern.
func (ex *expander) expand(w word, split bool) (fields []string, lastPattern string) {
	var text, pattern strings.Builder
	globbing, started := false, false

	emit := func() {
		if !started {
			return
		}
		lastPattern = pattern.String()
		matched := false
		if globbing && split {
			if matches, err := ex.sh.Glob(lastPattern); err == nil && len(matches) > 0 {
				sort.Strings(matches)
				fields = append(fields, matches...)
				matched = true
			}
			// Like a POSIX shell, a pattern that matches nothing is passed through.
		}
		if !matched {
			fields = append(fields, text.String())
		}
		text.Reset()
		pattern.Reset()
		globbing, started = false, false
	}
	// quoted adds text that stands for itself.
	quoted := func(s string) {
		text.WriteString(s)
		pattern.WriteString(globEscape(s))
		started = true
	}
	// several adds values that are each an argument of their own: the first
	// joins what came before it in the word, the last what comes after.
	several := func(items []string) {
		if !split {
			if len(items) > 0 {
				quoted(strings.Join(items, " "))
			}
			return
		}
		for i, item := range items {
			if i > 0 {
				emit()
			}
			quoted(item)
		}
	}

	for i, p := range w {
		if p.subst {
			out := ex.substitute(p.body)
			if p.mode == double || !split {
				quoted(out)
			} else {
				several(lines(out))
			}
			continue
		}

		switch p.mode {
		case literal:
			quoted(p.text)

		case double:
			if p.text == "" {
				started = true
			}
			for _, seg := range ex.vars(p.text) {
				if seg.list {
					several(seg.items)
				} else {
					quoted(seg.text)
				}
			}

		case raw:
			s := p.text
			if i == 0 {
				s = ex.sh.expandTilde(s)
			}
			// Variables are expanded piecewise so that only the text the user
			// typed - not a variable's value - can act as a glob pattern.
			for _, seg := range ex.vars(s) {
				switch {
				case seg.list:
					several(seg.items)
				case seg.fromVar:
					quoted(seg.text)
				default:
					text.WriteString(seg.text)
					pattern.WriteString(seg.text)
					started = true
					if strings.ContainsAny(seg.text, "*?[") {
						globbing = true
					}
				}
			}
		}
	}
	emit()
	return fields, lastPattern
}

// lines splits the output of a command substitution into its non-empty lines.
func lines(out string) []string {
	var items []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSuffix(line, "\r"); line != "" {
			items = append(items, line)
		}
	}
	return items
}

// substitute runs the commands of a $(...) and returns what they wrote to
// standard output, without the trailing newlines. They run as a subshell
// would: variables they set, options they change and a cd they make do not
// outlast them.
func (ex *expander) substitute(body []step) string {
	sh := ex.sh
	var out bytes.Buffer

	saved := sh.save()
	// The working directory belongs to the whole process. It is put back
	// afterwards, unless this is a pipeline stage, where cd changes nothing.
	wd, wdErr := "", os.ErrInvalid
	if sh.subshell == 0 {
		wd, wdErr = fsys.Getwd()
	}
	sh.loops = 0

	sh.runList(body, core.Streams{In: ex.s.In, Out: &out, Err: ex.s.Err})
	ex.substituted, ex.status = true, sh.status

	if wdErr == nil {
		_ = fsys.Chdir(wd)
	}
	sh.restore(saved)
	return strings.TrimRight(out.String(), "\r\n")
}

// segment is a piece of expanded text, tagged with where it came from. A list
// segment ($@) holds several values that each become an argument.
type segment struct {
	text    string
	fromVar bool
	list    bool
	items   []string
}

// vars substitutes the parameters in s: $NAME, ${NAME}, ${NAME:-default}, the
// positional $1 to $9 and ${10}, and $? $# $0 $@ $*.
func (ex *expander) vars(s string) []segment {
	var out []segment
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			out = append(out, segment{text: lit.String()})
			lit.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) {
			lit.WriteByte(s[i])
			continue
		}

		name, fallback, hasFallback, width := "", "", false, 0
		switch next := s[i+1]; {
		case strings.IndexByte("?#@*", next) >= 0, next >= '0' && next <= '9':
			name, width = string(next), 1
		case next == '{':
			if end := strings.IndexByte(s[i+2:], '}'); end >= 0 {
				name, width = s[i+2:i+2+end], end+2
				if at := strings.Index(name, ":-"); at >= 0 {
					name, fallback, hasFallback = name[:at], name[at+2:], true
				}
			}
		case isNameStart(next):
			j := i + 1
			for j < len(s) && isNameChar(s[j]) {
				j++
			}
			name, width = s[i+1:j], j-i-1
		}

		if width == 0 {
			lit.WriteByte(s[i])
			continue
		}
		flush()
		i += width

		if name == "@" {
			out = append(out, segment{list: true, items: ex.sh.args})
			continue
		}
		value, set := ex.sh.lookup(name)
		switch {
		case hasFallback && value == "":
			// The default is text of its own, with its own variables.
			var b strings.Builder
			for _, seg := range ex.vars(fallback) {
				if seg.list {
					b.WriteString(strings.Join(seg.items, " "))
				} else {
					b.WriteString(seg.text)
				}
			}
			value = b.String()
		case !set && ex.sh.NoUnset && !ex.failed:
			ex.sh.errorf(ex.s, "%s: unbound variable", name)
			ex.failed = true
		}
		out = append(out, segment{text: value, fromVar: true})
	}
	flush()
	return out
}

// lookup resolves a parameter: the special ones first, then shell variables,
// then $PWD (the live working directory, which cd changes), then the
// environment. set is false when nothing defines it, which -u refuses; an
// environment variable that is empty counts as unset.
func (sh *Shell) lookup(name string) (value string, set bool) {
	switch name {
	case "?":
		return strconv.Itoa(sh.status), true
	case "#":
		return strconv.Itoa(len(sh.args)), true
	case "*":
		return strings.Join(sh.args, " "), true
	case "0":
		return sh.name, true
	}
	if n, err := strconv.Atoi(name); err == nil && n > 0 {
		if n <= len(sh.args) {
			return sh.args[n-1], true
		}
		return "", false
	}
	if v, ok := sh.vars[name]; ok {
		return v, true
	}
	if name == "PWD" {
		if wd, err := fsys.Getwd(); err == nil {
			return wd, true
		}
	}
	v := sh.Getenv(name)
	return v, v != ""
}

// expandTilde replaces a leading "~" or "~/" with the home directory.
func (sh *Shell) expandTilde(s string) string {
	if s != "~" && !strings.HasPrefix(s, "~/") {
		return s
	}
	home, err := sh.HomeDir()
	if err != nil || home == "" {
		return s
	}
	return home + s[1:]
}

// globEscape quotes the glob metacharacters in s using bracket expressions,
// the one escape filepath.Match honors on every platform.
func globEscape(s string) string {
	if !strings.ContainsAny(s, "*?[") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '*', '?', '[':
			b.WriteByte('[')
			b.WriteByte(s[i])
			b.WriteByte(']')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// match reports whether s matches a case pattern: * is any text, ? any one
// character and [...] one of a set ([a-z] a range, [!...] its complement).
// Unlike a file name pattern, * and ? also match a slash, on every platform.
func match(pattern, s string) bool {
	return matchRunes([]rune(pattern), []rune(s))
}

func matchRunes(p, s []rune) bool {
	// star and mark remember the last * and where it was tried, so that a
	// mismatch backs up by letting the * take one more character.
	pi, si, star, mark := 0, 0, -1, 0
	for si < len(s) {
		if pi < len(p) {
			switch p[pi] {
			case '*':
				star, mark = pi, si
				pi++
				continue
			case '?':
				pi++
				si++
				continue
			case '[':
				if ok, width := matchClass(p[pi:], s[si]); width > 0 {
					if ok {
						pi += width
						si++
						continue
					}
					break
				}
				// No closing bracket: an ordinary character.
				fallthrough
			default:
				if p[pi] == s[si] {
					pi++
					si++
					continue
				}
			}
		}
		if star < 0 {
			return false
		}
		mark++
		pi, si = star+1, mark
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}

// matchClass matches c against the bracket expression that starts p. width is
// how much of p the expression takes, or 0 when it is never closed.
func matchClass(p []rune, c rune) (ok bool, width int) {
	i := 1
	negate := false
	if i < len(p) && (p[i] == '!' || p[i] == '^') {
		negate = true
		i++
	}
	for first := true; i < len(p); first = false {
		if p[i] == ']' && !first {
			return ok != negate, i + 1
		}
		lo, hi := p[i], p[i]
		if i+2 < len(p) && p[i+1] == '-' && p[i+2] != ']' {
			hi = p[i+2]
			i += 2
		}
		if lo <= c && c <= hi {
			ok = true
		}
		i++
	}
	return false, 0
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameChar(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}

// isName reports whether s is a valid variable name.
func isName(s string) bool {
	if s == "" || !isNameStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameChar(s[i]) {
			return false
		}
	}
	return true
}

// assignment splits a word written as NAME=value into the name and the word
// that is its value. The name and the = must be unquoted.
func assignment(w word) (name string, value word, ok bool) {
	if len(w) == 0 || w[0].mode != raw || w[0].subst {
		return "", nil, false
	}
	first := w[0].text
	eq := strings.IndexByte(first, '=')
	if eq <= 0 || !isName(first[:eq]) {
		return "", nil, false
	}
	if rest := first[eq+1:]; rest != "" {
		value = append(value, part{text: rest, mode: raw})
	}
	return first[:eq], append(value, w[1:]...), true
}
