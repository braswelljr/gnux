package basenamex

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/basename"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want basename.Options
	}{
		{"defaults", nil, basename.Options{}},
		{"multiple", []string{"-a"}, basename.Options{Multiple: true}},
		{"long multiple", []string{"--multiple"}, basename.Options{Multiple: true}},
		{"suffix implies multiple", []string{"-s", ".h"}, basename.Options{Multiple: true, Suffix: ".h"}},
		{"long suffix", []string{"--suffix=.h"}, basename.Options{Multiple: true, Suffix: ".h"}},
		{"empty suffix still implies multiple", []string{"--suffix="}, basename.Options{Multiple: true}},
		{"zero", []string{"-z"}, basename.Options{Zero: true}},
		{"long zero", []string{"--zero"}, basename.Options{Zero: true}},
		{"combined", []string{"-az", "-s", ".go"}, basename.Options{Multiple: true, Suffix: ".go", Zero: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(basename.Name, pflag.ContinueOnError)
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
