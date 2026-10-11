package ls_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ls"
)

// now is the fixed clock every test engine runs on.
var now = time.Date(2024, time.June, 15, 12, 0, 0, 0, time.UTC)

// newEngine builds an engine wired to buffers, with a fixed clock, UTC
// timestamps and an empty environment, so output depends only on the options
// and the files under test.
func newEngine(opts ls.Options) (*ls.Ls, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	l := ls.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	l.Now = func() time.Time { return now }
	l.Location = time.UTC
	l.Getenv = func(string) string { return "" }
	return l, &out, &errb
}

// list runs the engine and fails the test on any error or diagnostic.
func list(t *testing.T, opts ls.Options, args ...string) string {
	t.Helper()
	l, out, errb := newEngine(opts)
	if err := l.Run(args); err != nil {
		t.Fatalf("Run(%v): %v (stderr: %s)", args, err, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errb.String())
	}
	return out.String()
}

// touch creates an empty file (and its parent directories) under dir.
func touch(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// sized creates a sparse file of the given size.
func sized(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func setTime(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

func exitCode(err error) int {
	var ee *core.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.Code
	case errors.Is(err, core.ErrFailed):
		return 1
	}
	return -1
}

func TestHiddenEntries(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "b", "a", ".hidden", "Zed")

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		// Byte order: uppercase sorts before lowercase.
		{"default hides dot files", ls.Options{}, "Zed\na\nb\n"},
		{"almost all", ls.Options{AlmostAll: true}, ".hidden\nZed\na\nb\n"},
		{"all adds dot and dot-dot", ls.Options{All: true}, ".\n..\n.hidden\nZed\na\nb\n"},
		{"reverse", ls.Options{Reverse: true}, "b\na\nZed\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, dir); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNoOperandListsCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "one", "two")
	t.Chdir(dir)

	if got, want := list(t, ls.Options{}), "one\ntwo\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestOperands(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "f1", "f2", "d1/x", "d2/y")
	t.Chdir(dir)

	tests := []struct {
		name string
		opts ls.Options
		args []string
		want string
	}{
		{"single directory has no header", ls.Options{}, []string{"d1"}, "x\n"},
		{"single file", ls.Options{}, []string{"f1"}, "f1\n"},
		{"files come before directories", ls.Options{}, []string{"d2", "f2", "d1", "f1"},
			"f1\nf2\n\nd1:\nx\n\nd2:\ny\n"},
		{"two directories", ls.Options{}, []string{"d2", "d1"}, "d1:\nx\n\nd2:\ny\n"},
		{"directory flag lists the directory itself", ls.Options{Directory: true}, []string{"d1", "f1"}, "d1\nf1\n"},
		{"directory flag without operands", ls.Options{Directory: true}, nil, ".\n"},
		{"unsorted keeps operand order", ls.Options{Sort: ls.SortNone, Directory: true}, []string{"f2", "d1", "f1"}, "f2\nd1\nf1\n"},
		{"header is quoted like a name", ls.Options{Quoting: ls.QuoteC}, []string{"d1", "d2"},
			"\"d1\":\n\"x\"\n\n\"d2\":\n\"y\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, tt.args...); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecursive(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "top", "a/one", "a/deep/two", "b/three", ".git/config")
	t.Chdir(dir)

	t.Run("current directory", func(t *testing.T) {
		want := ".:\na\nb\ntop\n\n./a:\ndeep\none\n\n./a/deep:\ntwo\n\n./b:\nthree\n"
		if got := list(t, ls.Options{Recursive: true}); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("named directory", func(t *testing.T) {
		want := "a:\ndeep\none\n\na/deep:\ntwo\n"
		if got := list(t, ls.Options{Recursive: true}, "a"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("all does not recurse into dot or dot-dot", func(t *testing.T) {
		want := "b:\n.\n..\nthree\n"
		if got := list(t, ls.Options{Recursive: true, All: true}, "b"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("directory flag wins over recursion", func(t *testing.T) {
		if got, want := list(t, ls.Options{Recursive: true, Directory: true}, "a"), "a\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestSort(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "d/x")
	sized(t, filepath.Join(dir, "b.txt"), 30)
	sized(t, filepath.Join(dir, "a.go"), 10)
	sized(t, filepath.Join(dir, "c"), 20)
	sized(t, filepath.Join(dir, "v10.go"), 5)
	sized(t, filepath.Join(dir, "v9.go"), 5)
	for i, name := range []string{"c", "a.go", "v9.go", "b.txt", "v10.go"} {
		setTime(t, filepath.Join(dir, name), now.Add(-time.Duration(i+1)*time.Hour))
	}
	t.Chdir(dir)
	files := []string{"a.go", "b.txt", "c", "v10.go", "v9.go"}

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"name", ls.Options{}, "a.go\nb.txt\nc\nv10.go\nv9.go\n"},
		{"explicit name", ls.Options{Sort: ls.SortName, Time: ls.TimeAccessed}, "a.go\nb.txt\nc\nv10.go\nv9.go\n"},
		{"size, largest first, ties by name", ls.Options{Sort: ls.SortSize}, "b.txt\nc\na.go\nv10.go\nv9.go\n"},
		{"size reversed", ls.Options{Sort: ls.SortSize, Reverse: true}, "v9.go\nv10.go\na.go\nc\nb.txt\n"},
		{"time, newest first", ls.Options{Sort: ls.SortTime}, "c\na.go\nv9.go\nb.txt\nv10.go\n"},
		{"time reversed", ls.Options{Sort: ls.SortTime, Reverse: true}, "v10.go\nb.txt\nv9.go\na.go\nc\n"},
		{"extension", ls.Options{Sort: ls.SortExtension}, "c\na.go\nv10.go\nv9.go\nb.txt\n"},
		{"version", ls.Options{Sort: ls.SortVersion}, "a.go\nb.txt\nc\nv9.go\nv10.go\n"},
		{"width", ls.Options{Sort: ls.SortWidth}, "c\na.go\nb.txt\nv9.go\nv10.go\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.Directory = true
			if got := list(t, tt.opts, files...); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("group directories first", func(t *testing.T) {
		want := "d\na.go\nb.txt\nc\nv10.go\nv9.go\n"
		if got := list(t, ls.Options{GroupDirectories: true}, "."); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("group directories first survives reverse", func(t *testing.T) {
		want := "d\nv9.go\nv10.go\nc\nb.txt\na.go\n"
		if got := list(t, ls.Options{GroupDirectories: true, Reverse: true}, "."); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestVersionSort(t *testing.T) {
	dir := t.TempDir()
	// Already in the order GNU's version sort produces: dot files first,
	// letters before other characters, digit runs by value (file01 and file1
	// are equal as versions, so bytes decide), suffixes ignored at first.
	names := []string{".profile", "abc", "a-1.tar.gz", "a-1.10.tar.gz", "file01", "file1", "file2", "file10", "file10a"}
	touch(t, dir, names...)

	want := strings.Join(names, "\n") + "\n"
	if got := list(t, ls.Options{Sort: ls.SortVersion, AlmostAll: true}, dir); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIgnorePatterns(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "main.go", "main.o", "notes.txt", "notes.txt~", ".cache")

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"ignore backups", ls.Options{IgnoreBackups: true}, "main.go\nmain.o\nnotes.txt\n"},
		{"ignore pattern", ls.Options{Ignore: []string{"*.o", "*~"}}, "main.go\nnotes.txt\n"},
		{"ignore applies under almost-all", ls.Options{Ignore: []string{"main.*"}, AlmostAll: true}, ".cache\nnotes.txt\nnotes.txt~\n"},
		{"wildcard does not match a leading dot", ls.Options{Ignore: []string{"*"}, AlmostAll: true}, ".cache\n"},
		{"hide pattern", ls.Options{Hide: []string{"notes*"}}, "main.go\nmain.o\n"},
		{"hide is overridden by almost-all", ls.Options{Hide: []string{"notes*"}, AlmostAll: true},
			".cache\nmain.go\nmain.o\nnotes.txt\nnotes.txt~\n"},
		{"malformed pattern matches nothing", ls.Options{Ignore: []string{"[a"}}, "main.go\nmain.o\nnotes.txt\nnotes.txt~\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, dir); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestColumns(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf")

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"down columns", ls.Options{Format: ls.FormatColumns, Width: 30},
			"alpha  charlie  echo     golf\n" +
				"bravo  delta    foxtrot\n"},
		{"down columns, narrower", ls.Options{Format: ls.FormatColumns, Width: 20},
			"alpha    echo\n" +
				"bravo    foxtrot\n" +
				"charlie  golf\n" +
				"delta\n"},
		{"down columns with tabs", ls.Options{Format: ls.FormatColumns, Width: 20, TabSize: 8},
			"alpha\t echo\n" +
				"bravo\t foxtrot\n" +
				"charlie  golf\n" +
				"delta\n"},
		{"across", ls.Options{Format: ls.FormatAcross, Width: 31},
			"alpha  bravo    charlie  delta\n" +
				"echo   foxtrot  golf\n"},
		// Four columns need exactly 30, and a line must be shorter than the width.
		{"across, one column too tight", ls.Options{Format: ls.FormatAcross, Width: 30},
			"alpha  bravo  charlie\n" +
				"delta  echo   foxtrot\n" +
				"golf\n"},
		{"too narrow falls back to one column", ls.Options{Format: ls.FormatColumns, Width: 5},
			"alpha\nbravo\ncharlie\ndelta\necho\nfoxtrot\ngolf\n"},
		{"no width limit", ls.Options{Format: ls.FormatColumns, Width: -1},
			"alpha  bravo  charlie  delta  echo  foxtrot  golf\n"},
		{"commas", ls.Options{Format: ls.FormatCommas, Width: 30},
			"alpha, bravo, charlie, delta,\n" +
				"echo, foxtrot, golf\n"},
		{"commas without a limit", ls.Options{Format: ls.FormatCommas, Width: -1},
			"alpha, bravo, charlie, delta, echo, foxtrot, golf\n"},
		{"commas with indicators", ls.Options{Format: ls.FormatCommas, Width: -1, All: true, Indicator: ls.IndicatorSlash, Ignore: []string{"[b-g]*"}},
			"./, ../, alpha\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, dir); got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestWidthDetection(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "alpha", "bravo", "charlie", "delta")
	const (
		onePerLine = "alpha\nbravo\ncharlie\ndelta\n"
		twoColumns = "alpha  charlie\nbravo  delta\n"
		oneRow     = "alpha  bravo  charlie  delta\n"
	)

	tests := []struct {
		name     string
		opts     ls.Options
		terminal bool
		termCols int
		columns  string // COLUMNS
		want     string
	}{
		{"not a terminal: one per line", ls.Options{}, false, 0, "", onePerLine},
		{"terminal: columns at the terminal width", ls.Options{}, true, 16, "", twoColumns},
		{"terminal width beats COLUMNS", ls.Options{}, true, 16, "200", twoColumns},
		{"unknown terminal width falls back to COLUMNS", ls.Options{}, true, 0, "16", twoColumns},
		{"default width is 80", ls.Options{}, true, 0, "", oneRow},
		{"COLUMNS applies off a terminal", ls.Options{Format: ls.FormatColumns}, false, 0, "16", twoColumns},
		{"width option beats the terminal", ls.Options{Width: 16}, true, 200, "", twoColumns},
		{"terminal does not override an explicit format", ls.Options{Format: ls.FormatSingle}, true, 80, "", onePerLine},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, out, errb := newEngine(tt.opts)
			l.IsTerminal = tt.terminal
			l.TermWidth = func() int { return tt.termCols }
			l.Getenv = func(key string) string {
				if key == "COLUMNS" {
					return tt.columns
				}
				return ""
			}
			if err := l.Run([]string{dir}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			if got := out.String(); got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestQuoting(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a b", "plain", "caf\u00e9")
	special := []string{"q\"uote", "back\\slash", "tab\there", "new\nline"}
	for _, name := range special {
		// Not every filesystem accepts these names (Windows rejects them all).
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Skipf("cannot create %q: %v", name, err)
		}
	}

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"literal", ls.Options{}, "a b\nback\\slash\ncaf\u00e9\nnew\nline\nplain\nq\"uote\ntab\there\n"},
		{"hide control chars", ls.Options{HideControlChars: true}, "a b\nback\\slash\ncaf\u00e9\nnew?line\nplain\nq\"uote\ntab?here\n"},
		{"escape", ls.Options{Quoting: ls.QuoteEscape}, "a\\ b\nback\\\\slash\ncaf\u00e9\nnew\\nline\nplain\nq\"uote\ntab\\there\n"},
		{"quote name", ls.Options{Quoting: ls.QuoteC},
			"\"a b\"\n\"back\\\\slash\"\n\"caf\u00e9\"\n\"new\\nline\"\n\"plain\"\n\"q\\\"uote\"\n\"tab\\there\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, dir); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("a terminal hides control chars by default", func(t *testing.T) {
		l, out, _ := newEngine(ls.Options{Format: ls.FormatSingle})
		l.IsTerminal = true
		if err := l.Run([]string{dir}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "new?line\n") {
			t.Fatalf("got %q", out.String())
		}
	})
}

func TestIndicators(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "file", "sub/x")
	// Windows has no execute bit, so "*" can only be expected elsewhere.
	script, star := "", ""
	if runtime.GOOS != "windows" {
		if err := os.WriteFile(filepath.Join(dir, "run"), nil, 0o755); err != nil {
			t.Fatal(err)
		}
		script, star = "run\n", "run*\n"
	}

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"none", ls.Options{}, "file\n" + script + "sub\n"},
		{"slash", ls.Options{Indicator: ls.IndicatorSlash}, "file\n" + script + "sub/\n"},
		{"file type", ls.Options{Indicator: ls.IndicatorFileType}, "file\n" + script + "sub/\n"},
		{"classify", ls.Options{Indicator: ls.IndicatorClassify}, "file\n" + star + "sub/\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, dir); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSymlinks(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "real/inside", "file")
	symlink(t, "real", filepath.Join(dir, "dirlink"))
	symlink(t, "file", filepath.Join(dir, "filelink"))
	symlink(t, "missing", filepath.Join(dir, "dangling"))
	t.Chdir(dir)

	tests := []struct {
		name string
		opts ls.Options
		args []string
		want string
	}{
		{"operand link to a directory is followed", ls.Options{}, []string{"dirlink"}, "inside\n"},
		{"directory flag keeps the link", ls.Options{Directory: true}, []string{"dirlink"}, "dirlink\n"},
		{"classify keeps the link", ls.Options{Indicator: ls.IndicatorClassify}, []string{"dirlink"}, "dirlink@\n"},
		{"dereference-args follows it again", ls.Options{Indicator: ls.IndicatorClassify, DereferenceArgs: true}, []string{"dirlink"}, "inside\n"},
		{"links inside a directory are not followed", ls.Options{Indicator: ls.IndicatorClassify}, nil,
			"dangling@\ndirlink@\nfile\nfilelink@\nreal/\n"},
		{"dangling operand is listed as a link", ls.Options{}, []string{"dangling"}, "dangling\n"},
		{"dereference shows the target type", ls.Options{Indicator: ls.IndicatorClassify, Dereference: true, Ignore: []string{"dangling"}}, nil,
			"dirlink/\nfile\nfilelink\nreal/\n"},
		{"group directories first counts links to directories", ls.Options{GroupDirectories: true}, nil,
			"dirlink\nreal\ndangling\nfile\nfilelink\n"},
		{"recursion does not follow links", ls.Options{Recursive: true}, []string{"."},
			".:\ndangling\ndirlink\nfile\nfilelink\nreal\n\n./real:\ninside\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, tt.args...); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("long format shows the target", func(t *testing.T) {
		got := list(t, ls.Options{Format: ls.FormatLong, Indicator: ls.IndicatorClassify}, "dirlink", "filelink", "dangling")
		lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
		wantSuffixes := []string{" dangling -> missing", " dirlink -> real/", " filelink -> file"}
		if len(lines) != len(wantSuffixes) {
			t.Fatalf("got %q", got)
		}
		for i, want := range wantSuffixes {
			if !strings.HasPrefix(lines[i], "l") || !strings.HasSuffix(lines[i], want) {
				t.Errorf("line %d = %q, want an 'l' entry ending in %q", i, lines[i], want)
			}
		}
	})

	t.Run("dereferencing a dangling link inside a directory is a minor problem", func(t *testing.T) {
		l, out, errb := newEngine(ls.Options{Dereference: true, Format: ls.FormatLong, NoOwner: true, NoGroup: true, Ignore: []string{"[!d]*", "dirlink"}})
		err := l.Run([]string{"."})
		if exitCode(err) != 1 {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "lsx: cannot access './dangling': No such file or directory\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		if want := "total 0\nl????????? ? ?            ? dangling\n"; out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	})

	t.Run("dereferencing a dangling operand is serious", func(t *testing.T) {
		l, out, errb := newEngine(ls.Options{Dereference: true})
		if err := l.Run([]string{"dangling"}); exitCode(err) != 2 {
			t.Fatalf("err = %v, want exit status 2", err)
		}
		if want := "lsx: cannot access 'dangling': No such file or directory\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		if out.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", out.String())
		}
	})
}

func TestMissingOperand(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "here", "sub/x")
	t.Chdir(dir)

	l, out, errb := newEngine(ls.Options{})
	err := l.Run([]string{"nope", "here", "sub"})
	if exitCode(err) != 2 {
		t.Fatalf("err = %v, want exit status 2", err)
	}
	if want := "lsx: cannot access 'nope': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	// The remaining operands are still listed, and the directory keeps its
	// header because more than one operand was given.
	if want := "here\n\nsub:\nx\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestUnreadableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	dir := t.TempDir()
	touch(t, dir, "tree/ok/x")
	locked := filepath.Join(dir, "tree", "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	t.Chdir(dir)

	t.Run("as an operand it is serious", func(t *testing.T) {
		l, out, errb := newEngine(ls.Options{})
		if err := l.Run([]string{"tree/locked"}); exitCode(err) != 2 {
			t.Fatalf("err = %v, want exit status 2", err)
		}
		if want := "lsx: cannot open directory 'tree/locked': Permission denied\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		if out.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", out.String())
		}
	})

	t.Run("reached by recursion it is minor", func(t *testing.T) {
		l, out, errb := newEngine(ls.Options{Recursive: true})
		if err := l.Run([]string{"tree"}); exitCode(err) != 1 {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "lsx: cannot open directory 'tree/locked': Permission denied\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
		if want := "tree:\nlocked\nok\n\ntree/ok:\nx\n"; out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	})
}

func TestLongFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("asserts POSIX permission bits and link counts")
	}
	dir := t.TempDir()
	write := func(name string, size int64, perm os.FileMode, when time.Time) {
		path := filepath.Join(dir, name)
		sized(t, path, size)
		if err := os.Chmod(path, perm); err != nil {
			t.Fatal(err)
		}
		setTime(t, path, when)
	}
	write("recent", 5, 0o644, time.Date(2024, time.June, 1, 9, 5, 7, 0, time.UTC))
	write("old", 12345, 0o600, time.Date(2021, time.February, 3, 4, 5, 6, 0, time.UTC))
	write("future", 0, 0o755, time.Date(2024, time.July, 4, 0, 0, 0, 0, time.UTC))
	write("edge", 100, 0o444, now.Add(-183*24*time.Hour)) // just past six months
	t.Chdir(dir)
	files := []string{"recent", "old", "future", "edge"}

	// -g and -o drop the owner and group, whose names depend on the machine.
	base := ls.Options{Format: ls.FormatLong, NoOwner: true, NoGroup: true}
	with := func(mutate func(*ls.Options)) ls.Options {
		o := base
		mutate(&o)
		return o
	}

	tests := []struct {
		name string
		opts ls.Options
		want string
	}{
		{"locale style: recent shows the time, old and future show the year", base,
			"-r--r--r-- 1   100 Dec 15  2023 edge\n" +
				"-rwxr-xr-x 1     0 Jul  4  2024 future\n" +
				"-rw------- 1 12345 Feb  3  2021 old\n" +
				"-rw-r--r-- 1     5 Jun  1 09:05 recent\n"},
		{"long-iso", with(func(o *ls.Options) { o.TimeStyle = "long-iso" }),
			"-r--r--r-- 1   100 2023-12-15 12:00 edge\n" +
				"-rwxr-xr-x 1     0 2024-07-04 00:00 future\n" +
				"-rw------- 1 12345 2021-02-03 04:05 old\n" +
				"-rw-r--r-- 1     5 2024-06-01 09:05 recent\n"},
		{"full-iso", with(func(o *ls.Options) { o.TimeStyle = "full-iso" }),
			"-r--r--r-- 1   100 2023-12-15 12:00:00.000000000 +0000 edge\n" +
				"-rwxr-xr-x 1     0 2024-07-04 00:00:00.000000000 +0000 future\n" +
				"-rw------- 1 12345 2021-02-03 04:05:06.000000000 +0000 old\n" +
				"-rw-r--r-- 1     5 2024-06-01 09:05:07.000000000 +0000 recent\n"},
		{"iso", with(func(o *ls.Options) { o.TimeStyle = "iso" }),
			"-r--r--r-- 1   100 2023-12-15  edge\n" +
				"-rwxr-xr-x 1     0 2024-07-04  future\n" +
				"-rw------- 1 12345 2021-02-03  old\n" +
				"-rw-r--r-- 1     5 06-01 09:05 recent\n"},
		{"posix prefix is accepted", with(func(o *ls.Options) { o.TimeStyle = "posix-long-iso"; o.Ignore = nil }),
			"-r--r--r-- 1   100 2023-12-15 12:00 edge\n" +
				"-rwxr-xr-x 1     0 2024-07-04 00:00 future\n" +
				"-rw------- 1 12345 2021-02-03 04:05 old\n" +
				"-rw-r--r-- 1     5 2024-06-01 09:05 recent\n"},
		{"strftime format", with(func(o *ls.Options) { o.TimeStyle = "+%Y/%m/%d %H.%M.%S %a %j" }),
			"-r--r--r-- 1   100 2023/12/15 12.00.00 Fri 349 edge\n" +
				"-rwxr-xr-x 1     0 2024/07/04 00.00.00 Thu 186 future\n" +
				"-rw------- 1 12345 2021/02/03 04.05.06 Wed 034 old\n" +
				"-rw-r--r-- 1     5 2024/06/01 09.05.07 Sat 153 recent\n"},
		{"strftime format pair: old then recent", with(func(o *ls.Options) { o.TimeStyle = "+%Y\n%b %e" }),
			"-r--r--r-- 1   100 2023 edge\n" +
				"-rwxr-xr-x 1     0 2024 future\n" +
				"-rw------- 1 12345 2021 old\n" +
				"-rw-r--r-- 1     5 Jun  1 recent\n"},
		{"human readable", with(func(o *ls.Options) { o.HumanReadable = true; o.TimeStyle = "+" }),
			"-r--r--r-- 1 100  edge\n" +
				"-rwxr-xr-x 1   0  future\n" +
				"-rw------- 1 13K  old\n" +
				"-rw-r--r-- 1   5  recent\n"},
		{"sorted by time", with(func(o *ls.Options) { o.Sort = ls.SortTime; o.TimeStyle = "+" }),
			"-rwxr-xr-x 1     0  future\n" +
				"-rw-r--r-- 1     5  recent\n" +
				"-r--r--r-- 1   100  edge\n" +
				"-rw------- 1 12345  old\n"},
		{"classify", with(func(o *ls.Options) { o.Indicator = ls.IndicatorClassify; o.TimeStyle = "+" }),
			"-r--r--r-- 1   100  edge\n" +
				"-rwxr-xr-x 1     0  future*\n" +
				"-rw------- 1 12345  old\n" +
				"-rw-r--r-- 1     5  recent\n"},
		{"quoted names", with(func(o *ls.Options) { o.Quoting = ls.QuoteC; o.TimeStyle = "+" }),
			"-r--r--r-- 1   100  \"edge\"\n" +
				"-rwxr-xr-x 1     0  \"future\"\n" +
				"-rw------- 1 12345  \"old\"\n" +
				"-rw-r--r-- 1     5  \"recent\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := list(t, tt.opts, files...); got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}

	t.Run("TIME_STYLE supplies the default style", func(t *testing.T) {
		l, out, _ := newEngine(base)
		l.Getenv = func(key string) string {
			if key == "TIME_STYLE" {
				return "long-iso"
			}
			return ""
		}
		if err := l.Run([]string{"recent"}); err != nil {
			t.Fatal(err)
		}
		if want := "-rw-r--r-- 1 5 2024-06-01 09:05 recent\n"; out.String() != want {
			t.Fatalf("got %q, want %q", out.String(), want)
		}
	})

	t.Run("timestamps follow the engine's location", func(t *testing.T) {
		l, out, _ := newEngine(with(func(o *ls.Options) { o.TimeStyle = "full-iso" }))
		l.Location = time.FixedZone("EST", -5*60*60)
		if err := l.Run([]string{"recent"}); err != nil {
			t.Fatal(err)
		}
		if want := "-rw-r--r-- 1 5 2024-06-01 04:05:07.000000000 -0500 recent\n"; out.String() != want {
			t.Fatalf("got %q, want %q", out.String(), want)
		}
	})

	t.Run("numeric owner is right-aligned", func(t *testing.T) {
		opts := ls.Options{Format: ls.FormatLong, NumericIDs: true, NoGroup: true, TimeStyle: "+"}
		uid := strconv.Itoa(os.Getuid())
		want := regexp.MustCompile(`^-rw------- 1 ` + uid + ` 12345  old\n-rw-r--r-- 1 ` + uid + `     5  recent\n$`)
		if got := list(t, opts, "old", "recent"); !want.MatchString(got) {
			t.Fatalf("got:\n%s\nwant match for %s", got, want)
		}
	})

	t.Run("owner and group columns are present by default", func(t *testing.T) {
		got := list(t, ls.Options{Format: ls.FormatLong, TimeStyle: "+"}, "recent")
		if want := regexp.MustCompile(`^-rw-r--r-- 1 \S+ \S+ 5  recent\n$`); !want.MatchString(got) {
			t.Fatalf("got %q, want match for %s", got, want)
		}
	})

	t.Run("directory listing has a total line", func(t *testing.T) {
		empty := filepath.Join(dir, "empties")
		touch(t, empty, "a", "b")
		for _, name := range []string{"a", "b"} {
			setTime(t, filepath.Join(empty, name), now.Add(-time.Hour))
		}
		want := "total 0\n" +
			"-rw-r--r-- 1 0 Jun 15 11:00 a\n" +
			"-rw-r--r-- 1 0 Jun 15 11:00 b\n"
		if got := list(t, base, "empties"); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestModeString(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("asserts POSIX permission bits")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	tests := []struct {
		name  string
		isDir bool
		mode  os.FileMode
		want  string
	}{
		{"plain", false, 0o640, "-rw-r-----"},
		{"setuid with execute", false, os.ModeSetuid | 0o755, "-rwsr-xr-x"},
		{"setuid without execute", false, os.ModeSetuid | 0o644, "-rwSr--r--"},
		{"setgid with execute", false, os.ModeSetgid | 0o755, "-rwxr-sr-x"},
		{"setgid without execute", false, os.ModeSetgid | 0o644, "-rw-r-Sr--"},
		{"sticky directory", true, os.ModeSticky | 0o777, "drwxrwxrwt"},
		{"sticky without other-execute", true, os.ModeSticky | 0o770, "drwxrwx--T"},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := "f" + string(rune('a'+i))
			if tt.isDir {
				mkdir(t, name)
			} else {
				touch(t, dir, name)
			}
			if err := os.Chmod(name, tt.mode); err != nil {
				t.Skipf("chmod %v: %v", tt.mode, err)
			}
			if info, err := os.Lstat(name); err != nil || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != tt.mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) {
				t.Skipf("filesystem dropped the special bits (mode %v, err %v)", info.Mode(), err)
			}
			got := list(t, ls.Options{Format: ls.FormatLong, Directory: true}, name)
			if !strings.HasPrefix(got, tt.want+" ") {
				t.Fatalf("got %q, want mode %q", got, tt.want)
			}
		})
	}
}

func TestHumanReadableSizes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	tests := []struct {
		size int64
		iec  string // -h
		si   string // --si
	}{
		{0, "0", "0"},
		{999, "999", "999"},
		{1000, "1000", "1.0k"},
		{1023, "1023", "1.1k"},
		{1024, "1.0K", "1.1k"},
		{1025, "1.1K", "1.1k"},
		{10137, "9.9K", "11k"},
		{10138, "10K", "11k"},
		{10240, "10K", "11k"},
		{10241, "11K", "11k"},
		{1048575, "1.0M", "1.1M"},
		{1048576, "1.0M", "1.1M"},
		{1500000, "1.5M", "1.5M"},
		{3 << 20, "3.0M", "3.2M"},
	}
	for _, tt := range tests {
		name := "f"
		sized(t, name, tt.size)
		for _, mode := range []struct {
			opts ls.Options
			want string
		}{
			{ls.Options{HumanReadable: true}, tt.iec},
			{ls.Options{SI: true}, tt.si},
		} {
			opts := mode.opts
			opts.Format, opts.NoOwner, opts.NoGroup, opts.TimeStyle = ls.FormatLong, true, true, "+"
			fields := strings.Fields(list(t, opts, name))
			if len(fields) != 4 || fields[2] != mode.want {
				t.Errorf("size %d with %+v: got %q, want %q", tt.size, mode.opts, fields, mode.want)
			}
		}
	}
}

func TestBlocksAndInodes(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a", "bb")

	t.Run("size column and total for empty files", func(t *testing.T) {
		if got, want := list(t, ls.Options{ShowBlocks: true}, dir), "total 0\n0 a\n0 bb\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("no total for file operands", func(t *testing.T) {
		got := list(t, ls.Options{ShowBlocks: true, Format: ls.FormatCommas, Width: -1}, filepath.Join(dir, "a"))
		if want := "0 " + filepath.Join(dir, "a") + "\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("allocated blocks are counted in 1K units", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("block counts are placeholders on Windows")
		}
		sub := t.TempDir()
		if err := os.WriteFile(filepath.Join(sub, "f"), bytes.Repeat([]byte("x"), 3000), 0o644); err != nil {
			t.Fatal(err)
		}
		// The allocation depends on the filesystem's block size, but the total
		// must agree with the one entry and be at least the 3 KiB written.
		got := list(t, ls.Options{ShowBlocks: true}, sub)
		m := regexp.MustCompile(`^total (\d+)\n(\d+) f\n$`).FindStringSubmatch(got)
		if m == nil || m[1] != m[2] {
			t.Fatalf("got %q", got)
		}
		if blocks, _ := strconv.Atoi(m[1]); blocks < 3 {
			t.Fatalf("got %d blocks for a 3000-byte file, want at least 3", blocks)
		}
	})
	t.Run("inode column", func(t *testing.T) {
		got := list(t, ls.Options{Inode: true}, dir)
		if !regexp.MustCompile(`^ *\d+ a\n *\d+ bb\n$`).MatchString(got) {
			t.Fatalf("got %q", got)
		}
	})
}

func TestColor(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "notes.txt", "sub/x", "plain")
	const reset = "\x1b[0m"

	tests := []struct {
		name     string
		opts     ls.Options
		terminal bool
		env      map[string]string
		want     string
	}{
		{"never", ls.Options{Format: ls.FormatSingle}, true, nil, "notes.txt\nplain\nsub\n"},
		{"auto off a terminal", ls.Options{Color: ls.WhenAuto}, false, nil, "notes.txt\nplain\nsub\n"},
		{"auto on a terminal", ls.Options{Color: ls.WhenAuto, Format: ls.FormatSingle}, true, nil,
			"notes.txt\nplain\n" + reset + "\x1b[01;34msub" + reset + "\n"},
		{"auto on a dumb terminal", ls.Options{Color: ls.WhenAuto, Format: ls.FormatSingle}, true, map[string]string{"TERM": "dumb"},
			"notes.txt\nplain\nsub\n"},
		{"always", ls.Options{Color: ls.WhenAlways}, false, nil,
			"notes.txt\nplain\n" + reset + "\x1b[01;34msub" + reset + "\n"},
		{"indicator stays outside the color", ls.Options{Color: ls.WhenAlways, Indicator: ls.IndicatorSlash}, false, nil,
			"notes.txt\nplain\n" + reset + "\x1b[01;34msub" + reset + "/\n"},
		{"LS_COLORS overrides types and adds suffixes", ls.Options{Color: ls.WhenAlways}, false,
			map[string]string{"LS_COLORS": "di=35:*.txt=31:bogus"},
			reset + "\x1b[31mnotes.txt" + reset + "\nplain\n\x1b[35msub" + reset + "\n"},
		{"columns are sized by the visible name", ls.Options{Color: ls.WhenAlways, Format: ls.FormatColumns, Width: 40}, false, nil,
			"notes.txt  plain  " + reset + "\x1b[01;34msub" + reset + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, out, errb := newEngine(tt.opts)
			l.IsTerminal = tt.terminal
			l.Getenv = func(key string) string { return tt.env[key] }
			if err := l.Run([]string{dir}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			if got := out.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("symlinks and their targets", func(t *testing.T) {
		symlink(t, "sub", filepath.Join(dir, "link"))
		symlink(t, "gone", filepath.Join(dir, "broken"))
		opts := ls.Options{Color: ls.WhenAlways, Format: ls.FormatLong, Ignore: []string{"[!bl]*"}}
		lines := strings.Split(list(t, opts, dir), "\n")
		if len(lines) != 4 {
			t.Fatalf("got %q", lines)
		}
		if want := " " + reset + "\x1b[01;36mbroken" + reset + " -> gone"; !strings.HasSuffix(lines[1], want) {
			t.Errorf("broken link line = %q, want suffix %q", lines[1], want)
		}
		if want := " \x1b[01;36mlink" + reset + " -> \x1b[01;34msub" + reset; !strings.HasSuffix(lines[2], want) {
			t.Errorf("link line = %q, want suffix %q", lines[2], want)
		}
	})
}

func TestTimeKindSortsWithoutLongFormat(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a", "b")
	// Access times: b newer than a. Modification times: a newer than b.
	if err := os.Chtimes(filepath.Join(dir, "a"), now.Add(-2*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "b"), now.Add(-time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Reading the directory must not disturb the access times under test, so
	// the files are passed as operands.
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")

	t.Run("mtime sort", func(t *testing.T) {
		if got, want := list(t, ls.Options{Sort: ls.SortTime}, a, b), a+"\n"+b+"\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("access time without -l sorts by it", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("access times are not reliably settable")
		}
		if got, want := list(t, ls.Options{Time: ls.TimeAccessed}, a, b), b+"\n"+a+"\n"; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("access time with -l shows it and sorts by name", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("access times are not reliably settable")
		}
		opts := ls.Options{Time: ls.TimeAccessed, Format: ls.FormatLong, NoOwner: true, NoGroup: true, TimeStyle: "+%H:%M"}
		got := list(t, opts, a, b)
		want := regexp.MustCompile(`^\S+ 1 0 10:00 ` + regexp.QuoteMeta(a) + `\n\S+ 1 0 11:00 ` + regexp.QuoteMeta(b) + `\n$`)
		if !want.MatchString(got) {
			t.Fatalf("got %q, want match for %s", got, want)
		}
	})
}

func TestInvalidTimeStyle(t *testing.T) {
	l, _, _ := newEngine(ls.Options{TimeStyle: "bogus"})
	var usage *core.UsageError
	if err := l.Run(nil); !errors.As(err, &usage) {
		t.Fatalf("err = %v, want a usage error", err)
	}
}

func TestParseWords(t *testing.T) {
	t.Run("sort", func(t *testing.T) {
		tests := []struct {
			arg  string
			want ls.SortKey
			ok   bool
		}{
			{"none", ls.SortNone, true},
			{"name", ls.SortName, true},
			{"time", ls.SortTime, true},
			{"size", ls.SortSize, true},
			{"extension", ls.SortExtension, true},
			{"version", ls.SortVersion, true},
			{"width", ls.SortWidth, true},
			{"t", ls.SortTime, true}, // unambiguous prefix
			{"ext", ls.SortExtension, true},
			{"n", 0, false}, // none or name
			{"", 0, false},
			{"bogus", 0, false},
		}
		for _, tt := range tests {
			got, err := ls.ParseSort(tt.arg)
			if (err == nil) != tt.ok || (tt.ok && got != tt.want) {
				t.Errorf("ParseSort(%q) = %v, %v; want %v, ok=%v", tt.arg, got, err, tt.want, tt.ok)
			}
		}
	})

	t.Run("time", func(t *testing.T) {
		tests := []struct {
			arg  string
			want ls.TimeKind
			ok   bool
		}{
			{"atime", ls.TimeAccessed, true},
			{"access", ls.TimeAccessed, true},
			{"use", ls.TimeAccessed, true},
			{"a", ls.TimeAccessed, true}, // atime or access: the same thing
			{"ctime", ls.TimeChanged, true},
			{"status", ls.TimeChanged, true},
			{"mtime", ls.TimeModified, true},
			{"modification", ls.TimeModified, true},
			{"birth", ls.TimeBirth, true},
			{"creation", ls.TimeBirth, true},
			{"c", 0, false}, // ctime or creation
			{"never", 0, false},
		}
		for _, tt := range tests {
			got, err := ls.ParseTime(tt.arg)
			if (err == nil) != tt.ok || (tt.ok && got != tt.want) {
				t.Errorf("ParseTime(%q) = %v, %v; want %v, ok=%v", tt.arg, got, err, tt.want, tt.ok)
			}
		}
	})

	t.Run("when", func(t *testing.T) {
		tests := []struct {
			arg  string
			want ls.When
			ok   bool
		}{
			{"always", ls.WhenAlways, true},
			{"yes", ls.WhenAlways, true},
			{"force", ls.WhenAlways, true},
			{"never", ls.WhenNever, true},
			{"no", ls.WhenNever, true},
			{"none", ls.WhenNever, true},
			{"n", ls.WhenNever, true},
			{"auto", ls.WhenAuto, true},
			{"tty", ls.WhenAuto, true},
			{"if-tty", ls.WhenAuto, true},
			{"a", 0, false}, // always or auto
			{"sometimes", 0, false},
		}
		for _, tt := range tests {
			got, err := ls.ParseWhen("--color", tt.arg)
			if (err == nil) != tt.ok || (tt.ok && got != tt.want) {
				t.Errorf("ParseWhen(%q) = %v, %v; want %v, ok=%v", tt.arg, got, err, tt.want, tt.ok)
			}
		}
	})

	t.Run("error wording", func(t *testing.T) {
		_, err := ls.ParseWhen("--color", "sometimes")
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("err = %v, want a usage error", err)
		}
		if !strings.HasPrefix(usage.Msg, "invalid argument 'sometimes' for '--color'\nValid arguments are:\n  - 'always'\n") {
			t.Fatalf("message = %q", usage.Msg)
		}
		if _, err := ls.ParseWhen("--color", "a"); err == nil || !strings.HasPrefix(err.Error(), "ambiguous argument 'a' for '--color'") {
			t.Fatalf("err = %v, want an ambiguity error", err)
		}
	})

	t.Run("time style", func(t *testing.T) {
		for _, style := range []string{"", "full-iso", "long-iso", "iso", "locale", "posix-iso", "+%Y", "full", "lon"} {
			if err := ls.ValidateTimeStyle(style); err != nil {
				t.Errorf("ValidateTimeStyle(%q) = %v, want nil", style, err)
			}
		}
		for _, style := range []string{"bogus", "l", "iso8601"} {
			if err := ls.ValidateTimeStyle(style); err == nil {
				t.Errorf("ValidateTimeStyle(%q) = nil, want an error", style)
			}
		}
	})
}
