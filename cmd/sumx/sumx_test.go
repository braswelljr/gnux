package sumx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sum"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want sum.Options
	}{
		{"defaults", nil, sum.Options{}},
		{"bsd", []string{"-r"}, sum.Options{}},
		{"sysv", []string{"-s"}, sum.Options{SysV: true}},
		{"sysv long", []string{"--sysv"}, sum.Options{SysV: true}},
		{"sysv wins after bsd", []string{"-r", "-s"}, sum.Options{SysV: true}},
		{"sysv wins before bsd", []string{"-s", "-r"}, sum.Options{SysV: true}},
		{"bundled", []string{"-rs"}, sum.Options{SysV: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(sum.Name, pflag.ContinueOnError)
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
	if out, err := command(t, "hello\n", "-s"); err != nil || out != "542 1\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if out, err := command(t, "hello\n"); err != nil || out != "36979     1\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != sum.Name || !strings.HasPrefix(m.Use, sum.Name+" ") {
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
