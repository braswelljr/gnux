package seq_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/seq"
)

// FuzzOperands checks that no operands, format or separator make seqx panic:
// integers, decimals, exponents, hex floats, infinities and signs, with and
// without -w. Output is capped, so an endless sequence ends with a write
// error.
func FuzzOperands(f *testing.F) {
	for _, s := range [][3]string{
		{"1 10", "", ""}, {"5", "", ","}, {"1 0.5 3", "%.2f", ""}, {"-3 -1", "", ""}, {"10 -2 1", "", ""},
		{"1e2 1e2", "", ""}, {"0x10 0x12", "", ""}, {"1 inf", "", ""}, {"-inf", "", ""}, {"nan", "", ""},
		{"1 0 5", "", ""}, {"1.000 1.010", "", ""}, {"+1 +3", "%g", ""}, {"1 2", "%05.1e x", ""}, {"1 2", "%d", ""},
		{"1 2", "%", ""}, {"1 2", "%f%f", ""}, {"9999999999999999999 10000000000000000002", "", ""}, {".5 .5 2", "", ""},
	} {
		f.Add(s[0], s[1], s[2], false)
		f.Add(s[0], s[1], s[2], true)
	}
	f.Fuzz(func(t *testing.T, operands, format, sep string, equal bool) {
		if len(operands) > 80 || len(format) > 40 || len(sep) > 10 || tooLarge(format) {
			return
		}
		opts := seq.Options{Format: format, Separator: sep, SeparatorSet: sep != "", EqualWidth: equal}
		s := core.Streams{In: strings.NewReader(""), Out: &budget{left: 1 << 16}, Err: io.Discard}
		_ = seq.New(opts, s).Run(strings.Fields(operands))
	})
}

// budget is an output that fails once it has taken limit bytes, so that a
// fuzzed command line asking for endless output ends with a write error.
type budget struct{ left int }

func (b *budget) Write(p []byte) (int, error) {
	if len(p) > b.left {
		return 0, errors.New("output budget spent")
	}
	b.left -= len(p)
	return len(p), nil
}

// tooLarge reports a run of more than four digits, which a width,
// precision or repeat count would turn into a huge allocation: a size the
// user asked for, not a bug.
func tooLarge(ss ...string) bool {
	for _, s := range ss {
		run := 0
		for i := range len(s) {
			if s[i] >= '0' && s[i] <= '9' {
				if run++; run > 4 {
					return true
				}
			} else {
				run = 0
			}
		}
	}
	return false
}
