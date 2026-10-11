package ls

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// quoteName renders a file name for output according to -b, -Q and -q.
func (l *Ls) quoteName(name string) string {
	switch l.Quoting {
	case QuoteEscape:
		return escapeName(name, false)
	case QuoteC:
		return `"` + escapeName(name, true) + `"`
	}
	if !l.hideCtl {
		return name
	}
	return strings.Map(func(r rune) rune {
		// strings.Map hands every invalid byte over as RuneError, which is
		// exactly the one "?" per byte that ls prints.
		if r == utf8.RuneError || !unicode.IsPrint(r) {
			return '?'
		}
		return r
	}, name)
}

// escapeName applies C-style backslash escapes to nongraphic characters. In
// the quoted (-Q) form a double quote is escaped and a space is not; in the
// bare (-b) form it is the other way around.
func escapeName(name string, quoted bool) string {
	var b strings.Builder
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		switch {
		case r == utf8.RuneError && size <= 1:
			fmt.Fprintf(&b, `\%03o`, name[i])
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"' && quoted:
			b.WriteString(`\"`)
		case r == ' ' && !quoted:
			b.WriteString(`\ `)
		case r == '\a':
			b.WriteString(`\a`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\v':
			b.WriteString(`\v`)
		case unicode.IsPrint(r):
			b.WriteRune(r)
		default:
			for _, c := range []byte(name[i : i+size]) {
				fmt.Fprintf(&b, `\%03o`, c)
			}
		}
		i += size
	}
	return b.String()
}

// displayWidth estimates how many terminal columns s occupies: combining marks
// take none and East Asian wide characters take two. It is a small stand-in for
// wcwidth, enough to keep columns aligned for common scripts.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r < 0x7f:
			w++
		case unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf):
		case isWide(r):
			w += 2
		default:
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) || // Hangul Jamo
		(r >= 0x2e80 && r <= 0xa4cf) || // CJK radicals through Yi
		(r >= 0xac00 && r <= 0xd7a3) || // Hangul syllables
		(r >= 0xf900 && r <= 0xfaff) || // CJK compatibility ideographs
		(r >= 0xfe30 && r <= 0xfe6f) || // CJK compatibility forms
		(r >= 0xff00 && r <= 0xff60) || // fullwidth forms
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1faff) || // emoji and pictographs
		(r >= 0x20000 && r <= 0x3fffd) // supplementary ideographs
}
