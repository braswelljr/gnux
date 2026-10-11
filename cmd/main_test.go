package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// run executes gnux with argv and returns the exit status and both streams.
func run(t *testing.T, argv ...string) (status int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	status = Main(argv, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return status, out.String(), errb.String()
}

func TestProgName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"rmx", "rmx"},
		{"/usr/local/bin/rmx", "rmx"},
		{`C:\tools\rmx.exe`, "rmx"},
		{"./bin/gnux", "gnux"},
		{"RMX.EXE", "rmx"},
	}
	for _, tt := range tests {
		if got := progName(tt.in); got != tt.want {
			t.Errorf("progName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRegistryIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Tools() {
		name := s.Name()
		if name == "" || !strings.HasSuffix(name, "x") {
			t.Errorf("tool name %q must be the GNU name with an x suffix", name)
		}
		if s.GNU+"x" != name {
			t.Errorf("tool %q mirrors GNU %q; want name %q", name, s.GNU, s.GNU+"x")
		}
		if !strings.HasPrefix(s.Meta.Use, name) {
			t.Errorf("tool %q: Use %q must start with the tool name", name, s.Meta.Use)
		}
		if s.Meta.Short == "" || s.Meta.Long == "" || len(s.Meta.Examples) == 0 {
			t.Errorf("tool %q: Short, Long and at least one example are required", name)
		}
		if s.Run == nil {
			t.Errorf("tool %q has no Run", name)
		}
		for _, n := range []string{name, s.GNU} {
			if seen[n] {
				t.Errorf("name %q is registered twice", n)
			}
			seen[n] = true
		}
	}
}

func TestEveryToolAnswersHelpAndVersion(t *testing.T) {
	for _, s := range Tools() {
		name := s.Name()
		t.Run(name, func(t *testing.T) {
			status, out, errb := run(t, name, "--version")
			if status != 0 || !strings.Contains(out, name) {
				t.Errorf("--version: status=%d stdout=%q stderr=%q", status, out, errb)
			}
			status, out, errb = run(t, name, "--help")
			if status != 0 || !strings.Contains(out, name) {
				t.Errorf("--help: status=%d stdout=%q stderr=%q", status, out, errb)
			}
		})
	}
}

func TestDispatch(t *testing.T) {
	newFile := func(t *testing.T) string {
		t.Helper()
		f := filepath.Join(t.TempDir(), "f")
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return f
	}
	gone := func(t *testing.T, f string) {
		t.Helper()
		if _, err := os.Lstat(f); !os.IsNotExist(err) {
			t.Fatalf("%s should have been removed (err=%v)", f, err)
		}
	}

	t.Run("subcommand", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "gnux", "rmx", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("subcommand by GNU name", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "gnux", "rm", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("argv0", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "/usr/local/bin/rmx", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("argv0 by GNU name", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "rm", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})
}

func TestDiagnostics(t *testing.T) {
	t.Run("engine failure exits 1 without a usage hint", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope")
		status, _, errb := run(t, "rmx", missing)
		if status != 1 {
			t.Fatalf("status = %d, want 1", status)
		}
		if !strings.HasPrefix(errb, "rmx: cannot remove") || strings.Contains(errb, "--help") {
			t.Fatalf("stderr = %q", errb)
		}
	})

	t.Run("usage error names the tool", func(t *testing.T) {
		for _, argv := range [][]string{{"rmx"}, {"gnux", "rmx"}, {"rmx", "--no-such-flag"}} {
			status, _, errb := run(t, argv...)
			if status != 1 {
				t.Errorf("%v: status = %d, want 1", argv, status)
			}
			if !strings.HasPrefix(errb, "rmx: ") || !strings.Contains(errb, "Try 'rmx --help' for more information.") {
				t.Errorf("%v: stderr = %q", argv, errb)
			}
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "nosuchtool")
		if status != 1 || !strings.Contains(errb, "nosuchtool") {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
	})
}

func TestRootListAndVersion(t *testing.T) {
	status, out, _ := run(t, "gnux", "--list")
	if status != 0 {
		t.Fatalf("--list status = %d", status)
	}
	if got, want := len(strings.Fields(out)), len(Tools()); got != want {
		t.Fatalf("--list printed %d names, want %d", got, want)
	}

	status, out, _ = run(t, "gnux", "--version")
	if status != 0 || !strings.HasPrefix(out, "gnux ") {
		t.Fatalf("--version: status=%d out=%q", status, out)
	}
}
