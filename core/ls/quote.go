package ls

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// quoteName renders a file name for output in the quoting style in effect,
// and reports whether quoting changed it (which decides the alignment pad).
// With -q, nonprintable characters left as they are then become "?", as GNU
// does for the literal and shell styles.
func (l *Ls) quoteName(name string) (string, bool) {
	return l.quote(name, l.nameQuote)
}

// quoteDir renders a directory header, which also escapes a ':'.
func (l *Ls) quoteDir(name string) string {
	shown, _ := l.quote(name, l.dirQuote)
	return shown
}

func (l *Ls) quote(name string, o quotearg.Options) (string, bool) {
	shown := quotearg.Quote(name, o)
	quoted := shown != name
	if l.hideCtl {
		switch o.Style {
		case quotearg.Literal, quotearg.Shell, quotearg.ShellAlways:
			shown = questionMarks(shown)
		}
	}
	return shown, quoted
}

// questionMarks replaces each nonprintable character with "?".
func questionMarks(s string) string {
	return strings.Map(func(r rune) rune {
		// strings.Map hands every invalid byte over as RuneError, which is
		// exactly the one "?" per byte that ls prints.
		if r == utf8.RuneError || !unicode.IsPrint(r) {
			return '?'
		}
		return r
	}, s)
}

// displayWidth estimates how many terminal columns s occupies: combining marks
// take none and East Asian wide characters take two. It is a small stand-in for
// wcwidth, enough to keep columns aligned for common scripts.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r < 0x20 || r == 0x7f:
			// A control character left in a name takes no column.
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
