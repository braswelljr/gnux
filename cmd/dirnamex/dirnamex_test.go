package dirnamex

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/dirname"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want dirname.Options
	}{
		{"defaults", nil, dirname.Options{}},
		{"zero", []string{"-z"}, dirname.Options{Zero: true}},
		{"long zero", []string{"--zero"}, dirname.Options{Zero: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(dirname.Name, pflag.ContinueOnError)
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
