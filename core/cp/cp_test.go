package cp_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cp"
)

func newEngine(opts cp.Options) (*cp.Cp, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	c := cp.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return c, &out, &errb
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

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func isSymlink(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	return info.Mode()&os.ModeSymlink != 0
}

// run executes the engine and requires success with a silent stderr.
func run(t *testing.T, opts cp.Options, args ...string) string {
	t.Helper()
	c, out, errb := newEngine(opts)
	if err := c.Run(args); err != nil {
		t.Fatalf("Run(%v): %v (stderr: %s)", args, err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
	return out.String()
}

// fail executes the engine, requires core.ErrFailed and returns stderr.
func fail(t *testing.T, opts cp.Options, args ...string) string {
	t.Helper()
	c, _, errb := newEngine(opts)
	if err := c.Run(args); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("Run(%v) err = %v, want ErrFailed (stderr: %s)", args, err, errb.String())
	}
	return errb.String()
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, src, "hello")

	if out := run(t, cp.Options{}, src, dst); out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
	wantContent(t, dst, "hello")
	wantContent(t, src, "hello")
}

func TestCopyOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, src, "new")
	write(t, dst, "old, and longer")

	run(t, cp.Options{}, src, dst)
	wantContent(t, dst, "new")
}

func TestCopyIntoDirectoryVerbose(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	dest := mkdir(t, filepath.Join(dir, "dest"))
	write(t, a, "A")
	write(t, b, "B")

	out := run(t, cp.Options{Verbose: true}, a, b, dest)
	want := "'" + a + "' -> '" + filepath.Join(dest, "a") + "'\n" +
		"'" + b + "' -> '" + filepath.Join(dest, "b") + "'\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantContent(t, filepath.Join(dest, "a"), "A")
	wantContent(t, filepath.Join(dest, "b"), "B")
}

func TestUsageErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	write(t, file, "x")

	tests := []struct {
		name string
		opts cp.Options
		args []string
		want string
	}{
		{"no operands", cp.Options{}, nil, "missing file operand"},
		{"one operand", cp.Options{}, []string{"a"}, "missing destination file operand after 'a'"},
		{"target not a directory", cp.Options{}, []string{"a", "b", file}, "target '" + file + "' is not a directory"},
		{"extra operand with -T", cp.Options{NoTargetDirectory: true}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{"-t with -T", cp.Options{TargetDirectory: dir, NoTargetDirectory: true}, []string{"a"},
			"cannot combine --target-directory (-t) and --no-target-directory (-T)"},
		{"-l with -s", cp.Options{Link: true, SymbolicLink: true}, []string{"a", "b"}, "cannot make both hard and symbolic links"},
		{"--parents to a file", cp.Options{Parents: true}, []string{file, filepath.Join(dir, "new")},
			"with --parents, the destination must be a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, _ := newEngine(tt.opts)
			err := c.Run(tt.args)
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
	sub := mkdir(t, filepath.Join(dir, "sub"))
	inner := mkdir(t, filepath.Join(sub, "inner"))
	missing := filepath.Join(dir, "missing")
	write(t, file, "x")
	write(t, filepath.Join(inner, "file"), "y")

	tests := []struct {
		name string
		opts cp.Options
		args []string
		want string
	}{
		{"missing source", cp.Options{}, []string{missing, filepath.Join(dir, "out")},
			"cpx: cannot stat '" + missing + "': No such file or directory\n"},
		{"directory without -r", cp.Options{}, []string{sub, filepath.Join(dir, "out")},
			"cpx: -r not specified; omitting directory '" + sub + "'\n"},
		{"same file", cp.Options{}, []string{file, file},
			"cpx: '" + file + "' and '" + file + "' are the same file\n"},
		{"same file via directory", cp.Options{}, []string{file, dir},
			"cpx: '" + file + "' and '" + file + "' are the same file\n"},
		{"directory into itself", cp.Options{Recursive: true}, []string{sub, inner},
			"cpx: cannot copy a directory, '" + sub + "', into itself, '" + filepath.Join(inner, "sub") + "'\n"},
		{"directory into itself, new name", cp.Options{Recursive: true}, []string{sub, filepath.Join(sub, "copy")},
			"cpx: cannot copy a directory, '" + sub + "', into itself, '" + filepath.Join(sub, "copy") + "'\n"},
		{"file over directory", cp.Options{NoTargetDirectory: true}, []string{file, sub},
			"cpx: cannot overwrite directory '" + sub + "' with non-directory\n"},
		{"directory over file", cp.Options{Recursive: true, NoTargetDirectory: true}, []string{sub, file},
			"cpx: cannot overwrite non-directory '" + file + "' with directory '" + sub + "'\n"},
		{"missing target directory", cp.Options{TargetDirectory: missing}, []string{file},
			"cpx: target directory '" + missing + "': No such file or directory\n"},
		{"target directory is a file", cp.Options{TargetDirectory: file}, []string{file},
			"cpx: target directory '" + file + "': Not a directory\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fail(t, tt.opts, tt.args...); got != tt.want {
				t.Fatalf("stderr = %q, want %q", got, tt.want)
			}
		})
	}

	// Nothing may have been created inside the source by the refused copies.
	wantMissing(t, filepath.Join(inner, "sub"))
	wantMissing(t, filepath.Join(sub, "copy"))
}

func TestFailureDoesNotStopOtherOperands(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good")
	missing := filepath.Join(dir, "missing")
	dest := mkdir(t, filepath.Join(dir, "dest"))
	write(t, good, "ok")

	got := fail(t, cp.Options{}, missing, good, dest)
	if want := "cpx: cannot stat '" + missing + "': No such file or directory\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	wantContent(t, filepath.Join(dest, "good"), "ok")
}

func TestRecursiveCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "a.txt"), "A")
	write(t, filepath.Join(src, "sub", "b.txt"), "B")
	mkdir(t, filepath.Join(src, "empty"))
	dst := filepath.Join(dir, "dst")

	out := run(t, cp.Options{Recursive: true, Verbose: true}, src, dst)
	want := strings.Join([]string{
		"'" + src + "' -> '" + dst + "'",
		"'" + filepath.Join(src, "a.txt") + "' -> '" + filepath.Join(dst, "a.txt") + "'",
		"'" + filepath.Join(src, "empty") + "' -> '" + filepath.Join(dst, "empty") + "'",
		"'" + filepath.Join(src, "sub") + "' -> '" + filepath.Join(dst, "sub") + "'",
		"'" + filepath.Join(src, "sub", "b.txt") + "' -> '" + filepath.Join(dst, "sub", "b.txt") + "'",
	}, "\n") + "\n"
	if out != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", out, want)
	}
	wantContent(t, filepath.Join(dst, "a.txt"), "A")
	wantContent(t, filepath.Join(dst, "sub", "b.txt"), "B")
	if info, err := os.Stat(filepath.Join(dst, "empty")); err != nil || !info.IsDir() {
		t.Fatalf("empty directory not copied: %v", err)
	}

	t.Run("into an existing directory nests", func(t *testing.T) {
		run(t, cp.Options{Recursive: true}, src, dst)
		wantContent(t, filepath.Join(dst, "src", "sub", "b.txt"), "B")
	})

	t.Run("source ending in dot merges", func(t *testing.T) {
		merged := mkdir(t, filepath.Join(dir, "merged"))
		write(t, filepath.Join(merged, "keep"), "K")
		run(t, cp.Options{Recursive: true}, src+string(filepath.Separator)+".", merged)
		wantContent(t, filepath.Join(merged, "a.txt"), "A")
		wantContent(t, filepath.Join(merged, "keep"), "K")
		wantMissing(t, filepath.Join(merged, "src"))
	})
}

func TestReadOnlyDirectoryIsCopied(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "f"), "F")
	if err := os.Chmod(src, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(src, 0o755)
		_ = os.Chmod(filepath.Join(dir, "dst"), 0o755)
	})
	want, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "dst")
	run(t, cp.Options{Recursive: true}, src, dst)
	wantContent(t, filepath.Join(dst, "f"), "F")
	got, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != want.Mode().Perm() {
		t.Fatalf("mode = %v, want %v", got.Mode().Perm(), want.Mode().Perm())
	}
}

func TestOverwritePolicies(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name     string
		opts     cp.Options
		answer   bool
		srcTime  time.Time
		dstTime  time.Time
		want     string
		prompted bool
	}{
		{"no-clobber keeps destination", cp.Options{NoClobber: true}, true, recent, old, "old", false},
		{"update skips newer destination", cp.Options{Update: true}, true, old, recent, "old", false},
		{"update skips same-age destination", cp.Options{Update: true}, true, old, old, "old", false},
		{"update replaces older destination", cp.Options{Update: true}, true, recent, old, "new", false},
		{"interactive declined", cp.Options{Interactive: true}, false, recent, old, "old", true},
		{"interactive accepted", cp.Options{Interactive: true}, true, recent, old, "new", true},
		{"update decides before the prompt", cp.Options{Update: true, Interactive: true}, true, old, recent, "old", false},
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
			c, _, errb := newEngine(tt.opts)
			c.PromptFn = func(q string) bool {
				questions = append(questions, q)
				return tt.answer
			}
			if err := c.Run([]string{src, dst}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			wantContent(t, dst, tt.want)

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

func TestInteractiveReadsAnswerFromInput(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "new")
	write(t, dst, "old")

	var out, errb bytes.Buffer
	c := cp.New(cp.Options{Interactive: true}, core.Streams{In: strings.NewReader("y\n"), Out: &out, Err: &errb})
	if err := c.Run([]string{src, dst}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := "cpx: overwrite '" + dst + "'? "; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	wantContent(t, dst, "new")
}

func TestNewFileTakesSourceMode(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "x")
	// Owner-only bits survive any sane umask, so the result is predictable.
	if err := os.Chmod(src, 0o600); err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}

	run(t, cp.Options{}, src, dst)
	got, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != want.Mode().Perm() {
		t.Fatalf("mode = %v, want %v", got.Mode().Perm(), want.Mode().Perm())
	}
}

func TestPreserve(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "f"), "x")
	stamp := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	// World-writable bits are ones a typical umask would strip without -p.
	if err := os.Chmod(filepath.Join(src, "f"), 0o666); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(src, "f"), src} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	wantInfo, err := os.Stat(filepath.Join(src, "f"))
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "dst")
	run(t, cp.Options{Recursive: true, Preserve: cp.PreserveAll()}, src, dst)

	got, err := os.Stat(filepath.Join(dst, "f"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != wantInfo.Mode().Perm() {
		t.Fatalf("mode = %v, want %v", got.Mode().Perm(), wantInfo.Mode().Perm())
	}
	if !got.ModTime().Equal(stamp) {
		t.Fatalf("file mtime = %v, want %v", got.ModTime(), stamp)
	}
	gotDir, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !gotDir.ModTime().Equal(stamp) {
		t.Fatalf("directory mtime = %v, want %v", gotDir.ModTime(), stamp)
	}

	t.Run("without preserve the copy is new", func(t *testing.T) {
		plain := filepath.Join(dir, "plain")
		run(t, cp.Options{}, filepath.Join(src, "f"), plain)
		info, err := os.Stat(plain)
		if err != nil {
			t.Fatal(err)
		}
		if info.ModTime().Equal(stamp) {
			t.Fatal("mtime was preserved without --preserve=timestamps")
		}
	})
}

func TestPreserveLinks(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "one"), "data")
	if err := os.Link(filepath.Join(src, "one"), filepath.Join(src, "two")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}

	sameFile := func(t *testing.T, root string) bool {
		t.Helper()
		a, err := os.Stat(filepath.Join(root, "one"))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.Stat(filepath.Join(root, "two"))
		if err != nil {
			t.Fatal(err)
		}
		return os.SameFile(a, b)
	}

	plain := filepath.Join(dir, "plain")
	run(t, cp.Options{Recursive: true}, src, plain)
	if sameFile(t, plain) {
		t.Fatal("hard link preserved without --preserve=links")
	}
	wantContent(t, filepath.Join(plain, "two"), "data")

	linked := filepath.Join(dir, "linked")
	run(t, cp.Options{Recursive: true, Preserve: cp.Preserve{Links: true}}, src, linked)
	wantContent(t, filepath.Join(linked, "two"), "data")
	// Windows cannot identify hard links through os.FileInfo, so there the
	// names are copied independently; everywhere else they stay linked.
	if os.PathSeparator == '/' && !sameFile(t, linked) {
		t.Fatal("hard link not preserved with --preserve=links")
	}
}

func TestSymlinkPolicies(t *testing.T) {
	setup := func(t *testing.T) (dir, src string) {
		t.Helper()
		dir = t.TempDir()
		src = filepath.Join(dir, "src")
		write(t, filepath.Join(src, "real"), "data")
		symlink(t, "real", filepath.Join(src, "link"))
		return dir, src
	}

	t.Run("followed without -r", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		run(t, cp.Options{}, filepath.Join(src, "link"), dst)
		if isSymlink(t, dst) {
			t.Fatal("link was copied as a link; want its contents")
		}
		wantContent(t, dst, "data")
	})

	t.Run("kept with -P", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		run(t, cp.Options{Dereference: cp.DerefNever}, filepath.Join(src, "link"), dst)
		if got, err := os.Readlink(dst); err != nil || got != "real" {
			t.Fatalf("readlink = %q, %v; want \"real\"", got, err)
		}
	})

	t.Run("kept by -r", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		run(t, cp.Options{Recursive: true}, src, dst)
		if got, err := os.Readlink(filepath.Join(dst, "link")); err != nil || got != "real" {
			t.Fatalf("readlink = %q, %v; want \"real\"", got, err)
		}
		wantContent(t, filepath.Join(dst, "link"), "data")
	})

	t.Run("followed by -rL", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		run(t, cp.Options{Recursive: true, Dereference: cp.DerefAlways}, src, dst)
		if isSymlink(t, filepath.Join(dst, "link")) {
			t.Fatal("-L copied a link as a link")
		}
		wantContent(t, filepath.Join(dst, "link"), "data")
	})

	t.Run("-H follows only the operand", func(t *testing.T) {
		dir, src := setup(t)
		top := filepath.Join(dir, "top")
		symlink(t, src, top)
		dst := filepath.Join(dir, "out")
		run(t, cp.Options{Recursive: true, Dereference: cp.DerefCommandLine}, top, dst)
		if isSymlink(t, dst) {
			t.Fatal("-H did not follow the command-line link")
		}
		if !isSymlink(t, filepath.Join(dst, "link")) {
			t.Fatal("-H followed a link below the operand")
		}
	})

	t.Run("replaces an existing destination link", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		other := filepath.Join(dir, "other")
		write(t, other, "other")
		symlink(t, other, dst)
		run(t, cp.Options{Dereference: cp.DerefNever}, filepath.Join(src, "link"), dst)
		if got, _ := os.Readlink(dst); got != "real" {
			t.Fatalf("readlink = %q, want \"real\"", got)
		}
		wantContent(t, other, "other")
	})

	t.Run("refuses a dangling destination link", func(t *testing.T) {
		dir, src := setup(t)
		dst := filepath.Join(dir, "out")
		symlink(t, filepath.Join(dir, "nowhere"), dst)
		got := fail(t, cp.Options{}, filepath.Join(src, "real"), dst)
		if want := "cpx: not writing through dangling symlink '" + dst + "'\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		wantMissing(t, filepath.Join(dir, "nowhere"))
	})

	t.Run("-L refuses to follow a link into the destination", func(t *testing.T) {
		dir, src := setup(t)
		symlink(t, dir, filepath.Join(src, "up"))
		dst := filepath.Join(dir, "out")
		got := fail(t, cp.Options{Recursive: true, Dereference: cp.DerefAlways}, src, dst)
		want := "cpx: cannot copy a directory, '" + filepath.Join(src, "up") + "', into itself, '" + filepath.Join(dst, "up") + "'\n"
		if got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestLinkModes(t *testing.T) {
	t.Run("hard link", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
		write(t, src, "data")
		c, out, errb := newEngine(cp.Options{Link: true, Verbose: true})
		if err := c.Run([]string{src, dst}); err != nil {
			t.Skipf("hard links unavailable: %v (%s)", err, errb.String())
		}
		if want := "'" + src + "' -> '" + dst + "'\n"; out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
		a, _ := os.Stat(src)
		b, _ := os.Stat(dst)
		if !os.SameFile(a, b) {
			t.Fatal("dst is not a hard link to src")
		}

		// Linking again is already done, and needs no -f.
		run(t, cp.Options{Link: true}, src, dst)

		other := filepath.Join(dir, "other")
		write(t, other, "other")
		got := fail(t, cp.Options{Link: true}, other, dst)
		if want := "cpx: cannot create hard link '" + dst + "' to '" + other + "': File exists\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		run(t, cp.Options{Link: true, Force: true}, other, dst)
		wantContent(t, dst, "other")
		wantContent(t, src, "data")
	})

	t.Run("symbolic link", func(t *testing.T) {
		dir := t.TempDir()
		src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
		write(t, src, "data")
		symlink(t, src, filepath.Join(dir, "probe"))

		run(t, cp.Options{SymbolicLink: true}, src, dst)
		if got, _ := os.Readlink(dst); got != src {
			t.Fatalf("readlink = %q, want %q", got, src)
		}

		got := fail(t, cp.Options{SymbolicLink: true}, src, dst)
		if want := "cpx: cannot create symbolic link '" + dst + "' to '" + src + "': File exists\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		run(t, cp.Options{SymbolicLink: true, Force: true}, src, dst)
	})

	t.Run("relative symbolic link outside the current directory", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "src"), "data")
		mkdir(t, filepath.Join(dir, "sub"))
		t.Chdir(dir)

		dst := filepath.Join("sub", "dst")
		got := fail(t, cp.Options{SymbolicLink: true, NoTargetDirectory: true}, "src", dst)
		if want := "cpx: '" + dst + "': can make relative symbolic links only in current directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestTargetDirectoryOptions(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, a, "A")
	write(t, b, "B")

	t.Run("-t", func(t *testing.T) {
		dest := mkdir(t, filepath.Join(dir, "dest"))
		run(t, cp.Options{TargetDirectory: dest}, a, b)
		wantContent(t, filepath.Join(dest, "a"), "A")
		wantContent(t, filepath.Join(dest, "b"), "B")
	})

	t.Run("-T merges instead of nesting", func(t *testing.T) {
		src := filepath.Join(dir, "tree")
		write(t, filepath.Join(src, "f"), "F")
		dest := mkdir(t, filepath.Join(dir, "existing"))
		run(t, cp.Options{Recursive: true, NoTargetDirectory: true}, src, dest)
		wantContent(t, filepath.Join(dest, "f"), "F")
		wantMissing(t, filepath.Join(dest, "tree"))
	})
}

func TestParents(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a", "b", "file"), "data")
	dest := mkdir(t, filepath.Join(dir, "dest"))
	t.Chdir(dir)

	src := filepath.Join("a", "b", "file")
	out := run(t, cp.Options{Parents: true, Verbose: true}, src, "dest")
	want := "'a' -> '" + filepath.Join("dest", "a") + "'\n" +
		"'" + filepath.Join("a", "b") + "' -> '" + filepath.Join("dest", "a", "b") + "'\n" +
		"'" + src + "' -> '" + filepath.Join("dest", src) + "'\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantContent(t, filepath.Join(dest, "a", "b", "file"), "data")
}

func TestStripTrailingSlashes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	write(t, filepath.Join(src, "f"), "F")
	dst := filepath.Join(dir, "dst")

	out := run(t, cp.Options{Recursive: true, Verbose: true, StripTrailingSlashes: true}, src+string(filepath.Separator), dst)
	if want := "'" + src + "' -> '" + dst + "'\n"; !strings.HasPrefix(out, want) {
		t.Fatalf("stdout = %q, want prefix %q", out, want)
	}
}

func TestJustCreated(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "one", "f"), filepath.Join(dir, "two", "f")
	write(t, first, "first")
	write(t, second, "second")
	dest := mkdir(t, filepath.Join(dir, "dest"))

	got := fail(t, cp.Options{}, first, second, dest)
	want := "cpx: will not overwrite just-created '" + filepath.Join(dest, "f") + "' with '" + second + "'\n"
	if got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	wantContent(t, filepath.Join(dest, "f"), "first")
}

func TestForceReplacesUnwritableDestination(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	write(t, src, "new")
	write(t, dst, "old")
	if err := os.Chmod(dst, 0o444); err != nil {
		t.Fatal(err)
	}
	if f, err := os.OpenFile(dst, os.O_WRONLY, 0); err == nil {
		_ = f.Close()
		t.Skip("read-only files are writable here (running as root?)")
	}

	got := fail(t, cp.Options{}, src, dst)
	if want := "cpx: cannot create regular file '" + dst + "': Permission denied\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	wantContent(t, dst, "old")

	c, _, errb := newEngine(cp.Options{Force: true})
	if err := c.Run([]string{src, dst}); err != nil {
		// Windows will not unlink a read-only file; that is the platform's
		// refusal, not a missing retry.
		if !strings.Contains(errb.String(), "cannot remove") {
			t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
		}
		t.Skipf("cannot remove a read-only file here: %s", errb.String())
	}
	wantContent(t, dst, "new")
}

func TestNewNamedAndCopy(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	var out, errb bytes.Buffer
	c := cp.NewNamed("mvx", cp.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})

	if c.Copy(missing, filepath.Join(dir, "out")) {
		t.Fatal("Copy of a missing file reported success")
	}
	if want := "mvx: cannot stat '" + missing + "': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}

	// Copy never treats the destination as a directory to copy into.
	src := filepath.Join(dir, "src")
	write(t, src, "x")
	errb.Reset()
	if c.Copy(src, dir) {
		t.Fatal("Copy onto a directory reported success")
	}
	if want := "mvx: cannot overwrite directory '" + dir + "' with non-directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestPathHelpers(t *testing.T) {
	sep := string(filepath.Separator)
	joins := []struct {
		dir, src, want string
	}{
		{"d", "a", filepath.Join("d", "a")},
		{"d", "x" + sep + "a", filepath.Join("d", "a")},
		{"d", "a" + sep, filepath.Join("d", "a")},
		{"d", "a" + sep + ".", "d"},
		{"d", "..", "d"},
		{"d", ".", "d"},
	}
	for _, tt := range joins {
		if got := cp.JoinInto(tt.dir, tt.src); got != tt.want {
			t.Errorf("JoinInto(%q, %q) = %q, want %q", tt.dir, tt.src, got, tt.want)
		}
	}

	if got := cp.StripTrailingSlashes("a" + sep + sep); got != "a" {
		t.Errorf("StripTrailingSlashes = %q, want \"a\"", got)
	}
	if got := cp.StripTrailingSlashes(sep); got != sep {
		t.Errorf("StripTrailingSlashes(root) = %q, want %q", got, sep)
	}

	dir := t.TempDir()
	within := []struct {
		path string
		want bool
	}{
		{dir, true},
		{filepath.Join(dir, "not", "yet", "there"), true},
		{filepath.Dir(dir), false},
		{dir + "-sibling", false},
	}
	for _, tt := range within {
		if got := cp.IsWithin(dir, tt.path); got != tt.want {
			t.Errorf("IsWithin(%q, %q) = %v, want %v", dir, tt.path, got, tt.want)
		}
	}
}
