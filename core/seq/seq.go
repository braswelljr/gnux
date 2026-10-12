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

	q.out = bufio.NewWriter(q.streams.Out)
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

	if directive == "" && first.isInt && step.isInt && last.inf == 0 && last.isInt &&
		fitsInt64(first.val) && fitsInt64(step.val) && fitsInt64(last.val) {
		return q.integers(first.val.Num().Int64(), step.val.Num().Int64(), last.val.Num().Int64(), sep, width)
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

// integers is the fast path for a sequence of integers that fit in 64 bits.
func (q *Seq) integers(x, step, last int64, sep string, width int) error {
	printed := false
	var buf []byte
	for (step > 0 && x <= last) || (step < 0 && x >= last) {
		buf = buf[:0]
		if printed {
			buf = append(buf, sep...)
		}
		if width > 0 {
			buf = append(buf, zeroPad(strconv.FormatInt(x, 10), width)...)
		} else {
			buf = strconv.AppendInt(buf, x, 10)
		}
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

// fitsInt64 reports whether the integer r fits in an int64.
func fitsInt64(r *big.Rat) bool {
	return r.Num().IsInt64()
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
