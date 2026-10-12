package split_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/split"
)

// run splits stdin in a fresh directory and returns the files made, in
// name order, with their contents.
func run(t *testing.T, opts split.Options, stdin string, args ...string) (files map[string]string, names []string, stdout, stderr string, err error) {
	t.Helper()
	t.Chdir(t.TempDir())
	var out, errb bytes.Buffer
	err = split.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	entries, _ := os.ReadDir(".")
	files = map[string]string{}
	for _, e := range entries {
		data, _ := os.ReadFile(e.Name())
		files[e.Name()] = string(data)
		names = append(names, e.Name())
	}
	slices.Sort(names)
	return files, names, out.String(), errb.String(), err
}

func TestLines(t *testing.T) {
	files, names, _, _, err := run(t, split.Options{Count: 2}, "1\n2\n3\n4\n5\n")
	if err != nil || !slices.Equal(names, []string{"xaa", "xab", "xac"}) {
		t.Fatalf("names = %v, err = %v", names, err)
	}
	if files["xaa"] != "1\n2\n" || files["xab"] != "3\n4\n" || files["xac"] != "5\n" {
		t.Fatalf("files = %q", files)
	}
}

func TestDefaultIs1000Lines(t *testing.T) {
	_, names, _, _, _ := run(t, split.Options{}, strings.Repeat("x\n", 1001))
	if !slices.Equal(names, []string{"xaa", "xab"}) {
		t.Fatalf("names = %v", names)
	}
}

func TestBytesAndPrefix(t *testing.T) {
	files, names, _, _, _ := run(t, split.Options{Mode: split.Bytes, Count: 4}, "abcdefghij", "-", "part.")
	if !slices.Equal(names, []string{"part.aa", "part.ab", "part.ac"}) || files["part.ac"] != "ij" {
		t.Fatalf("files = %q", files)
	}
}

func TestLineBytes(t *testing.T) {
	files, names, _, _, _ := run(t, split.Options{Mode: split.LineBytes, Count: 6}, "ab\ncd\nefghijkl\nm\n")
	want := map[string]string{"xaa": "ab\ncd\n", "xab": "efghij", "xac": "kl\nm\n"}
	if len(names) != 3 || files["xaa"] != want["xaa"] || files["xab"] != want["xab"] || files["xac"] != want["xac"] {
		t.Fatalf("files = %q", files)
	}
}

func TestSuffixes(t *testing.T) {
	_, names, _, _, _ := run(t, split.Options{Mode: split.Bytes, Count: 1, Suffixes: split.Numeric}, "abc")
	if !slices.Equal(names, []string{"x00", "x01", "x02"}) {
		t.Fatalf("-d: %v", names)
	}
	_, names, _, _, _ = run(t, split.Options{Mode: split.Bytes, Count: 1, Suffixes: split.Hex, SuffixStart: "0e"}, "abc")
	if !slices.Equal(names, []string{"x0e", "x0f", "x10"}) {
		t.Fatalf("-x=0e: %v", names)
	}
	_, names, _, _, _ = run(t, split.Options{Mode: split.Bytes, Count: 1, SuffixLength: 1, AdditionalSuffix: ".txt"}, "ab")
	if !slices.Equal(names, []string{"xa.txt", "xb.txt"}) {
		t.Fatalf("-a 1 --additional-suffix: %v", names)
	}
}

func TestSuffixAutoWidens(t *testing.T) {
	// 10*9 = 90 two-digit names, then the prefix takes the 9.
	_, names, _, _, err := run(t, split.Options{Mode: split.Bytes, Count: 1, Suffixes: split.Numeric}, strings.Repeat("x", 92))
	if err != nil {
		t.Fatal(err)
	}
	if names[89] != "x89" || names[90] != "x9000" || names[91] != "x9001" {
		t.Fatalf("names[89:] = %v", names[89:])
	}
}

func TestSuffixesExhausted(t *testing.T) {
	_, names, _, errb, err := run(t, split.Options{Mode: split.Bytes, Count: 1, SuffixLength: 1, Suffixes: split.Numeric}, strings.Repeat("x", 11))
	if !errors.Is(err, core.ErrFailed) || errb != "splitx: output file suffixes exhausted\n" || len(names) != 10 {
		t.Fatalf("err = %v, stderr = %q, %d files", err, errb, len(names))
	}
}

func TestChunkBytes(t *testing.T) {
	files, names, _, _, _ := run(t, split.Options{Mode: split.ChunkBytes, Count: 3}, "abcdefghij")
	if !slices.Equal(names, []string{"xaa", "xab", "xac"}) ||
		files["xaa"] != "abcd" || files["xab"] != "efg" || files["xac"] != "hij" {
		t.Fatalf("files = %q", files)
	}
	_, _, out, _, _ := run(t, split.Options{Mode: split.ChunkBytes, Count: 3, Chunk: 2}, "abcdefghij")
	if out != "efg" {
		t.Fatalf("-n 2/3: %q", out)
	}
	// More chunks than bytes: empty files, unless elided.
	_, names, _, _, _ = run(t, split.Options{Mode: split.ChunkBytes, Count: 4}, "ab")
	if len(names) != 4 {
		t.Fatalf("-n 4 of 2 bytes: %v", names)
	}
	_, names, _, _, _ = run(t, split.Options{Mode: split.ChunkBytes, Count: 4, ElideEmpty: true}, "ab")
	if !slices.Equal(names, []string{"xaa", "xab"}) {
		t.Fatalf("-e: %v", names)
	}
}

func TestChunkLines(t *testing.T) {
	files, names, _, _, _ := run(t, split.Options{Mode: split.ChunkLines, Count: 2}, "a\nbbbbbbbb\nc\nd\n")
	if !slices.Equal(names, []string{"xaa", "xab"}) || files["xaa"] != "a\nbbbbbbbb\n" || files["xab"] != "c\nd\n" {
		t.Fatalf("files = %q", files)
	}
	_, _, out, _, _ := run(t, split.Options{Mode: split.ChunkLines, Count: 2, Chunk: 2}, "a\nbbbbbbbb\nc\nd\n")
	if out != "c\nd\n" {
		t.Fatalf("-n l/2/2: %q", out)
	}
}

func TestRoundRobin(t *testing.T) {
	files, names, _, _, _ := run(t, split.Options{Mode: split.RoundRobin, Count: 2}, "1\n2\n3\n4\n5\n")
	if !slices.Equal(names, []string{"xaa", "xab"}) || files["xaa"] != "1\n3\n5\n" || files["xab"] != "2\n4\n" {
		t.Fatalf("files = %q", files)
	}
	_, _, out, _, _ := run(t, split.Options{Mode: split.RoundRobin, Count: 3, Chunk: 2}, "1\n2\n3\n4\n5\n")
	if out != "2\n5\n" {
		t.Fatalf("-n r/2/3: %q", out)
	}
}

func TestSeparatorAndVerbose(t *testing.T) {
	files, _, out, _, _ := run(t, split.Options{Count: 1, Separator: ',', SeparatorSet: true, Verbose: true}, "a,b,")
	if files["xaa"] != "a," || files["xab"] != "b," || out != "creating file 'xaa'\ncreating file 'xab'\n" {
		t.Fatalf("files = %q, out = %q", files, out)
	}
}

func TestWouldOverwriteInput(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("xaa", []byte("1\n2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	err := split.New(split.Options{Count: 1}, core.Streams{Out: &bytes.Buffer{}, Err: &errb}).Run([]string{"xaa"})
	if !errors.Is(err, core.ErrFailed) || errb.String() != "splitx: 'xaa' would overwrite input; aborting\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
	if data, _ := os.ReadFile("xaa"); string(data) != "1\n2\n" {
		t.Fatalf("input changed: %q", data)
	}
}

func TestErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, _, _, errb, err := run(t, split.Options{}, "", missing)
	if !errors.Is(err, core.ErrFailed) || errb != "splitx: cannot open '"+missing+"' for reading: No such file or directory\n" {
		t.Fatalf("missing: %v %q", err, errb)
	}
	_, _, _, errb, err = run(t, split.Options{Mode: split.ChunkBytes, Count: 1000, SuffixLength: 1}, "x")
	if !errors.Is(err, core.ErrFailed) || errb != "splitx: the suffix length needs to be at least 3\n" {
		t.Fatalf("suffix length: %v %q", err, errb)
	}
	for _, tt := range []struct {
		opts split.Options
		args []string
		msg  string
	}{
		{split.Options{}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{split.Options{AdditionalSuffix: "a/b"}, nil, "invalid suffix 'a/b', contains directory separator"},
		{split.Options{Suffixes: split.Numeric, SuffixStart: "1x"}, nil, "'1x': invalid start value for numerical suffix"},
		{split.Options{Suffixes: split.Numeric, SuffixStart: "123"}, nil, "numerical suffix start value is too large for the suffix length"},
	} {
		_, _, _, _, err := run(t, tt.opts, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("%+v %q: %v, want %q", tt.opts, tt.args, err, tt.msg)
		}
	}
}
