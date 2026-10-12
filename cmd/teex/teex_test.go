package teex

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tee"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (tee.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(tee.Name, pflag.ContinueOnError)
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
		want tee.Options
	}{
		{"defaults", nil, tee.Options{}},
		{"append", []string{"-a"}, tee.Options{Append: true}},
		{"append long", []string{"--append"}, tee.Options{Append: true}},
		{"ignore interrupts", []string{"-i"}, tee.Options{IgnoreInterrupts: true}},
		{"ignore interrupts long", []string{"--ignore-interrupts"}, tee.Options{IgnoreInterrupts: true}},
		{"p", []string{"-p"}, tee.Options{OutputError: tee.OutputErrorWarnNoPipe}},
		{"output-error bare", []string{"--output-error"}, tee.Options{OutputError: tee.OutputErrorWarnNoPipe}},
		{"output-error=warn", []string{"--output-error=warn"}, tee.Options{OutputError: tee.OutputErrorWarn}},
		{"output-error=warn-nopipe", []string{"--output-error=warn-nopipe"}, tee.Options{OutputError: tee.OutputErrorWarnNoPipe}},
		{"output-error=exit", []string{"--output-error=exit"}, tee.Options{OutputError: tee.OutputErrorExit}},
		{"output-error=exit-nopipe", []string{"--output-error=exit-nopipe"}, tee.Options{OutputError: tee.OutputErrorExitNoPipe}},
		// An explicit MODE wins over -p, whatever the order.
		{"mode after p", []string{"-p", "--output-error=exit"}, tee.Options{OutputError: tee.OutputErrorExit}},
		{"mode before p", []string{"--output-error=exit", "-p"}, tee.Options{OutputError: tee.OutputErrorExit}},
		{"combined", []string{"-aip"}, tee.Options{Append: true, IgnoreInterrupts: true, OutputError: tee.OutputErrorWarnNoPipe}},
	}
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

func TestBuildOptionsInvalid(t *testing.T) {
	for _, mode := range []string{"sometimes", "", "WARN"} {
		_, err := options(t, "--output-error="+mode)
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("--output-error=%s: err = %v, want a UsageError", mode, err)
		}
		if want := "invalid argument '" + mode + "' for '--output-error'\nValid arguments are:"; !strings.HasPrefix(usage.Msg, want) {
			t.Fatalf("message = %q, want prefix %q", usage.Msg, want)
		}
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")

	for _, args := range [][]string{{path}, {"-a", path}} {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader("line\n"), Out: &out, Err: &errb})
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatalf("Execute(%v): %v (stderr %q)", args, err, errb.String())
		}
		if out.String() != "line\n" {
			t.Fatalf("stdout = %q", out.String())
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "line\nline\n" {
		t.Fatalf("file = %q, want the second run appended", got)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != tee.Name || !strings.HasPrefix(m.Use, tee.Name+" ") {
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
