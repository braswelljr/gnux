package shredx

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shred"
	"github.com/braswelljr/gnux/internal/tool"
)

func parse(t *testing.T, args ...string) (shred.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(shred.Name, pflag.ContinueOnError)
	Register(fs)
	if err := tool.ParseFlagSet(fs, args, false); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want shred.Options
	}{
		{nil, shred.Options{}},
		{[]string{"-fvxz"}, shred.Options{Force: true, Verbose: true, Exact: true, Zero: true}},
		{[]string{"-n", "5"}, shred.Options{Iterations: 5}},
		{[]string{"-n0"}, shred.Options{Iterations: -1}},
		{[]string{"-u"}, shred.Options{Remove: shred.WipeSync}},
		{[]string{"-uv"}, shred.Options{Remove: shred.WipeSync, Verbose: true}},
		{[]string{"--remove"}, shred.Options{Remove: shred.WipeSync}},
		{[]string{"--remove=unlink"}, shred.Options{Remove: shred.Unlink}},
		{[]string{"--remove=wipe"}, shred.Options{Remove: shred.Wipe}},
		{[]string{"-s", "10c"}, shred.Options{Size: 10, SizeSet: true}},
		{[]string{"-s", "2K"}, shred.Options{Size: 2048, SizeSet: true}},
		{[]string{"-s", "2KB"}, shred.Options{Size: 2000, SizeSet: true}},
		{[]string{"-s", "1b"}, shred.Options{Size: 512, SizeSet: true}},
		{[]string{"--random-source=r", "--random-source=r"}, shred.Options{RandomSource: "r"}},
	}
	for _, tt := range tests {
		got, err := parse(t, tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
		}
	}
}

func TestBuildOptionsMistakes(t *testing.T) {
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-n", "x"}, "invalid number of passes: 'x'"},
		{[]string{"-s", "1x"}, "invalid file size: '1x'"},
		{[]string{"--random-source=a", "--random-source=b"}, "multiple random sources specified"},
		{[]string{"--remove=w"}, "ambiguous argument 'w' for '--remove'\nValid arguments are:\n  - 'unlink'\n  - 'wipe'\n  - 'wipesync'"},
	} {
		_, err := parse(t, tt.args...)
		var f *fatal
		var usage *core.UsageError
		matched := (errors.As(err, &f) && f.msg == tt.msg) || (errors.As(err, &usage) && usage.Msg == tt.msg)
		if !matched {
			t.Errorf("%v: err = %v, want %q", tt.args, err, tt.msg)
		}
	}
}
