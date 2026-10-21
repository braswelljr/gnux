package pathchkx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/pathchk"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want pathchk.Options
	}{
		{"defaults", nil, pathchk.Options{}},
		{"basic", []string{"-p"}, pathchk.Options{Portable: true}},
		{"extra", []string{"-P"}, pathchk.Options{Extra: true}},
		{"both letters", []string{"-pP"}, pathchk.Options{Portable: true, Extra: true}},
		{"portability", []string{"--portability"}, pathchk.Options{Portable: true, Extra: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(pathchk.Name, pflag.ContinueOnError)
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
