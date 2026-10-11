package rmx

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (rm.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet("rmx", pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func mustOptions(t *testing.T, args ...string) rm.Options {
	t.Helper()
	o, err := options(t, args...)
	if err != nil {
		t.Fatalf("buildOptions %v: %v", args, err)
	}
	return o
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want rm.Options
	}{
		{"defaults", nil, rm.Options{Interactive: rm.PromptDefault}},
		{"force", []string{"-f"}, rm.Options{Force: true, Interactive: rm.PromptNever}},
		{"interactive each", []string{"-i"}, rm.Options{Interactive: rm.PromptAlways}},
		{"interactive once", []string{"-I"}, rm.Options{Interactive: rm.PromptOnce}},
		{"recursive short", []string{"-r"}, rm.Options{Recursive: true}},
		{"recursive upper", []string{"-R"}, rm.Options{Recursive: true}},
		{"dir", []string{"-d"}, rm.Options{Dir: true}},
		{"verbose", []string{"-v"}, rm.Options{Verbose: true}},
		{"one file system", []string{"--one-file-system"}, rm.Options{OneFileSystem: true}},
		{"no preserve root", []string{"--no-preserve-root", "-r"}, rm.Options{Recursive: true, NoPreserveRoot: true}},
		{"interactive=never", []string{"--interactive=never"}, rm.Options{Interactive: rm.PromptNever}},
		{"interactive=once", []string{"--interactive=once"}, rm.Options{Interactive: rm.PromptOnce}},
		{"interactive bare", []string{"--interactive"}, rm.Options{Interactive: rm.PromptAlways}},
		{"combined -rvf", []string{"-rvf"}, rm.Options{Force: true, Recursive: true, Verbose: true, Interactive: rm.PromptNever}},

		// Precedence: -i (always) is the most cautious and wins over -f/-I.
		{"force then -i wins", []string{"-f", "-i"}, rm.Options{Force: true, Interactive: rm.PromptAlways}},
		{"force then -I", []string{"-fI"}, rm.Options{Force: true, Interactive: rm.PromptOnce}},
		{"-I then -i wins", []string{"-Ii"}, rm.Options{Interactive: rm.PromptAlways}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustOptions(t, tt.args...)
			if got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalidInteractive(t *testing.T) {
	if _, err := options(t, "--interactive=bogus"); err == nil {
		t.Fatal("expected an error for --interactive=bogus")
	}
}

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: io.Discard})
	c.SetArgs([]string{"--version"})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	if !strings.Contains(out.String(), "rmx") {
		t.Fatalf("version output = %q, want it to mention rmx", out.String())
	}
}

func TestMissingOperand(t *testing.T) {
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})
	c.SetArgs(nil)
	var usage *core.UsageError
	if err := c.Execute(); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error", err)
	}
}
