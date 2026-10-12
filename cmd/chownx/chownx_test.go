package chownx

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/chown"
)

func options(t *testing.T, args ...string) chown.Options {
	t.Helper()
	fs := pflag.NewFlagSet(chown.Name, pflag.ContinueOnError)
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
		want chown.Options
	}{
		{"defaults", nil, chown.Options{}},
		{"-cfvR", []string{"-cfvR"}, chown.Options{Changes: true, Silent: true, Verbose: true, Recursive: true}},
		{"--quiet", []string{"--quiet"}, chown.Options{Silent: true}},
		{"-h", []string{"-h"}, chown.Options{NoDereference: true}},
		{"--dereference", []string{"--dereference"}, chown.Options{Dereference: true}},
		{"-h wins", []string{"--dereference", "-h"}, chown.Options{NoDereference: true}},
		{"-h wins before", []string{"-h", "--dereference"}, chown.Options{NoDereference: true}},
		{"--reference", []string{"--reference", "r"}, chown.Options{Reference: "r"}},
		{"--from", []string{"--from=bob:"}, chown.Options{From: "bob:"}},
		{"-H", []string{"-H"}, chown.Options{Traversal: chown.TraverseCommandLine}},
		{"-L", []string{"-L"}, chown.Options{Traversal: chown.TraverseLogical}},
		{"-P beats -L", []string{"-L", "-P"}, chown.Options{Traversal: chown.TraversePhysical}},
		{"-P beats -H before", []string{"-P", "-H"}, chown.Options{Traversal: chown.TraversePhysical}},
		{"-H beats -L", []string{"-H", "-L"}, chown.Options{Traversal: chown.TraverseCommandLine}},
		{"no-preserve-root", []string{"--no-preserve-root"}, chown.Options{NoPreserveRoot: true}},
		{"preserve-root wins", []string{"--no-preserve-root", "--preserve-root"}, chown.Options{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options(t, tt.args...); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != chown.Name || !strings.HasPrefix(m.Use, chown.Name+" ") {
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
