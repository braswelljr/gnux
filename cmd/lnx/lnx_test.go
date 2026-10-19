package lnx

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ln"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want ln.Options
	}{
		{"defaults", nil, ln.Options{Physical: true}},
		{"symbolic", []string{"-s"}, ln.Options{Symbolic: true, Physical: true}},
		{"symbolic long", []string{"--symbolic"}, ln.Options{Symbolic: true, Physical: true}},
		{"force", []string{"-f"}, ln.Options{Force: true, Physical: true}},
		{"interactive", []string{"-i"}, ln.Options{Interactive: true, Physical: true}},
		{"no-dereference", []string{"-n"}, ln.Options{NoDereference: true, Physical: true}},
		{"no-dereference long", []string{"--no-dereference"}, ln.Options{NoDereference: true, Physical: true}},
		{"verbose", []string{"-v"}, ln.Options{Verbose: true, Physical: true}},
		{"relative", []string{"-r"}, ln.Options{Relative: true, Physical: true}},
		{"target directory", []string{"-t", "dir"}, ln.Options{TargetDirectory: "dir", Physical: true}},
		{"target directory long", []string{"--target-directory=dir"}, ln.Options{TargetDirectory: "dir", Physical: true}},
		{"no target directory", []string{"-T"}, ln.Options{NoTargetDirectory: true, Physical: true}},
		{"logical", []string{"-L"}, ln.Options{Logical: true}},
		{"physical", []string{"-P"}, ln.Options{Physical: true}},
		{"combined -sfn", []string{"-sfn"}, ln.Options{Symbolic: true, Force: true, NoDereference: true, Physical: true}},
		{"combined -srv", []string{"-srv"}, ln.Options{Symbolic: true, Relative: true, Verbose: true, Physical: true}},

		// Precedence: order is invisible, so -i beats -f and -L beats -P.
		{"-i beats -f", []string{"-i", "-f"}, ln.Options{Interactive: true, Physical: true}},
		{"-i beats -f reversed", []string{"-f", "-i"}, ln.Options{Interactive: true, Physical: true}},
		{"-L beats -P", []string{"-P", "-L"}, ln.Options{Logical: true}},

		{"-b", []string{"-b"}, ln.Options{Physical: true, Backup: core.Backup{Control: core.ExistingBackup}}},
		{"--backup=numbered", []string{"--backup=numbered"}, ln.Options{Physical: true, Backup: core.Backup{Control: core.NumberedBackup}}},
		{"-S", []string{"-S", ".old"}, ln.Options{Physical: true, Backup: core.Backup{Control: core.ExistingBackup, Suffix: ".old"}}},
	}
	t.Setenv("VERSION_CONTROL", "")
	t.Setenv("SIMPLE_BACKUP_SUFFIX", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(ln.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got, err := buildOptions(fs); err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	spec := Spec()
	if spec.Name() != "lnx" || spec.GNU != "ln" {
		t.Fatalf("spec = %q replacing %q, want lnx replacing ln", spec.Name(), spec.GNU)
	}
	if !strings.HasPrefix(spec.Meta.Use, "lnx ") {
		t.Fatalf("Use = %q, want it to start with the tool name", spec.Meta.Use)
	}

	c := spec.Command(core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
	c.SetArgs(nil)
	var usage *core.UsageError
	if err := c.Execute(); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error for a missing operand", err)
	}
}
