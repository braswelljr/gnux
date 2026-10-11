package env

import (
	"errors"
	"fmt"
)

// SplitString breaks the argument of -S/--split-string into separate
// arguments, following GNU env's rules rather than a shell's:
//
//   - whitespace separates arguments, except inside single or double quotes;
//   - ${NAME} expands to the variable's value (lookup supplies it) everywhere
//     but inside single quotes, and no other form of '$' is allowed;
//   - a '#' at the start of an argument begins a comment that runs to the end;
//   - the escapes \" \# \$ \' \\ stand for the character itself, \f \n \r \t \v
//     for the control character, \_ for a space inside double quotes and an
//     argument separator outside them, and \c ends the string. Inside single
//     quotes only \\ and \' are escapes.
//
// It exists because a "#!" line passes everything after the interpreter as a
// single argument.
func SplitString(s string, lookup func(name string) (string, bool)) ([]string, error) {
	var (
		args   []string
		cur    []byte
		inArg  bool // an argument has been started (it may still be empty)
		single bool
		double bool
	)
	flush := func() {
		if inArg {
			args = append(args, string(cur))
			cur, inArg = cur[:0], false
		}
	}
	put := func(c byte) {
		cur = append(cur, c)
		inArg = true
	}

scan:
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !double:
			single, inArg = !single, true

		case c == '"' && !single:
			double, inArg = !double, true

		case isSpace(c) && !single && !double:
			flush()

		case c == '#' && !inArg:
			break scan

		case c == '\\' && (!single || next(s, i) == '\\' || next(s, i) == '\''):
			if i+1 == len(s) {
				return nil, errors.New("invalid backslash at end of string in -S")
			}
			i++
			switch e := s[i]; e {
			case '"', '#', '$', '\'', '\\':
				put(e)
			case '_':
				if double {
					put(' ')
				} else {
					flush()
				}
			case 'c':
				if double {
					return nil, errors.New(`'\c' must not appear in double-quoted -S string`)
				}
				break scan
			case 'f':
				put('\f')
			case 'n':
				put('\n')
			case 'r':
				put('\r')
			case 't':
				put('\t')
			case 'v':
				put('\v')
			default:
				return nil, fmt.Errorf(`invalid sequence '\%c' in -S`, e)
			}

		case c == '$' && !single:
			name, ok := varName(s[i:])
			if !ok {
				return nil, fmt.Errorf("only ${VARNAME} expansion is supported, error at: %s", s[i:])
			}
			// An unset variable contributes nothing, not even an empty argument.
			if value, set := lookup(name); set {
				cur = append(cur, value...)
				inArg = true
			}
			i += len(name) + 2 // lands on the closing brace

		default:
			put(c)
		}
	}

	if single || double {
		return nil, errors.New("no terminating quote in -S string")
	}
	flush()
	return args, nil
}

// next returns the byte after position i of s, or 0 at the end.
func next(s string, i int) byte {
	if i+1 < len(s) {
		return s[i+1]
	}
	return 0
}

// isSpace reports whether c separates arguments.
func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// varName extracts NAME from a string beginning with "${NAME}". NAME must be a
// letter or underscore followed by letters, digits and underscores.
func varName(s string) (string, bool) {
	if len(s) < 4 || s[1] != '{' {
		return "", false
	}
	for i := 2; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '}':
			return s[2:i], i > 2
		case c == '_', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9' && i > 2:
		default:
			return "", false
		}
	}
	return "", false
}
