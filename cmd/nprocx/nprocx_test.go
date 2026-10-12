package nprocx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nproc"
)

func options(t *testing.T, args ...string) (nproc.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(nproc.Name, pflag.ContinueOnError)
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
		want nproc.Options
	}{
		{"defaults", nil, nproc.Options{}},
		{"all", []string{"--all"}, nproc.Options{All: true}},
		{"ignore", []string{"--ignore=2"}, nproc.Options{Ignore: 2}},
		{"ignore separate", []string{"--ignore", "3"}, nproc.Options{Ignore: 3}},
		{"both", []string{"--all", "--ignore=1"}, nproc.Options{All: true, Ignore: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	for _, v := range []string{"x", "-1", "", "1.5"} {
		_, err := options(t, "--ignore="+v)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "invalid number: '"+v+"'" {
			t.Fatalf("--ignore=%s: err = %v", v, err)
		}
	}
}

func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"--ignore=100000"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if out.String() != "1\n" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != nproc.Name || !strings.HasPrefix(m.Use, nproc.Name+" ") {
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
