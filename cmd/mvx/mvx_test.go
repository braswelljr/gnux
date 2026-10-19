package mvx

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mv"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (mv.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(mv.Name, pflag.ContinueOnError)
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
		want mv.Options
	}{
		{"defaults", nil, mv.Options{}},
		{"force", []string{"-f"}, mv.Options{Force: true}},
		{"force long", []string{"--force"}, mv.Options{Force: true}},
		{"interactive", []string{"-i"}, mv.Options{Interactive: true}},
		{"interactive long", []string{"--interactive"}, mv.Options{Interactive: true}},
		{"no-clobber", []string{"-n"}, mv.Options{NoClobber: true}},
		{"no-clobber long", []string{"--no-clobber"}, mv.Options{NoClobber: true}},
		{"update short", []string{"-u"}, mv.Options{Update: true}},
		{"update bare", []string{"--update"}, mv.Options{Update: true}},
		{"update=older", []string{"--update=older"}, mv.Options{Update: true}},
		{"update=none", []string{"--update=none"}, mv.Options{NoClobber: true}},
		{"update=all", []string{"--update=all"}, mv.Options{}},
		{"verbose", []string{"-v"}, mv.Options{Verbose: true}},
		{"target directory", []string{"-t", "dir"}, mv.Options{TargetDirectory: "dir"}},
		{"target directory long", []string{"--target-directory=dir"}, mv.Options{TargetDirectory: "dir"}},
		{"no target directory", []string{"-T"}, mv.Options{NoTargetDirectory: true}},
		{"strip trailing slashes", []string{"--strip-trailing-slashes"}, mv.Options{StripTrailingSlashes: true}},
		{"no copy", []string{"--no-copy"}, mv.Options{NoCopy: true}},
		{"-b", []string{"-b"}, mv.Options{Backup: core.Backup{Control: core.ExistingBackup}}},
		{"--backup=t", []string{"--backup=t"}, mv.Options{Backup: core.Backup{Control: core.NumberedBackup}}},
		{"--backup bare", []string{"--backup"}, mv.Options{Backup: core.Backup{Control: core.ExistingBackup}}},
		{"-S", []string{"-S", ".bak"}, mv.Options{Backup: core.Backup{Control: core.ExistingBackup, Suffix: ".bak"}}},
		{"--suffix", []string{"--suffix=.k"}, mv.Options{Backup: core.Backup{Control: core.ExistingBackup, Suffix: ".k"}}},
		{"combined -uv keeps the default WHEN", []string{"-uv"}, mv.Options{Update: true, Verbose: true}},

		// Precedence: the most cautious option wins, whatever the order.
		{"-i beats -f", []string{"-i", "-f"}, mv.Options{Interactive: true}},
		{"-i beats -f reversed", []string{"-f", "-i"}, mv.Options{Interactive: true}},
		{"-n beats -i", []string{"-n", "-i"}, mv.Options{NoClobber: true}},
		{"-n beats -f", []string{"-f", "-n"}, mv.Options{NoClobber: true}},
		{"-n beats both", []string{"-fin"}, mv.Options{NoClobber: true}},
		{"update=none beats -f", []string{"-f", "--update=none"}, mv.Options{NoClobber: true}},
	}
	t.Setenv("VERSION_CONTROL", "")
	t.Setenv("SIMPLE_BACKUP_SUFFIX", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalidUpdate(t *testing.T) {
	_, err := options(t, "--update=bogus")
	var usage *core.UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error", err)
	}
	if !strings.Contains(usage.Msg, "invalid argument 'bogus' for '--update'") {
		t.Fatalf("usage = %q", usage.Msg)
	}
}

func TestSpec(t *testing.T) {
	spec := Spec()
	if spec.Name() != "mvx" || spec.GNU != "mv" {
		t.Fatalf("spec = %q replacing %q, want mvx replacing mv", spec.Name(), spec.GNU)
	}
	if !strings.HasPrefix(spec.Meta.Use, "mvx ") {
		t.Fatalf("Use = %q, want it to start with the tool name", spec.Meta.Use)
	}

	c := spec.Command(core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
	c.SetArgs(nil)
	var usage *core.UsageError
	if err := c.Execute(); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error for a missing operand", err)
	}
}
