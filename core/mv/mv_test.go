package mv_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mv"
)

func newEngine(opts mv.Options) (*mv.Mv, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	m := mv.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return m, &out, &errb
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func wantContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content of %s = %q, want %q", path, got, want)
	}
}

func wantMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s should not exist (err = %v)", path, err)
	}
}

// run executes the engine and requires success with a silent stderr.
func run(t *testing.T, opts mv.Options, args ...string) string {
	t.Helper()
	m, out, errb := newEngine(opts)
	if err := m.Run(args); err != nil {
		t.Fatalf("Run(%v): %v (stderr: %s)", args, err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
	return out.String()
}

// fail executes the engine, requires core.ErrFailed and returns stderr.
func fail(t *testing.T, opts mv.Options, args ...string) string {
	t.Helper()
	m, _, errb := newEngine(opts)
	if err := m.Run(args); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("Run(%v) err = %v, want ErrFailed (stderr: %s)", args, err, errb.String())
	}
	return errb.String()
}

// crossDevice makes every rename fail the way rename(2) does between
// filesystems, forcing the copy-and-remove path.
func crossDevice(oldpath, newpath string) error {
	return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.EXDEV}
}

func TestRename(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, src, "data")

	out := run(t, mv.Options{Verbose: true}, src, dst)
	if want := "renamed '" + src + "' -> '" + dst + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantContent(t, dst, "data")
	wantMissing(t, src)
}

func TestMoveIntoDirectory(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	tree := filepath.Join(dir, "tree")
	write(t, a, "A")
	write(t, b, "B")
	write(t, filepath.Join(tree, "f"), "F")
	dest := mkdir(t, filepath.Join(dir, "dest"))

	if out := run(t, mv.Options{}, a, b, tree+string(filepath.Separator), dest); out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	wantContent(t, filepath.Join(dest, "a"), "A")
	wantContent(t, filepath.Join(dest, "b"), "B")
	wantContent(t, filepath.Join(dest, "tree", "f"), "F")
	for _, p := range []string{a, b, tree} {
		wantMissing(t, p)
	}

	t.Run("-t", func(t *testing.T) {
		back := mkdir(t, filepath.Join(dir, "back"))
		run(t, mv.Options{TargetDirectory: back}, filepath.Join(dest, "a"), filepath.Join(dest, "b"))
		wantContent(t, filepath.Join(back, "a"), "A")
		wantContent(t, filepath.Join(back, "b"), "B")
	})

	t.Run("-T replaces an empty directory instead of nesting", func(t *testing.T) {
		empty := mkdir(t, filepath.Join(dir, "empty"))
		run(t, mv.Options{NoTargetDirectory: true}, filepath.Join(dest, "tree"), empty)
		wantContent(t, filepath.Join(empty, "f"), "F")
		wantMissing(t, filepath.Join(empty, "tree"))
	})
}

func TestUsageErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	write(t, file, "x")

	tests := []struct {
		name string
		opts mv.Options
		args []string
		want string
	}{
		{"no operands", mv.Options{}, nil, "missing file operand"},
		{"one operand", mv.Options{}, []string{"a"}, "missing destination file operand after 'a'"},
		{"target not a directory", mv.Options{}, []string{"a", "b", file}, "target '" + file + "' is not a directory"},
		{"extra operand with -T", mv.Options{NoTargetDirectory: true}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{"-t with -T", mv.Options{TargetDirectory: dir, NoTargetDirectory: true}, []string{"a"},
			"cannot combine --target-directory (-t) and --no-target-directory (-T)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newEngine(tt.opts)
			err := m.Run(tt.args)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a usage error", err)
			}
			if usage.Msg != tt.want {
				t.Fatalf("usage = %q, want %q", usage.Msg, tt.want)
			}
		})
	}
}

func TestDiagnostics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	other := filepath.Join(dir, "other")
	sub := mkdir(t, filepath.Join(dir, "sub"))
	inner := mkdir(t, filepath.Join(sub, "inner"))
	full := mkdir(t, filepath.Join(dir, "full"))
	missing := filepath.Join(dir, "missing")
	write(t, file, "x")
	write(t, other, "y")
	write(t, filepath.Join(full, "sub", "keep"), "k")

	tests := []struct {
		name string
		opts mv.Options
		args []string
		want string
	}{
		{"missing source", mv.Options{}, []string{missing, filepath.Join(dir, "out")},
			"mvx: cannot stat '" + missing + "': No such file or directory\n"},
		{"same file", mv.Options{}, []string{file, file},
			"mvx: '" + file + "' and '" + file + "' are the same file\n"},
		{"same file via directory", mv.Options{}, []string{file, dir},
			"mvx: '" + file + "' and '" + file + "' are the same file\n"},
		{"directory into itself", mv.Options{}, []string{sub, inner},
			"mvx: cannot move '" + sub + "' to a subdirectory of itself, '" + filepath.Join(inner, "sub") + "'\n"},
		{"directory into itself, new name", mv.Options{}, []string{sub, filepath.Join(sub, "new")},
			"mvx: cannot move '" + sub + "' to a subdirectory of itself, '" + filepath.Join(sub, "new") + "'\n"},
		{"file over directory", mv.Options{NoTargetDirectory: true}, []string{file, sub},
			"mvx: cannot overwrite directory '" + sub + "' with non-directory\n"},
		{"directory over file", mv.Options{NoTargetDirectory: true}, []string{sub, file},
			"mvx: cannot overwrite non-directory '" + file + "' with directory '" + sub + "'\n"},
		{"directory over non-empty directory", mv.Options{}, []string{sub, full},
			"mvx: cannot move '" + sub + "' to '" + filepath.Join(full, "sub") + "': Directory not empty\n"},
		{"missing target directory", mv.Options{TargetDirectory: missing}, []string{file},
			"mvx: target directory '" + missing + "': No such file or directory\n"},
		{"just created", mv.Options{TargetDirectory: full}, []string{file, filepath.Join(dir, ".", "file")},
			"mvx: will not overwrite just-created '" + filepath.Join(full, "file") + "' with '" + filepath.Join(dir, ".", "file") + "'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fail(t, tt.opts, tt.args...); got != tt.want {
				t.Fatalf("stderr = %q, want %q", got, tt.want)
			}
		})
	}

	wantContent(t, other, "y")
	wantContent(t, filepath.Join(full, "sub", "keep"), "k")
	if _, err := os.Stat(inner); err != nil {
		t.Fatalf("source directory was disturbed: %v", err)
	}
}

func TestOverwritePolicies(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name     string
		opts     mv.Options
		answer   bool
		srcTime  time.Time
		dstTime  time.Time
		moved    bool
		prompted bool
	}{
		{"default replaces", mv.Options{}, true, recent, old, true, false},
		{"force replaces", mv.Options{Force: true}, false, recent, old, true, false},
		{"no-clobber keeps destination", mv.Options{NoClobber: true}, true, recent, old, false, false},
		{"update skips newer destination", mv.Options{Update: true}, true, old, recent, false, false},
		{"update replaces older destination", mv.Options{Update: true}, true, recent, old, true, false},
		{"interactive declined", mv.Options{Interactive: true}, false, recent, old, false, true},
		{"interactive accepted", mv.Options{Interactive: true}, true, recent, old, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
			write(t, src, "new")
			write(t, dst, "old")
			if err := os.Chtimes(src, tt.srcTime, tt.srcTime); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(dst, tt.dstTime, tt.dstTime); err != nil {
				t.Fatal(err)
			}

			var questions []string
			m, out, errb := newEngine(tt.opts)
			m.Verbose = true
			m.PromptFn = func(q string) bool {
				questions = append(questions, q)
				return tt.answer
			}
			if err := m.Run([]string{src, dst}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}

			if tt.moved {
				wantContent(t, dst, "new")
				wantMissing(t, src)
				if want := "renamed '" + src + "' -> '" + dst + "'\n"; out.String() != want {
					t.Fatalf("stdout = %q, want %q", out.String(), want)
				}
			} else {
				wantContent(t, dst, "old")
				wantContent(t, src, "new")
				if out.Len() != 0 {
					t.Fatalf("stdout = %q, want empty", out.String())
				}
			}

			if tt.prompted {
				if want := "overwrite '" + dst + "'?"; len(questions) != 1 || questions[0] != want {
					t.Fatalf("questions = %q, want [%q]", questions, want)
				}
			} else if len(questions) != 0 {
				t.Fatalf("unexpected prompt %q", questions)
			}
		})
	}
}

func TestMoveReplacesSymlinkNotItsTarget(t *testing.T) {
	dir := t.TempDir()
	src, target, link := filepath.Join(dir, "src"), filepath.Join(dir, "target"), filepath.Join(dir, "link")
	write(t, src, "new")
	write(t, target, "target")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, tt := range []struct {
		name   string
		rename func(string, string) error
	}{
		{"rename", os.Rename},
		{"cross-device", crossDevice},
	} {
		t.Run(tt.name, func(t *testing.T) {
			write(t, src, "new")
			m, _, errb := newEngine(mv.Options{})
			m.RenameFn = tt.rename
			if err := m.Run([]string{src, link}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantContent(t, target, "target")
			wantContent(t, link, "new")
			if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink != 0 {
				t.Fatal("destination is still a symlink")
			}
			// Put the link back for the next case.
			if err := os.Remove(link); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStripTrailingSlashes(t *testing.T) {
	dir := t.TempDir()
	src := mkdir(t, filepath.Join(dir, "src"))
	dst := filepath.Join(dir, "dst")

	out := run(t, mv.Options{Verbose: true, StripTrailingSlashes: true}, src+string(filepath.Separator), dst)
	if want := "renamed '" + src + "' -> '" + dst + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestRenameFailure(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "data")

	m, _, errb := newEngine(mv.Options{})
	m.RenameFn = func(oldpath, newpath string) error {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: os.ErrPermission}
	}
	if err := m.Run([]string{src, dst}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "mvx: cannot move '" + src + "' to '" + dst + "': Permission denied\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	wantContent(t, src, "data")
}

func TestCrossDeviceFile(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "data")
	write(t, dst, "old")
	stamp := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(src, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	m, out, errb := newEngine(mv.Options{Verbose: true})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dst}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	want := "copied '" + src + "' -> '" + dst + "'\nremoved '" + src + "'\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	wantContent(t, dst, "data")
	wantMissing(t, src)
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("mtime = %v, want %v (attributes must survive the copy)", info.ModTime(), stamp)
	}
}

func TestCrossDeviceDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "a"), "A")
	write(t, filepath.Join(src, "sub", "b"), "B")
	linked := os.Symlink("a", filepath.Join(src, "link")) == nil
	dest := mkdir(t, filepath.Join(dir, "dest"))

	m, _, errb := newEngine(mv.Options{})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dest}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	moved := filepath.Join(dest, "src")
	wantContent(t, filepath.Join(moved, "a"), "A")
	wantContent(t, filepath.Join(moved, "sub", "b"), "B")
	if linked {
		if got, err := os.Readlink(filepath.Join(moved, "link")); err != nil || got != "a" {
			t.Fatalf("readlink = %q, %v; want the symlink itself to be moved", got, err)
		}
	}
	wantMissing(t, src)
}

func TestCrossDeviceNeverMergesIntoNonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "a"), "A")
	dst := filepath.Join(dir, "dst")
	write(t, filepath.Join(dst, "keep"), "K")

	m, _, errb := newEngine(mv.Options{NoTargetDirectory: true})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dst}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	want := "mvx: cannot move '" + src + "' to '" + dst + "': Directory not empty\n"
	if errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	wantContent(t, filepath.Join(src, "a"), "A")
	wantContent(t, filepath.Join(dst, "keep"), "K")
}

func TestCrossDeviceKeepsSourceWhenCopyFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, src, "data")
	dst := filepath.Join(dir, "no-such-dir", "dst")

	m, _, errb := newEngine(mv.Options{})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dst}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	want := "mvx: cannot create regular file '" + dst + "': No such file or directory\n"
	if errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	wantContent(t, src, "data")
}

func TestNoCopy(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "data")

	m, out, errb := newEngine(mv.Options{NoCopy: true, Verbose: true})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dst}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "mvx: cannot move '" + src + "' to '" + dst + "': Invalid cross-device link\n"; errb.String() != want || out.Len() != 0 {
		t.Fatalf("stderr = %q, stdout = %q", errb.String(), out.String())
	}
	wantContent(t, src, "data")
	wantMissing(t, dst)

	// On one file system it is an ordinary rename.
	m, _, errb = newEngine(mv.Options{NoCopy: true})
	if err := m.Run([]string{src, dst}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantContent(t, dst, "data")
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	simple := core.Backup{Control: core.SimpleBackup}

	write(t, a, "A")
	write(t, b, "B")
	m, out, errb := newEngine(mv.Options{Backup: simple, Verbose: true})
	if err := m.Run([]string{a, b}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	if want := "renamed '" + a + "' -> '" + b + "' (backup: '" + b + "~')\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	wantContent(t, b, "A")
	wantContent(t, b+"~", "B")
	wantMissing(t, a)

	// A directory is backed up too, and one kind may replace the other.
	src, e := mkdir(t, filepath.Join(dir, "src")), mkdir(t, filepath.Join(dir, "e"))
	write(t, filepath.Join(e, "keep"), "K")
	m, _, errb = newEngine(mv.Options{Backup: simple, NoTargetDirectory: true})
	if err := m.Run([]string{src, e}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantContent(t, filepath.Join(e+"~", "keep"), "K")
	write(t, a, "A2")
	m, _, errb = newEngine(mv.Options{Backup: core.Backup{Control: core.NumberedBackup}, NoTargetDirectory: true})
	if err := m.Run([]string{a, e}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	wantContent(t, e, "A2")
	if info, err := os.Stat(e + ".~1~"); err != nil || !info.IsDir() {
		t.Fatalf("numbered backup of the directory: %v", err)
	}

	// Moving a file's simple backup onto it would destroy the source.
	m, _, errb = newEngine(mv.Options{Backup: simple})
	if err := m.Run([]string{b + "~", b}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "mvx: backing up '" + b + "' might destroy source;  '" + b + "~' not moved\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}

	// The same name is still refused; -n cannot be combined.
	m, _, errb = newEngine(mv.Options{Backup: simple})
	if err := m.Run([]string{b, b}); !errors.Is(err, core.ErrFailed) || !strings.Contains(errb.String(), "are the same file") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
	m, _, _ = newEngine(mv.Options{Backup: simple, NoClobber: true})
	var usage *core.UsageError
	if err := m.Run([]string{a, b}); !errors.As(err, &usage) {
		t.Fatalf("err = %v", err)
	}
}

func TestBackupAcrossDevices(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "new")
	write(t, dst, "old")
	m, out, errb := newEngine(mv.Options{Backup: core.Backup{Control: core.SimpleBackup}, Verbose: true})
	m.RenameFn = crossDevice
	if err := m.Run([]string{src, dst}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	want := "copied '" + src + "' -> '" + dst + "' (backup: '" + dst + "~')\nremoved '" + src + "'\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	wantContent(t, dst, "new")
}
