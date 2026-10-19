// Package factor implements the engine behind factorx, a GNU-compatible
// implementation of the GNU factor command: print the prime factors of each
// number.
package factor

import (
	"bufio"
	"errors"
	"io"
	"math/big"
	"slices"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "factorx"

// Options mirrors the GNU factor flag surface. The zero value is plain factor.
type Options struct {
	// Exponents (-h, --exponents) prints a repeated factor once, as p^e.
	Exponents bool
}

// Factor is the engine. Construct it with New.
type Factor struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a factor engine reading from and writing to s.
func New(opts Options, s core.Streams) *Factor {
	return &Factor{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run factors each operand, or each whitespace-separated number read from
// standard input when there is none. A number that is not a positive integer
// is reported and the rest are still factored. It returns core.ErrFailed
// when any number was invalid, and nil otherwise.
func (f *Factor) Run(args []string) (err error) {
	f.out = core.NewWriter(f.streams.Out)
	defer core.FlushOutput(f.out, f.diag, &err)

	if len(args) > 0 {
		for _, a := range args {
			if !f.print(a) {
				break
			}
		}
		return f.diag.Result()
	}

	in := core.NewReader(f.streams.In)
	var tok []byte
	for {
		c, rerr := in.ReadByte()
		if rerr == nil && c != ' ' && c != '\t' && c != '\n' {
			tok = append(tok, c)
			continue
		}
		if len(tok) > 0 {
			if !f.print(string(tok)) {
				break
			}
			tok = tok[:0]
			core.FlushIfIdle(f.out, in)
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				f.diag.Errorf("-: %s", core.ErrText(rerr))
			}
			break
		}
	}
	return f.diag.Result()
}

// print factors one number and writes its line. It reports false once the
// output has failed, which ends the run.
func (f *Factor) print(input string) bool {
	n, ok := parse(input)
	if !ok {
		// The output goes first, so that it and the diagnostic stay in
		// order on a shared terminal.
		if err := f.out.Flush(); err != nil {
			return false
		}
		f.diag.Errorf("'%s' is not a valid positive integer", input)
		return true
	}

	line := n.Append(f.out.AvailableBuffer(), 10)
	line = append(line, ':')
	factors := Of(n)
	for i := 0; i < len(factors); {
		p := factors[i]
		e := 1
		for i+e < len(factors) && factors[i+e].Cmp(p) == 0 {
			e++
		}
		if f.Exponents {
			line = append(line, ' ')
			line = p.Append(line, 10)
			if e > 1 {
				line = append(line, '^')
				line = core.AppendInt(line, int64(e), 0)
			}
		} else {
			for range e {
				line = append(line, ' ')
				line = p.Append(line, 10)
			}
		}
		i += e
	}
	line = append(line, '\n')
	_, err := f.out.Write(line)
	return err == nil
}

// parse reads a number as GNU factor does: leading spaces and one '+' are
// skipped, and what is left must be decimal digits only.
func parse(s string) (*big.Int, bool) {
	s = strings.TrimLeft(s, " ")
	s = strings.TrimPrefix(s, "+")
	if s == "" {
		return nil, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return nil, false
		}
	}
	n, ok := new(big.Int).SetString(s, 10)
	return n, ok
}

// Of returns the prime factors of n in ascending order, a repeated factor
// as many times as it divides n. It returns none for 0 and 1, as GNU factor
// prints none. n must not be negative.
func Of(n *big.Int) []*big.Int {
	var factors []*big.Int
	if n.Sign() <= 0 || n.Cmp(one) == 0 {
		return nil
	}
	if n.IsUint64() {
		for _, p := range factor64(n.Uint64()) {
			factors = append(factors, new(big.Int).SetUint64(p))
		}
		return factors
	}
	factors = factorBig(new(big.Int).Set(n), factors)
	slices.SortFunc(factors, func(a, b *big.Int) int { return a.Cmp(b) })
	return factors
}
