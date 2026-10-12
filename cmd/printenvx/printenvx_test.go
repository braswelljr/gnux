package printenvx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printenv"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want printenv.Options
	}{
		{"defaults", nil, printenv.Options{}},
		{"null", []string{"-0"}, printenv.Options{Null: true}},
		{"null long", []string{"--null"}, printenv.Options{Null: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(printenv.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	t.Setenv("GNUX_PRINTENV_TEST", "value")
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-0", "GNUX_PRINTENV_TEST"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if out.String() != "value\x00" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != printenv.Name || !strings.HasPrefix(m.Use, printenv.Name+" ") {
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
