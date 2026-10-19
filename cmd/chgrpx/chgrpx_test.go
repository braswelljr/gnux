package chgrpx

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/chgrp"
)

func options(t *testing.T, args ...string) chgrp.Options {
	t.Helper()
	fs := pflag.NewFlagSet(chgrp.Name, pflag.ContinueOnError)
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
		want chgrp.Options
	}{
		{"defaults", nil, chgrp.Options{}},
		{"-cfvR", []string{"-cfvR"}, chgrp.Options{Changes: true, Silent: true, Verbose: true, Recursive: true}},
		{"--quiet", []string{"--quiet"}, chgrp.Options{Silent: true}},
		{"-h", []string{"-h"}, chgrp.Options{NoDereference: true}},
		{"--dereference", []string{"--dereference"}, chgrp.Options{Dereference: true}},
		{"-h wins", []string{"--dereference", "-h"}, chgrp.Options{NoDereference: true}},
		{"-h wins before", []string{"-h", "--dereference"}, chgrp.Options{NoDereference: true}},
		{"--reference", []string{"--reference", "r"}, chgrp.Options{Reference: "r"}},
		{"--from", []string{"--from=alice:staff"}, chgrp.Options{From: "alice:staff"}},

		{"-H", []string{"-H"}, chgrp.Options{Traversal: chgrp.TraverseCommandLine}},
		{"-L", []string{"-L"}, chgrp.Options{Traversal: chgrp.TraverseLogical}},
		{"-P beats -L", []string{"-L", "-P"}, chgrp.Options{Traversal: chgrp.TraversePhysical}},
		{"-P beats -H before", []string{"-P", "-H"}, chgrp.Options{Traversal: chgrp.TraversePhysical}},
		{"-H beats -L", []string{"-H", "-L"}, chgrp.Options{Traversal: chgrp.TraverseCommandLine}},
		{"no-preserve-root", []string{"--no-preserve-root"}, chgrp.Options{NoPreserveRoot: true}},
		{"preserve-root wins", []string{"--no-preserve-root", "--preserve-root"}, chgrp.Options{}},
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
	if m.Name != chgrp.Name || !strings.HasPrefix(m.Use, chgrp.Name+" ") {
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
