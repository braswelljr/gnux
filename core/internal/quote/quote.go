// Package quote renders file names in gnulib's shell-escape quoting style,
// the style GNU stat (and ls without -N) uses: a name is left alone when a
// shell would read it back unchanged, single-quoted otherwise, with control
// characters written as $'\n' outside the quotes.
package quote

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// safe are the ASCII characters that never need quoting.
const safe = "%+,-./:=@_"

// ShellEscape quotes name for a POSIX shell.
func ShellEscape(name string) string {
	if name == "" {
		return "''"
	}
	if !needsQuotes(name) {
		return name
	}
	var b strings.Builder
	quoted := false
	open := func() {
		if !quoted {
			b.WriteByte('\'')
			quoted = true
		}
	}
	shut := func() {
		if quoted {
			b.WriteByte('\'')
			quoted = false
		}
	}
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		switch {
		case r == '\'':
			shut()
			b.WriteString(`\'`)
		case r == utf8.RuneError && size <= 1, !unicode.IsPrint(r):
			shut()
			b.WriteString("$'")
			for _, c := range []byte(name[i : i+size]) {
				b.WriteString(escape(c))
			}
			b.WriteByte('\'')
		default:
			open()
			b.WriteString(name[i : i+size])
		}
		i += size
	}
	shut()
	return b.String()
}

func needsQuotes(name string) bool {
	if name[0] == '~' || name[0] == '#' {
		return true
	}
	for _, r := range name {
		switch {
		case r < 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
		case r < 0x80 && strings.ContainsRune(safe, r):
		case r >= 0x80 && r != utf8.RuneError && unicode.IsPrint(r):
		default:
			return true
		}
	}
	return false
}

// escape writes one byte as an ANSI-C escape.
func escape(c byte) string {
	switch c {
	case '\a':
		return `\a`
	case '\b':
		return `\b`
	case '\f':
		return `\f`
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\v':
		return `\v`
	case 0x1b:
		return `\E`
	}
	return fmt.Sprintf(`\%03o`, c)
}
