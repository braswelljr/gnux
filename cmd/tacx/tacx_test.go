package tacx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tac"
)

func options(t *testing.T, args ...string) tac.Options {
	t.Helper()
	fs := pflag.NewFlagSet(tac.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want tac.Options
	}{
		{"defaults", nil, tac.Options{}},
		{"-b", []string{"-b"}, tac.Options{Before: true}},
		{"-r", []string{"-r"}, tac.Options{Regex: true}},
		{"-s", []string{"-s", ","}, tac.Options{Separator: ",", SeparatorSet: true}},
		{"empty -s", []string{"--separator="}, tac.Options{SeparatorSet: true}},
		{"all", []string{"-br", "-s", "x+"}, tac.Options{Before: true, Regex: true, Separator: "x+", SeparatorSet: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options(t, tt.args...); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("a,b,"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-s", ","})
	if err := c.Execute(); err != nil || out.String() != "b,a," {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != tac.Name || !strings.HasPrefix(m.Use, tac.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatal("Meta is missing prose")
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
