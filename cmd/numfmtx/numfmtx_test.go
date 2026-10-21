package numfmtx

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/numfmt"
	"github.com/braswelljr/gnux/internal/tool"
)

func parse(t *testing.T, args ...string) (numfmt.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(numfmt.Name, pflag.ContinueOnError)
	Register(fs)
	if err := tool.ParseFlagSet(fs, args, false); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want numfmt.Options
	}{
		{nil, numfmt.Options{}},
		{[]string{"--to=si", "--from=auto"}, numfmt.Options{To: numfmt.ScaleSI, From: numfmt.ScaleAuto}},
		{[]string{"--to=iec-i", "--from=iec"}, numfmt.Options{To: numfmt.ScaleIECI, From: numfmt.ScaleIEC}},
		{[]string{"--to=s"}, numfmt.Options{To: numfmt.ScaleSI}},
		{[]string{"--round=near"}, numfmt.Options{Round: numfmt.RoundNearest}},
		{[]string{"--invalid=w"}, numfmt.Options{Invalid: numfmt.InvalidWarn}},
		{[]string{"--from-unit=1K", "--to-unit=1Ki"}, numfmt.Options{FromUnit: 1000, ToUnit: 1024}},
		{[]string{"--padding=-8"}, numfmt.Options{Padding: -8}},
		{[]string{"--header"}, numfmt.Options{Header: 1}},
		{[]string{"--header=3"}, numfmt.Options{Header: 3}},
		{[]string{"-d:", "--field=2-"}, numfmt.Options{Delimiter: ':', DelimiterSet: true, Fields: "2-"}},
		{[]string{"-d", ""}, numfmt.Options{DelimiterSet: true}},
		{[]string{"-z", "--suffix=B", "--grouping", "--debug", "--format=%.2f"},
			numfmt.Options{ZeroTerminated: true, Suffix: "B", Grouping: true, Debug: true, Format: "%.2f"}},
	}
	for _, tt := range tests {
		got, err := parse(t, tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
		}
	}
}

func TestBuildOptionsMistakes(t *testing.T) {
	tests := []struct {
		args  []string
		msg   string
		usage bool
	}{
		{[]string{"--to=auto"}, "invalid argument 'auto' for '--to'\nValid arguments are:\n  - 'none'\n  - 'si'\n  - 'iec'\n  - 'iec-i'", true},
		{[]string{"--to=i"}, "ambiguous argument 'i' for '--to'\nValid arguments are:\n  - 'none'\n  - 'si'\n  - 'iec'\n  - 'iec-i'", true},
		{[]string{"--padding=0"}, "invalid padding value '0'", false},
		{[]string{"--padding=x"}, "invalid padding value 'x'", false},
		{[]string{"--header=0"}, "invalid header value '0'", false},
		{[]string{"--from-unit=0"}, "invalid unit size: '0'", false},
		{[]string{"--from-unit=1KiB"}, "invalid unit size: '1KiB'", false},
		{[]string{"-d", "ab"}, "the delimiter must be a single character", false},
		{[]string{"--field=1", "--field=2"}, "multiple field specifications", false},
	}
	for _, tt := range tests {
		_, err := parse(t, tt.args...)
		var usage *core.UsageError
		var f *fatal
		switch {
		case tt.usage && errors.As(err, &usage) && usage.Msg == tt.msg:
		case !tt.usage && errors.As(err, &f) && f.msg == tt.msg:
		default:
			t.Errorf("%v: err = %v, want %q", tt.args, err, tt.msg)
		}
	}
}
