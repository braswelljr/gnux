package syncx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sync"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want sync.Options
	}{
		{"defaults", nil, sync.Options{}},
		{"data", []string{"-d"}, sync.Options{Data: true}},
		{"file system", []string{"--file-system"}, sync.Options{FileSystem: true}},
		{"both, refused by the engine", []string{"-df"}, sync.Options{Data: true, FileSystem: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(sync.Name, pflag.ContinueOnError)
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
