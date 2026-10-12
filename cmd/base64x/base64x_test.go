package base64x

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/base64"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(base64.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-d", "-i"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (base64.Options{Decode: true, IgnoreGarbage: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	for _, tt := range []struct {
		args     []string
		in, want string
	}{
		{nil, "hello", "aGVsbG8=\n"},
		{[]string{"-w", "0"}, "hello", "aGVsbG8="},
		{[]string{"-d"}, "aGVsbG8=\n", "hello"},
	} {
		var out bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(tt.in), Out: &out, Err: &bytes.Buffer{}})
		c.SetArgs(tt.args)
		if err := c.Execute(); err != nil || out.String() != tt.want {
			t.Errorf("%v: %v, stdout = %q, want %q", tt.args, err, out.String(), tt.want)
		}
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != base64.Name || Spec().GNU != "base64" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
