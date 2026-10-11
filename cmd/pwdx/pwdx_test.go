package pwdx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/pwd"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want pwd.Options
	}{
		{"defaults", nil, pwd.Options{}},
		{"logical", []string{"-L"}, pwd.Options{Logical: true}},
		{"physical", []string{"-P"}, pwd.Options{}},
		{"long logical", []string{"--logical"}, pwd.Options{Logical: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(pwd.Name, pflag.ContinueOnError)
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
