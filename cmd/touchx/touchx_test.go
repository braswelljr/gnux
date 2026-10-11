package touchx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/touch"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want touch.Options
	}{
		{"defaults", nil, touch.Options{}},
		{"access", []string{"-a"}, touch.Options{Access: true}},
		{"modify", []string{"-m"}, touch.Options{Modify: true}},
		{"no create", []string{"-c"}, touch.Options{NoCreate: true}},
		{"long no create", []string{"--no-create"}, touch.Options{NoCreate: true}},
		{"date", []string{"-d", "2024-01-02 15:04"}, touch.Options{Date: "2024-01-02 15:04"}},
		{"long date", []string{"--date=@0"}, touch.Options{Date: "@0"}},
		{"stamp", []string{"-t", "202401021504.05"}, touch.Options{Stamp: "202401021504.05"}},
		{"reference", []string{"-r", "ref"}, touch.Options{Reference: "ref"}},
		{"long reference", []string{"--reference=ref"}, touch.Options{Reference: "ref"}},
		{"no dereference", []string{"-h"}, touch.Options{NoDereference: true}},
		{"long no dereference", []string{"--no-dereference"}, touch.Options{NoDereference: true}},
		{"time", []string{"--time=atime"}, touch.Options{Time: "atime"}},
		{"combined", []string{"-amch", "-r", "ref"}, touch.Options{Access: true, Modify: true, NoCreate: true, NoDereference: true, Reference: "ref"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(touch.Name, pflag.ContinueOnError)
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
