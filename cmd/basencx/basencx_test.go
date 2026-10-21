package basencx

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basenc"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		enc  basenc.Encoding
		opts basenc.Options
	}{
		{"base16", []string{"--base16"}, basenc.Base16, basenc.Options{}},
		{"z85 decoding", []string{"--z85", "-d"}, basenc.Z85, basenc.Options{Decode: true}},
		{"the last encoding wins", []string{"--base16", "--base64"}, basenc.Base64, basenc.Options{}},
		{"the last encoding wins either way", []string{"--base64", "--base16"}, basenc.Base16, basenc.Options{}},
		{"an abbreviation", []string{"--base64u", "-w", "0"}, basenc.Base64URL, basenc.Options{NoWrap: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(basenc.Name, pflag.ContinueOnError)
			Register(fs)
			if err := tool.ParseFlagSet(fs, tt.args, false); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			enc, opts, err := buildOptions(fs)
			if err != nil || enc != tt.enc || opts != tt.opts {
				t.Fatalf("buildOptions(%v) = %v, %+v, %v", tt.args, enc, opts, err)
			}
		})
	}
}

func TestMissingEncoding(t *testing.T) {
	fs := pflag.NewFlagSet(basenc.Name, pflag.ContinueOnError)
	Register(fs)
	_, _, err := buildOptions(fs)
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing encoding type" {
		t.Fatalf("err = %v", err)
	}
}
