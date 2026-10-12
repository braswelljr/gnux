package printf

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

// errMissingHex is GNU's diagnostic for \x, \u or \U without enough digits.
var errMissingHex = errors.New("missing hexadecimal number in escape")

// escape interprets the backslash escape at s[i] (a '\\'), appends its value
// to buf and returns the index just past it. stop reports a \c, which ends
// all output. octal0 selects the %b form, where an octal escape may be
// written \0NNN as well as \NNN.
func escape(buf *bytes.Buffer, s string, i int, octal0 bool) (next int, stop bool, err error) {
	i++ // past the backslash
	if i >= len(s) {
		buf.WriteByte('\\')
		return i, false, nil
	}

	switch c := s[i]; c {
	case '"':
		buf.WriteByte('"')
	case '\\':
		buf.WriteByte('\\')
	case 'a':
		buf.WriteByte('\a')
	case 'b':
		buf.WriteByte('\b')
	case 'c':
		return i + 1, true, nil
	case 'e':
		buf.WriteByte(0x1b)
	case 'f':
		buf.WriteByte('\f')
	case 'n':
		buf.WriteByte('\n')
	case 'r':
		buf.WriteByte('\r')
	case 't':
		buf.WriteByte('\t')
	case 'v':
		buf.WriteByte('\v')
	case 'x':
		v, n := 0, 0
		for n < 2 && i+1+n < len(s) && hexVal(s[i+1+n]) >= 0 {
			v = v*16 + hexVal(s[i+1+n])
			n++
		}
		if n == 0 {
			return i + 1, false, errMissingHex
		}
		buf.WriteByte(byte(v))
		return i + 1 + n, false, nil
	case 'u', 'U':
		digits := 4
		if c == 'U' {
			digits = 8
		}
		v := 0
		for n := range digits {
			if i+1+n >= len(s) || hexVal(s[i+1+n]) < 0 {
				return i + 1 + n, false, errMissingHex
			}
			v = v*16 + hexVal(s[i+1+n])
		}
		// The C99 rule for universal character names.
		if (v < 0xa0 && v != 0x24 && v != 0x40 && v != 0x60) ||
			(v >= 0xd800 && v <= 0xdfff) || v > utf8.MaxRune {
			return i + 1 + digits, false, fmt.Errorf("invalid universal character name \\%c%0*x", c, digits, v)
		}
		buf.WriteRune(rune(v))
		return i + 1 + digits, false, nil
	default:
		if c >= '0' && c <= '7' {
			j := i
			if octal0 && c == '0' {
				j++
			}
			v := 0
			for n := 0; n < 3 && j < len(s) && s[j] >= '0' && s[j] <= '7'; n++ {
				v = v*8 + int(s[j]-'0')
				j++
			}
			buf.WriteByte(byte(v))
			return j, false, nil
		}
		// An unknown escape is printed as written.
		buf.WriteByte('\\')
		buf.WriteByte(c)
	}
	return i + 1, false, nil
}

// hexVal returns the value of a hexadecimal digit, or -1.
func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
