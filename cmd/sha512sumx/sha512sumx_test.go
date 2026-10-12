package sha512sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sha512sum"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(sha512sum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-c", "--quiet"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (sha512sum.Options{Check: true, Quiet: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.String() != "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f  -\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != sha512sum.Name || Spec().GNU != "sha512sum" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
