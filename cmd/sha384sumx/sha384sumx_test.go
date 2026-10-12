package sha384sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha384sum"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(sha384sum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-c", "--quiet"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (sha384sum.Options{Check: true, Quiet: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.String() != "cb00753f45a35e8bb5a03d699ac65007272c32ab0eded1631a8b605a43ff5bed8086072ba1e7cc2358baeca134c825a7  -\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != sha384sum.Name || Spec().GNU != "sha384sum" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
