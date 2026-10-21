package numfmt

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// GNU numfmt computes in long double. These are the limits of the x86-64
// long double (a 64-bit mantissa), which the arithmetic below reproduces:
// more digits than unscaledDigits lose precision, and a value of more than
// acceptableDigits digits cannot be handled at all.
const (
	mantissa         = 64
	unscaledDigits   = 18
	acceptableDigits = 33
)

func newFloat() *big.Float { return new(big.Float).SetPrec(mantissa) }

func fromInt(i int64) *big.Float   { return newFloat().SetInt64(i) }
func fromUint(u uint64) *big.Float { return newFloat().SetUint64(u) }
func mul(a, b *big.Float) *big.Float {
	return newFloat().Mul(a, b)
}
func quo(a, b *big.Float) *big.Float { return newFloat().Quo(a, b) }

// power is base**x by repeated multiplication, rounding at each step as
// GNU's powerld does.
func power(base int64, x int) *big.Float {
	r := fromInt(1)
	b := fromInt(base)
	for range x {
		r = mul(r, b)
	}
	return r
}

func abs(v *big.Float) *big.Float { return newFloat().Abs(v) }

func less(a *big.Float, b int64) bool { return a.Cmp(fromInt(b)) < 0 }

// expld divides v by base until it is below base, and returns the result and
// how many times it divided.
func expld(v *big.Float, base int64) (*big.Float, int) {
	p := 0
	if v.IsInf() {
		return v, 0
	}
	for !less(abs(v), base) {
		p++
		v = quo(v, fromInt(base))
	}
	return v, p
}

// trunc is a C conversion of a long double to an integer: toward zero.
func trunc(v *big.Float) *big.Float {
	i, _ := v.Int(nil)
	return newFloat().SetInt(i)
}

func ceiling(v *big.Float) *big.Float {
	i := trunc(v)
	if i.Cmp(v) < 0 {
		i = newFloat().Add(i, fromInt(1))
	}
	return i
}

func floor(v *big.Float) *big.Float { return newFloat().Neg(ceiling(newFloat().Neg(v))) }

// round rounds v to an integer, the way GNU's simple_round does.
func round(v *big.Float, r Round) *big.Float {
	maxInt := fromInt(math.MaxInt64)
	m := trunc(quo(v, maxInt))
	v = newFloat().Sub(v, mul(maxInt, m))
	var rv *big.Float
	switch r {
	case RoundUp:
		rv = ceiling(v)
	case RoundDown:
		rv = floor(v)
	case RoundTowardsZero:
		rv = trunc(v)
	case RoundNearest:
		half := newFloat().SetFloat64(0.5)
		if v.Sign() < 0 {
			rv = trunc(newFloat().Sub(v, half))
		} else {
			rv = trunc(newFloat().Add(v, half))
		}
	default:
		if v.Sign() < 0 {
			rv = floor(v)
		} else {
			rv = ceiling(v)
		}
	}
	return newFloat().Add(mul(maxInt, m), rv)
}

// parseError is why a number could not be read.
type parseError int

const (
	parseOK parseError = iota
	parsePrecisionLoss
	parseOverflow
	parseInvalidNumber
	parseForbiddenSuffix
	parseInvalidSuffix
	parseMissingI
)

// parseInt reads an optional '-' and a run of digits from s, and returns the
// value, whether it was negative, and how much of s it used.
func parseInt(s string) (val *big.Float, negative bool, used int, e parseError) {
	val = fromInt(0)
	i := 0
	if strings.HasPrefix(s, "-") {
		negative = true
		i++
	}
	digits := 0
	found := false
	ten := fromInt(10)
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		d := int64(s[i] - '0')
		found = true
		if val.Sign() != 0 || d != 0 {
			digits++
		}
		if digits > unscaledDigits {
			e = parsePrecisionLoss
		}
		if digits > acceptableDigits {
			return nil, negative, i, parseOverflow
		}
		val = newFloat().Add(mul(val, ten), fromInt(d))
	}
	if !found && !strings.HasPrefix(s[i:], ".") {
		return nil, negative, i, parseInvalidNumber
	}
	if negative {
		val = newFloat().Neg(val)
	}
	return val, negative, i, e
}

// parseFloat reads "N[.N]" and returns the value, the number of digits after
// the point, and how much of s it used.
func parseFloat(s string) (val *big.Float, precision, used int, e parseError) {
	val, negative, used, e := parseInt(s)
	if e != parseOK && e != parsePrecisionLoss {
		return nil, 0, used, e
	}
	if !strings.HasPrefix(s[used:], ".") {
		return val, 0, used, e
	}
	used++
	frac, negFrac, n, e2 := parseInt(s[used:])
	if e2 != parseOK && e2 != parsePrecisionLoss {
		return nil, 0, used + n, e2
	}
	if e2 == parsePrecisionLoss {
		e = e2
	}
	if negFrac {
		return nil, 0, used + n, parseInvalidNumber
	}
	frac = quo(frac, power(10, n))
	if negative {
		val = newFloat().Sub(val, frac)
	} else {
		val = newFloat().Add(val, frac)
	}
	return val, n, used + n, e
}

func scaleBase(s Scale) int64 {
	if s == ScaleIEC || s == ScaleIECI {
		return 1024
	}
	return 1000
}

const validSuffixes = "KkMGTPEZYRQ"

func suffixPower(c byte) int {
	if c == 'k' {
		return 1
	}
	return strings.IndexByte("KMGTPEZYRQ", c) + 1
}

// parseHumanNumber reads a number with an optional unit suffix, as --from
// allows, and returns its value, the precision of its input, and what of s
// is left over.
func (n *Numfmt) parseHumanNumber(s string) (val *big.Float, precision int, rest string, e parseError) {
	base := scaleBase(n.From)
	val, precision, used, e := parseFloat(s)
	if e != parseOK && e != parsePrecisionLoss {
		return nil, 0, "", e
	}
	rest = s[used:]
	p := 0
	if rest != "" {
		rest = strings.TrimLeft(rest, " \t")
		// Blanks with nothing after them pass GNU's suffix check, since
		// its test for a valid suffix character finds the string's end.
		if rest != "" && !strings.Contains(validSuffixes, rest[:1]) {
			return nil, 0, "", parseInvalidSuffix
		}
		if n.From == ScaleNone {
			return nil, 0, "", parseForbiddenSuffix
		}
		if rest == "" {
			return val, 0, "", e
		}
		p = suffixPower(rest[0])
		rest = rest[1:]
		switch {
		case n.From == ScaleAuto && strings.HasPrefix(rest, "i"):
			base = 1024
			rest = rest[1:]
		case n.From == ScaleIECI:
			if !strings.HasPrefix(rest, "i") {
				return nil, 0, "", parseMissingI
			}
			rest = rest[1:]
		}
		precision = 0
	}
	return mul(val, power(base, p)), precision, rest, e
}

// parseHuman is parse_human_number: it reports an invalid number as
// --invalid says.
func (n *Numfmt) parseHuman(s string) (*big.Float, int, bool) {
	val, precision, rest, e := n.parseHumanNumber(s)
	switch e {
	case parseOK, parsePrecisionLoss:
	case parseOverflow:
		n.invalid("value too large to be converted: %s", quote(s))
	case parseInvalidNumber:
		n.invalid("invalid number: %s", quote(s))
	case parseForbiddenSuffix:
		n.invalid("rejecting suffix in input: %s (consider using --from)", quote(s))
	case parseInvalidSuffix:
		n.invalid("invalid suffix in input: %s", quote(s))
	case parseMissingI:
		n.invalid("missing 'i' suffix in input: %s (e.g Ki/Mi/Gi)", quote(s))
	}
	if e != parseOK && e != parsePrecisionLoss {
		return nil, 0, false
	}
	if rest != "" {
		n.invalid("invalid suffix in input %s: %s", quote(s), quote(rest))
		return nil, 0, false
	}
	if e == parsePrecisionLoss && n.Debug {
		n.diag.Warnf("large input value %s: possible precision loss", quote(s))
	}
	return val, precision, true
}

// format turns a value into its output text, or reports why it cannot.
func (n *Numfmt) format(val *big.Float, precision int) (string, bool) {
	if n.precision != -1 {
		precision = n.precision
	}
	_, x := expld(val, 10)
	if n.To == ScaleNone && x+precision > unscaledDigits {
		if precision != 0 {
			n.invalid("value/precision too large to be printed: '%s/%d' (consider using --to)", cG(val), precision)
		} else {
			n.invalid("value too large to be printed: '%s' (consider using --to)", cG(val))
		}
		return "", false
	}
	if x > acceptableDigits-1 {
		n.invalid("value too large to be printed: '%s' (cannot handle values > 999Q)", cG(val))
		return "", false
	}
	return n.human(val, precision), true
}

// human is double_to_human: the value as printf("%.*Lf") prints it with no
// scaling, or scaled down to a unit suffix.
func (n *Numfmt) human(val *big.Float, precision int) string {
	if n.To == ScaleNone {
		p := power(10, precision)
		val = quo(round(mul(val, p), n.Round), p)
		return n.pad(val.Text('f', precision)) + n.Suffix
	}

	base := scaleBase(n.To)
	val, pow := expld(val, base)
	adjust := 0
	switch {
	case n.precision != -1:
		adjust = min(pow*3, n.precision)
	case less(abs(val), 10):
		adjust = 1
	}
	p := power(10, adjust)
	val = quo(round(mul(val, p), n.Round), p)
	if !less(abs(val), base) {
		val = quo(val, fromInt(base))
		pow++
	}
	prec := 0
	switch {
	case n.precision != -1:
		prec = n.precision
	case val.Sign() != 0 && less(abs(val), 10) && pow > 0:
		prec = 1
	}
	unit := ""
	if pow > 0 {
		unit = string("KMGTPEZYRQ"[pow-1])
		if pow == 1 && n.To == ScaleSI {
			unit = "k"
		}
		if n.To == ScaleIECI {
			unit += "i"
		}
	}
	return n.pad(val.Text('f', prec)) + unit + n.Suffix
}

// pad applies the zero padding of a --format such as %08f.
func (n *Numfmt) pad(num string) string {
	if len(num) >= n.zeroPad {
		return num
	}
	sign := ""
	if strings.HasPrefix(num, "-") {
		sign, num = "-", num[1:]
	}
	return sign + strings.Repeat("0", n.zeroPad-len(sign)-len(num)) + num
}

// cG formats v as C's %Lg does.
func cG(v *big.Float) string {
	f, _ := v.Float64()
	if f == 0 {
		return "0"
	}
	exp := int(math.Floor(math.Log10(math.Abs(f))))
	// The exponent after rounding to six significant digits.
	if r, err := strconv.ParseFloat(strconv.FormatFloat(f, 'e', 5, 64), 64); err == nil && r != 0 {
		exp = int(math.Floor(math.Log10(math.Abs(r))))
	}
	if exp < -4 || exp >= 6 {
		s := strconv.FormatFloat(f, 'e', 5, 64)
		mant, e, _ := strings.Cut(s, "e")
		if strings.Contains(mant, ".") {
			mant = strings.TrimRight(strings.TrimRight(mant, "0"), ".")
		}
		sign := e[0]
		digits := strings.TrimLeft(e[1:], "0")
		for len(digits) < 2 {
			digits = "0" + digits
		}
		return mant + "e" + string(sign) + digits
	}
	s := strconv.FormatFloat(f, 'f', max(5-exp, 0), 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// parseFormat reads a --format of the form [PREFIX]%[0]['][-][N][.][N]f[SUFFIX]
// and returns the message of the mistake in it, if any.
func (n *Numfmt) parseFormat(f string) string {
	q := quote(f)
	// GNU keeps as many bytes of the text before the directive as it took
	// steps to pass it, counting %% as one step: "%%%f" keeps just "%".
	i, steps := 0, 0
	for i >= len(f) || f[i] != '%' || (i+1 < len(f) && f[i+1] == '%') {
		if i >= len(f) {
			return "format " + q + " has no % directive"
		}
		if f[i] == '%' {
			i++
		}
		i++
		steps++
	}
	prefix := f[:steps]
	i++
	zero := false
	for {
		skip := len(f[i:]) - len(strings.TrimLeft(f[i:], " "))
		i += skip
		if i < len(f) && f[i] == '\'' {
			n.Grouping = true
			i++
			continue
		}
		if i < len(f) && f[i] == '0' {
			zero = true
			i++
			continue
		}
		if skip == 0 {
			break
		}
	}
	if pad, used := cInt(f[i:]); used > 0 {
		if pad != 0 {
			if n.Debug && n.padding != 0 && (!zero || pad < 0) {
				n.diag.Warnf("--format padding overriding --padding")
			}
			switch {
			case pad < 0:
				n.padding = pad
			case zero:
				n.zeroPad = int(min(pad, math.MaxInt32))
			default:
				n.padding = pad
			}
		}
		i += used
	}
	if i >= len(f) {
		return "format " + q + " ends in %"
	}
	if f[i] == '.' {
		i++
		if i < len(f) && (f[i] == ' ' || f[i] == '\t' || f[i] == '+') {
			return "invalid precision in format " + q
		}
		p, used := cInt(f[i:])
		if p < 0 || p > math.MaxInt32 {
			return "invalid precision in format " + q
		}
		n.precision = int(p)
		i += used
	}
	if i >= len(f) || f[i] != 'f' {
		return "invalid format " + q + ", directive must be %[0]['][-][N][.][N]f"
	}
	i++
	for j := i; j < len(f); j++ {
		if f[j] == '%' {
			if j+1 >= len(f) || f[j+1] != '%' {
				return "format " + q + " has too many % directives"
			}
			j++
		}
	}
	// What follows the directive is printed as it is, %% and all, as GNU does.
	n.prefix, n.suffixText = prefix, f[i:]
	return ""
}

// cInt reads an integer as strtoimax does at the start of s (blanks, an
// optional sign, digits) and returns it and how many bytes it used: none
// when there are no digits. A value out of range saturates.
func cInt(s string) (int64, int) {
	i := len(s) - len(strings.TrimLeft(s, " \t\n\v\f\r"))
	neg := false
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	start := i
	var v int64
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		d := int64(s[i] - '0')
		if v > (math.MaxInt64-d)/10 {
			v = math.MaxInt64
			continue
		}
		v = v*10 + d
	}
	if i == start {
		return 0, 0
	}
	if neg {
		v = -v
	}
	return v, i
}

// parseFields reads a --field list: cut's syntax, plus "-" for every field.
func parseFields(list string) ([]span, error) {
	var spans []span
	for item := range strings.FieldsFuncSeq(list, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
		if item == "-" {
			spans = append(spans, span{1, math.MaxUint64})
			continue
		}
		if i := strings.IndexFunc(item, func(r rune) bool { return r != '-' && (r < '0' || r > '9') }); i >= 0 {
			return nil, core.Usagef("invalid field value '%s'", item[i:])
		}
		number := func(s string) (uint64, error) {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return 0, core.Usagef("field number '%s' is too large", s)
			}
			if v == 0 {
				return 0, core.Usagef("fields are numbered from 1")
			}
			return v, nil
		}
		lo, hi, isRange := strings.Cut(item, "-")
		s := span{1, math.MaxUint64}
		var err error
		switch {
		case !isRange:
			if s.lo, err = number(lo); err != nil {
				return nil, err
			}
			s.hi = s.lo
		case strings.Contains(hi, "-"):
			return nil, core.Usagef("invalid field range")
		default:
			if lo != "" {
				if s.lo, err = number(lo); err != nil {
					return nil, err
				}
			}
			if hi != "" {
				if s.hi, err = number(hi); err != nil {
					return nil, err
				}
			}
			if s.hi < s.lo {
				return nil, core.Usagef("invalid decreasing range")
			}
		}
		spans = append(spans, s)
	}
	if len(spans) == 0 {
		return nil, core.Usagef("missing list of fields")
	}
	return spans, nil
}
