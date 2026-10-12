package printf

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The numeric arguments are read the way GNU printf reads them, with C's
// strtoimax, strtoumax and strtold: leading white space and a sign are
// allowed, 0x introduces hexadecimal and a leading 0 octal (integers only),
// and an argument that starts with a quote is the character code of what
// follows it. What could not be converted is diagnosed; the value converted
// so far is still printed.

// signed reads a %d or %i argument. ok is false when it was diagnosed.
func (p *Printf) signed(s string) (int64, bool) {
	if c, ok := p.charConst(s); ok {
		return c, true
	}
	mag, neg, end, overflow := scanInt(s)
	var v int64
	switch {
	case !neg && (overflow || mag > math.MaxInt64):
		v, overflow = math.MaxInt64, true
	case neg && (overflow || mag > 1<<63):
		v, overflow = math.MinInt64, true
	case neg && mag == 1<<63:
		v = math.MinInt64
	case neg:
		v = -int64(mag)
	default:
		v = int64(mag)
	}
	return v, p.verify(s, end, overflow)
}

// unsigned reads a %o, %u, %x or %X argument. As with strtoumax, a negative
// value wraps around.
func (p *Printf) unsigned(s string) (uint64, bool) {
	if c, ok := p.charConst(s); ok {
		return uint64(c), true
	}
	mag, neg, end, overflow := scanInt(s)
	if overflow {
		return math.MaxUint64, p.verify(s, end, true)
	}
	if neg {
		mag = -mag
	}
	return mag, p.verify(s, end, false)
}

// float reads a floating-point argument.
func (p *Printf) float(s string) (float64, bool) {
	if c, ok := p.charConst(s); ok {
		return float64(c), true
	}
	v, end, overflow := ParseFloatPrefix(s)
	return v, p.verify(s, end, overflow)
}

// charConst handles an argument written as a quote followed by a character,
// whose value is that character's code point. Anything after the character
// is ignored with a warning.
func (p *Printf) charConst(s string) (int64, bool) {
	if len(s) < 2 || (s[0] != '\'' && s[0] != '"') {
		return 0, false
	}
	r, n := utf8.DecodeRuneInString(s[1:])
	if r == utf8.RuneError && n == 1 {
		r = rune(s[1]) // an invalid byte stands for itself
	}
	if rest := s[1+n:]; rest != "" {
		p.diag.Warnf("warning: %s: character(s) following character constant have been ignored", rest)
	}
	return int64(r), true
}

// verify diagnoses an argument that was out of range, not a number, or not
// completely converted, and reports whether it was clean.
func (p *Printf) verify(s string, end int, overflow bool) bool {
	switch {
	case overflow:
		p.diag.Errorf("%s: Numerical result out of range", quote(s))
	case end < len(s) && end == 0:
		p.diag.Errorf("%s: expected a numeric value", quote(s))
	case end < len(s):
		p.diag.Errorf("%s: value not completely converted", quote(s))
	default:
		return true
	}
	return false
}

// scanInt reads the integer at the start of s as strtoumax does with base 0,
// returning its magnitude, sign, the index where conversion stopped (0 when
// nothing was converted) and whether the magnitude overflowed 64 bits.
func scanInt(s string) (mag uint64, neg bool, end int, overflow bool) {
	i := skipSpace(s)
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	base := uint64(10)
	switch {
	case len(s)-i > 2 && s[i] == '0' && (s[i+1] == 'x' || s[i+1] == 'X') && hexVal(s[i+2]) >= 0:
		base = 16
		i += 2
	case i < len(s) && s[i] == '0':
		base = 8
	}
	start := i
	for i < len(s) {
		d := hexVal(s[i])
		if d < 0 || uint64(d) >= base {
			break
		}
		if mag > (math.MaxUint64-uint64(d))/base {
			overflow = true
		} else {
			mag = mag*base + uint64(d)
		}
		i++
	}
	if i == start {
		return 0, false, 0, false
	}
	return mag, neg, i, overflow
}

// ParseFloatPrefix reads the floating-point number at the start of s as C's
// strtod does: leading white space, a sign, then a decimal or hexadecimal
// number (the binary exponent optional), "inf", "infinity" or "nan". It
// returns the value, the index where conversion stopped (0 when nothing was
// converted) and whether the value was out of range.
func ParseFloatPrefix(s string) (v float64, end int, overflow bool) {
	i := skipSpace(s)
	start := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	body := i
	lower := strings.ToLower(s[i:])

	switch {
	case strings.HasPrefix(lower, "infinity"):
		i += len("infinity")
	case strings.HasPrefix(lower, "inf"):
		i += len("inf")
	case strings.HasPrefix(lower, "nan"):
		i += len("nan")
		if j := strings.IndexByte(s[i:], ')'); i < len(s) && s[i] == '(' && j > 0 {
			i += j + 1
		}
		v = math.NaN()
		if s[start] == '-' {
			v = math.Copysign(v, -1)
		}
		return v, i, false
	case strings.HasPrefix(lower, "0x") && (hexDigits(s[i+2:]) > 0 || (strings.HasPrefix(s[i+2:], ".") && hexDigits(s[i+3:]) > 0)):
		i += 2
		i += hexDigits(s[i:])
		if i < len(s) && s[i] == '.' {
			i++
			i += hexDigits(s[i:])
		}
		mant := i
		i = exponent(s, i, 'p')
		text := s[start:i]
		if i == mant {
			text += "p0"
		}
		v, err := strconv.ParseFloat(text, 64)
		return v, i, err != nil && math.IsInf(v, 0)
	default:
		n := decDigits(s[i:])
		i += n
		frac := 0
		if i < len(s) && s[i] == '.' {
			frac = decDigits(s[i+1:])
			if n == 0 && frac == 0 {
				return 0, 0, false
			}
			i += 1 + frac
		}
		if n == 0 && frac == 0 {
			return 0, 0, false
		}
		i = exponent(s, i, 'e')
	}
	if i == body {
		return 0, 0, false
	}
	v, err := strconv.ParseFloat(s[start:i], 64)
	// Underflow to zero or a denormal is not an error, as with strtod.
	return v, i, err != nil && math.IsInf(v, 0)
}

// exponent consumes an exponent introduced by marker (either case) at s[i],
// if one with at least one digit is there.
func exponent(s string, i int, marker byte) int {
	if i >= len(s) || (s[i]|0x20) != marker {
		return i
	}
	j := i + 1
	if j < len(s) && (s[j] == '+' || s[j] == '-') {
		j++
	}
	if n := decDigits(s[j:]); n > 0 {
		return j + n
	}
	return i
}

func decDigits(s string) int {
	n := 0
	for n < len(s) && isDigit(s[n]) {
		n++
	}
	return n
}

func hexDigits(s string) int {
	n := 0
	for n < len(s) && hexVal(s[n]) >= 0 {
		n++
	}
	return n
}

// skipSpace returns the index of the first byte of s that is not C white space.
func skipSpace(s string) int {
	i := 0
	for i < len(s) && strings.IndexByte(" \t\n\v\f\r", s[i]) >= 0 {
		i++
	}
	return i
}

// ParseRat reads a whole operand as an exact rational number, accepting what
// ParseFloatPrefix accepts except infinities and NaN. ok is false when s is
// not, in its entirety, a finite number.
func ParseRat(s string) (r *big.Rat, ok bool) {
	_, end, _ := ParseFloatPrefix(s)
	if end == 0 || end != len(s) {
		return nil, false
	}
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	t = strings.TrimPrefix(t, "+")
	lower := strings.ToLower(strings.TrimPrefix(t, "-"))
	if strings.HasPrefix(lower, "inf") || strings.HasPrefix(lower, "nan") {
		return nil, false
	}
	if strings.HasPrefix(lower, "0x") && !strings.ContainsAny(lower, "p") {
		t += "p0"
	}
	r, ok = new(big.Rat).SetString(t)
	return r, ok
}
