package sha224sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha224sum"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(sha224sum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-c", "--quiet"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (sha224sum.Options{Check: true, Quiet: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.String() != "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7  -\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != sha224sum.Name || Spec().GNU != "sha224sum" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
