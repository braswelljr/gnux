package rmdirx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rmdir"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want rmdir.Options
	}{
		{"defaults", nil, rmdir.Options{}},
		{"parents", []string{"-p"}, rmdir.Options{Parents: true}},
		{"long parents", []string{"--parents"}, rmdir.Options{Parents: true}},
		{"verbose", []string{"-v"}, rmdir.Options{Verbose: true}},
		{"long verbose", []string{"--verbose"}, rmdir.Options{Verbose: true}},
		{"ignore non-empty", []string{"--ignore-fail-on-non-empty"}, rmdir.Options{IgnoreNonEmpty: true}},
		{"combined", []string{"-pv", "--ignore-fail-on-non-empty"}, rmdir.Options{Parents: true, Verbose: true, IgnoreNonEmpty: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(rmdir.Name, pflag.ContinueOnError)
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
