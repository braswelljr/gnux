package fsys_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core/fsys"
)

// layout makes rw/, ro/ and out/ in a temporary directory and a Rooted file
// system of rw (writable) and ro (read-only), working in rw.
func layout(t *testing.T) (f *fsys.Rooted, rw, ro, out string) {
	t.Helper()
	base := t.TempDir()
	if real, err := filepath.EvalSymlinks(base); err == nil {
		base = real
	}
	rw, ro, out = filepath.Join(base, "rw"), filepath.Join(base, "ro"), filepath.Join(base, "out")
	for _, d := range []string{rw, ro, out} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "f"), []byte(filepath.Base(d)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f, err := fsys.NewRooted([]string{rw}, []string{ro}, rw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(f.Close)
	return f, rw, ro, out
}

func denied(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS)
}

func TestRootedReachesOnlyItsRoots(t *testing.T) {
	f, rw, ro, out := layout(t)

	for name, want := range map[string]string{"f": "rw", filepath.Join(rw, "f"): "rw", filepath.Join(ro, "f"): "ro", "../ro/f": "ro"} {
		data, err := f.ReadFile(name)
		if err != nil || string(data) != want {
			t.Errorf("ReadFile(%q) = %q, %v; want %q", name, data, err, want)
		}
	}
	// The top of the disk: "/" is relative on Windows, so name its volume.
	top := filepath.VolumeName(rw) + string(filepath.Separator)
	for _, name := range []string{filepath.Join(out, "f"), "../out/f", top} {
		if _, err := f.ReadFile(name); !denied(err) {
			t.Errorf("ReadFile(%q) err = %v, want permission denied", name, err)
		}
	}
	if err := f.WriteFile(filepath.Join(ro, "g"), nil, 0o644); !denied(err) {
		t.Errorf("writing in the read-only root: %v", err)
	}
	if err := f.Remove(filepath.Join(ro, "f")); !denied(err) {
		t.Errorf("removing in the read-only root: %v", err)
	}
	if err := f.WriteFile("new", []byte("x"), 0o644); err != nil {
		t.Errorf("writing in the writable root: %v", err)
	}
}

func TestRootedSymlinks(t *testing.T) {
	f, rw, ro, out := layout(t)
	links := map[string]string{
		"abs-in":   filepath.Join(rw, "f"), // absolute, inside: os.Root alone refuses it
		"abs-ro":   filepath.Join(ro, "f"), // into the other root
		"rel-ro":   filepath.Join("..", "ro", "f"),
		"abs-out":  filepath.Join(out, "f"), // out of every root
		"rel-out":  filepath.Join("..", "out", "f"),
		"dir-link": ro,
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(rw, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	for name, want := range map[string]string{"abs-in": "rw", "abs-ro": "ro", "rel-ro": "ro", filepath.Join("dir-link", "f"): "ro"} {
		if data, err := f.ReadFile(name); err != nil || string(data) != want {
			t.Errorf("through %s: %q, %v; want %q", name, data, err, want)
		}
	}
	for _, name := range []string{"abs-out", "rel-out"} {
		if _, err := f.ReadFile(name); !denied(err) {
			t.Errorf("through %s err = %v, want permission denied", name, err)
		}
		// The link itself is in the root: it can be examined and removed.
		if info, err := f.Lstat(name); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("Lstat(%s) = %v, %v", name, info, err)
		}
	}
	if err := f.WriteFile(filepath.Join("dir-link", "g"), nil, 0o644); !denied(err) {
		t.Errorf("writing through a link into the read-only root: %v", err)
	}
	if real, err := f.EvalSymlinks("abs-ro"); err != nil || real != filepath.Join(ro, "f") {
		t.Errorf("EvalSymlinks = %q, %v", real, err)
	}
}

func TestRootedRenameAcrossRoots(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	for _, d := range []string{a, b} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(a, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := fsys.NewRooted([]string{a, b}, nil, a)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Two roots are two devices to a rename: mv then copies.
	if err := f.Rename("f", filepath.Join(b, "f")); !errors.Is(err, syscall.EXDEV) {
		t.Errorf("Rename across roots err = %v, want EXDEV", err)
	}
	if err := f.Rename("f", "g"); err != nil {
		t.Errorf("Rename within a root: %v", err)
	}
}

func TestRootedWorkingDirectory(t *testing.T) {
	f, rw, ro, out := layout(t)
	if err := os.Mkdir(filepath.Join(rw, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := f.Chdir("sub"); err != nil {
		t.Fatal(err)
	}
	if wd, _ := f.Getwd(); wd != filepath.Join(rw, "sub") {
		t.Errorf("Getwd = %q", wd)
	}
	if data, err := f.ReadFile("../f"); err != nil || string(data) != "rw" {
		t.Errorf("relative to the new directory: %q, %v", data, err)
	}
	if err := f.Chdir(ro); err != nil {
		t.Errorf("Chdir into the read-only root: %v", err)
	}
	if err := f.Chdir(out); !denied(err) {
		t.Errorf("Chdir out of the roots err = %v", err)
	}
	// The process's own directory never moved.
	if wd, _ := os.Getwd(); wd == filepath.Join(rw, "sub") {
		t.Error("Chdir changed the process's directory")
	}
}

func TestRootedReadDirInfo(t *testing.T) {
	f, rw, _, _ := layout(t)
	entries, err := f.ReadDir(rw)
	if err != nil || len(entries) != 1 {
		t.Fatalf("ReadDir = %v, %v", entries, err)
	}
	// Info examines the file when asked, through the roots: a file removed
	// meanwhile is reported missing.
	if err := os.Remove(filepath.Join(rw, "f")); err != nil {
		t.Fatal(err)
	}
	if _, err := entries[0].Info(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Info after removal err = %v, want not exist", err)
	}
}

func TestRootedTempFiles(t *testing.T) {
	f, rw, _, out := layout(t)
	dir, err := f.MkdirTemp(rw, "t-*")
	if err != nil || filepath.Dir(dir) != rw {
		t.Fatalf("MkdirTemp = %q, %v", dir, err)
	}
	file, err := f.CreateTemp(dir, "x*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if _, err := f.CreateTemp(out, "x"); !denied(err) {
		t.Errorf("CreateTemp out of the roots err = %v", err)
	}
}
