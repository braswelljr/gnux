package chmodx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chmod"
)

func parse(t *testing.T, args ...string) (parsed, error) {
	t.Helper()
	fs := pflag.NewFlagSet(chmod.Name, pflag.ContinueOnError)
	fs.Bool("help", false, "")
	fs.Bool("version", false, "")
	Register(fs)
	return parseArgs(fs, args)
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		opts     chmod.Options
		operands []string
	}{
		{"plain", []string{"755", "f"}, chmod.Options{}, []string{"755", "f"}},
		{"flags", []string{"-cfvR", "u+x", "f"}, chmod.Options{Changes: true, Silent: true, Verbose: true, Recursive: true}, []string{"u+x", "f"}},
		{"quiet", []string{"--quiet", "644", "f"}, chmod.Options{Silent: true}, []string{"644", "f"}},
		{"reference", []string{"--reference", "r", "f"}, chmod.Options{Reference: "r"}, []string{"f"}},
		{"reference=", []string{"--reference=r", "f"}, chmod.Options{Reference: "r"}, []string{"f"}},
		{"mode as option", []string{"-w", "f"}, chmod.Options{DiagnoseSurprises: true}, []string{"-w", "f"}},
		{"modes as options combine", []string{"-rw", "f", "-u+x,g-w"}, chmod.Options{DiagnoseSurprises: true}, []string{"-rw,-u+x,g-w", "f"}},
		{"mode option with flags", []string{"-R", "-x", "d"}, chmod.Options{Recursive: true, DiagnoseSurprises: true}, []string{"-x", "d"}},
		{"after --", []string{"--", "-w", "f"}, chmod.Options{}, []string{"-w", "f"}},
		{"-P", []string{"-R", "-P", "u+w", "d"}, chmod.Options{Recursive: true, Traversal: chmod.TraversePhysical}, []string{"u+w", "d"}},
		{"-L", []string{"-RL", "u+w", "d"}, chmod.Options{Recursive: true, Traversal: chmod.TraverseLogical}, []string{"u+w", "d"}},
		{"-P beats -L", []string{"-RLP", "u+w", "d"}, chmod.Options{Recursive: true, Traversal: chmod.TraversePhysical}, []string{"u+w", "d"}},
		{"-P beats -L in any order", []string{"-RPL", "u+w", "d"}, chmod.Options{Recursive: true, Traversal: chmod.TraversePhysical}, []string{"u+w", "d"}},
		{"-H beats -L", []string{"-RLH", "u+w", "d"}, chmod.Options{Recursive: true, Traversal: chmod.TraverseCommandLine}, []string{"u+w", "d"}},
		{"no-preserve-root", []string{"--no-preserve-root", "-R", "u+w", "/"}, chmod.Options{Recursive: true, NoPreserveRoot: true}, []string{"u+w", "/"}},
		{"preserve-root wins", []string{"--no-preserve-root", "--preserve-root", "u+w", "/"}, chmod.Options{}, []string{"u+w", "/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parse(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if p.opts != tt.opts || !reflect.DeepEqual(p.operands, tt.operands) {
				t.Fatalf("parseArgs(%q) = %+v %q, want %+v %q", tt.args, p.opts, p.operands, tt.opts, tt.operands)
			}
		})
	}
}

func TestParseArgsInvalid(t *testing.T) {
	_, err := parse(t, "-w", "--reference=r", "f")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "cannot combine mode and --reference options" {
		t.Fatalf("mode and reference: %v", err)
	}
	if _, err := parse(t, "-Z", "f"); !errors.As(err, &usage) {
		t.Fatalf("unknown option: %v", err)
	}
	if p, _ := parse(t, "--help"); !p.help {
		t.Fatal("--help not seen")
	}
}

func TestCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not map onto Windows")
	}
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-v", "u+x", f})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if info, _ := os.Stat(f); info.Mode().Perm() != 0o700 {
		t.Fatalf("mode = %v", info.Mode())
	}
	if want := "mode of '" + f + "' changed from 0600 (rw-------) to 0700 (rwx------)\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	out.Reset()
	c = Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-R", "--help"})
	if err := c.Execute(); err != nil || !strings.Contains(out.String(), "--reference") {
		t.Fatalf("--help: %v %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != chmod.Name || !strings.HasPrefix(m.Use, chmod.Name+" ") {
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
