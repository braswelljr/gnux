package ttyx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tty"
)

func TestBuildOptions(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want tty.Options
	}{
		{nil, tty.Options{}},
		{[]string{"-s"}, tty.Options{Silent: true}},
		{[]string{"--silent"}, tty.Options{Silent: true}},
		{[]string{"--quiet"}, tty.Options{Silent: true}},
	} {
		fs := pflag.NewFlagSet(tty.Name, pflag.ContinueOnError)
		Register(fs)
		if err := fs.Parse(tt.args); err != nil {
			t.Fatal(err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs(nil)
	err := c.Execute()
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != tty.ExitNotTTY || out.String() != "not a tty\n" {
		t.Fatalf("err = %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != tty.Name || !strings.HasPrefix(m.Use, tty.Name+" ") || Spec().UsageExit != 2 {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
}
