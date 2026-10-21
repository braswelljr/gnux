package mkfifox

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/mkfifo"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want mkfifo.Options
	}{
		{nil, mkfifo.Options{}},
		{[]string{"-m", "600"}, mkfifo.Options{Mode: "600"}},
		{[]string{"-Z"}, mkfifo.Options{}},
		{[]string{"--context"}, mkfifo.Options{}},
		{[]string{"--context=user_u:object_r:tmp_t"}, mkfifo.Options{Context: "user_u:object_r:tmp_t"}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(mkfifo.Name, pflag.ContinueOnError)
		Register(fs)
		if err := tool.ParseFlagSet(fs, tt.args, false); err != nil {
			t.Fatalf("parse %v: %v", tt.args, err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
