// Package quotearg quotes strings as gnulib's quotearg module does: the
// quoting styles behind GNU ls --quoting-style, and the quotes GNU tools put
// around names in their messages. Multibyte input is read as UTF-8.
package quotearg

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Style is one of gnulib's quoting styles.
type Style int

const (
	// Literal outputs the string as it is.
	Literal Style = iota
	// Shell quotes for the shell only when needed: 'a b'.
	Shell
	// ShellAlways always quotes for the shell: 'abc'.
	ShellAlways
	// ShellEscape is Shell, with nonprintable characters as $'\n' escapes.
	ShellEscape
	// ShellEscapeAlways is ShellAlways, with $'\n' escapes.
	ShellEscapeAlways
	// C quotes as a C string: "a\nb".
	C
	// CMaybe quotes as a C string only when needed.
	CMaybe
	// Escape is C without the surrounding quotes: a\nb.
	Escape
	// Locale quotes with the locale's quotation marks: 'a' or ‘a’.
	Locale
	// CLocale is Locale, falling back to "a" rather than 'a'.
	CLocale
)

// Names are the styles' names, as GNU's --quoting-style takes them, indexed
// by Style.
var Names = []string{
	"literal", "shell", "shell-always", "shell-escape", "shell-escape-always",
	"c", "c-maybe", "escape", "locale", "clocale",
}

// Options says how to quote.
type Options struct {
	Style Style
	// Also is a set of further characters to escape (or, in the shell
	// styles, to quote for), as gnulib's quote_these_too.
	Also string
	// LeftQuote and RightQuote are the marks of the Locale and CLocale
	// styles. Empty means the C locale's: ' and ', or " and " for CLocale.
	LeftQuote, RightQuote string
}

// UTF8Quotes are the quotation marks GNU uses in a UTF-8 locale.
const (
	UTF8LeftQuote  = "‘"
	UTF8RightQuote = "’"
)

// Quote returns s quoted as o says.
func Quote(s string, o Options) string {
	// The common cases cost nothing: the literal style changes nothing, and
	// a name made only of safe characters needs nothing in the styles that
	// add quotes only when needed.
	switch o.Style {
	case Literal:
		return s
	case Shell, ShellEscape, CMaybe, Escape:
		if s != "" && allSafe(s, o.Also) {
			return s
		}
	}
	q := quoter{opts: o, also: o.Also}
	return q.run(s, o.Style, false)
}

// Needed reports whether Quote changes s.
func Needed(s string, o Options) bool {
	return Quote(s, o) != s
}

type quoter struct {
	opts Options
	also string
}

// run is quotearg_buffer_restyled. elide asks for the outer quotes to be left
// off when nothing needs them; a character that does restarts the run with
// them (force_outer_quoting_style).
func (q *quoter) run(arg string, style Style, elide bool) string {
	var (
		b                 strings.Builder
		quoteString       string
		backslashEscapes  bool
		encounteredSingle bool
		allCompat         = true
		pendingShellEnd   bool
		leftQuote         = q.opts.LeftQuote
		rightQuote        = q.opts.RightQuote
	)
	force := func() string {
		// Restart with the outer quotes, which quote the extra characters too.
		if style == ShellAlways && backslashEscapes {
			style = ShellEscapeAlways
		}
		r := quoter{opts: q.opts}
		return r.run(arg, style, false)
	}

	switch style {
	case CMaybe:
		style = C
		elide = true
		fallthrough
	case C:
		if !elide {
			b.WriteByte('"')
		}
		backslashEscapes = true
		quoteString = `"`
	case Escape:
		backslashEscapes = true
		elide = false
	case Locale, CLocale:
		if leftQuote == "" || rightQuote == "" {
			leftQuote, rightQuote = "'", "'"
			if style == CLocale {
				leftQuote, rightQuote = `"`, `"`
			}
		}
		if !elide {
			b.WriteString(leftQuote)
		}
		backslashEscapes = true
		quoteString = rightQuote
	case ShellEscape:
		backslashEscapes = true
		elide = true
		style = ShellAlways
		quoteString = q.shellStart(&b, elide)
	case Shell:
		elide = true
		style = ShellAlways
		quoteString = q.shellStart(&b, elide)
	case ShellEscapeAlways:
		if !elide {
			backslashEscapes = true
		}
		style = ShellAlways
		quoteString = q.shellStart(&b, elide)
	case ShellAlways:
		quoteString = q.shellStart(&b, elide)
	case Literal:
		elide = false
	}

	escaping := false
	startEsc := func() bool {
		if elide {
			return false
		}
		escaping = true
		if style == ShellAlways && !pendingShellEnd {
			b.WriteString("'$'")
			pendingShellEnd = true
		}
		b.WriteByte('\\')
		return true
	}
	endEsc := func() {
		if pendingShellEnd && !escaping {
			b.WriteString("''")
			pendingShellEnd = false
		}
	}

	n := len(arg)
	for i := 0; i < n; i++ {
		isRightQuote := false
		escaping = false
		compat := false

		if backslashEscapes && style != ShellAlways && quoteString != "" &&
			strings.HasPrefix(arg[i:], quoteString) {
			if elide {
				return force()
			}
			isRightQuote = true
		}

		c := arg[i]
		storeEscape, storeC := false, false
		switch c {
		case 0:
			if backslashEscapes {
				if !startEsc() {
					return force()
				}
				if style != ShellAlways && i+1 < n && arg[i+1] >= '0' && arg[i+1] <= '9' {
					b.WriteString("00")
				}
				c = '0'
			}
		case '?':
			if style == ShellAlways && elide {
				return force()
			}
		case '\a', '\b', '\f', '\n', '\r', '\t', '\v', '\\':
			esc := cEscape(c)
			if c == '\\' {
				if style == ShellAlways {
					if elide {
						return force()
					}
					storeC = true
					break
				}
				if backslashEscapes && elide && quoteString != "" {
					storeC = true
					break
				}
			}
			if (c == '\n' || c == '\r' || c == '\t' || c == '\\') && style == ShellAlways && elide {
				return force()
			}
			if backslashEscapes {
				c = esc
				storeEscape = true
			}
		case '{', '}', '#', '~', ' ', '!', '"', '$', '&', '(', ')', '*', ';', '<', '=', '>', '[', '^', '`', '|':
			special := true
			switch c {
			case '{', '}':
				special = n == 1
				if special && i != 0 {
					special = false
				}
				if special {
					compat = true
				}
			case '#', '~':
				special = i == 0
				if special {
					compat = true
				}
			case ' ':
				compat = true
			}
			if special && style == ShellAlways && elide {
				return force()
			}
		case '\'':
			encounteredSingle = true
			compat = true
			if style == ShellAlways {
				if elide {
					return force()
				}
				// Close the quote, escape the apostrophe; it is stored below.
				b.WriteString(`'\'`)
				pendingShellEnd = false
			}
		default:
			if isSafe(c) {
				compat = true
				break
			}
			// A multibyte sequence, or a byte that is not one.
			r, size := utf8.DecodeRuneInString(arg[i:])
			printable := true
			m := size
			switch {
			case r == utf8.RuneError && size <= 1:
				printable, m = false, 0
			case !unicode.IsPrint(r):
				printable = false
			}
			compat = printable
			if m > 1 || (backslashEscapes && !printable) {
				ilim := i + m
				for {
					if backslashEscapes && !printable {
						if !startEsc() {
							return force()
						}
						b.WriteByte('0' + c>>6)
						b.WriteByte('0' + (c>>3)&7)
						c = '0' + c&7
					} else if isRightQuote {
						b.WriteByte('\\')
						isRightQuote = false
					}
					if ilim <= i+1 {
						break
					}
					endEsc()
					b.WriteByte(c)
					i++
					c = arg[i]
				}
				storeC = true
			}
		}

		if !storeEscape && !storeC {
			inAlso := strings.IndexByte(q.also, c) >= 0
			if ((backslashEscapes && style != ShellAlways) || elide) && inAlso || isRightQuote {
				storeEscape = true
			} else {
				storeC = true
			}
		}
		if storeEscape {
			if !startEsc() {
				return force()
			}
		}
		endEsc()
		b.WriteByte(c)
		if !compat {
			allCompat = false
		}
	}

	if b.Len() == 0 && style == ShellAlways && elide {
		return force()
	}
	// An apostrophe is common in names; C style quotes it more readably than
	// the shell's 'it'\''s'.
	if style == ShellAlways && !elide && encounteredSingle && allCompat {
		r := quoter{opts: q.opts, also: q.also}
		return r.run(arg, C, false)
	}
	if quoteString != "" && !elide {
		b.WriteString(quoteString)
	}
	return b.String()
}

// shellStart opens a shell-quoted string and returns its closing quote.
func (q *quoter) shellStart(b *strings.Builder, elide bool) string {
	if !elide {
		b.WriteByte('\'')
	}
	return "'"
}

// cEscape is the letter C writes after a backslash for the control
// character c, or c itself for a backslash.
func cEscape(c byte) byte {
	switch c {
	case '\a':
		return 'a'
	case '\b':
		return 'b'
	case '\f':
		return 'f'
	case '\n':
		return 'n'
	case '\r':
		return 'r'
	case '\t':
		return 't'
	case '\v':
		return 'v'
	}
	return c
}

// allSafe reports whether every byte of s is safe and not in also.
func allSafe(s, also string) bool {
	for i := 0; i < len(s); i++ {
		if !isSafe(s[i]) || strings.IndexByte(also, s[i]) >= 0 {
			return false
		}
	}
	return true
}

// isSafe reports whether c needs no quoting in any style.
func isSafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("%+,-./:]_", c) >= 0
}

// ParseStyle reads a style name as gnulib's argmatch does: the name, or an
// abbreviation of only one. ok is false for anything else.
func ParseStyle(word string) (style Style, ok bool) {
	found := -1
	for i, name := range Names {
		if name == word {
			return Style(i), true
		}
		if word != "" && strings.HasPrefix(name, word) {
			if found >= 0 {
				return 0, false
			}
			found = i
		}
	}
	return Style(found), found >= 0
}
