package ls_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/ls"
)

// sgr matches a terminal color sequence.
var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// tree builds src/{a.go,lib/{b.go,deep/c.go},z.txt} under a temporary directory.
func tree(t *testing.T) string {
	dir := t.TempDir()
	touch(t, dir, "src/a.go", "src/lib/b.go", "src/lib/deep/c.go", "src/z.txt")
	return dir
}

func TestTree(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	got := list(t, ls.Options{Tree: true}, "src")
	want := "src\n" +
		"├── a.go\n" +
		"├── lib\n" +
		"│   ├── b.go\n" +
		"│   └── deep\n" +
		"│       └── c.go\n" +
		"└── z.txt\n"
	if got != want {
		t.Errorf("tree:\n%s\nwant:\n%s", got, want)
	}

	got = list(t, ls.Options{Tree: true, Level: 1}, "src")
	if want := "src\n├── a.go\n├── lib\n└── z.txt\n"; got != want {
		t.Errorf("--level=1:\n%s\nwant:\n%s", got, want)
	}

	got = list(t, ls.Options{Tree: true, Only: ls.OnlyDirs, GroupDirectories: true}, "src")
	if want := "src\n└── lib\n    └── deep\n"; got != want {
		t.Errorf("--only-dirs:\n%s\nwant:\n%s", got, want)
	}

	// Several operands are several roots; -a shows no "." or "..".
	got = list(t, ls.Options{Tree: true, All: true, Level: 1}, "src/lib", "src/a.go")
	if want := "src/a.go\nsrc/lib\n├── b.go\n└── deep\n"; got != want {
		t.Errorf("two roots:\n%s\nwant:\n%s", got, want)
	}
}

func TestTreeASCII(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	l, out, _ := newEngine(ls.Options{Tree: true, Level: 1})
	l.Getenv = func(k string) string {
		if k == "LANG" {
			return "C"
		}
		return ""
	}
	if err := l.Run([]string{"src"}); err != nil {
		t.Fatal(err)
	}
	if want := "src\n|-- a.go\n|-- lib\n`-- z.txt\n"; out.String() != want {
		t.Errorf("ASCII tree:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestTreeLong(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	got := list(t, ls.Options{Tree: true, Format: ls.FormatLong, Level: 1, NoOwner: true, NoGroup: true}, "src")
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("want a row per entry and no total line:\n%s", got)
	}
	for i, suffix := range []string{" src", " ├── a.go", " ├── lib", " └── z.txt"} {
		if !strings.HasSuffix(lines[i], suffix) {
			t.Errorf("row %d = %q, want it to end %q", i, lines[i], suffix)
		}
	}
	// One table: the names all start in the same column.
	at := strings.Index(lines[0], " src")
	if strings.Index(lines[1], " ├── a.go") != at {
		t.Errorf("columns do not line up:\n%s", got)
	}
}

func TestRecursiveLevel(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	got := list(t, ls.Options{Recursive: true, Level: 2}, "src")
	want := "src:\na.go\nlib\nz.txt\n\nsrc/lib:\nb.go\ndeep\n"
	if got != want {
		t.Errorf("-R --level=2:\n%s\nwant:\n%s", got, want)
	}
}

func TestOnlyFilesStillRecurses(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	got := list(t, ls.Options{Recursive: true, Only: ls.OnlyFiles}, "src")
	want := "src:\na.go\nz.txt\n\nsrc/lib:\nb.go\n\nsrc/lib/deep:\nc.go\n"
	if got != want {
		t.Errorf("-R --only-files:\n%s\nwant:\n%s", got, want)
	}
}

func TestIcons(t *testing.T) {
	dir := tree(t)
	t.Chdir(dir)
	got := list(t, ls.Options{Icons: ls.WhenAlways}, "src")
	want := "\ue627 a.go\n\ue5ff lib\n\uf15c z.txt\n"
	if got != want {
		t.Errorf("icons:\n%q\nwant:\n%q", got, want)
	}
	// auto is off when the output is not a terminal.
	if got := list(t, ls.Options{Icons: ls.WhenAuto}, "src"); got != "a.go\nlib\nz.txt\n" {
		t.Errorf("--icons=auto off a terminal: %q", got)
	}
}

func TestIconsInColumns(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "aaaa", "bbbb", "cccc")
	t.Chdir(dir)
	// Each cell is the name plus the icon and its space.
	got := list(t, ls.Options{Format: ls.FormatAcross, Width: 20, Icons: ls.WhenAlways})
	want := "\uf15b aaaa  \uf15b bbbb\n\uf15b cccc\n"
	if got != want {
		t.Errorf("icons in columns:\n%q\nwant:\n%q", got, want)
	}
}

func TestHeaderAndOctal(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "f")
	if err := os.Chmod(filepath.Join(dir, "f"), 0o640); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	got := list(t, ls.Options{Format: ls.FormatLong, Header: true, OctalPermissions: true, NoOwner: true, NoGroup: true, TimeStyle: "long-iso"}, "f")
	want := "Octal Permissions Links Size Modified         Name\n" +
		"0640  -rw-r-----      1    0 "
	if !strings.HasPrefix(got, want) || !strings.HasSuffix(got, " f\n") {
		t.Errorf("header:\n%s\nwant it to start:\n%s", got, want)
	}
}

func TestTotalSize(t *testing.T) {
	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, "d", "e"))
	sized(t, filepath.Join(dir, "d", "x"), 1000)
	sized(t, filepath.Join(dir, "d", "e", "y"), 234)
	sized(t, filepath.Join(dir, "small"), 2000)
	t.Chdir(dir)
	got := list(t, ls.Options{Format: ls.FormatLong, TotalSize: true, NoOwner: true, NoGroup: true, Sort: ls.SortSize})
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")[1:]
	if len(lines) != 2 || !strings.HasSuffix(lines[0], " small") || !strings.Contains(lines[1], " 1234 ") {
		t.Errorf("--total-size -S:\n%s", got)
	}
}

func TestPrettyLong(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "f")
	t.Chdir(dir)
	plain := list(t, ls.Options{Format: ls.FormatLong, Color: ls.WhenAlways}, "f")
	pretty := list(t, ls.Options{Format: ls.FormatLong, Color: ls.WhenAlways, Pretty: true}, "f")
	if plain == pretty || !strings.Contains(pretty, "\x1b[1;33mr\x1b[0m") {
		t.Errorf("pretty should color the mode letters:\n%q", pretty)
	}
	// Under the colors the text is exactly GNU's.
	if sgr.ReplaceAllString(pretty, "") != sgr.ReplaceAllString(plain, "") {
		t.Errorf("pretty changed the text:\n%q\n%q", pretty, plain)
	}
	// Without color, Pretty adds nothing.
	if got := list(t, ls.Options{Format: ls.FormatLong, Pretty: true}, "f"); strings.Contains(got, "\x1b") {
		t.Errorf("pretty without color: %q", got)
	}

	l, out, _ := newEngine(ls.Options{Format: ls.FormatLong, Color: ls.WhenAlways, Pretty: true})
	l.Getenv = func(k string) string {
		if k == "GNUX_THEME" {
			return "mono"
		}
		return ""
	}
	if err := l.Run([]string{"f"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[1mr\x1b[0m") {
		t.Errorf("GNUX_THEME=mono: %q", out.String())
	}
}

func TestUnknownTheme(t *testing.T) {
	l, _, errb := newEngine(ls.Options{Color: ls.WhenAlways, Pretty: true, Theme: "neon"})
	if err := l.Run([]string{t.TempDir()}); err == nil || !strings.Contains(errb.String(), "invalid theme 'neon'") {
		t.Errorf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	touch(t, dir, "kept", "build/out.o", "notes.log")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "kept", ".gitignore")
	git("commit", "-q", "-m", "one")
	touch(t, dir, "new")
	t.Chdir(dir)

	got := list(t, ls.Options{GitIgnore: true})
	if got != "kept\nnew\n" {
		t.Errorf("--git-ignore:\n%s", got)
	}

	got = list(t, ls.Options{Format: ls.FormatLong, Git: true, NoOwner: true, NoGroup: true})
	for name, status := range map[string]string{"kept": "--", "new": "-N", "notes.log": "-I", "build": "-I"} {
		if !regexp.MustCompile(" " + status + " " + regexp.QuoteMeta(name) + "\n").MatchString(got) {
			t.Errorf("--git: %s should be %s:\n%s", name, status, got)
		}
	}

	// Outside a repository the column is blank.
	t.Chdir(t.TempDir())
	touch(t, ".", "x")
	got = list(t, ls.Options{Format: ls.FormatLong, Git: true, NoOwner: true, NoGroup: true})
	if !strings.HasSuffix(got, "    x\n") {
		t.Errorf("--git outside a repository:\n%q", got)
	}
}
