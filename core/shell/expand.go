package shell

import (
	"os"
	"sort"
	"strconv"
	"strings"
)

// expand turns a word into its final arguments: variables are substituted,
// a leading ~ becomes the home directory, and an unquoted glob pattern is
// replaced by the files it matches. A variable's value is never re-split or
// re-globbed, so a path with spaces stays one argument without quoting.
func (sh *Shell) expand(w word) []string {
	var text, pattern strings.Builder
	globbing := false

	for i, p := range w {
		s := p.text
		switch p.mode {
		case raw:
			if i == 0 {
				s = sh.expandTilde(s)
			}
			// Variables are expanded piecewise so that only the text the user
			// typed - not a variable's value - can act as a glob pattern.
			for _, seg := range sh.expandVars(s) {
				text.WriteString(seg.text)
				if seg.fromVar {
					pattern.WriteString(globEscape(seg.text))
					continue
				}
				if strings.ContainsAny(seg.text, "*?[") {
					globbing = true
				}
				pattern.WriteString(seg.text)
			}
			continue
		case double:
			var b strings.Builder
			for _, seg := range sh.expandVars(s) {
				b.WriteString(seg.text)
			}
			s = b.String()
		case literal:
		}
		text.WriteString(s)
		pattern.WriteString(globEscape(s))
	}

	if globbing {
		if matches, err := sh.Glob(pattern.String()); err == nil && len(matches) > 0 {
			sort.Strings(matches)
			return matches
		}
		// Like a POSIX shell, a pattern that matches nothing is passed through.
	}
	return []string{text.String()}
}

// segment is a piece of expanded text, tagged with where it came from.
type segment struct {
	text    string
	fromVar bool
}

// expandVars substitutes $NAME, ${NAME} and $? in s.
func (sh *Shell) expandVars(s string) []segment {
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

		name, width := "", 0
		switch next := s[i+1]; {
		case next == '?':
			name, width = "?", 1
		case next == '{':
			if end := strings.IndexByte(s[i+2:], '}'); end >= 0 {
				name, width = s[i+2:i+2+end], end+2
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
		out = append(out, segment{text: sh.lookup(name), fromVar: true})
		i += width
	}
	flush()
	return out
}

// lookup resolves a variable: $? first, then shell variables, then $PWD (the
// live working directory, which cd changes), then the environment.
func (sh *Shell) lookup(name string) string {
	if name == "?" {
		return strconv.Itoa(sh.status)
	}
	if v, ok := sh.vars[name]; ok {
		return v
	}
	if name == "PWD" {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
	}
	return sh.Getenv(name)
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

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameChar(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}

// assignment splits "NAME=value" when name is a valid variable name.
func assignment(s string) (name, value string, ok bool) {
	eq := strings.IndexByte(s, '=')
	if eq <= 0 || !isNameStart(s[0]) {
		return "", "", false
	}
	for i := 1; i < eq; i++ {
		if !isNameChar(s[i]) {
			return "", "", false
		}
	}
	return s[:eq], s[eq+1:], true
}
