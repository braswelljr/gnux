package mknodx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/mknod"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want mknod.Options
	}{
		{nil, mknod.Options{}},
		{[]string{"--mode=u=rw"}, mknod.Options{Mode: "u=rw"}},
		{[]string{"-Z", "-m", "644"}, mknod.Options{Mode: "644"}},
		{[]string{"--context=ctx"}, mknod.Options{Context: "ctx"}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(mknod.Name, pflag.ContinueOnError)
		Register(fs)
		if err := tool.ParseFlagSet(fs, tt.args, false); err != nil {
			t.Fatalf("parse %v: %v", tt.args, err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
