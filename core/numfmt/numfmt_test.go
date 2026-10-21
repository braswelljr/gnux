package numfmt_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/numfmt"
)

func run(opts numfmt.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = numfmt.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func status(err error) int {
	var exit *core.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.Code
	}
	return 1
}

// TestAgainstGNU holds numfmtx to what GNU numfmt 9.7 printed for the same
// command lines.
func TestAgainstGNU(t *testing.T) {
	si := numfmt.Options{To: numfmt.ScaleSI}
	tests := []struct {
		name   string
		opts   numfmt.Options
		stdin  string
		args   []string
		out    string
		err    string
		status int
	}{
		{"si", si, "", []string{"1000", "1500000", "9999", "999999", "1", "0", "-1500"}, "1.0k\n1.5M\n10k\n1.0M\n1\n0\n-1.5k\n", "", 0},
		{"iec", numfmt.Options{To: numfmt.ScaleIEC}, "", []string{"1024", "2047"}, "1.0K\n2.0K\n", "", 0},
		{"iec-i", numfmt.Options{To: numfmt.ScaleIECI}, "", []string{"1048576"}, "1.0Mi\n", "", 0},
		{"from si", numfmt.Options{From: numfmt.ScaleSI}, "", []string{"1K", "1k", "0.5K"}, "1000\n1000\n500\n", "", 0},
		{"from auto", numfmt.Options{From: numfmt.ScaleAuto}, "", []string{"1Ki", "1K"}, "1024\n1000\n", "", 0},
		{"from iec-i needs the i", numfmt.Options{From: numfmt.ScaleIECI}, "", []string{"1K"}, "", "numfmtx: missing 'i' suffix in input: '1K' (e.g Ki/Mi/Gi)\n", 2},
		{"si to iec", numfmt.Options{From: numfmt.ScaleSI, To: numfmt.ScaleIEC}, "", []string{"1M"}, "977K\n", "", 0},
		{"suffix rejected", numfmt.Options{}, "", []string{"1K"}, "", "numfmtx: rejecting suffix in input: '1K' (consider using --from)\n", 2},
		{"not a number", numfmt.Options{}, "", []string{"abc", "5"}, "", "numfmtx: invalid number: 'abc'\n", 2},
		{"bad suffix", si, "", []string{"1e3"}, "", "numfmtx: invalid suffix in input: '1e3'\n", 2},
		{"input precision kept", numfmt.Options{}, "", []string{"1.5", "1.50", "0005", ".5"}, "1.5\n1.50\n5\n0.5\n", "", 0},
		{"no digits after the point", numfmt.Options{}, "", []string{"5."}, "", "numfmtx: invalid number: '5.'\n", 2},
		{"round down", numfmt.Options{To: numfmt.ScaleSI, Round: numfmt.RoundDown}, "", []string{"1999"}, "1.9k\n", "", 0},
		{"round nearest", numfmt.Options{To: numfmt.ScaleSI, Round: numfmt.RoundNearest}, "", []string{"1450", "-1450"}, "1.5k\n-1.5k\n", "", 0},
		{"round towards zero", numfmt.Options{To: numfmt.ScaleSI, Round: numfmt.RoundTowardsZero}, "", []string{"-1999"}, "-1.9k\n", "", 0},
		{"round up", numfmt.Options{To: numfmt.ScaleIEC, Round: numfmt.RoundUp}, "", []string{"1025"}, "1.1K\n", "", 0},
		{"rounding carries into the next unit", si, "", []string{"999500", "9950", "99950", "1.25"}, "1.0M\n10k\n100k\n1\n", "", 0},
		{"huge", si, "", []string{"123456789012345678901234567890", "1000000000000000000000000000000"}, "124R\n1.0Q\n", "", 0},
		{"too long without --to", numfmt.Options{}, "", []string{"12345678901234567890"}, "",
			"numfmtx: value too large to be printed: '1.23457e+19' (consider using --to)\n", 2},
		{"width", numfmt.Options{Format: "%10f"}, "", []string{"1234"}, "      1234\n", "", 0},
		{"left and text", numfmt.Options{Format: "%-10fX"}, "", []string{"1234"}, "1234      X\n", "", 0},
		{"zero padding", numfmt.Options{Format: "%010f"}, "", []string{"1234", "-12.5"}, "0000001234\n-0000012.5\n", "", 0},
		{"precision", numfmt.Options{Format: "%.3f"}, "", []string{"1.23456"}, "1.235\n", "", 0},
		{"precision with a unit", numfmt.Options{Format: "%.1f", To: numfmt.ScaleSI}, "", []string{"1234567"}, "1.3M\n", "", 0},
		{"no decimals with a unit", numfmt.Options{Format: "%.0f", To: numfmt.ScaleSI}, "", []string{"1234567"}, "2M\n", "", 0},
		{"text around", numfmt.Options{Format: "pre %f post"}, "", []string{"5"}, "pre 5 post\n", "", 0},
		{"percent signs kept as GNU keeps them", numfmt.Options{Format: "%%%f%%"}, "", []string{"5"}, "%5%%\n", "", 0},
		{"padding", numfmt.Options{Padding: 8}, "", []string{"42"}, "      42\n", "", 0},
		{"left padding", numfmt.Options{Padding: -8}, "", []string{"42"}, "42      \n", "", 0},
		{"suffix", numfmt.Options{To: numfmt.ScaleIEC, Suffix: "B"}, "", []string{"1024"}, "1.0KB\n", "", 0},
		{"suffix on input", numfmt.Options{From: numfmt.ScaleIEC, Suffix: "B"}, "", []string{"1KB"}, "1024B\n", "", 0},
		{"units", numfmt.Options{FromUnit: 512}, "", []string{"4"}, "2048\n", "", 0},
		{"to unit", numfmt.Options{ToUnit: 1000}, "", []string{"5000"}, "5\n", "", 0},
		{"warn", numfmt.Options{Invalid: numfmt.InvalidWarn}, "", []string{"abc", "5"}, "abc\n5\n", "numfmtx: invalid number: 'abc'\n", 0},
		{"fail", numfmt.Options{Invalid: numfmt.InvalidFail}, "", []string{"abc", "5"}, "abc\n5\n", "numfmtx: invalid number: 'abc'\n", 2},
		{"ignore", numfmt.Options{Invalid: numfmt.InvalidIgnore}, "", []string{"abc", "5"}, "abc\n5\n", "", 0},
		{"blank-led fields keep their width", si, "  1000 2000\n 300 4\n", nil, "  1.0k 2000\n 300 4\n", "", 0},
		{"header", numfmt.Options{To: numfmt.ScaleSI, Header: 1, Fields: "2"}, "size 1000\nx 2000\n", nil, "size 1000\nx 2.0k\n", "", 0},
		{"later fields keep their width", numfmt.Options{To: numfmt.ScaleSI, Fields: "2"}, "a 1000 b\n", nil, "a 1.0k b\n", "", 0},
		{"tabs become spaces", numfmt.Options{To: numfmt.ScaleSI, Fields: "3"}, "a\tb\t1000\n", nil, "a b 1.0k\n", "", 0},
		{"field ranges", numfmt.Options{To: numfmt.ScaleSI, Fields: "2-"}, "", []string{"1 2000 3000"}, "1 2.0k 3.0k\n", "", 0},
		{"every field", numfmt.Options{To: numfmt.ScaleSI, Fields: "-"}, "", []string{"1000 2000"}, "1.0k 2.0k\n", "", 0},
		{"delimiter", numfmt.Options{To: numfmt.ScaleSI, Fields: "2", Delimiter: ':', DelimiterSet: true}, "", []string{"a:1000:b"}, "a:1.0k:b\n", "", 0},
		{"zero terminated", numfmt.Options{To: numfmt.ScaleSI, ZeroTerminated: true}, "1000\x002000\x00", nil, "1.0k\x002.0k\x00", "", 0},
		{"no final newline", si, "1000", nil, "1.0k", "", 0},
		{"an empty line is not a number", numfmt.Options{}, "\n\n", nil, "", "numfmtx: invalid number: ''\n", 2},
		{"abort stops mid-line", numfmt.Options{To: numfmt.ScaleSI, Fields: "1-2"}, "1000 2000\n3000 x\n4000\n", nil, "1.0k 2.0k\n3.0k ", "numfmtx: invalid number: 'x'\n", 2},
		{"debug", numfmt.Options{Debug: true, Invalid: numfmt.InvalidFail}, "", []string{"x"}, "x\n",
			"numfmtx: no conversion option specified\nnumfmtx: invalid number: 'x'\nnumfmtx: failed to convert some of the input numbers\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.stdin, tt.args...)
			if out != tt.out || errb != tt.err || status(err) != tt.status {
				t.Fatalf("got stdout %q stderr %q status %d (%v)\nwant stdout %q stderr %q status %d", out, errb, status(err), err, tt.out, tt.err, tt.status)
			}
		})
	}
}

func TestOptionMistakes(t *testing.T) {
	tests := []struct {
		opts  numfmt.Options
		err   string
		usage bool
	}{
		{numfmt.Options{Format: "%d"}, "numfmtx: invalid format '%d', directive must be %[0]['][-][N][.][N]f\n", false},
		{numfmt.Options{Format: "%f%f"}, "numfmtx: format '%f%f' has too many % directives\n", false},
		{numfmt.Options{Format: "%"}, "numfmtx: format '%' ends in %\n", false},
		{numfmt.Options{Format: "x"}, "numfmtx: format 'x' has no % directive\n", false},
		{numfmt.Options{Format: "%.-1f"}, "numfmtx: invalid precision in format '%.-1f'\n", false},
		{numfmt.Options{Grouping: true, To: numfmt.ScaleSI}, "numfmtx: grouping cannot be combined with --to\n", false},
		{numfmt.Options{Grouping: true, Format: "%f"}, "numfmtx: --grouping cannot be combined with --format\n", false},
		{numfmt.Options{Fields: "0"}, "", true},
		{numfmt.Options{To: numfmt.ScaleAuto}, "", true},
	}
	for _, tt := range tests {
		_, errb, err := run(tt.opts, "", "5")
		var usage *core.UsageError
		if tt.usage {
			if !errors.As(err, &usage) {
				t.Errorf("%+v: err = %v, want a usage error", tt.opts, err)
			}
			continue
		}
		if !errors.Is(err, core.ErrFailed) || errb != tt.err {
			t.Errorf("%+v: err=%v stderr=%q, want %q", tt.opts, err, errb, tt.err)
		}
	}
}
