package test_test

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
	"github.com/braswelljr/gnux/core/test"
)

// run evaluates args and returns the exit status and the error stream.
func run(args ...string) (int, string) {
	var out, errb bytes.Buffer
	err := test.New(test.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return status(err), errb.String()
}

func status(err error) int {
	if err == nil {
		return 0
	}
	var exit *core.ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return -1
}

func TestStrings(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"no arguments is false", nil, 1},
		{"one non-empty", []string{"x"}, 0},
		{"one empty", []string{""}, 1},
		{"a lone operator is a string", []string{"-n"}, 0},
		{"a lone ! is a string", []string{"!"}, 0},
		{"-n non-empty", []string{"-n", "a"}, 0},
		{"-n empty", []string{"-n", ""}, 1},
		{"-z empty", []string{"-z", ""}, 0},
		{"-z non-empty", []string{"-z", "a"}, 1},
		{"! empty", []string{"!", ""}, 0},
		{"! non-empty", []string{"!", "a"}, 1},
		{"equal", []string{"a", "=", "a"}, 0},
		{"== is equal", []string{"a", "==", "a"}, 0},
		{"not equal", []string{"a", "!=", "b"}, 0},
		{"equal fails", []string{"a", "=", "b"}, 1},
		{"less", []string{"abc", "<", "abd"}, 0},
		{"greater", []string{"b", ">", "a"}, 0},
		{"bytes compare (C locale)", []string{"B", "<", "a"}, 0},
		{"three args: operator wins over !", []string{"!", "=", "!"}, 0},
		{"three args: ! two", []string{"!", "-z", "a"}, 0},
		{"three args: parenthesized", []string{"(", "x", ")"}, 0},
		{"three args: -a", []string{"a", "-a", ""}, 1},
		{"three args: -o", []string{"a", "-o", ""}, 0},
		{"four args: ! three", []string{"!", "a", "=", "b"}, 0},
		{"four args: parenthesized two", []string{"(", "-n", "", ")"}, 1},
		{"and", []string{"-n", "a", "-a", "-z", ""}, 0},
		{"and false", []string{"-n", "a", "-a", "-n", ""}, 1},
		{"or", []string{"-n", "", "-o", "-n", "a"}, 0},
		{"-a binds tighter than -o", []string{"x", "-o", "", "-a", ""}, 0},
		{"parentheses group", []string{"(", "x", "-o", "", ")", "-a", ""}, 1},
		{"! in the full grammar", []string{"!", "-n", "", "-a", "x"}, 0},
		{"! twice", []string{"!", "!", "a", "=", "a"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := run(tt.args...)
			if got != tt.want || errs != "" {
				t.Fatalf("test %q = %d (stderr %q), want %d", tt.args, got, errs, tt.want)
			}
		})
	}
}

func TestIntegers(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"1", "-eq", "1"}, 0},
		{[]string{"1", "-eq", "2"}, 1},
		{[]string{"1", "-ne", "2"}, 0},
		{[]string{"-3", "-lt", "2"}, 0},
		{[]string{"2", "-le", "2"}, 0},
		{[]string{"3", "-gt", "2"}, 0},
		{[]string{"2", "-ge", "3"}, 1},
		{[]string{" 7 ", "-eq", "+7"}, 0},
		{[]string{"007", "-eq", "7"}, 0},
		{[]string{"99999999999999999999999", "-gt", "99999999999999999999998"}, 0},
	}
	for _, tt := range tests {
		if got, errs := run(tt.args...); got != tt.want {
			t.Errorf("test %q = %d (stderr %q), want %d", tt.args, got, errs, tt.want)
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"a", "b"}, "testx: 'a': unary operator expected\n"},
		{[]string{"a", "b", "c"}, "testx: 'b': binary operator expected\n"},
		{[]string{"1", "-eq", "x"}, "testx: invalid integer 'x'\n"},
		{[]string{"1.5", "-lt", "2"}, "testx: invalid integer '1.5'\n"},
		{[]string{"a", "=", "b", "c", "d"}, "testx: extra argument 'c'\n"},
		{[]string{"-n", "a", "-a", "-f"}, "testx: missing argument after '-f'\n"},
		{[]string{"x", "-a", "y", "-a"}, "testx: missing argument after '-a'\n"},
		{[]string{"(", "a", "-a", "b"}, "testx: ')' expected\n"},
		{[]string{"(", "a", "=", "a", "b"}, "testx: ')' expected, found 'b'\n"},
		{[]string{"-t", "x"}, "testx: invalid integer 'x'\n"},
	}
	for _, tt := range tests {
		got, errs := run(tt.args...)
		if got != 2 || errs != tt.want {
			t.Errorf("test %q = %d, stderr %q; want 2, %q", tt.args, got, errs, tt.want)
		}
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	tests := []struct {
		args []string
		want int
	}{
		{[]string{"-e", file}, 0},
		{[]string{"-e", missing}, 1},
		{[]string{"-f", file}, 0},
		{[]string{"-f", dir}, 1},
		{[]string{"-d", dir}, 0},
		{[]string{"-d", file}, 1},
		{[]string{"-s", file}, 0},
		{[]string{"-s", empty}, 1},
		{[]string{"-r", file}, 0},
		{[]string{"-r", missing}, 1},
		{[]string{"-w", file}, 0},
		{[]string{"-x", dir}, 0},
		{[]string{"-b", file}, 1},
		{[]string{"-c", file}, 1},
		{[]string{"-p", file}, 1},
		{[]string{"-S", file}, 1},
		{[]string{"-h", file}, 1},
		{[]string{"-u", file}, 1},
		{[]string{"-g", file}, 1},
		{[]string{"-k", file}, 1},
		{[]string{file, "-ef", file}, 0},
		{[]string{file, "-ef", empty}, 1},
		{[]string{file, "-ef", missing}, 1},
		{[]string{file, "-nt", missing}, 0},
		{[]string{missing, "-nt", file}, 1},
		{[]string{missing, "-ot", file}, 0},
		{[]string{"-f", file, "-a", "-d", dir}, 0},
	}
	for _, tt := range tests {
		if got, errs := run(tt.args...); got != tt.want {
			t.Errorf("test %q = %d (stderr %q), want %d", tt.args, got, errs, tt.want)
		}
	}
}

func TestNewerOlder(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "older")
	newer := filepath.Join(dir, "newer")
	for _, p := range []string{older, newer} {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(older, base, base); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, base.Add(time.Hour), base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got, _ := run(newer, "-nt", older); got != 0 {
		t.Errorf("-nt = %d, want 0", got)
	}
	if got, _ := run(older, "-nt", newer); got != 1 {
		t.Errorf("reverse -nt = %d, want 1", got)
	}
	if got, _ := run(older, "-ot", newer); got != 0 {
		t.Errorf("-ot = %d, want 0", got)
	}
}

func TestSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, op := range []string{"-h", "-L"} {
		if got, _ := run(op, link); got != 0 {
			t.Errorf("%s link = %d, want 0", op, got)
		}
	}
	if got, _ := run("-f", link); got != 0 {
		t.Errorf("-f follows the link: got %d", got)
	}
}

func TestOwnership(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no owners on Windows")
	}
	file := filepath.Join(t.TempDir(), "mine")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := run("-O", file); got != 0 {
		t.Errorf("-O = %d, want 0", got)
	}
	if got, _ := run("-G", file); got != 0 {
		t.Errorf("-G = %d, want 0", got)
	}
}

func TestInjection(t *testing.T) {
	tt := test.New(test.Options{}, core.Discard())
	tt.IsTerminal = func(fd int) bool { return fd == 5 }
	tt.Access = func(string, uint32) bool { return false }
	if ok, err := tt.Eval([]string{"-t", "5"}); !ok || err != nil {
		t.Fatalf("-t 5 = %v, %v", ok, err)
	}
	if ok, _ := tt.Eval([]string{"-t", "1"}); ok {
		t.Fatal("-t 1 should be false")
	}
	if ok, _ := tt.Eval([]string{"-r", "."}); ok {
		t.Fatal("-r should follow the injected Access")
	}
	// -t on the default streams: buffers are never terminals.
	if got, _ := run("-t", "0"); got != 1 {
		t.Fatalf("-t 0 on a buffer = %d, want 1", got)
	}
}

func TestLength(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"-l", "abc", "-eq", "3"}, 0},
		{[]string{"-l", "", "-eq", "0"}, 0},
		{[]string{"-l", "abc", "-gt", "5"}, 1},
		{[]string{"5", "-gt", "-l", "abc"}, 0},
		{[]string{"-l", "ab", "-lt", "-l", "abc"}, 0},
		{[]string{"!", "-l", "abc", "-eq", "3"}, 1},
		{[]string{"-l", "abc", "-eq", "3", "-a", "x"}, 0},
		{[]string{"-l", "abc"}, 2},
		{[]string{"-l", "-eq", "3"}, 2},
		// The string comparisons, as GNU's parser reads them.
		{[]string{"-l", "abc", "=", "3"}, 1},
		{[]string{"-l", "abc", "=", "abc"}, 0},
		{[]string{"3", "=", "-l", "abc"}, 1},
		{[]string{"3", "=", "-l", "="}, 0},
		{[]string{"-l", "x", "!=", "-l", "y"}, 0},
	}
	for _, tt := range tests {
		if got, errOut := run(tt.args...); got != tt.want {
			t.Errorf("test %q = %d (stderr %q); want %d", tt.args, got, errOut, tt.want)
		}
	}
	if got, errOut := run("-l", "abc", "-nt", "x"); got != 2 || errOut != "testx: -nt does not accept -l\n" {
		t.Errorf("-l with -nt = %d, %q", got, errOut)
	}
}

func TestBracket(t *testing.T) {
	bracket := func(args ...string) (int, string) {
		var errb bytes.Buffer
		err := test.New(test.Options{Bracket: true}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb}).Run(args)
		return status(err), errb.String()
	}
	tests := []struct {
		args    []string
		want    int
		wantErr string
	}{
		{[]string{"1", "-eq", "1", "]"}, 0, ""},
		{[]string{"a", "=", "b", "]"}, 1, ""},
		{[]string{"]"}, 1, ""},
		{[]string{"]", "]"}, 0, ""},
		{[]string{"-n", "x", "]"}, 0, ""},
		{[]string{"1", "-eq", "1"}, 2, "[: missing ']'\n"},
		{nil, 2, "[: missing ']'\n"},
		{[]string{"1", "-eq", "x", "]"}, 2, "[: invalid integer 'x'\n"},
	}
	for _, tt := range tests {
		got, errOut := bracket(tt.args...)
		if got != tt.want || errOut != tt.wantErr {
			t.Errorf("[ %q = %d, %q; want %d, %q", tt.args, got, errOut, tt.want, tt.wantErr)
		}
	}
}
