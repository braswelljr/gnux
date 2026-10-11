package mkdir_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mkdir"
)

func newEngine(opts mkdir.Options) (*mkdir.Mkdir, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	m := mkdir.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return m, &out, &errb
}

func mustDir(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
	return info
}

func TestCreate(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")

	m, out, errb := newEngine(mkdir.Options{})
	if err := m.Run([]string{a, b}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustDir(t, a)
	mustDir(t, b)
	if out.Len() != 0 || errb.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
	}
}

func TestCreateErrors(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(dir, "missing", "child")

	tests := []struct {
		name string
		opts mkdir.Options
		path string
		want string
	}{
		{"existing directory", mkdir.Options{}, existing, "mkdirx: cannot create directory '" + existing + "': File exists\n"},
		{"existing file", mkdir.Options{}, file, "mkdirx: cannot create directory '" + file + "': File exists\n"},
		{"existing file with parents", mkdir.Options{Parents: true}, file, "mkdirx: cannot create directory '" + file + "': File exists\n"},
		{"missing parent", mkdir.Options{}, orphan, "mkdirx: cannot create directory '" + orphan + "': No such file or directory\n"},
		{"file as parent", mkdir.Options{Parents: true}, filepath.Join(file, "x", "y"), "mkdirx: cannot create directory '" + file + "': File exists\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, out, errb := newEngine(tt.opts)
			if err := m.Run([]string{tt.path}); !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if errb.String() != tt.want {
				t.Fatalf("stderr = %q, want %q", errb.String(), tt.want)
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
		})
	}
}

func TestFailureDoesNotStopLaterOperands(t *testing.T) {
	dir := t.TempDir()
	bad, good := filepath.Join(dir, "x", "y"), filepath.Join(dir, "good")

	m, _, _ := newEngine(mkdir.Options{})
	if err := m.Run([]string{bad, good}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	mustDir(t, good)
}

func TestParents(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	ab := filepath.Join(a, "b")
	abc := filepath.Join(ab, "c")

	m, out, errb := newEngine(mkdir.Options{Parents: true, Verbose: true})
	if err := m.Run([]string{abc}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustDir(t, abc)
	want := "mkdirx: created directory '" + a + "'\n" +
		"mkdirx: created directory '" + ab + "'\n" +
		"mkdirx: created directory '" + abc + "'\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}

	// A second run finds everything in place: no error and nothing reported.
	m, out, errb = newEngine(mkdir.Options{Parents: true, Verbose: true})
	if err := m.Run([]string{abc, abc + string(filepath.Separator)}); err != nil {
		t.Fatalf("second Run: %v (stderr: %s)", err, errb.String())
	}
	if out.Len() != 0 || errb.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
	}
}

func TestParentsThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	m, _, errb := newEngine(mkdir.Options{Parents: true})
	if err := m.Run([]string{filepath.Join(link, "sub"), link}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustDir(t, filepath.Join(real, "sub"))
}

func TestVerbose(t *testing.T) {
	a := filepath.Join(t.TempDir(), "a")

	m, out, _ := newEngine(mkdir.Options{Verbose: true})
	if err := m.Run([]string{a}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := "mkdirx: created directory '" + a + "'\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts mkdir.Options
		args []string
		want string
	}{
		{"missing operand", mkdir.Options{}, nil, "missing operand"},
		{"invalid octal mode", mkdir.Options{Mode: "999"}, []string{"x"}, "invalid mode '999'"},
		{"invalid symbolic mode", mkdir.Options{Mode: "u+q"}, []string{"x"}, "invalid mode 'u+q'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			m, _, _ := newEngine(tt.opts)
			err := m.Run(tt.args)
			var uerr *core.UsageError
			if !errors.As(err, &uerr) {
				t.Fatalf("err = %v, want UsageError", err)
			}
			if uerr.Msg != tt.want {
				t.Fatalf("usage = %q, want %q", uerr.Msg, tt.want)
			}
			if _, err := os.Lstat("x"); err == nil {
				t.Fatal("directory created despite the usage error")
			}
		})
	}
}
