package codec

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/base64"
	"github.com/braswelljr/gnux/core/basenc"
)

var b64 = Tool{Name: base64.Name, GNU: "base64", Encoding: "base64", Sibling: "base32x",
	Example: "hello", Encoded: "aGVsbG8=", New: base64.New}

func options(t *testing.T, args ...string) (basenc.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return BuildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want basenc.Options
	}{
		{nil, basenc.Options{}},
		{[]string{"-d"}, basenc.Options{Decode: true}},
		{[]string{"--ignore-garbage"}, basenc.Options{IgnoreGarbage: true}},
		{[]string{"-w", "10"}, basenc.Options{Wrap: 10}},
		{[]string{"-w", "0"}, basenc.Options{NoWrap: true}},
	}
	for _, tt := range tests {
		got, err := options(t, tt.args...)
		if err != nil || got != tt.want {
			t.Errorf("BuildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
		}
	}
	for _, w := range []string{"x", "-1", ""} {
		_, err := options(t, "--wrap="+w)
		var fatal *fatalError
		if !errors.As(err, &fatal) || fatal.msg != "invalid wrap size: '"+w+"'" {
			t.Errorf("-w %q: %v", w, err)
		}
	}
}

func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := b64.Spec().Command(core.Streams{In: strings.NewReader("hello"), Out: &out, Err: &errb})
	c.SetArgs([]string{"-w", "x"})
	if err := c.Execute(); !errors.Is(err, core.ErrFailed) || errb.String() != "base64x: invalid wrap size: 'x'\n" {
		t.Fatalf("-w x: %v %q", err, errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := b64.Meta()
	if m.Name != base64.Name || !strings.HasPrefix(m.Use, base64.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example %+v", e)
		}
	}
}
