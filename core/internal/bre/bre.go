// Package bre translates POSIX basic regular expressions (BRE), the syntax
// GNU expr, nl and csplit take, into Go's RE2 syntax. Back references, which
// RE2 cannot express, are refused.
package bre

import (
	"regexp"
	"strings"
)

// Error is a pattern GNU would reject, with GNU's message.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Translate rewrites a POSIX basic regular expression, with the GNU
// extensions \+ \? \|, into Go syntax. The result is not anchored: a
// leading ^ and a trailing $ keep their meaning. On error it returns a
// *Error carrying GNU's message.
func Translate(pattern string) (string, error) {
	var b strings.Builder
	atStart := true // where a * is literal
	depth := 0
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\':
			if i+1 >= len(pattern) {
				return "", &Error{"Trailing backslash"}
			}
			i++
			d := pattern[i]
			switch d {
			case '(':
				b.WriteByte('(')
				depth++
				atStart = true
				continue
			case ')':
				if depth == 0 {
					return "", &Error{"Unmatched ) or \\)"}
				}
				depth--
				b.WriteByte(')')
			case '{':
				b.WriteByte('{')
			case '}':
				b.WriteByte('}')
			case '|':
				b.WriteByte('|')
				atStart = true
				continue
			case '+', '?':
				b.WriteByte(d)
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				return "", &Error{"back references are not supported"}
			case 'n':
				b.WriteString(`\n`)
			case 'w', 'W', 's', 'S', 'b', 'B':
				b.WriteByte('\\')
				b.WriteByte(d)
			default:
				b.WriteString(regexp.QuoteMeta(string(d)))
			}
		case c == '[':
			j := i + 1
			if j < len(pattern) && pattern[j] == '^' {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				if pattern[j] == '[' && j+1 < len(pattern) && strings.IndexByte(":.=", pattern[j+1]) >= 0 {
					end := strings.Index(pattern[j+2:], string(pattern[j+1])+"]")
					if end < 0 {
						return "", &Error{"Unmatched [, [^, [:, [., or [="}
					}
					j += end + 4
					continue
				}
				j++
			}
			if j >= len(pattern) {
				return "", &Error{"Unmatched [, [^, [:, [., or [="}
			}
			class := pattern[i : j+1]
			// Go needs a backslash before a literal ] or \ inside a class.
			body := class[1 : len(class)-1]
			neg := strings.HasPrefix(body, "^")
			body = strings.TrimPrefix(body, "^")
			body = strings.ReplaceAll(body, `\`, `\\`)
			if strings.HasPrefix(body, "]") {
				body = `\]` + body[1:]
			}
			b.WriteByte('[')
			if neg {
				b.WriteByte('^')
			}
			b.WriteString(body)
			b.WriteByte(']')
			i = j
		case c == '*' && atStart:
			b.WriteString(`\*`)
		case c == '*':
			b.WriteByte('*')
		case c == '^' && i == 0:
			// An anchor, after which a * is literal.
			b.WriteByte('^')
			continue
		case c == '$' && i == len(pattern)-1:
			b.WriteByte('$')
		case c == '.':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
		atStart = false
	}
	if depth > 0 {
		return "", &Error{"Unmatched ( or \\("}
	}
	return b.String(), nil
}

// Compile translates pattern and compiles it to match leftmost-longest, as
// POSIX requires. A dot matches a newline too, as in POSIX.
func Compile(pattern string) (*regexp.Regexp, error) {
	return compile("(?s:", pattern)
}

// CompileAnchored is Compile, with the match anchored at the start of the
// text, as expr's ':' operator wants.
func CompileAnchored(pattern string) (*regexp.Regexp, error) {
	return compile("^(?s:", pattern)
}

func compile(prefix, pattern string) (*regexp.Regexp, error) {
	expr, err := Translate(pattern)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(prefix + expr + ")")
	if err != nil {
		return nil, &Error{"Invalid regular expression"}
	}
	re.Longest()
	return re, nil
}
