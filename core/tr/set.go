package tr

import (
	"fmt"
	"strconv"
)

// setError is a problem with a SET operand. GNU reports these as plain
// diagnostics (exit status 1), not as usage errors.
type setError struct{ msg string }

func (e *setError) Error() string { return e.msg }

func setErrorf(format string, a ...any) error {
	return &setError{msg: fmt.Sprintf(format, a...)}
}

// symbol is one byte of a SET after backslash escapes are decoded. Escaped
// bytes never start or close a bracket construct or a range.
type symbol struct {
	b       byte
	escaped bool
}

// kind is the type of a SET element.
type kind int

const (
	kindChar   kind = iota // a single byte
	kindRange              // lo-hi
	kindClass              // [:name:]
	kindEquiv              // [=c=]
	kindRepeat             // [c*n] or [c*]
)

// element is one construct of a parsed SET.
type element struct {
	kind   kind
	lo, hi byte   // kindChar uses lo; kindRange both; kindEquiv and kindRepeat lo
	class  string // kindClass
	count  int    // kindRepeat with an explicit count
	fill   bool   // kindRepeat written [c*] or [c*0]: fill to the length of SET1
}

// classes lists the POSIX character classes in the C locale, which is the
// byte-oriented locale GNU tr effectively works in.
var classes = map[string]func(b byte) bool{
	"alnum":  func(b byte) bool { return isAlpha(b) || isDigit(b) },
	"alpha":  isAlpha,
	"blank":  func(b byte) bool { return b == ' ' || b == '\t' },
	"cntrl":  func(b byte) bool { return b < 32 || b == 127 },
	"digit":  isDigit,
	"graph":  func(b byte) bool { return b > 32 && b < 127 },
	"lower":  isLower,
	"print":  func(b byte) bool { return b >= 32 && b < 127 },
	"punct":  func(b byte) bool { return b > 32 && b < 127 && !isAlpha(b) && !isDigit(b) },
	"space":  func(b byte) bool { return b == ' ' || (b >= '\t' && b <= '\r') },
	"upper":  isUpper,
	"xdigit": func(b byte) bool { return isDigit(b) || (b|0x20 >= 'a' && b|0x20 <= 'f') },
}

func isLower(b byte) bool { return b >= 'a' && b <= 'z' }
func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
func isAlpha(b byte) bool { return isLower(b) || isUpper(b) }
func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// members returns the bytes of a class in ascending order.
func members(class string) []byte {
	in := classes[class]
	var out []byte
	for c := range 256 {
		if in(byte(c)) {
			out = append(out, byte(c))
		}
	}
	return out
}

// unescape decodes the backslash escapes of a SET. warn receives GNU's
// portability warnings.
func unescape(s string, warn func(string)) []symbol {
	var out []symbol
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			out = append(out, symbol{b: c})
			continue
		}
		if i+1 == len(s) {
			warn("warning: an unescaped backslash at end of string is not portable")
			out = append(out, symbol{b: '\\'})
			continue
		}
		i++
		c = s[i]
		switch c {
		case 'a':
			c = '\a'
		case 'b':
			c = '\b'
		case 'f':
			c = '\f'
		case 'n':
			c = '\n'
		case 'r':
			c = '\r'
		case 't':
			c = '\t'
		case 'v':
			c = '\v'
		case '0', '1', '2', '3', '4', '5', '6', '7':
			j := i
			for j < len(s) && j < i+3 && s[j] >= '0' && s[j] <= '7' {
				j++
			}
			v, _ := strconv.ParseUint(s[i:j], 8, 16)
			if v > 0377 {
				// \400 and above do not fit a byte: GNU reads two digits
				// and leaves the third as a literal.
				warn(fmt.Sprintf("warning: the ambiguous octal escape \\%s is being\n"+
					"interpreted as the 2-byte sequence \\0%s, %c", s[i:j], s[i:j-1], s[j-1]))
				j--
				v, _ = strconv.ParseUint(s[i:j], 8, 16)
			}
			c = byte(v)
			i = j - 1
		}
		out = append(out, symbol{b: c, escaped: true})
	}
	return out
}

// text renders symbols back as plain text, for diagnostics.
func text(syms []symbol) string {
	b := make([]byte, len(syms))
	for i, s := range syms {
		b[i] = s.b
	}
	return string(b)
}

// is reports whether syms[i] is the unescaped byte c.
func is(syms []symbol, i int, c byte) bool {
	return i < len(syms) && !syms[i].escaped && syms[i].b == c
}

// closing finds the unescaped two-byte terminator a, ']' at or after from.
func closing(syms []symbol, from int, a byte) int {
	for j := from; j+1 < len(syms); j++ {
		if is(syms, j, a) && is(syms, j+1, ']') {
			return j
		}
	}
	return -1
}

// parse splits a SET into its elements.
func parse(set string, warn func(string)) ([]element, error) {
	syms := unescape(set, warn)
	var out []element
	for i := 0; i < len(syms); {
		if is(syms, i, '[') {
			switch {
			case is(syms, i+1, ':'):
				if j := closing(syms, i+2, ':'); j >= 0 {
					name := text(syms[i+2 : j])
					if _, ok := classes[name]; !ok {
						return nil, setErrorf("invalid character class '%s'", name)
					}
					out = append(out, element{kind: kindClass, class: name})
					i = j + 2
					continue
				}
			case is(syms, i+1, '='):
				if j := closing(syms, i+2, '='); j >= 0 {
					if j-(i+2) != 1 {
						return nil, setErrorf("%s: equivalence class operand must be a single character",
							text(syms[i+2:j]))
					}
					out = append(out, element{kind: kindEquiv, lo: syms[i+2].b})
					i = j + 2
					continue
				}
			case i+2 < len(syms) && is(syms, i+2, '*'):
				if e, n, ok, err := repeat(syms[i:]); ok || err != nil {
					if err != nil {
						return nil, err
					}
					out = append(out, e)
					i += n
					continue
				}
			}
		}
		if i+2 < len(syms) && is(syms, i+1, '-') {
			lo, hi := syms[i].b, syms[i+2].b
			if lo > hi {
				return nil, setErrorf("range-endpoints of '%s' are in reverse collating sequence order",
					text(syms[i:i+3]))
			}
			out = append(out, element{kind: kindRange, lo: lo, hi: hi})
			i += 3
			continue
		}
		out = append(out, element{kind: kindChar, lo: syms[i].b})
		i++
	}
	return out, nil
}

// repeat parses [c*n] or [c*] at the start of syms. ok is false when the
// bracket is not closed, in which case the '[' is an ordinary byte.
func repeat(syms []symbol) (e element, n int, ok bool, err error) {
	end := -1
	for j := 3; j < len(syms); j++ {
		if is(syms, j, ']') {
			end = j
			break
		}
	}
	if end < 0 {
		return element{}, 0, false, nil
	}
	e = element{kind: kindRepeat, lo: syms[1].b}
	digits := text(syms[3:end])
	if digits == "" {
		e.fill = true
	} else {
		base := 10
		if digits[0] == '0' {
			base = 8
		}
		v, perr := strconv.ParseUint(digits, base, 31)
		if perr != nil {
			return element{}, 0, false, setErrorf("invalid repeat count '%s' in [c*n] construct", digits)
		}
		e.count = int(v)
		e.fill = v == 0
	}
	return e, end + 1, true, nil
}

// expanded is a SET written out byte by byte. caseAt records where an
// [:upper:] or [:lower:] class starts, so that translations between the two
// can be checked for alignment.
type expanded struct {
	bytes  []byte
	caseAt map[int]string
}

// expand writes elements out. fill is the length a [c*] repeat takes.
func expand(elems []element, fill int) expanded {
	x := expanded{caseAt: map[int]string{}}
	for _, e := range elems {
		switch e.kind {
		case kindChar, kindEquiv:
			x.bytes = append(x.bytes, e.lo)
		case kindRange:
			for c := int(e.lo); c <= int(e.hi); c++ {
				x.bytes = append(x.bytes, byte(c))
			}
		case kindClass:
			if e.class == "upper" || e.class == "lower" {
				x.caseAt[len(x.bytes)] = e.class
			}
			x.bytes = append(x.bytes, members(e.class)...)
		case kindRepeat:
			n := e.count
			if e.fill {
				n = fill
			}
			for range n {
				x.bytes = append(x.bytes, e.lo)
			}
		}
	}
	return x
}
