package b2sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/b2sum"
)

func TestBuildOptions(t *testing.T) {
	fs := pflag.NewFlagSet(b2sum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse([]string{"-c", "--quiet"}); err != nil {
		t.Fatal(err)
	}
	got, err := buildOptions(fs)
	if err != nil || got != (b2sum.Options{Check: true, Quiet: true}) {
		t.Fatalf("buildOptions = %+v, %v", got, err)
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.String() != "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923  -\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	if m := Meta(); m.Name != b2sum.Name || Spec().GNU != "b2sum" {
		t.Fatalf("Name = %q, GNU = %q", m.Name, Spec().GNU)
	}
}
