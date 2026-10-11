package rmdir_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rmdir"
)

func newEngine(opts rmdir.Options) (*rmdir.Rmdir, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	r := rmdir.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return r, &out, &errb
}

func mustExist(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Lstat(path)
	if got := err == nil; got != want {
		t.Fatalf("existence of %q = %v, want %v", path, got, want)
	}
}

// tree creates dir/a/b/c inside a fresh working directory and returns the
// relative names, so -p has a path it can walk all the way up.
func tree(t *testing.T) (a, ab, abc string) {
	t.Helper()
	t.Chdir(t.TempDir())
	a = "a"
	ab = filepath.Join(a, "b")
	abc = filepath.Join(ab, "c")
	if err := os.MkdirAll(abc, 0o755); err != nil {
		t.Fatal(err)
	}
	return a, ab, abc
}

func TestRemoveEmpty(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	for _, d := range []string{a, b} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	r, out, errb := newEngine(rmdir.Options{})
	if err := r.Run([]string{a, b}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustExist(t, a, false)
	mustExist(t, b, false)
	if out.Len() != 0 || errb.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
	}
}

func TestRemoveErrors(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "full")
	if err := os.MkdirAll(filepath.Join(full, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	tests := []struct {
		name, path, want string
	}{
		{"non-empty", full, "rmdirx: failed to remove '" + full + "': Directory not empty\n"},
		{"regular file", file, "rmdirx: failed to remove '" + file + "': Not a directory\n"},
		{"missing", missing, "rmdirx: failed to remove '" + missing + "': No such file or directory\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, errb := newEngine(rmdir.Options{})
			if err := r.Run([]string{tt.path}); !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if errb.String() != tt.want {
				t.Fatalf("stderr = %q, want %q", errb.String(), tt.want)
			}
		})
	}
	mustExist(t, full, true)
	mustExist(t, file, true)
}

func TestIgnoreFailOnNonEmpty(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "full")
	if err := os.MkdirAll(filepath.Join(full, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	r, _, errb := newEngine(rmdir.Options{IgnoreNonEmpty: true})
	if err := r.Run([]string{full}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
	mustExist(t, full, true)

	// Only the non-empty failure is ignored; other errors still count.
	r, _, errb = newEngine(rmdir.Options{IgnoreNonEmpty: true})
	if err := r.Run([]string{missing}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "rmdirx: failed to remove '" + missing + "': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestParents(t *testing.T) {
	a, ab, abc := tree(t)

	r, out, errb := newEngine(rmdir.Options{Parents: true, Verbose: true})
	if err := r.Run([]string{abc}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustExist(t, a, false)
	want := "rmdirx: removing directory, '" + abc + "'\n" +
		"rmdirx: removing directory, '" + ab + "'\n" +
		"rmdirx: removing directory, '" + a + "'\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestParentsStopsAtNonEmpty(t *testing.T) {
	a, ab, abc := tree(t)
	if err := os.WriteFile(filepath.Join(a, "keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rmdir.Options{Parents: true})
	if err := r.Run([]string{abc}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "rmdirx: failed to remove directory '" + a + "': Directory not empty\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	mustExist(t, ab, false)
	mustExist(t, a, true)
}

func TestParentsIgnoreNonEmpty(t *testing.T) {
	a, ab, abc := tree(t)
	if err := os.WriteFile(filepath.Join(a, "keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rmdir.Options{Parents: true, IgnoreNonEmpty: true})
	if err := r.Run([]string{abc + string(filepath.Separator)}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
	mustExist(t, ab, false)
	mustExist(t, a, true)
}

func TestParentsAbsoluteStopsBelowRoot(t *testing.T) {
	dir := t.TempDir()
	leaf := filepath.Join(dir, "x", "y")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}
	// A sibling keeps the temporary directory itself from being removed, so
	// the walk ends there with a diagnostic instead of climbing further.
	if err := os.WriteFile(filepath.Join(dir, "keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rmdir.Options{Parents: true})
	if err := r.Run([]string{leaf}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "rmdirx: failed to remove directory '" + dir + "': Directory not empty\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	mustExist(t, filepath.Join(dir, "x"), false)
	mustExist(t, dir, true)
}

func TestSymlinkToDirectoryIsNotADirectory(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	r, _, errb := newEngine(rmdir.Options{})
	if err := r.Run([]string{link}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "rmdirx: failed to remove '" + link + "': Not a directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	mustExist(t, link, true)
	mustExist(t, real, true)
}

func TestMissingOperand(t *testing.T) {
	r, _, _ := newEngine(rmdir.Options{})
	err := r.Run(nil)
	var uerr *core.UsageError
	if !errors.As(err, &uerr) || uerr.Msg != "missing operand" {
		t.Fatalf("err = %v, want usage error %q", err, "missing operand")
	}
}
