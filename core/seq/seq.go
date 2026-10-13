// Package seq implements the engine behind seqx, a GNU-compatible
// implementation of the GNU/UNIX seq command: print a sequence of numbers.
package seq

import (
	"bufio"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printf"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "seqx"

// Options mirrors the GNU seq flag surface. The zero value is plain seq: one
// number per line, as few digits as the operands call for.
type Options struct {
	// Format (-f) prints each number with a printf-style floating-point
	// directive (%e, %f, %g or %a, with optional flags, width and precision)
	// surrounded by literal text.
	Format string
	// Separator (-s) goes between the numbers instead of a newline. An empty
	// Separator means a newline unless SeparatorSet is true, which makes it
	// mean no separator at all (seq -s '').
	Separator    string
	SeparatorSet bool
	// EqualWidth (-w) pads the numbers with leading zeros to equal width.
	EqualWidth bool
}

// Seq is the engine. Construct it with New.
type Seq struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a seq engine writing to s.
func New(opts Options, s core.Streams) *Seq {
	return &Seq{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// operand is one parsed number: its exact value, or an infinity.
type operand struct {
	text  string
	val   *big.Rat
	inf   int // +1 or -1 for an infinity, 0 otherwise
	prec  int // digits after the decimal point it calls for
	isInt bool
}

// Run prints the sequence described by the operands: LAST, FIRST LAST or
// FIRST INCREMENT LAST. FIRST and INCREMENT default to 1. Nothing is printed
// when FIRST is already past LAST.
func (q *Seq) Run(args []string) error {
	switch {
	case len(args) == 0:
		return core.Usagef("missing operand")
	case len(args) > 3:
		return core.Usagef("extra operand '%s'", args[3])
	}
	if q.Format != "" && q.EqualWidth {
		return core.Usagef("format string may not be specified when printing equal width strings")
	}

	ops := make([]operand, len(args))
	for i, a := range args {
		op, err := parse(a)
		if err != nil {
			return err
		}
		ops[i] = op
	}
	first := operand{text: "1", val: big.NewRat(1, 1), isInt: true}
	step := first
	var last operand
	switch len(ops) {
	case 1:
		last = ops[0]
	case 2:
		first, last = ops[0], ops[1]
	case 3:
		first, step, last = ops[0], ops[1], ops[2]
	}
	if first.inf != 0 || step.inf != 0 {
		// An infinite FIRST or INCREMENT can only repeat one value forever.
		bad := first
		if step.inf != 0 {
			bad = step
		}
		return core.Usagef("invalid floating point argument: '%s'", bad.text)
	}
	if step.val.Sign() == 0 {
		return core.Usagef("invalid Zero increment value: '%s'", step.text)
	}

	var prefix, suffix, directive string
	if q.Format != "" {
		var err error
		if prefix, directive, suffix, err = splitFormat(q.Format); err != nil {
			return err
		}
	}

	sep := "\n"
	if q.Separator != "" || q.SeparatorSet {
		sep = q.Separator
	}

	prec := max(first.prec, step.prec)
	width := 0
	if q.EqualWidth {
		width = len(decimal(first.val, prec))
		if last.inf == 0 {
			width = max(width, len(decimal(last.val, prec)))
		}
	}

	q.out = core.NewWriter(q.streams.Out)
	format := func(x *big.Rat) string {
		if directive != "" {
			f, _ := x.Float64()
			s, _ := printf.FormatFloat(directive, f)
			return prefix + s + suffix
		}
		return zeroPad(decimal(x, prec), width)
	}

	ascending := step.val.Sign() > 0
	past := func(x *big.Rat) bool {
		if last.inf != 0 {
			return (last.inf > 0) != ascending
		}
		if ascending {
			return x.Cmp(last.val) > 0
		}
		return x.Cmp(last.val) < 0
	}

	if directive == "" && last.inf == 0 {
		if x, by, end, ok := fixedPoint(first.val, step.val, last.val, prec); ok {
			return q.fixed(x, by, end, prec, sep, width)
		}
	}

	x := new(big.Rat).Set(first.val)
	printed := false
	for !past(x) {
		if printed {
			_, _ = q.out.WriteString(sep)
		}
		if _, err := q.out.WriteString(format(x)); err != nil {
			return q.writeErr(err)
		}
		printed = true
		x.Add(x, step.val)
	}
	return q.end(printed)
}

// maxFixedPrec is the most digits after the point the fast path carries: a
// 64-bit count of 10^-18 units still spans the integers up to nine.
const maxFixedPrec = 18

// fixedPoint expresses a sequence in units of 10^-prec, when FIRST and
// INCREMENT are whole numbers of them and everything fits in 64 bits. Every
// term is then a whole number of units too, and the sequence is plain integer
// arithmetic. LAST only bounds the sequence, so it is rounded toward FIRST.
func fixedPoint(first, step, last *big.Rat, prec int) (x, by, end int64, ok bool) {
	if prec > maxFixedPrec {
		return 0, 0, 0, false
	}
	unit := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(prec)), nil))
	units := func(r *big.Rat) *big.Rat { return new(big.Rat).Mul(r, unit) }

	f, s, l := units(first), units(step), units(last)
	// Div rounds toward negative infinity: the bound of an ascending sequence.
	bound := new(big.Int).Div(l.Num(), l.Denom())
	if s.Sign() < 0 && !l.IsInt() {
		bound.Add(bound, big.NewInt(1))
	}
	if !f.IsInt() || !s.IsInt() || !f.Num().IsInt64() || !s.Num().IsInt64() || !bound.IsInt64() {
		return 0, 0, 0, false
	}
	return f.Num().Int64(), s.Num().Int64(), bound.Int64(), true
}

// fixed is the fast path: a sequence counted in 64-bit units of 10^-prec
// (see fixedPoint), which is every sequence of integers and most others.
func (q *Seq) fixed(x, step, last int64, prec int, sep string, width int) error {
	printed := false
	var buf []byte
	for (step > 0 && x <= last) || (step < 0 && x >= last) {
		buf = buf[:0]
		if printed {
			buf = append(buf, sep...)
		}
		buf = appendFixed(buf, x, prec, width)
		if _, err := q.out.Write(buf); err != nil {
			return q.writeErr(err)
		}
		printed = true
		// Stop before x+step would overflow: it would be past last anyway.
		if (step > 0 && x > math.MaxInt64-step) || (step < 0 && x < math.MinInt64-step) {
			break
		}
		x += step
	}
	return q.end(printed)
}

// appendFixed appends x units of 10^-prec as a decimal number with prec digits
// after the point, padded with zeros after any sign to width: what
// zeroPad(decimal(x/10^prec, prec), width) renders.
func appendFixed(dst []byte, x int64, prec, width int) []byte {
	u := uint64(x)
	if x < 0 {
		u = -u
		dst = append(dst, '-')
		width--
	}
	var digits [20]byte // the longest uint64
	text := strconv.AppendUint(digits[:0], u, 10)

	// At least one digit stands before the point: 5 units of 0.01 are "0.05".
	whole := max(len(text)-prec, 1)
	if prec > 0 {
		width -= prec + 1
	}
	for zeros := width - whole; zeros > 0; zeros-- {
		dst = append(dst, '0')
	}
	if len(text) <= prec {
		dst = append(dst, '0', '.')
		for zeros := prec - len(text); zeros > 0; zeros-- {
			dst = append(dst, '0')
		}
		return append(dst, text...)
	}
	dst = append(dst, text[:whole]...)
	if prec > 0 {
		dst = append(dst, '.')
		dst = append(dst, text[whole:]...)
	}
	return dst
}

// end terminates the output with a newline and flushes it.
func (q *Seq) end(printed bool) error {
	if printed {
		_, _ = q.out.WriteString("\n")
	}
	if err := q.out.Flush(); err != nil {
		return q.writeErr(err)
	}
	return nil
}

// writeErr reports a write failure, except a reader that has gone away,
// which ends seq quietly (seqx 1 inf | headx).
func (q *Seq) writeErr(err error) error {
	if core.IsClosedPipe(err) {
		return nil
	}
	q.diag.Errorf("write error: %s", core.ErrText(err))
	return q.diag.Result()
}

// parse reads one operand. NaN is refused; an infinity is kept as a sign.
func parse(s string) (operand, error) {
	op := operand{text: s}
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	neg := strings.HasPrefix(t, "-")
	body := strings.ToLower(strings.TrimLeft(t, "+-"))

	_, end, _ := printf.ParseFloatPrefix(s)
	if end == 0 || end != len(s) {
		return op, core.Usagef("invalid floating point argument: '%s'", s)
	}
	switch {
	case strings.HasPrefix(body, "nan"):
		return op, core.Usagef("invalid 'not-a-number' argument: '%s'", s)
	case strings.HasPrefix(body, "inf"):
		op.inf = 1
		if neg {
			op.inf = -1
		}
		return op, nil
	}

	r, ok := printf.ParseRat(s)
	if !ok {
		return op, core.Usagef("invalid floating point argument: '%s'", s)
	}
	op.val = r
	op.isInt = r.IsInt()

	// The precision GNU derives from the text: the digits after the point,
	// less a positive exponent, plus a negative one.
	if strings.HasPrefix(body, "0x") {
		if !r.IsInt() {
			op.prec = fracDigits(r)
		}
		return op, nil
	}
	mant, exp, hasExp := strings.Cut(body, "e")
	if _, frac, ok := strings.Cut(mant, "."); ok {
		op.prec = len(frac)
	}
	if hasExp {
		e, _ := strconv.Atoi(exp)
		if e < 0 {
			op.prec -= e
		} else {
			op.prec -= min(op.prec, e)
		}
	}
	return op, nil
}

// fracDigits returns how many decimal digits r needs after the point; r's
// denominator is a power of two, so the expansion terminates.
func fracDigits(r *big.Rat) int {
	n := 0
	x := new(big.Rat).Set(r)
	ten := big.NewRat(10, 1)
	for !x.IsInt() && n < 64 {
		x.Mul(x, ten)
		n++
	}
	return n
}

// decimal renders x with prec digits after the point.
func decimal(x *big.Rat, prec int) string {
	if x.IsInt() && prec == 0 {
		return x.Num().String()
	}
	return x.FloatString(prec)
}

// zeroPad pads s with zeros after any sign to width.
func zeroPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	zeros := strings.Repeat("0", width-len(s))
	if strings.HasPrefix(s, "-") {
		return "-" + zeros + s[1:]
	}
	return zeros + s
}

// splitFormat checks a -f format and splits it around its one directive.
// "%%" is a literal percent sign in the text around it.
func splitFormat(f string) (prefix, directive, suffix string, err error) {
	var text strings.Builder
	found := false
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			text.WriteByte(f[i])
			continue
		}
		if i+1 < len(f) && f[i+1] == '%' {
			text.WriteByte('%')
			i++
			continue
		}
		if found {
			return "", "", "", core.Usagef("format '%s' has too many %% directives", f)
		}
		j := i + 1
		for j < len(f) && strings.IndexByte("-+ #0'", f[j]) >= 0 {
			j++
		}
		for j < len(f) && f[j] >= '0' && f[j] <= '9' {
			j++
		}
		if j < len(f) && f[j] == '.' {
			j++
			for j < len(f) && f[j] >= '0' && f[j] <= '9' {
				j++
			}
		}
		if j >= len(f) {
			return "", "", "", core.Usagef("format '%s' ends in %%", f)
		}
		if strings.IndexByte("eEfFgGaA", f[j]) < 0 {
			return "", "", "", core.Usagef("format '%s' has unknown %%%c directive", f, f[j])
		}
		prefix = text.String()
		text.Reset()
		directive = f[i : j+1]
		found = true
		i = j
	}
	if !found {
		return "", "", "", core.Usagef("format '%s' has no %% directive", f)
	}
	return prefix, directive, text.String(), nil
}
