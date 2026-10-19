package ln_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ln"
	"github.com/braswelljr/gnux/core/printf"
)

func newEngine(opts ln.Options) (*ln.Ln, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	l := ln.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return l, &out, &errb
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

// requireSymlinks skips the test where the platform refuses to create
// symbolic links (unprivileged Windows).
func requireSymlinks(t *testing.T) {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.Symlink("target", probe); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// requireHardLinks skips the test where the filesystem has no hard links.
func requireHardLinks(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a"), "")
	if err := os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
}

func wantLink(t *testing.T, link, want string) {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("readlink %s: %v", link, err)
	}
	if got != want {
		t.Fatalf("readlink %s = %q, want %q", link, got, want)
	}
}

func wantSameFile(t *testing.T, a, b string) {
	t.Helper()
	ai, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	bi, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(ai, bi) {
		t.Fatalf("%s and %s are not the same file", a, b)
	}
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

// noStrays fails if a replacement left its temporary name behind in dir.
func noStrays(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "."+ln.Name) {
			t.Fatalf("temporary entry %s left behind", e.Name())
		}
	}
}

// run executes the engine and requires success with a silent stderr.
func run(t *testing.T, opts ln.Options, args ...string) string {
	t.Helper()
	l, out, errb := newEngine(opts)
	if err := l.Run(args); err != nil {
		t.Fatalf("Run(%v): %v (stderr: %s)", args, err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
	return out.String()
}

// fail executes the engine, requires core.ErrFailed and returns stderr.
func fail(t *testing.T, opts ln.Options, args ...string) string {
	t.Helper()
	l, _, errb := newEngine(opts)
	if err := l.Run(args); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("Run(%v) err = %v, want ErrFailed (stderr: %s)", args, err, errb.String())
	}
	return errb.String()
}

func TestSymbolicLink(t *testing.T) {
	requireSymlinks(t)
	dir := t.TempDir()
	target, link := filepath.Join(dir, "target"), filepath.Join(dir, "link")
	write(t, target, "data")

	out := run(t, ln.Options{Symbolic: true, Verbose: true}, target, link)
	if want := "'" + link + "' -> '" + target + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantLink(t, link, target)

	t.Run("target is stored verbatim and need not exist", func(t *testing.T) {
		dangling := filepath.Join(dir, "dangling")
		run(t, ln.Options{Symbolic: true}, "no/such/thing", dangling)
		// Verbatim but for the separator: on Windows, Go stores a link's
		// slashes as backslashes, the only separator a Windows link follows.
		wantLink(t, dangling, filepath.FromSlash("no/such/thing"))
	})

	t.Run("existing destination", func(t *testing.T) {
		got := fail(t, ln.Options{Symbolic: true}, target, link)
		if want := "lnx: failed to create symbolic link '" + link + "': File exists\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestHardLink(t *testing.T) {
	requireHardLinks(t)
	dir := t.TempDir()
	target, link := filepath.Join(dir, "target"), filepath.Join(dir, "link")
	write(t, target, "data")

	out := run(t, ln.Options{Verbose: true}, target, link)
	if want := "'" + link + "' => '" + target + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantSameFile(t, target, link)

	t.Run("existing destination", func(t *testing.T) {
		got := fail(t, ln.Options{}, target, link)
		if want := "lnx: failed to create hard link '" + link + "': File exists\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})

	t.Run("force on an existing link to the same file is a no-op", func(t *testing.T) {
		out := run(t, ln.Options{Force: true, Verbose: true}, target, link)
		if want := "'" + link + "' => '" + target + "'\n"; out != want {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
		wantSameFile(t, target, link)
		noStrays(t, dir)
	})

	t.Run("missing target", func(t *testing.T) {
		missing := filepath.Join(dir, "missing")
		got := fail(t, ln.Options{}, missing, filepath.Join(dir, "out"))
		if want := "lnx: failed to access '" + missing + "': No such file or directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})

	t.Run("directory target", func(t *testing.T) {
		sub := mkdir(t, filepath.Join(dir, "sub"))
		got := fail(t, ln.Options{NoTargetDirectory: true}, sub, filepath.Join(dir, "out"))
		if want := "lnx: " + printf.ShellQuote(sub) + ": hard link not allowed for directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})

	t.Run("destination directory missing", func(t *testing.T) {
		dst := filepath.Join(dir, "nope", "out")
		got := fail(t, ln.Options{}, target, dst)
		if want := "lnx: failed to create hard link '" + dst + "' => '" + target + "': No such file or directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestLogicalAndPhysical(t *testing.T) {
	requireSymlinks(t)
	requireHardLinks(t)
	dir := t.TempDir()
	target, sym := filepath.Join(dir, "target"), filepath.Join(dir, "sym")
	write(t, target, "data")
	if err := os.Symlink(target, sym); err != nil {
		t.Fatal(err)
	}

	logical := filepath.Join(dir, "logical")
	run(t, ln.Options{Logical: true}, sym, logical)
	info, err := os.Lstat(logical)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("-L linked to the symlink, want its referent")
	}
	wantSameFile(t, target, logical)
}

func TestSingleOperandLinksIntoCurrentDirectory(t *testing.T) {
	requireSymlinks(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere", "target")
	write(t, target, "data")
	cwd := mkdir(t, filepath.Join(dir, "cwd"))
	t.Chdir(cwd)

	out := run(t, ln.Options{Symbolic: true, Verbose: true}, target)
	if want := "'." + string(filepath.Separator) + "target' -> '" + target + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	wantLink(t, filepath.Join(cwd, "target"), target)
}

func TestLinkIntoDirectory(t *testing.T) {
	requireSymlinks(t)
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, a, "A")
	write(t, b, "B")

	t.Run("TARGET... DIRECTORY", func(t *testing.T) {
		dest := mkdir(t, filepath.Join(dir, "dest"))
		out := run(t, ln.Options{Symbolic: true, Verbose: true}, a, b, dest)
		want := "'" + filepath.Join(dest, "a") + "' -> '" + a + "'\n" +
			"'" + filepath.Join(dest, "b") + "' -> '" + b + "'\n"
		if out != want {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
		wantLink(t, filepath.Join(dest, "a"), a)
		wantLink(t, filepath.Join(dest, "b"), b)
	})

	t.Run("TARGET DIRECTORY", func(t *testing.T) {
		dest := mkdir(t, filepath.Join(dir, "single"))
		run(t, ln.Options{Symbolic: true}, a, dest)
		wantLink(t, filepath.Join(dest, "a"), a)
	})

	t.Run("-t DIRECTORY TARGET...", func(t *testing.T) {
		dest := mkdir(t, filepath.Join(dir, "tdest"))
		run(t, ln.Options{Symbolic: true, TargetDirectory: dest}, a, b)
		wantLink(t, filepath.Join(dest, "a"), a)
		wantLink(t, filepath.Join(dest, "b"), b)
	})

	t.Run("one failure does not stop the rest", func(t *testing.T) {
		dest := mkdir(t, filepath.Join(dir, "partial"))
		write(t, filepath.Join(dest, "a"), "in the way")
		got := fail(t, ln.Options{Symbolic: true}, a, b, dest)
		if want := "lnx: failed to create symbolic link '" + filepath.Join(dest, "a") + "': File exists\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		wantLink(t, filepath.Join(dest, "b"), b)
	})
}

func TestUsageAndTargetErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	missing := filepath.Join(dir, "missing")
	write(t, file, "x")

	usage := []struct {
		name string
		opts ln.Options
		args []string
		want string
	}{
		{"no operands", ln.Options{}, nil, "missing file operand"},
		{"-T with one operand", ln.Options{NoTargetDirectory: true}, []string{"a"}, "missing destination file operand after 'a'"},
		{"-T with three operands", ln.Options{NoTargetDirectory: true}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{"target not a directory", ln.Options{}, []string{"a", "b", file}, "target '" + file + "' is not a directory"},
		{"-r without -s", ln.Options{Relative: true}, []string{"a", "b"}, "cannot do --relative without --symbolic"},
		{"-t with -T", ln.Options{TargetDirectory: dir, NoTargetDirectory: true}, []string{"a"},
			"cannot combine --target-directory and --no-target-directory"},
	}
	for _, tt := range usage {
		t.Run(tt.name, func(t *testing.T) {
			l, _, _ := newEngine(tt.opts)
			err := l.Run(tt.args)
			var u *core.UsageError
			if !errors.As(err, &u) {
				t.Fatalf("err = %v, want a usage error", err)
			}
			if u.Msg != tt.want {
				t.Fatalf("usage = %q, want %q", u.Msg, tt.want)
			}
		})
	}

	t.Run("missing target directory", func(t *testing.T) {
		got := fail(t, ln.Options{Symbolic: true, TargetDirectory: missing}, file)
		if want := "lnx: target directory '" + missing + "': No such file or directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
	t.Run("target directory is a file", func(t *testing.T) {
		got := fail(t, ln.Options{Symbolic: true, TargetDirectory: file}, file)
		if want := "lnx: target directory '" + file + "': Not a directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestForceReplaces(t *testing.T) {
	requireSymlinks(t)
	dir := t.TempDir()
	oldTarget, newTarget := filepath.Join(dir, "old"), filepath.Join(dir, "new")
	link := filepath.Join(dir, "link")
	write(t, oldTarget, "old")
	write(t, newTarget, "new")
	if err := os.Symlink(oldTarget, link); err != nil {
		t.Fatal(err)
	}

	run(t, ln.Options{Symbolic: true, Force: true}, newTarget, link)
	wantLink(t, link, newTarget)
	wantContent(t, oldTarget, "old")
	noStrays(t, dir)

	t.Run("re-running is idempotent", func(t *testing.T) {
		run(t, ln.Options{Symbolic: true, Force: true}, newTarget, link)
		wantLink(t, link, newTarget)
		wantContent(t, newTarget, "new")
	})

	t.Run("replaces a regular file", func(t *testing.T) {
		file := filepath.Join(dir, "file")
		write(t, file, "in the way")
		run(t, ln.Options{Symbolic: true, Force: true}, newTarget, file)
		wantLink(t, file, newTarget)
	})

	t.Run("refuses to replace a file with a link to itself", func(t *testing.T) {
		got := fail(t, ln.Options{Symbolic: true, Force: true}, newTarget, newTarget)
		if want := "lnx: '" + newTarget + "' and '" + newTarget + "' are the same file\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
		wantContent(t, newTarget, "new")
	})

	t.Run("refuses to replace a directory", func(t *testing.T) {
		sub := mkdir(t, filepath.Join(dir, "sub"))
		got := fail(t, ln.Options{Symbolic: true, Force: true, NoTargetDirectory: true}, newTarget, sub)
		if want := "lnx: " + printf.ShellQuote(sub) + ": cannot overwrite directory\n"; got != want {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	})
}

func TestForceHardLinkRefusesSameFile(t *testing.T) {
	requireHardLinks(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	write(t, file, "data")

	got := fail(t, ln.Options{Force: true}, file, file)
	if want := "lnx: '" + file + "' and '" + file + "' are the same file\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	wantContent(t, file, "data")
	noStrays(t, dir)
}

func TestInteractive(t *testing.T) {
	requireSymlinks(t)
	for _, tt := range []struct {
		name   string
		answer bool
	}{
		{"declined", false},
		{"accepted", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			target, link := filepath.Join(dir, "target"), filepath.Join(dir, "link")
			write(t, target, "data")
			write(t, link, "in the way")

			var questions []string
			l, _, errb := newEngine(ln.Options{Symbolic: true, Interactive: true})
			l.PromptFn = func(q string) bool {
				questions = append(questions, q)
				return tt.answer
			}
			if err := l.Run([]string{target, link}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			if want := "replace '" + link + "'?"; len(questions) != 1 || questions[0] != want {
				t.Fatalf("questions = %q, want [%q]", questions, want)
			}
			if tt.answer {
				wantLink(t, link, target)
			} else {
				wantContent(t, link, "in the way")
			}
		})
	}

	t.Run("no prompt when the destination is free", func(t *testing.T) {
		dir := t.TempDir()
		l, _, _ := newEngine(ln.Options{Symbolic: true, Interactive: true})
		l.PromptFn = func(q string) bool {
			t.Fatalf("unexpected prompt %q", q)
			return false
		}
		if err := l.Run([]string{"target", filepath.Join(dir, "link")}); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
}

func TestNoDereference(t *testing.T) {
	requireSymlinks(t)
	setup := func(t *testing.T) (dir, v1, v2, current string) {
		t.Helper()
		dir = t.TempDir()
		v1, v2 = mkdir(t, filepath.Join(dir, "v1")), mkdir(t, filepath.Join(dir, "v2"))
		current = filepath.Join(dir, "current")
		if err := os.Symlink(v1, current); err != nil {
			t.Fatal(err)
		}
		return dir, v1, v2, current
	}

	t.Run("without -n the link is created inside the directory", func(t *testing.T) {
		_, v1, v2, current := setup(t)
		run(t, ln.Options{Symbolic: true, Force: true}, v2, current)
		wantLink(t, current, v1)
		wantLink(t, filepath.Join(v1, "v2"), v2)
	})

	t.Run("with -n the link itself is replaced", func(t *testing.T) {
		_, v1, v2, current := setup(t)
		run(t, ln.Options{Symbolic: true, Force: true, NoDereference: true}, v2, current)
		wantLink(t, current, v2)
		if _, err := os.Lstat(filepath.Join(v1, "v2")); !os.IsNotExist(err) {
			t.Fatalf("link was created inside the old directory (err = %v)", err)
		}
	})

	t.Run("-T behaves the same", func(t *testing.T) {
		_, _, v2, current := setup(t)
		run(t, ln.Options{Symbolic: true, Force: true, NoTargetDirectory: true}, v2, current)
		wantLink(t, current, v2)
	})
}

func TestRelative(t *testing.T) {
	requireSymlinks(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "data", "file")
	write(t, target, "data")
	bin := mkdir(t, filepath.Join(dir, "a", "bin"))

	tests := []struct {
		name, target, link, want string
	}{
		{"up and across", target, filepath.Join(bin, "file"), filepath.Join("..", "..", "data", "file")},
		{"sibling", target, filepath.Join(dir, "data", "alias"), "file"},
		{"missing target", filepath.Join(dir, "data", "later"), filepath.Join(bin, "later"), filepath.Join("..", "..", "data", "later")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := run(t, ln.Options{Symbolic: true, Relative: true, Verbose: true}, tt.target, tt.link)
			if want := "'" + tt.link + "' -> '" + tt.want + "'\n"; out != want {
				t.Fatalf("stdout = %q, want %q", out, want)
			}
			wantLink(t, tt.link, tt.want)
		})
	}
	wantContent(t, filepath.Join(bin, "file"), "data")

	t.Run("relative operands resolve against the working directory", func(t *testing.T) {
		t.Chdir(dir)
		link := filepath.Join("a", "bin", "cwd")
		run(t, ln.Options{Symbolic: true, Relative: true}, filepath.Join("data", "file"), link)
		wantLink(t, link, filepath.Join("..", "..", "data", "file"))
	})
}

func TestBackup(t *testing.T) {
	requireSymlinks(t)
	requireHardLinks(t)
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	write(t, a, "A")
	write(t, b, "B")
	simple := core.Backup{Control: core.SimpleBackup}

	// A backup replaces like -f does, and -v names it.
	out := run(t, ln.Options{Backup: simple, Verbose: true}, a, b)
	if want := "'" + b + "~' ~ '" + b + "' => '" + a + "'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if data, _ := os.ReadFile(b + "~"); string(data) != "B" {
		t.Fatalf("backup holds %q", data)
	}
	ai, _ := os.Stat(a)
	bi, _ := os.Stat(b)
	if !os.SameFile(ai, bi) {
		t.Fatal("b is not a hard link to a")
	}

	// b is now a hard link to a: with a backup it is replaced again, not
	// skipped as -f would.
	run(t, ln.Options{Backup: core.Backup{Control: core.NumberedBackup}}, a, b)
	if _, err := os.Lstat(b + ".~1~"); err != nil {
		t.Fatalf("numbered backup: %v", err)
	}

	// A symbolic link, even to the very same name, backs up what was there.
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	out = run(t, ln.Options{Backup: simple, Symbolic: true, Verbose: true}, "a", a)
	if want := "'" + a + "~' ~ '" + a + "' -> 'a'\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if data, _ := os.ReadFile(a + "~"); string(data) != "A" {
		t.Fatalf("backup holds %q", data)
	}

	// A hard link onto its own name is still refused; a new name needs no
	// backup.
	if err := os.Remove(a); err != nil {
		t.Fatal(err)
	}
	write(t, a, "A")
	c, _, errb := newEngine(ln.Options{Backup: simple})
	if err := c.Run([]string{a, a}); !errors.Is(err, core.ErrFailed) || !strings.Contains(errb.String(), "are the same file") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
	run(t, ln.Options{Backup: simple}, a, filepath.Join(dir, "new"))
	if _, err := os.Lstat(filepath.Join(dir, "new~")); !os.IsNotExist(err) {
		t.Fatal("a backup was made of nothing")
	}
}
