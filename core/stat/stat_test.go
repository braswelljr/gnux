package stat_test

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
	"github.com/braswelljr/gnux/core/stat"
)

func run(opts stat.Options, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	s := stat.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	s.Location = time.UTC
	err = s.Run(args)
	return out.String(), errb.String(), err
}

func format(f string) stat.Options { return stat.Options{Format: f, FormatSet: true} }

// fixture makes a 5-byte file with a known modification time.
func fixture(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1700000000, 500000000)
	if err := os.Chtimes(f, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFileDirectives(t *testing.T) {
	f := fixture(t)
	tests := []struct {
		format, want string
	}{
		{"%n", f},
		{"%s", "5"},
		{"%F", "regular file"},
		{"%-6s|", "5     |"},
		{"%6s|", "     5|"},
		{"%Y", "1700000000"},
		{"%.3Y", "1700000000.500"},
		{"%.Y", "1700000000.500000000"},
		{"%15.3Y|", " 1700000000.500|"},
		{"%-15.3Y|", "1700000000.500 |"},
		{"%y", "2023-11-14 22:13:20.500000000 +0000"},
		{"%B", "512"},
		{"100%%", "100%"},
		{"%Z %X", ""},
		{"%k", "?"},
	}
	for _, tt := range tests {
		out, errb, err := run(format(tt.format), f)
		if err != nil {
			t.Fatalf("%q: %v (%q)", tt.format, err, errb)
		}
		if tt.want != "" && out != tt.want+"\n" {
			t.Errorf("%q = %q, want %q", tt.format, out, tt.want+"\n")
		}
	}
}

func TestModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no POSIX permission bits")
	}
	f := fixture(t)
	if err := os.Chmod(f, 0o755|os.ModeSetuid); err != nil {
		t.Fatal(err)
	}
	out, _, _ := run(format("%a %05a %#a %A"), f)
	if out != "4755 04755 04755 -rwsr-xr-x\n" {
		t.Fatalf("modes: %q", out)
	}
	out, _, _ = run(format("%A %F"), filepath.Dir(f))
	if !strings.HasPrefix(out, "d") || !strings.HasSuffix(out, " directory\n") {
		t.Fatalf("directory: %q", out)
	}
	empty := filepath.Join(filepath.Dir(f), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, _, _ = run(format("%a %A %F"), empty); out != "600 -rw------- regular empty file\n" {
		t.Fatalf("empty: %q", out)
	}
}

func TestSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "my link")
	if err := os.Symlink("target", link); err != nil {
		t.Skip("symlinks unavailable")
	}
	out, _, _ := run(format("%N|%F"), link)
	if want := "'" + link + "' -> 'target'|symbolic link\n"; out != want { // GNU quotes the target too
		t.Fatalf("%%N = %q, want %q", out, want)
	}
	out, _, _ = run(stat.Options{Format: "%F %s", FormatSet: true, Dereference: true}, link)
	if out != "regular file 1\n" {
		t.Fatalf("-L: %q", out)
	}
	if out, _, _ = run(format("%Qn"), link); out != "'"+link+"'\n" {
		t.Fatalf("%%Qn: %q", out)
	}
}

func TestPrintf(t *testing.T) {
	f := fixture(t)
	out, _, _ := run(stat.Options{Printf: `%s\t\x41\101\n`, PrintfSet: true}, f)
	if out != "5\tAA\n" {
		t.Fatalf("--printf: %q", out)
	}
	// -c does not interpret escapes.
	if out, _, _ = run(format(`%s\t`), f); out != "5\\t\n" {
		t.Fatalf("-c: %q", out)
	}
	_, errb, _ := run(stat.Options{Printf: `\q`, PrintfSet: true}, f)
	if errb != "statx: warning: unrecognized escape '\\q'\n" {
		t.Fatalf("bad escape: %q", errb)
	}
}

func TestDefaultAndTerse(t *testing.T) {
	f := fixture(t)
	out, _, err := run(stat.Options{}, f)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 8 || lines[0] != "  File: "+f || // the default format names the file as it is, as GNU does
		!strings.HasPrefix(lines[1], "  Size: 5         \tBlocks: ") ||
		!strings.HasSuffix(lines[1], " regular file") ||
		!strings.HasPrefix(lines[2], "Device: ") ||
		lines[5] != "Modify: 2023-11-14 22:13:20.500000000 +0000" ||
		!strings.HasPrefix(lines[7], " Birth: ") {
		t.Fatalf("default format:\n%s", out)
	}
	out, _, _ = run(stat.Options{Terse: true}, f)
	if fields := strings.Fields(out); len(fields) != 16 || fields[1] != "5" || fields[12] != "1700000000" {
		t.Fatalf("terse: %q", out)
	}
}

func TestFileSystem(t *testing.T) {
	dir := t.TempDir()
	out, errb, err := run(stat.Options{FileSystem: true, Format: "%n %T", FormatSet: true}, dir)
	if err != nil {
		if strings.Contains(errb, "not supported") {
			t.Skip("no file system statistics here")
		}
		t.Fatalf("-f: %v %q", err, errb)
	}
	if !strings.HasPrefix(out, dir+" ") || len(out) <= len(dir)+2 {
		t.Fatalf("-f: %q", out)
	}
	out, _, _ = run(stat.Options{FileSystem: true, Terse: true}, dir)
	if fields := strings.Fields(out); len(fields) != 11 {
		t.Fatalf("-f -t: %q", out)
	}
	_, errb, err = run(stat.Options{FileSystem: true}, "-")
	if !errors.Is(err, core.ErrFailed) || errb != "statx: using '-' to denote standard input does not work in file system mode\n" {
		t.Fatalf("-f -: %v %q", err, errb)
	}
}

func TestErrors(t *testing.T) {
	f := fixture(t)
	missing := filepath.Join(t.TempDir(), "missing")
	out, errb, err := run(format("%s"), missing, f)
	if !errors.Is(err, core.ErrFailed) || errb != "statx: cannot stat '"+missing+"': No such file or directory\n" || out != "5\n" {
		t.Fatalf("missing: %v %q %q", err, errb, out)
	}
	_, errb, err = run(format("%5%"), f)
	if !errors.Is(err, core.ErrFailed) || errb != "statx: '%5%': invalid directive\n" {
		t.Fatalf("invalid directive: %v %q", err, errb)
	}
	_, errb, err = run(format("%C"), f)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "failed to get security context") {
		t.Fatalf("%%C: %v %q", err, errb)
	}
	_, _, err = run(stat.Options{})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("no operand: %v", err)
	}
}
