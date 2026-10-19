package sort_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sort"
)

// run executes the engine against in-memory streams.
func run(opts sort.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = sort.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exitCode(err error) int {
	var e *core.ExitError
	if errors.As(err, &e) {
		return e.Code
	}
	return -1
}

func TestOrdering(t *testing.T) {
	tests := []struct {
		name  string
		opts  sort.Options
		input string
		want  string
	}{
		{"default byte order", sort.Options{}, "b\na\nC\nc\n", "C\na\nb\nc\n"},
		{"missing final newline", sort.Options{}, "b\na", "a\nb\n"},
		{"empty input", sort.Options{}, "", ""},
		{"empty lines first", sort.Options{}, "b\n\na\n", "\na\nb\n"},
		{"reverse", sort.Options{Reverse: true}, "a\nc\nb\n", "c\nb\na\n"},
		{"ignore case", sort.Options{IgnoreCase: true}, "b\nA\na\nB\n", "A\na\nB\nb\n"},
		{"ignore case stable keeps input order", sort.Options{IgnoreCase: true, Stable: true}, "b\na\nA\nB\n", "a\nA\nb\nB\n"},
		{"dictionary order", sort.Options{DictionaryOrder: true}, "-b\na\n_c\n", "a\n-b\n_c\n"},
		{"ignore nonprinting", sort.Options{IgnoreNonprinting: true}, "\x01b\na\n", "a\n\x01b\n"},
		{"ignore leading blanks", sort.Options{IgnoreLeadingBlanks: true}, "  b\na\n c\n", "a\n  b\n c\n"},
		{"blanks count without -b", sort.Options{}, "a\n b\n", " b\na\n"},

		{"numeric", sort.Options{Numeric: true}, "10\n9\n-1\n1.5\nx\n", "-1\nx\n1.5\n9\n10\n"},
		{"numeric zeros", sort.Options{Numeric: true, Stable: true}, "0\n-0\n0.0\n.0\n", "0\n-0\n0.0\n.0\n"},
		{"numeric negative fractions", sort.Options{Numeric: true}, "-1.5\n-1.25\n-2\n", "-2\n-1.5\n-1.25\n"},
		{"numeric huge", sort.Options{Numeric: true}, "123456789012345678901234567890\n99\n", "99\n123456789012345678901234567890\n"},
		{"numeric leading blanks", sort.Options{Numeric: true}, "  10\n 9\n", " 9\n  10\n"},
		{"numeric reverse", sort.Options{Numeric: true, Reverse: true}, "1\n3\n2\n", "3\n2\n1\n"},
		{"numeric ties fall back to the line", sort.Options{Numeric: true}, "1b\n1a\n", "1a\n1b\n"},

		{"general numeric", sort.Options{GeneralNumeric: true}, "1e3\n-inf\n2.5\nnan\nabc\n10\n", "abc\nnan\n-inf\n2.5\n10\n1e3\n"},
		{"human numeric", sort.Options{HumanNumeric: true}, "2K\n1G\n1000\n3M\n-1K\n", "-1K\n1000\n2K\n3M\n1G\n"},
		{"month", sort.Options{Month: true}, "Mar\nfeb\nxyz\n  jan\nDecember\n", "xyz\n  jan\nfeb\nMar\nDecember\n"},
		{"version", sort.Options{Version: true}, "v1.10\nv1.2\nv1.9\n", "v1.2\nv1.9\nv1.10\n"},
		{"version file names", sort.Options{Version: true}, "a-1.10.tar.gz\na-1.2.tar.gz\n.hidden\n", ".hidden\na-1.2.tar.gz\na-1.10.tar.gz\n"},
		{"version tilde before release", sort.Options{Version: true}, "1.0\n1.0~rc1\n", "1.0~rc1\n1.0\n"},

		{"unique", sort.Options{Unique: true}, "b\na\nb\na\n", "a\nb\n"},
		{"unique by numeric value keeps the first", sort.Options{Unique: true, Numeric: true}, "01\n1\n2\n", "01\n2\n"},
		{"unique ignore case", sort.Options{Unique: true, IgnoreCase: true}, "a\nA\nb\n", "a\nb\n"},

		{"zero terminated", sort.Options{ZeroTerminated: true}, "b\x00a\nz\x00", "a\nz\x00b\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			if err != nil {
				t.Fatalf("Run: %v (stderr %q)", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		name  string
		opts  sort.Options
		input string
		want  string
	}{
		{"second field", sort.Options{Keys: []string{"2"}}, "x b\ny a\n", "y a\nx b\n"},
		{"second field only, then the line", sort.Options{Keys: []string{"2,2"}}, "b a 2\na a 1\n", "a a 1\nb a 2\n"},
		{"numeric key modifier", sort.Options{Keys: []string{"2n"}}, "a 10\nb 9\n", "b 9\na 10\n"},
		{"numeric reverse key", sort.Options{Keys: []string{"2,2nr"}}, "a 1\nb 3\nc 2\n", "b 3\nc 2\na 1\n"},
		{"two keys", sort.Options{Keys: []string{"1,1", "2,2nr"}}, "a 1\nb 1\na 2\n", "a 2\na 1\nb 1\n"},
		{"global options apply to a plain key", sort.Options{Numeric: true, Keys: []string{"2"}}, "a 10\nb 9\n", "b 9\na 10\n"},
		{"a key with options ignores global ones", sort.Options{Reverse: true, Keys: []string{"2n"}}, "a 1\nb 2\n", "a 1\nb 2\n"},
		{"separator", sort.Options{FieldSeparator: ":", Keys: []string{"3n"}}, "root:x:0\nme:x:1000\nbin:x:2\n", "root:x:0\nbin:x:2\nme:x:1000\n"},
		{"separator with empty fields", sort.Options{FieldSeparator: ",", Keys: []string{"2,2"}}, "a,,z\nb,y,a\n", "a,,z\nb,y,a\n"},
		{"character position", sort.Options{Keys: []string{"1.2"}}, "ab\nba\n", "ba\nab\n"},
		{"character range", sort.Options{Keys: []string{"1.2,1.2"}}, "xbz\nyaz\n", "yaz\nxbz\n"},
		{"blank-separated fields keep leading blanks", sort.Options{Keys: []string{"2,2"}}, "a  b\nc a\n", "a  b\nc a\n"},
		{"b skips leading blanks of the key", sort.Options{Keys: []string{"2b,2"}}, "a  b\nc a\n", "c a\na  b\n"},
		{"stable keeps input order on equal keys", sort.Options{Stable: true, Keys: []string{"1,1"}}, "a 2\na 1\n", "a 2\na 1\n"},
		{"unique on a key", sort.Options{Unique: true, Keys: []string{"1,1"}}, "a 2\na 1\nb 3\n", "a 2\nb 3\n"},
		{"field past the end", sort.Options{Keys: []string{"5"}}, "b\na\n", "a\nb\n"},
		{"month key", sort.Options{Keys: []string{"2M"}}, "x Mar\ny Jan\n", "y Jan\nx Mar\n"},
		{"NUL separator", sort.Options{FieldSeparator: `\0`, Keys: []string{"2"}}, "a\x00b\nb\x00a\n", "b\x00a\na\x00b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			if err != nil {
				t.Fatalf("Run: %v (stderr %q)", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestBadKeys(t *testing.T) {
	tests := []struct {
		key, want string
	}{
		{"0", "sortx: field number is zero: invalid field specification '0'\n"},
		{"1.0", "sortx: character offset is zero: invalid field specification '1.0'\n"},
		{"x", "sortx: invalid number at field start: invalid field specification 'x'\n"},
		{"1,x", "sortx: invalid number after ',': invalid field specification '1,x'\n"},
		{"1.x", "sortx: invalid number after '.': invalid field specification '1.x'\n"},
		{"1,0", "sortx: field number is zero: invalid field specification '1,0'\n"},
		{"1q", "sortx: stray character in field spec: invalid field specification '1q'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			out, errb, err := run(sort.Options{Keys: []string{tt.key}}, "a\n")
			if exitCode(err) != 2 {
				t.Fatalf("err = %v, want exit status 2", err)
			}
			if errb != tt.want || out != "" {
				t.Fatalf("stderr = %q, stdout = %q; want %q", errb, out, tt.want)
			}
		})
	}
}

func TestIncompatible(t *testing.T) {
	tests := []struct {
		name string
		opts sort.Options
		want string
	}{
		{"global", sort.Options{Numeric: true, GeneralNumeric: true}, "sortx: options '-gn' are incompatible\n"},
		{"dictionary and numeric", sort.Options{Numeric: true, DictionaryOrder: true}, "sortx: options '-dn' are incompatible\n"},
		{"in a key", sort.Options{Keys: []string{"1hM"}}, "sortx: options '-hM' are incompatible\n"},
		{"multi-character tab", sort.Options{FieldSeparator: "ab"}, "sortx: multi-character tab 'ab'\n"},
		{"check with output", sort.Options{Check: sort.CheckDiagnose, Output: "x"}, "sortx: options '-co' are incompatible\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errb, err := run(tt.opts, "a\n")
			if exitCode(err) != 2 || errb != tt.want {
				t.Fatalf("err = %v, stderr = %q; want status 2 and %q", err, errb, tt.want)
			}
		})
	}

	// Two orderings for different keys are fine.
	if out, errb, err := run(sort.Options{Keys: []string{"1n", "2M"}}, "1 Feb\n1 Jan\n"); err != nil || out != "1 Jan\n1 Feb\n" {
		t.Fatalf("separate keys: out %q, stderr %q, err %v", out, errb, err)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name   string
		opts   sort.Options
		input  string
		status int
		stderr string
	}{
		{"sorted", sort.Options{Check: sort.CheckDiagnose}, "a\nb\nb\n", 0, ""},
		{"disorder", sort.Options{Check: sort.CheckDiagnose}, "a\nc\nb\nd\n", 1, "sortx: -:3: disorder: b\n"},
		{"quiet", sort.Options{Check: sort.CheckQuiet}, "b\na\n", 1, ""},
		{"unique needs strict order", sort.Options{Check: sort.CheckDiagnose, Unique: true}, "a\na\n", 1, "sortx: -:2: disorder: a\n"},
		{"numeric", sort.Options{Check: sort.CheckDiagnose, Numeric: true}, "9\n10\n", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			status := 0
			if err != nil {
				status = exitCode(err)
			}
			if status != tt.status || errb != tt.stderr || out != "" {
				t.Fatalf("status %d, stderr %q, stdout %q; want %d, %q", status, errb, out, tt.status, tt.stderr)
			}
		})
	}

	dir := t.TempDir()
	f := write(t, dir, "f", "b\na\n")
	_, errb, err := run(sort.Options{Check: sort.CheckDiagnose}, "", f)
	if exitCode(err) != 1 || errb != "sortx: "+f+":2: disorder: a\n" {
		t.Fatalf("file: err %v, stderr %q", err, errb)
	}

	_, _, err = run(sort.Options{Check: sort.CheckQuiet}, "", f, f)
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand '"+f+"' not allowed with -C" {
		t.Fatalf("two operands: err = %v", err)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "c\na\n")
	b := write(t, dir, "b", "b\n")

	out, errb, err := run(sort.Options{}, "z\n", a, "-", b)
	if err != nil || out != "a\nb\nc\nz\n" {
		t.Fatalf("out %q, stderr %q, err %v", out, errb, err)
	}

	// A missing operand stops everything before output is written.
	out, errb, err = run(sort.Options{}, "", a, filepath.Join(dir, "missing"))
	if exitCode(err) != 2 || out != "" {
		t.Fatalf("missing: out %q, err %v", out, err)
	}
	if want := "sortx: cannot read: " + filepath.Join(dir, "missing") + ": No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}

	_, errb, err = run(sort.Options{}, "", dir)
	if want := "sortx: read failed: " + dir + ": Is a directory\n"; exitCode(err) != 2 || errb != want {
		t.Fatalf("directory: err %v, stderr %q, want %q", err, errb, want)
	}
}

func TestMerge(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1 a\n3 a\n5 a\n")
	b := write(t, dir, "b", "1 b\n2 b\n6 b\n")

	out, _, err := run(sort.Options{Merge: true, Keys: []string{"1,1n"}, Stable: true}, "", a, b)
	if err != nil {
		t.Fatal(err)
	}
	if want := "1 a\n1 b\n2 b\n3 a\n5 a\n6 b\n"; out != want {
		t.Fatalf("merge = %q, want %q", out, want)
	}

	// Merging does not sort: unsorted input stays as it is.
	c := write(t, dir, "c", "b\na\n")
	out, _, _ = run(sort.Options{Merge: true}, "", c)
	if out != "b\na\n" {
		t.Fatalf("merge of one unsorted file = %q", out)
	}

	out, _, _ = run(sort.Options{Merge: true, Unique: true}, "", write(t, dir, "d", "a\nb\n"), write(t, dir, "e", "a\nc\n"))
	if out != "a\nb\nc\n" {
		t.Fatalf("merge -u = %q", out)
	}
}

func TestOutput(t *testing.T) {
	dir := t.TempDir()
	f := write(t, dir, "f", "c\nb\na\n")

	// Sorting a file in place: the output is also the input.
	out, errb, err := run(sort.Options{Output: f}, "", f)
	if err != nil || out != "" {
		t.Fatalf("out %q, stderr %q, err %v", out, errb, err)
	}
	if got, _ := os.ReadFile(f); string(got) != "a\nb\nc\n" {
		t.Fatalf("file = %q", got)
	}

	// A shorter result leaves no stale bytes behind.
	if _, _, err := run(sort.Options{Output: f, Unique: true}, "a\na\n"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(f); string(got) != "a\n" {
		t.Fatalf("file = %q", got)
	}

	// A missing input never truncates the output.
	_, _, err = run(sort.Options{Output: f}, "", filepath.Join(dir, "missing"))
	if exitCode(err) != 2 {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(f); string(got) != "a\n" {
		t.Fatalf("file after a failed run = %q", got)
	}

	_, errb, err = run(sort.Options{Output: filepath.Join(dir, "no", "such")}, "a\n")
	if exitCode(err) != 2 || !strings.HasPrefix(errb, "sortx: open failed: ") {
		t.Fatalf("err %v, stderr %q", err, errb)
	}
}

func TestFiles0From(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "b\n")
	b := write(t, dir, "b", "a\n")
	list := write(t, dir, "list", a+"\x00"+b+"\x00")

	out, errb, err := run(sort.Options{Files0From: list}, "")
	if err != nil || out != "a\nb\n" {
		t.Fatalf("out %q, stderr %q, err %v", out, errb, err)
	}

	out, _, err = run(sort.Options{Files0From: "-"}, a+"\x00"+b)
	if err != nil || out != "a\nb\n" {
		t.Fatalf("from stdin: out %q, err %v", out, err)
	}

	_, _, err = run(sort.Options{Files0From: list}, "", a)
	var usage *core.UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("operand with --files0-from: err = %v", err)
	}
}

func TestRandom(t *testing.T) {
	dir := t.TempDir()
	seed := write(t, dir, "seed", "0123456789abcdefXYZ")
	const in = "a\nb\nc\nd\ne\nf\nb\nA\n"
	tests := []struct {
		name string
		opts sort.Options
		in   string
		want string
	}{
		// The orders GNU sort 9.7 gives for the same seed file.
		{"whole lines", sort.Options{Random: true, RandomSource: seed}, in, "f\nd\nA\nb\nb\na\nc\ne\n"},
		{"folded", sort.Options{Random: true, IgnoreCase: true, RandomSource: seed}, in, "e\nA\na\nf\nb\nb\nd\nc\n"},
		{"a key", sort.Options{Keys: []string{"2,2R"}, RandomSource: seed}, "1 x\n2 y\n3 x\n4 z\n", "1 x\n3 x\n4 z\n2 y\n"},
	}
	for _, tt := range tests {
		out, errs, err := run(tt.opts, tt.in)
		if err != nil || out != tt.want {
			t.Errorf("%s: out = %q, want %q (err %v, stderr %q)", tt.name, out, tt.want, err, errs)
		}
	}

	// Without a source the order changes from run to run, but equal lines
	// stay together.
	out, _, err := run(sort.Options{Random: true}, "x\ny\nx\nz\nx\n")
	if err != nil || !strings.Contains(out, "x\nx\nx\n") {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestRandomSourceErrors(t *testing.T) {
	dir := t.TempDir()
	short := write(t, dir, "short", "short")
	missing := filepath.Join(dir, "missing")
	tests := []struct {
		name string
		opts sort.Options
		want string
	}{
		{"short", sort.Options{Random: true, RandomSource: short}, "sortx: '" + short + "': end of file\n"},
		{"missing", sort.Options{Random: true, RandomSource: missing}, "sortx: open failed: " + missing + ": No such file or directory\n"},
		{"incompatible", sort.Options{Random: true, Numeric: true}, "sortx: options '-nR' are incompatible\n"},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name string
			opts sort.Options
			want string
		}{"directory", sort.Options{Random: true, RandomSource: dir}, "sortx: '" + dir + "': read error: Is a directory\n"})
	}
	for _, tt := range tests {
		_, errs, err := run(tt.opts, "a\n")
		if exitCode(err) != 2 || errs != tt.want {
			t.Errorf("%s: stderr = %q, want %q (err %v)", tt.name, errs, tt.want, err)
		}
	}
	// The source is not read at all without -R.
	if out, _, err := run(sort.Options{RandomSource: missing}, "b\na\n"); err != nil || out != "a\nb\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}
