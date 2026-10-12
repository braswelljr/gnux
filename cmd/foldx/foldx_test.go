package foldx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fold"
)

func options(t *testing.T, args ...string) (fold.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(fold.Name, pflag.ContinueOnError)
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
		want fold.Options
	}{
		{"defaults", nil, fold.Options{}},
		{"-b", []string{"-b"}, fold.Options{Bytes: true}},
		{"-s", []string{"--spaces"}, fold.Options{Spaces: true}},
		{"-w", []string{"-w", "40"}, fold.Options{Width: 40}},
		{"obsolete", []string{"-72"}, fold.Options{Width: 72}},
		{"obsolete with flags", []string{"-bs20"}, fold.Options{Bytes: true, Spaces: true, Width: 20}},
		{"-w then obsolete", []string{"-w", "40", "-12"}, fold.Options{Width: 12}},
		{"obsolete then -w", []string{"-12", "-w", "40"}, fold.Options{Width: 40}},
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
	for _, w := range []string{"0", "x", "-3", "99999999999"} {
		_, err := options(t, "--width="+w)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "invalid number of columns: '"+w+"'" {
			t.Errorf("-w %s: %v", w, err)
		}
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("abcdef\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-4"})
	if err := c.Execute(); err != nil || out.String() != "abcd\nef\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != fold.Name || !strings.HasPrefix(m.Use, fold.Name+" ") {
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
