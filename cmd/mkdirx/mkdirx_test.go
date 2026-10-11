package mkdirx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/mkdir"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want mkdir.Options
	}{
		{"defaults", nil, mkdir.Options{}},
		{"parents", []string{"-p"}, mkdir.Options{Parents: true}},
		{"long parents", []string{"--parents"}, mkdir.Options{Parents: true}},
		{"mode", []string{"-m", "755"}, mkdir.Options{Mode: "755"}},
		{"attached mode", []string{"-m700"}, mkdir.Options{Mode: "700"}},
		{"long symbolic mode", []string{"--mode=u=rwx,go=rx"}, mkdir.Options{Mode: "u=rwx,go=rx"}},
		{"verbose", []string{"-v"}, mkdir.Options{Verbose: true}},
		{"long verbose", []string{"--verbose"}, mkdir.Options{Verbose: true}},
		{"combined", []string{"-pvm", "a+x"}, mkdir.Options{Parents: true, Verbose: true, Mode: "a+x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(mkdir.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}
