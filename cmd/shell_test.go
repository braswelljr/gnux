package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shell"
)

func TestRunTool(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if status, ok := RunTool("rmx", []string{f}, core.Discard()); !ok || status != 0 {
		t.Fatalf("RunTool(rmx) = %d, %v", status, ok)
	}
	if _, err := os.Lstat(f); !os.IsNotExist(err) {
		t.Fatalf("file should be gone (err=%v)", err)
	}
	if _, ok := RunTool("nosuchtool", nil, core.Discard()); ok {
		t.Fatal("RunTool must report an unknown tool")
	}
}

func TestShellCommand(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.tmp"), filepath.Join(dir, "b.tmp")
	for _, f := range []string{a, b} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	glob := filepath.ToSlash(dir) + "/*.tmp"

	t.Run("-c runs tools by either name, with globbing", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "shell", "-c", "rm "+glob+" && pwdx")
		if status != 0 {
			t.Fatalf("status=%d stderr=%q", status, errb)
		}
		for _, f := range []string{a, b} {
			if _, err := os.Lstat(f); !os.IsNotExist(err) {
				t.Errorf("%s should be gone", f)
			}
		}
	})

	t.Run("status of a failing tool propagates", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "shell", "-c", "rmx "+filepath.ToSlash(dir)+"/missing")
		if status != 1 || !strings.HasPrefix(errb, "rmx: cannot remove") {
			t.Fatalf("status=%d stderr=%q", status, errb)
		}
	})

	t.Run("script file", func(t *testing.T) {
		script := filepath.Join(dir, "s.gnux")
		if err := os.WriteFile(script, []byte("# demo\npwdx\nexit 5\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		status, out, _ := run(t, "gnux", "shell", script)
		if status != 5 || strings.TrimSpace(out) == "" {
			t.Fatalf("status=%d out=%q", status, out)
		}
	})

	t.Run("missing script", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "shell", filepath.Join(dir, "nope"))
		if status != 127 || !strings.HasPrefix(errb, "gnux: ") {
			t.Fatalf("status=%d stderr=%q", status, errb)
		}
	})

	t.Run("usage error names gnux shell", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "shell", "a", "b")
		if status != 2 || !strings.Contains(errb, "Try 'gnux shell --help'") {
			t.Fatalf("status=%d stderr=%q", status, errb)
		}
	})

	t.Run("shell is not a tool", func(t *testing.T) {
		if _, ok := Lookup("shell"); ok {
			t.Fatal("shell must not be reachable by argv[0] or listed as a tool")
		}
	})
}

func TestNewShell(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	sh := NewShell(shell.Options{ErrExit: true}, core.Discard())
	sh.Set("F", f)
	if err := sh.RunString("rmx $F"); err != nil {
		t.Fatalf("RunString: %v", err)
	}
	if _, err := os.Lstat(f); !os.IsNotExist(err) {
		t.Fatalf("file should be gone (err=%v)", err)
	}
}

func TestShellHelpAndGnux(t *testing.T) {
	tests := []struct {
		script, want string
	}{
		{"help rmx", "rmx (GNU rm) - Remove"},
		{"help rm -f", "rmx force - "},
		{"gnux help catx", "catx (GNU cat)"},
		{"gnux --list | headx -1", "basenamex\n"},
		{"help", "Builtins: cd [DIR], exit [STATUS], help [TOOL [OPTION]]."},
	}
	for _, tt := range tests {
		status, out, errb := run(t, "gnux", "shell", "-c", tt.script)
		if status != 0 || errb != "" || !strings.Contains(out, tt.want) {
			t.Errorf("%q: status=%d stdout=%q stderr=%q", tt.script, status, out, errb)
		}
	}
	if status, _, errb := run(t, "gnux", "shell", "-c", "help nosuchtool"); status != 1 ||
		!strings.HasPrefix(errb, "gnux help: no help for 'nosuchtool'") {
		t.Errorf("help nosuchtool: status=%d stderr=%q", status, errb)
	}
}
