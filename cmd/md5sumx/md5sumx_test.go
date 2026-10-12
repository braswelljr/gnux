package md5sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/md5sum"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(md5sum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-c", "--quiet"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (md5sum.Options{Check: true, Quiet: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.String() != "900150983cd24fb0d6963f7d28e17f72  -\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != md5sum.Name || Spec().GNU != "md5sum" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
