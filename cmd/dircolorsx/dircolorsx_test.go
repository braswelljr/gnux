package dircolorsx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/dircolors"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want dircolors.Options
	}{
		{nil, dircolors.Options{}},
		{[]string{"-b"}, dircolors.Options{Shell: dircolors.Bourne}},
		{[]string{"--bourne-shell"}, dircolors.Options{Shell: dircolors.Bourne}},
		{[]string{"--c-shell"}, dircolors.Options{Shell: dircolors.C}},
		{[]string{"-b", "-c"}, dircolors.Options{Shell: dircolors.C}},
		{[]string{"-c", "-b"}, dircolors.Options{Shell: dircolors.Bourne}},
		{[]string{"--cs"}, dircolors.Options{Shell: dircolors.C}},
		{[]string{"-p"}, dircolors.Options{PrintDatabase: true}},
		{[]string{"--print-l"}, dircolors.Options{PrintLsColors: true}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(dircolors.Name, pflag.ContinueOnError)
		Register(fs)
		if err := tool.ParseFlagSet(fs, tt.args, false); err != nil {
			t.Fatalf("parse %v: %v", tt.args, err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
