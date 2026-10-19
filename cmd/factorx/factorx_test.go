package factorx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/factor"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want factor.Options
	}{
		{"defaults", nil, factor.Options{}},
		{"exponents", []string{"-h"}, factor.Options{Exponents: true}},
		{"exponents long", []string{"--exponents"}, factor.Options{Exponents: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(factor.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v; want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	if out, err := command(t, "", "-h", "360"); err != nil || out != "360: 2^3 3^2 5\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if out, err := command(t, "6\n"); err != nil || out != "6: 2 3\n" {
		t.Fatalf("stdin out = %q, err = %v", out, err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != factor.Name || !strings.HasPrefix(m.Use, factor.Name+" ") {
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

// command runs the tool through its cobra command, as the binary does.
func command(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}
