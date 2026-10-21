package ptxx

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/ptx"
	"github.com/braswelljr/gnux/internal/tool"
)

func parse(t *testing.T, args ...string) (ptx.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(ptx.Name, pflag.ContinueOnError)
	Register(fs)
	if err := tool.ParseFlagSet(fs, args, false); err != nil {
		return ptx.Options{}, err
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want ptx.Options
	}{
		{nil, ptx.Options{}},
		{[]string{"-O"}, ptx.Options{Format: ptx.Roff}},
		{[]string{"-O", "-T"}, ptx.Options{Format: ptx.TeX}},
		{[]string{"-T", "--format=roff"}, ptx.Options{Format: ptx.Roff}},
		{[]string{"--format=t"}, ptx.Options{Format: ptx.TeX}},
		{[]string{"-AGRfr", "-M", "ix"}, ptx.Options{AutoReference: true, Traditional: true, RightSideRefs: true, IgnoreCase: true, References: true, MacroName: "ix"}},
		{[]string{"-F", ""}, ptx.Options{FlagTruncationSet: true}},
		{[]string{"-S", `\n`, "-W", `[a-z]+`}, ptx.Options{SentenceRegexp: `\n`, SentenceRegexpSet: true, WordRegexp: "[a-z]+"}},
		{[]string{"-g", "5", "-w", "0x40"}, ptx.Options{GapSize: 5, Width: 64}},
		{[]string{"-b", "b", "-i", "i", "-o", "o"}, ptx.Options{BreakFile: "b", IgnoreFile: "i", OnlyFile: "o"}},
	}
	for _, tt := range tests {
		got, err := parse(t, tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("%v: %+v %v, want %+v", tt.args, got, err, tt.want)
		}
	}
}

func TestBuildOptionsMistakes(t *testing.T) {
	for args, msg := range map[string]string{"-g": "invalid gap width: '0'", "-w": "invalid line width: '0'"} {
		_, err := parse(t, args, "0")
		var f *fatal
		if !errors.As(err, &f) || f.msg != msg {
			t.Errorf("%s 0: %v", args, err)
		}
	}
	if _, err := parse(t, "--format=x"); err == nil {
		t.Error("--format=x accepted")
	}
}
