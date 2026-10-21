package stdbufx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/stdbuf"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want stdbuf.Options
	}{
		{[]string{"-oL"}, stdbuf.Options{Output: "L"}},
		{[]string{"-i", "0", "--error=4K"}, stdbuf.Options{Input: "0", Error: "4K"}},
		{[]string{"--out=L", "cmd", "-o"}, stdbuf.Options{Output: "L"}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(stdbuf.Name, pflag.ContinueOnError)
		Register(fs)
		fs.SetInterspersed(false)
		if err := tool.ParseFlagSet(fs, tt.args, true); err != nil {
			t.Fatalf("parse %v: %v", tt.args, err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
