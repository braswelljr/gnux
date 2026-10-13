package seq_test

import (
	"bytes"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/seq"
)

// run executes the engine against in-memory streams.
func run(opts seq.Options, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = seq.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func lines(s ...string) string {
	return strings.Join(s, "\n") + "\n"
}

func TestSequence(t *testing.T) {
	comma := seq.Options{Separator: ",", SeparatorSet: true}
	tests := []struct {
		name string
		opts seq.Options
		args []string
		want string
	}{
		{"last", seq.Options{}, []string{"3"}, lines("1", "2", "3")},
		{"first last", seq.Options{}, []string{"3", "5"}, lines("3", "4", "5")},
		{"increment", seq.Options{}, []string{"1", "3", "10"}, lines("1", "4", "7", "10")},
		{"count down", seq.Options{}, []string{"10", "-3", "1"}, lines("10", "7", "4", "1")},
		{"negative first", seq.Options{}, []string{"-2", "1"}, lines("-2", "-1", "0", "1")},
		{"first past last", seq.Options{}, []string{"5", "1"}, ""},
		{"last below one", seq.Options{}, []string{"0"}, ""},
		{"single number", seq.Options{}, []string{"4", "4"}, lines("4")},
		{"exact decimals", seq.Options{}, []string{"0.1", "0.1", "0.3"}, lines("0.1", "0.2", "0.3")},
		{"precision from increment", seq.Options{}, []string{"0", "0.25", "1"}, lines("0.00", "0.25", "0.50", "0.75", "1.00")},
		{"last's precision is ignored", seq.Options{}, []string{"1", "2.5"}, lines("1", "2")},
		{"positive exponent", seq.Options{}, []string{"1e2", "1e2", "3e2"}, lines("100", "200", "300")},
		{"negative exponent", seq.Options{}, []string{"1e-2", "1e-2", "3e-2"}, lines("0.01", "0.02", "0.03")},
		{"hex operands", seq.Options{}, []string{"0x10", "0x11"}, lines("16", "17")},
		{"plus sign", seq.Options{}, []string{"+2"}, lines("1", "2")},
		{"big integers", seq.Options{}, []string{"99999999999999999998", "99999999999999999999"}, lines("99999999999999999998", "99999999999999999999")},
		{"precision of a whole first", seq.Options{}, []string{"1.0", "3"}, lines("1.0", "2.0", "3.0")},
		{"decimals counting down", seq.Options{}, []string{"1", "-0.5", "-1.3"}, lines("1.0", "0.5", "0.0", "-0.5", "-1.0")},
		{"decimals below one", seq.Options{}, []string{"-0.05", "0.05", "0.05"}, lines("-0.05", "0.00", "0.05")},
		{"last between two terms", seq.Options{}, []string{"2", "-0.5", "0.25"}, lines("2.0", "1.5", "1.0", "0.5")},
		{"more decimals than 64 bits hold", seq.Options{}, []string{"0.0000000000000000001", "0.0000000000000000001", "0.0000000000000000002"}, lines("0.0000000000000000001", "0.0000000000000000002")},
		{"near the int64 limit", seq.Options{}, []string{"9223372036854775806", "9223372036854775807"}, lines("9223372036854775806", "9223372036854775807")},

		{"separator", comma, []string{"3"}, "1,2,3\n"},
		{"empty separator", seq.Options{SeparatorSet: true}, []string{"3"}, "123\n"},
		{"long separator", seq.Options{Separator: " - "}, []string{"2"}, "1 - 2\n"},

		{"equal width", seq.Options{EqualWidth: true}, []string{"8", "10"}, lines("08", "09", "10")},
		{"equal width from last operand", seq.Options{EqualWidth: true}, []string{"1", "4", "10"}, lines("01", "05", "09")},
		{"equal width negative", seq.Options{EqualWidth: true}, []string{"-1", "1"}, lines("-1", "00", "01")},
		{"equal width decimals", seq.Options{EqualWidth: true}, []string{"1", "4.5", "10"}, lines("01.0", "05.5", "10.0")},
		{"equal width negative decimals", seq.Options{EqualWidth: true}, []string{"-0.5", "0.5", "1"}, lines("-0.5", "00.0", "00.5", "01.0")},

		{"format", seq.Options{Format: "%.2f"}, []string{"1", "0.5", "2"}, lines("1.00", "1.50", "2.00")},
		{"format text and percent", seq.Options{Format: "n=%g%%"}, []string{"2"}, lines("n=1%", "n=2%")},
		{"format width", seq.Options{Format: "[%3.0f]"}, []string{"2"}, lines("[  1]", "[  2]")},
		{"format exponent", seq.Options{Format: "%e"}, []string{"1"}, lines("1.000000e+00")},
		{"format hex float", seq.Options{Format: "%a"}, []string{"2", "2"}, lines("0x1p+1")},
		{"format with separator", seq.Options{Format: "%g", Separator: " "}, []string{"3"}, "1 2 3\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.args...)
			if err != nil {
				t.Fatalf("Run(%q): %v (stderr %q)", tt.args, err, errb)
			}
			if out != tt.want {
				t.Fatalf("Run(%q) = %q, want %q", tt.args, out, tt.want)
			}
		})
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts seq.Options
		args []string
		want string
	}{
		{"no operand", seq.Options{}, nil, "missing operand"},
		{"extra operand", seq.Options{}, []string{"1", "2", "3", "4"}, "extra operand '4'"},
		{"not a number", seq.Options{}, []string{"abc"}, "invalid floating point argument: 'abc'"},
		{"trailing garbage", seq.Options{}, []string{"1", "2x"}, "invalid floating point argument: '2x'"},
		{"fraction", seq.Options{}, []string{"1/2"}, "invalid floating point argument: '1/2'"},
		{"nan", seq.Options{}, []string{"1", "nan"}, "invalid 'not-a-number' argument: 'nan'"},
		{"zero increment", seq.Options{}, []string{"1", "0", "5"}, "invalid Zero increment value: '0'"},
		{"zero increment written as a decimal", seq.Options{}, []string{"1", "0.0", "5"}, "invalid Zero increment value: '0.0'"},
		{"infinite first", seq.Options{}, []string{"inf", "5"}, "invalid floating point argument: 'inf'"},
		{"no directive", seq.Options{Format: "x"}, []string{"1"}, "format 'x' has no % directive"},
		{"two directives", seq.Options{Format: "%f%f"}, []string{"1"}, "format '%f%f' has too many % directives"},
		{"integer directive", seq.Options{Format: "%d"}, []string{"1"}, "format '%d' has unknown %d directive"},
		{"ends in percent", seq.Options{Format: "%5"}, []string{"1"}, "format '%5' ends in %"},
		{"format with equal width", seq.Options{Format: "%g", EqualWidth: true}, []string{"1"}, "format string may not be specified when printing equal width strings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a UsageError", err)
			}
			if usage.Msg != tt.want {
				t.Fatalf("message = %q, want %q", usage.Msg, tt.want)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want nothing", out)
			}
		})
	}
}

// limited accepts n bytes, then fails as a pipe whose reader has gone.
type limited struct {
	bytes.Buffer
	n int
}

func (l *limited) Write(p []byte) (int, error) {
	if l.Len()+len(p) > l.n {
		return 0, syscall.EPIPE
	}
	return l.Buffer.Write(p)
}

// TestInfinite checks that an endless sequence stops quietly when its reader
// goes away, as in seqx 1 inf | headx.
func TestInfinite(t *testing.T) {
	for _, args := range [][]string{{"1", "inf"}, {"0", "-1", "-inf"}} {
		out := &limited{n: 1 << 20}
		var errb bytes.Buffer
		err := seq.New(seq.Options{}, core.Streams{In: strings.NewReader(""), Out: out, Err: &errb}).Run(args)
		if err != nil || errb.Len() != 0 {
			t.Fatalf("Run(%q): err = %v, stderr = %q; want a quiet stop", args, err, errb.String())
		}
		if out.Len() == 0 {
			t.Fatalf("Run(%q) wrote nothing", args)
		}
	}

	// An infinite LAST in the other direction prints nothing.
	out, _, err := run(seq.Options{}, "5", "-inf")
	if err != nil || out != "" {
		t.Fatalf("seq 5 -inf = (%q, %v), want nothing", out, err)
	}
}

// failing fails every write with an error worth reporting.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, syscall.ENOSPC }

func TestWriteError(t *testing.T) {
	var errb bytes.Buffer
	err := seq.New(seq.Options{}, core.Streams{In: strings.NewReader(""), Out: failing{}, Err: &errb}).Run([]string{"3"})
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb.String(), "seqx: write error: ") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}
