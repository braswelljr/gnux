package install_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/install"
)

func run(opts install.Options, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	err := install.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// unixMode skips a mode assertion where only the read-only bit exists.
func unixMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %v, want %v", path, got, want)
	}
}

func TestFileToFile(t *testing.T) {
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "a"), "hello")
	dst := filepath.Join(dir, "b")
	if _, msg, err := run(install.Options{}, src, dst); err != nil {
		t.Fatal(err, msg)
	}
	if read(t, dst) != "hello" {
		t.Fatal("content not copied")
	}
	unixMode(t, dst, 0o755)
}

func TestReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "a"), "new")
	dst := write(t, filepath.Join(dir, "b"), "old content")
	if _, _, err := run(install.Options{Mode: "644"}, src, dst); err != nil {
		t.Fatal(err)
	}
	if read(t, dst) != "new" {
		t.Fatal("not replaced")
	}
	unixMode(t, dst, 0o644)
}

func TestIntoDirectory(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	b := write(t, filepath.Join(dir, "b"), "B")
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(install.Options{Verbose: true}, a, b, bin)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(bin, "a")) != "A" || read(t, filepath.Join(bin, "b")) != "B" {
		t.Fatal("not copied into the directory")
	}
	want := "'" + a + "' -> '" + filepath.Join(bin, "a") + "'\n'" + b + "' -> '" + filepath.Join(bin, "b") + "'\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestTargetDirectoryAndLeading(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	target := filepath.Join(dir, "x", "y")
	_, msg, err := run(install.Options{TargetDirectory: target}, a)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(msg, "failed to access '"+target+"'") {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
	out, _, err := run(install.Options{TargetDirectory: target, CreateLeading: true, Verbose: true}, a)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(target, "a")) != "A" {
		t.Fatal("not installed")
	}
	if !strings.Contains(out, "installx: creating directory '"+filepath.Join(dir, "x")+"'") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestCreateLeadingForFile(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	dst := filepath.Join(dir, "p", "q", "a.conf")
	if _, _, err := run(install.Options{CreateLeading: true}, a, dst); err != nil {
		t.Fatal(err)
	}
	if read(t, dst) != "A" {
		t.Fatal("not installed")
	}
}

func TestDirectoryMode(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "a", "b")
	out, _, err := run(install.Options{Directory: true, Mode: "u=rwx,go=", Verbose: true}, d)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(d); err != nil || !info.IsDir() {
		t.Fatal("not created")
	}
	unixMode(t, d, 0o700)
	unixMode(t, filepath.Join(dir, "a"), os.FileMode(0o755&^umask()))
	want := "installx: creating directory '" + filepath.Join(dir, "a") + "'\ninstallx: creating directory '" + d + "'\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestNoTargetDirectory(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	_, msg, err := run(install.Options{NoTargetDirectory: true}, a, sub)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(msg, "cannot overwrite directory '"+sub+"' with non-directory") {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestCompareLeavesIdenticalFileAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("modes other than read-only are not representable")
	}
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "a"), "same")
	dst := filepath.Join(dir, "b")
	if _, _, err := run(install.Options{Mode: "644"}, src, dst); err != nil {
		t.Fatal(err)
	}
	old := time.Unix(1_000_000_000, 0)
	if err := os.Chtimes(dst, old, old); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(install.Options{Mode: "644", Compare: true, Verbose: true}, src, dst)
	if err != nil || out != "" {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if info, _ := os.Stat(dst); !info.ModTime().Equal(old) {
		t.Fatal("-C rewrote an identical file")
	}
	write(t, src, "different")
	if _, _, err := run(install.Options{Mode: "644", Compare: true}, src, dst); err != nil {
		t.Fatal(err)
	}
	if read(t, dst) != "different" {
		t.Fatal("-C left a changed file alone")
	}
}

func TestPreserveTimestamps(t *testing.T) {
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "a"), "A")
	stamp := time.Unix(1_500_000_000, 0)
	if err := os.Chtimes(src, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "b")
	if _, _, err := run(install.Options{PreserveTimestamps: true}, src, dst); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(dst); !info.ModTime().Equal(stamp) {
		t.Fatalf("mtime = %v, want %v", info.ModTime(), stamp)
	}
}

func TestErrorsContinue(t *testing.T) {
	dir := t.TempDir()
	good := write(t, filepath.Join(dir, "good"), "G")
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	_, msg, err := run(install.Options{}, missing, dir, good, bin)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	want := "installx: cannot stat '" + missing + "': No such file or directory\ninstallx: omitting directory '" + dir + "'\n"
	if msg != want {
		t.Fatalf("stderr = %q, want %q", msg, want)
	}
	if read(t, filepath.Join(bin, "good")) != "G" {
		t.Fatal("the good operand was not installed")
	}
}

func TestSameFile(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	_, msg, err := run(install.Options{}, a, a)
	if !errors.Is(err, core.ErrFailed) || msg != "installx: '"+a+"' and '"+a+"' are the same file\n" {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestNotADirectory(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	b := write(t, filepath.Join(dir, "b"), "B")
	c := filepath.Join(dir, "c")
	_, msg, err := run(install.Options{}, a, b, c)
	if !errors.Is(err, core.ErrFailed) || msg != "installx: target '"+c+"' is not a directory\n" {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestStripWarns(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "A")
	_, msg, err := run(install.Options{Strip: true}, a, filepath.Join(dir, "b"))
	if err != nil || !strings.Contains(msg, "stripping is not supported") {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestInvalidOwner(t *testing.T) {
	_, msg, err := run(install.Options{Owner: "no-such-user-gnux"}, "a", "b")
	if !errors.Is(err, core.ErrFailed) || msg != "installx: invalid user 'no-such-user-gnux'\n" {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
	_, msg, err = run(install.Options{Group: "no-such-group-gnux"}, "a", "b")
	if !errors.Is(err, core.ErrFailed) || msg != "installx: invalid group 'no-such-group-gnux'\n" {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts install.Options
		args []string
		want string
	}{
		{"no operands", install.Options{}, nil, "missing file operand"},
		{"no destination", install.Options{}, []string{"a"}, "missing destination file operand after 'a'"},
		{"t and T", install.Options{TargetDirectory: "d", NoTargetDirectory: true}, []string{"a"}, "cannot combine --target-directory (-t) and --no-target-directory (-T)"},
		{"d and t", install.Options{Directory: true, TargetDirectory: "d"}, []string{"a"}, "target directory not allowed when installing a directory"},
		{"T extra", install.Options{NoTargetDirectory: true}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{"bad mode", install.Options{Mode: "8"}, []string{"a", "b"}, "invalid mode '8'"},
		{"C and p", install.Options{Compare: true, PreserveTimestamps: true}, []string{"a", "b"}, "options --compare (-C) and --preserve-timestamps are mutually exclusive"},
		{"C and s", install.Options{Compare: true, Strip: true}, []string{"a", "b"}, "options --compare (-C) and --strip are mutually exclusive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(tt.opts, tt.args...)
			var u *core.UsageError
			if !errors.As(err, &u) || u.Msg != tt.want {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "src"), "new")
	dest := write(t, filepath.Join(dir, "dest"), "old")

	out, errs, err := run(install.Options{Backup: core.Backup{Control: core.SimpleBackup, Suffix: ".old"}, Verbose: true}, src, dest)
	if err != nil {
		t.Fatalf("err = %v, stderr = %q", err, errs)
	}
	if want := "'" + src + "' -> '" + dest + "' (backup: '" + dest + ".old')\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	for path, want := range map[string]string{dest: "new", dest + ".old": "old"} {
		if data, _ := os.ReadFile(path); string(data) != want {
			t.Fatalf("%s holds %q, want %q", path, data, want)
		}
	}

	// Nothing to replace, nothing to back up.
	fresh := filepath.Join(dir, "fresh")
	if _, _, err := run(install.Options{Backup: core.Backup{Control: core.NumberedBackup}}, src, fresh); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(fresh + ".~1~"); !os.IsNotExist(err) {
		t.Fatal("a backup was made of nothing")
	}
}
