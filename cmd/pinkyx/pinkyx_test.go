package pinkyx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/pinky"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want pinky.Options
	}{
		{nil, pinky.Options{}},
		{[]string{"-l"}, pinky.Options{Long: true}},
		{[]string{"-l", "-s"}, pinky.Options{}},
		{[]string{"-s", "-l"}, pinky.Options{Long: true}},
		{[]string{"-bhp"}, pinky.Options{NoHomeAndShell: true, NoProject: true, NoPlan: true}},
		{[]string{"-f", "-w"}, pinky.Options{NoHeading: true, NoFullName: true}},
		{[]string{"-i"}, pinky.Options{NoFullName: true, NoWhere: true}},
		{[]string{"-q"}, pinky.Options{NoFullName: true, NoWhere: true, NoIdle: true}},
		{[]string{"--lookup"}, pinky.Options{Lookup: true}},
		{[]string{"--l"}, pinky.Options{Lookup: true}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(pinky.Name, pflag.ContinueOnError)
		Register(fs)
		if err := tool.ParseFlagSet(fs, tt.args, false); err != nil {
			t.Fatalf("parse %v: %v", tt.args, err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
