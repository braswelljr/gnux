package rm_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

// newEngine builds an engine wired to buffers with a non-terminal stdin, so
// PromptDefault never prompts unless a PromptFn is injected.
func newEngine(opts rm.Options) (*rm.Rm, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	r := rm.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return r, &out, &errb
}

func mustExist(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Lstat(path)
	got := err == nil
	if got != want {
		t.Fatalf("existence of %q = %v, want %v", path, got, want)
	}
}

func TestRemoveFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rm.Options{})
	if err := r.Run([]string{f}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustExist(t, f, false)
}

func TestRemoveNonexistent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")

	t.Run("without force is an error", func(t *testing.T) {
		r, _, errb := newEngine(rm.Options{})
		err := r.Run([]string{missing})
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if !strings.Contains(errb.String(), "No such file or directory") {
			t.Fatalf("stderr = %q", errb.String())
		}
	})

	t.Run("with force is silent", func(t *testing.T) {
		r, _, errb := newEngine(rm.Options{Force: true})
		if err := r.Run([]string{missing}); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if errb.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errb.String())
		}
	})
}

func TestRemoveDirectoryRequiresRecursive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rm.Options{})
	if err := r.Run([]string{dir}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if !strings.Contains(errb.String(), "Is a directory") {
		t.Fatalf("stderr = %q", errb.String())
	}
	mustExist(t, dir, true)
}

func TestRemoveEmptyDirWithDirFlag(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rm.Options{Dir: true})
	if err := r.Run([]string{empty}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustExist(t, empty, false)
}

func TestRemoveNonEmptyDirWithDirFlag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "full")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	r, _, errb := newEngine(rm.Options{Dir: true})
	if err := r.Run([]string{dir}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if !strings.Contains(errb.String(), "Directory not empty") {
		t.Fatalf("stderr = %q", errb.String())
	}
	mustExist(t, dir, true)
}

func TestRecursiveRemoval(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(root, "top.txt"),
		filepath.Join(root, "a", "mid.txt"),
		filepath.Join(nested, "leaf.txt"),
	} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	r, _, errb := newEngine(rm.Options{Recursive: true})
	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	mustExist(t, root, false)
}

func TestInteractivePrompt(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.txt")
	drop := filepath.Join(dir, "drop.txt")
	for _, f := range []string{keep, drop} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	r, _, _ := newEngine(rm.Options{Interactive: rm.PromptAlways})
	// Answer "yes" only for drop.txt.
	r.PromptFn = func(q string) bool { return strings.Contains(q, "drop.txt") }

	if err := r.Run([]string{keep, drop}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	mustExist(t, keep, true)
	mustExist(t, drop, false)
}

func TestPromptOnceDecline(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	if err := os.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}

	r, _, _ := newEngine(rm.Options{Recursive: true, Interactive: rm.PromptOnce})
	asked := 0
	r.PromptFn = func(string) bool { asked++; return false }

	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if asked != 1 {
		t.Fatalf("prompted %d times, want exactly 1", asked)
	}
	mustExist(t, root, true) // declined → nothing removed
}

func TestVerboseOutput(t *testing.T) {
	f := filepath.Join(t.TempDir(), "v.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, out, _ := newEngine(rm.Options{Verbose: true})
	if err := r.Run([]string{f}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "removed '"+f+"'") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestConcurrentBulkRemoval(t *testing.T) {
	dir := t.TempDir()
	var paths []string
	for _, name := range []string{"1", "2", "3", "4", "5", "6"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}

	r, _, errb := newEngine(rm.Options{Force: true, Interactive: rm.PromptNever})
	if err := r.Run(paths); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	for _, p := range paths {
		mustExist(t, p, false)
	}
}

func TestRefusesDotDirectory(t *testing.T) {
	r, _, errb := newEngine(rm.Options{Recursive: true})
	if err := r.Run([]string{"."}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if !strings.Contains(errb.String(), "refusing to remove") {
		t.Fatalf("stderr = %q", errb.String())
	}
}

func TestInjectedRemoveFn(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	if err := os.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var removed []string
	r, _, _ := newEngine(rm.Options{Recursive: true})
	r.RemoveFn = func(p string) error { removed = append(removed, p); return nil }

	if err := r.Run([]string{root}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Post-order: file, then its parent, then the root.
	want := []string{
		filepath.Join(root, "a", "f.txt"),
		filepath.Join(root, "a"),
		root,
	}
	if strings.Join(removed, "\n") != strings.Join(want, "\n") {
		t.Fatalf("removed = %v, want %v", removed, want)
	}
	mustExist(t, root, true) // RemoveFn was a no-op recorder
}
