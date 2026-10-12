package base32x

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/base32"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(base32.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-d", "-i"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (base32.Options{Decode: true, IgnoreGarbage: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	for _, tt := range []struct {
		args     []string
		in, want string
	}{
		{nil, "hello", "NBSWY3DP\n"},
		{[]string{"-w", "0"}, "hello", "NBSWY3DP"},
		{[]string{"-d"}, "NBSWY3DP\n", "hello"},
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
	if m := Meta(); m.Name != base32.Name || Spec().GNU != "base32" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
