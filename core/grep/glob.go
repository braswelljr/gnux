package grep

import (
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// fileGlob is one compiled --include or --exclude pattern.
type fileGlob struct {
	re      *regexp.Regexp
	include bool
}

// compileGlob turns a wildcard pattern into an anchored regexp with the
// meaning fnmatch(3) gives it without FNM_PATHNAME: '*' and '?' match any
// character, a slash included, '[...]' is a set ('!' or '^' negates it) and a
// backslash takes the next character literally.
func compileGlob(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("(?s)^")
	for i := 0; i < len(glob); {
		c := glob[i]
		switch c {
		case '*':
			b.WriteString(".*")
			i++
		case '?':
			b.WriteByte('.')
			i++
		case '[':
			set, next, ok := globSet(glob, i)
			if !ok {
				b.WriteString(`\[`)
				i++
				continue
			}
			b.WriteString(set)
			i = next
		case '\\':
			if i+1 < len(glob) {
				i++
			}
			fallthrough
		default:
			_, size := utf8.DecodeRuneInString(glob[i:])
			b.WriteString(regexp.QuoteMeta(glob[i : i+size]))
			i += size
		}
	}
	b.WriteByte('$')
	re, err := regexp.Compile(b.String())
	if err != nil {
		// Not expressible (an inverted range, say): match the text itself.
		return regexp.MustCompile("^" + regexp.QuoteMeta(glob) + "$")
	}
	return re
}

// globSet translates the set starting at glob[i]. ok is false when the
// bracket is not closed, which makes it an ordinary character.
func globSet(glob string, i int) (set string, next int, ok bool) {
	j := i + 1
	var b strings.Builder
	b.WriteByte('[')
	if j < len(glob) && (glob[j] == '!' || glob[j] == '^') {
		b.WriteByte('^')
		j++
	}
	for first := true; ; first = false {
		if j >= len(glob) {
			return "", 0, false
		}
		c := glob[j]
		if c == ']' && !first {
			break
		}
		if c == '\\' && j+1 < len(glob) {
			j++
			c = glob[j]
			b.WriteByte('\\')
		} else if strings.IndexByte(`\]^[`, c) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
		j++
	}
	b.WriteByte(']')
	return b.String(), j + 1, true
}

// matches reports whether the glob matches name. A name from the command
// line also matches by any trailing part that follows a slash, as GNU's
// unanchored exclude patterns do; a name met while descending is a base name
// and must match whole.
func (f fileGlob) matches(name string, commandLine bool) bool {
	if f.re.MatchString(name) {
		return true
	}
	if !commandLine {
		return false
	}
	for i := 0; i+1 < len(name); i++ {
		if isSep(name[i]) && !isSep(name[i+1]) && f.re.MatchString(name[i+1:]) {
			return true
		}
	}
	return false
}

// excluded applies GNU's rule to a list of globs: the last one that matches
// decides, and when none does the name is kept unless the first glob is an
// --include (which then reads as "only these").
func excluded(globs []fileGlob, name string, commandLine bool) bool {
	if len(globs) == 0 {
		return false
	}
	for i := len(globs) - 1; i >= 0; i-- {
		if globs[i].matches(name, commandLine) {
			return !globs[i].include
		}
	}
	return globs[0].include
}

func isSep(c byte) bool { return c == '/' || c == os.PathSeparator }

// trimSeps removes redundant trailing separators, keeping a lone root.
func trimSeps(name string) string {
	for len(name) > 1 && isSep(name[len(name)-1]) {
		name = name[:len(name)-1]
	}
	return name
}
