package ptx

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"strings"
)

// Unescape turns GNU ptx's backslash escapes in an option value into the
// characters they stand for: \a \b \f \n \r \t \v, \xHHH and \0OOO (at most
// three digits each), and \c, which drops the rest. Any other escaped
// character keeps its backslash.
func Unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i == len(s) {
			break // a lone backslash at the end is dropped
		}
		switch c = s[i]; c {
		case 'x':
			v, n := 0, 0
			for ; n < 3 && i+1 < len(s) && isHex(s[i+1]); n++ {
				i++
				v = v*16 + hexValue(s[i])
			}
			if n == 0 {
				b.WriteString(`\x`)
			} else {
				b.WriteByte(byte(v))
			}
		case '0':
			v := 0
			for n := 0; n < 3 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7'; n++ {
				i++
				v = v*8 + int(s[i]-'0')
			}
			b.WriteByte(byte(v))
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'c':
			return b.String()
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'v':
			b.WriteByte('\v')
		default:
			b.WriteByte('\\')
			b.WriteByte(c)
		}
	}
	return b.String()
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexValue(c byte) int {
	switch {
	case c >= 'a':
		return int(c-'a') + 10
	case c >= 'A':
		return int(c-'A') + 10
	}
	return int(c - '0')
}

// translateEmacs turns a regular expression in the Emacs syntax GNU ptx
// compiles with into Go's syntax: \( \) \| group and alternate, ( ) | { }
// are literal, * + ? repeat, ^ and $ are anchors only at the start and end
// of a pattern or an alternative, and \w \W \b \B \< \> \` \' keep their
// Emacs meanings as closely as Go allows. A back reference cannot be run.
func translateEmacs(pat string) (string, error) {
	var b strings.Builder
	// atStart reports that a repetition operator here would have nothing to
	// repeat, which in Emacs makes it literal.
	atStart := true
	for i := 0; i < len(pat); i++ {
		c := pat[i]
		switch c {
		case '\\':
			if i+1 == len(pat) {
				return "", errors.New("Trailing backslash") //nolint:revive,staticcheck // GNU regex's message
			}
			i++
			switch d := pat[i]; d {
			case '(':
				b.WriteString("(")
				atStart = true
				continue
			case ')':
				b.WriteString(")")
			case '|':
				b.WriteString("|")
				atStart = true
				continue
			case 'w':
				b.WriteString("[0-9A-Za-z]")
			case 'W':
				b.WriteString("[^0-9A-Za-z]")
			case 'b', '<', '>':
				b.WriteString(`\b`)
			case 'B':
				b.WriteString(`\B`)
			case '`':
				b.WriteString(`\A`)
			case '\'':
				b.WriteString(`\z`)
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				return "", errBackReference
			default:
				b.WriteString(regexp.QuoteMeta(string(d)))
			}
		case '[':
			end, class, err := bracket(pat, i)
			if err != nil {
				return "", err
			}
			b.WriteString(class)
			i = end
		case '^':
			if atStart {
				b.WriteString("^")
				continue
			}
			b.WriteString(`\^`)
		case '$':
			rest := pat[i+1:]
			if rest == "" || strings.HasPrefix(rest, `\)`) || strings.HasPrefix(rest, `\|`) {
				b.WriteString("$")
			} else {
				b.WriteString(`\$`)
			}
		case '*', '+', '?':
			if atStart {
				b.WriteString(regexp.QuoteMeta(string(c)))
			} else {
				b.WriteByte(c)
			}
		case '.':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
		atStart = false
	}
	return b.String(), nil
}

// bracket translates the bracket expression starting at pat[i] and returns
// the index of its closing ']'. A backslash in it is literal, as in POSIX.
func bracket(pat string, i int) (int, string, error) {
	var b strings.Builder
	b.WriteByte('[')
	j := i + 1
	if j < len(pat) && pat[j] == '^' {
		b.WriteByte('^')
		j++
	}
	first := true
	for ; j < len(pat); j++ {
		c := pat[j]
		switch {
		case c == ']' && !first:
			b.WriteByte(']')
			return j, b.String(), nil
		case c == '[' && j+1 < len(pat) && pat[j+1] == ':':
			if end := strings.Index(pat[j+2:], ":]"); end >= 0 {
				b.WriteString(pat[j : j+2+end+2])
				j += 2 + end + 1
				first = false
				continue
			}
			b.WriteString(`\[`)
		case c == '-' && !first && j+1 < len(pat) && pat[j+1] != ']':
			b.WriteByte('-')
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80:
			b.WriteByte(c)
		default:
			b.WriteByte('\\')
			b.WriteByte(c)
		}
		first = false
	}
	return 0, "", errors.New("Unmatched [, [^, [:, [., or [=") //nolint:revive,staticcheck // GNU regex's message
}

// regexMessage words a compile error as GNU regex's messages read.
func regexMessage(err error) string {
	var se *syntax.Error
	if errors.As(err, &se) {
		switch se.Code {
		case syntax.ErrMissingParen, syntax.ErrUnexpectedParen:
			return "Unmatched ( or \\("
		case syntax.ErrMissingBracket:
			return "Unmatched [, [^, [:, [., or [="
		case syntax.ErrInvalidCharRange:
			return "Invalid range end"
		case syntax.ErrInvalidCharClass:
			return "Invalid character class name"
		case syntax.ErrMissingRepeatArgument:
			return "Invalid preceding regular expression"
		}
		return "Invalid regular expression"
	}
	return err.Error()
}
