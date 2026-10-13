package shuf_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shuf"
)

// identity is an injected generator that always draws 0, so the partial
// Fisher-Yates shuffle returns the input in order.
func identity(uint64) (uint64, error) { return 0, nil }

// last always draws the largest index: c, then a (now last), then b.
func last(n uint64) (uint64, error) { return n - 1, nil }

func run(opts shuf.Options, rnd func(uint64) (uint64, error), stdin string, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	sh := shuf.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	sh.Rand = rnd
	err := sh.Run(args)
	return out.String(), errb.String(), err
}

func TestDeterministicOrders(t *testing.T) {
	tests := []struct {
		name  string
		opts  shuf.Options
		rnd   func(uint64) (uint64, error)
		stdin string
		args  []string
		want  string
	}{
		{"identity keeps order", shuf.Options{}, identity, "a\nb\nc\n", nil, "a\nb\nc\n"},
		{"last swaps from the end", shuf.Options{}, last, "a\nb\nc\n", nil, "c\na\nb\n"},
		{"unterminated last line", shuf.Options{}, identity, "a\nb", nil, "a\nb\n"},
		{"empty input", shuf.Options{}, identity, "", nil, ""},
		{"echo operands", shuf.Options{Echo: true}, identity, "", []string{"x", "y z"}, "x\ny z\n"},
		{"echo nothing", shuf.Options{Echo: true}, identity, "", nil, ""},
		{"range", shuf.Options{InputRange: "3-5"}, identity, "", nil, "3\n4\n5\n"},
		{"empty range", shuf.Options{InputRange: "5-4"}, identity, "", nil, ""},
		{"head count", shuf.Options{HeadCount: 2, HeadCountSet: true}, identity, "a\nb\nc\n", nil, "a\nb\n"},
		{"head count zero", shuf.Options{HeadCountSet: true}, identity, "a\nb\n", nil, ""},
		{"head count beyond input", shuf.Options{HeadCount: 9, HeadCountSet: true}, identity, "a\n", nil, "a\n"},
		{"repeat with count", shuf.Options{Repeat: true, HeadCount: 3, HeadCountSet: true}, identity, "a\nb\n", nil, "a\na\na\n"},
		{"zero terminated", shuf.Options{ZeroTerminated: true}, last, "a\x00b\x00", nil, "b\x00a\x00"},
		{"huge range, small sample", shuf.Options{InputRange: "0-18446744073709551614", HeadCount: 2, HeadCountSet: true}, identity, "", nil, "0\n1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.rnd, tt.stdin, tt.args...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestPermutation checks with the real generator that every line comes out
// exactly once.
func TestPermutation(t *testing.T) {
	var in strings.Builder
	var want []string
	for i := range 200 {
		s := strings.Repeat("x", i%7) + string(rune('a'+i%26)) + strings.Repeat("y", i/26)
		want = append(want, s)
		in.WriteString(s + "\n")
	}
	out, _, err := run(shuf.Options{}, nil, in.String())
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatal("output is not a permutation of the input")
	}
}

// TestHeadCountIsAPrefix checks that -n only cuts the shuffle short. A short
// and a full shuffle store their permutation differently, and the same draws
// must still give the same lines.
func TestHeadCountIsAPrefix(t *testing.T) {
	var in strings.Builder
	for i := range 100 {
		in.WriteString(strings.Repeat("l", i%5) + string(rune('a'+i%26)) + "\n")
	}
	// A fixed sequence of draws, restarted for each run.
	draws := func() func(uint64) (uint64, error) {
		state := uint64(42)
		return func(n uint64) (uint64, error) {
			state = state*6364136223846793005 + 1442695040888963407
			return (state >> 33) % n, nil
		}
	}

	full, _, err := run(shuf.Options{}, draws(), in.String())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int64{1, 7, 49, 50, 99} {
		got, _, err := run(shuf.Options{HeadCount: n, HeadCountSet: true}, draws(), in.String())
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Join(strings.SplitAfter(full, "\n")[:n], "")
		if got != want {
			t.Fatalf("-n %d = %q, want the first %d lines of the full shuffle %q", n, got, n, want)
		}
	}
}

func TestFileAndOutput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	if err := os.WriteFile(in, []byte("1\n2\n3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// -o may name the input: it is read before the output is opened.
	out, errb, err := run(shuf.Options{Output: in}, last, "", in)
	if err != nil || out != "" || errb != "" {
		t.Fatalf("err=%v out=%q stderr=%q", err, out, errb)
	}
	got, _ := os.ReadFile(in)
	if string(got) != "3\n1\n2\n" {
		t.Fatalf("file = %q", got)
	}
}

func TestRandomSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "seed")
	if err := os.WriteFile(src, bytes.Repeat([]byte{7, 200, 13, 99}, 64), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := shuf.Options{InputRange: "1-50", RandomSource: src}
	first, _, err := run(opts, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := run(opts, nil, "")
	if first != second {
		t.Fatal("the same random source gave two different orders")
	}

	// A source that runs dry is GNU's "end of file".
	short := filepath.Join(dir, "short")
	_ = os.WriteFile(short, []byte{1}, 0o644)
	_, errb, err := run(shuf.Options{InputRange: "1-50", RandomSource: short}, nil, "")
	if !errors.Is(err, core.ErrFailed) || errb != "shufx: '"+short+"': end of file\n" {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name   string
		opts   shuf.Options
		args   []string
		stderr string
		usage  string
	}{
		{"missing file", shuf.Options{}, []string{"nope"}, "shufx: nope: No such file or directory\n", ""},
		{"extra operand", shuf.Options{}, []string{"a", "b"}, "", "extra operand 'b'"},
		{"operand with range", shuf.Options{InputRange: "1-2"}, []string{"a"}, "", "extra operand 'a'"},
		{"echo and range", shuf.Options{Echo: true, InputRange: "1-2"}, nil, "", "cannot combine -e and -i options"},
		{"bad range", shuf.Options{InputRange: "9-x"}, nil, "shufx: invalid input range: '9-x'\n", ""},
		{"backward range", shuf.Options{InputRange: "9-2"}, nil, "shufx: invalid input range: '9-2'\n", ""},
		{"nothing to repeat", shuf.Options{Repeat: true}, nil, "shufx: no lines to repeat\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errb, err := run(tt.opts, identity, "", tt.args...)
			if tt.usage != "" {
				var u *core.UsageError
				if !errors.As(err, &u) || u.Msg != tt.usage {
					t.Fatalf("err = %v, want usage %q", err, tt.usage)
				}
				return
			}
			if !errors.Is(err, core.ErrFailed) || errb != tt.stderr {
				t.Fatalf("err=%v stderr=%q, want %q", err, errb, tt.stderr)
			}
		})
	}
}

func TestParseCount(t *testing.T) {
	if n, err := shuf.ParseCount("12"); err != nil || n != 12 {
		t.Fatalf("12: %d %v", n, err)
	}
	if _, err := shuf.ParseCount("-1"); err == nil || err.Error() != "invalid line count: '-1'" {
		t.Fatalf("-1: %v", err)
	}
	if n, _ := shuf.ParseCount("99999999999999999999999"); n <= 0 {
		t.Fatal("a huge count should clamp")
	}
}
