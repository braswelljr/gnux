package head_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
)

// run executes the engine against in-memory streams.
func run(opts head.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = head.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
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

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"0", 0, true},
		{"10", 10, true},
		{"007", 7, true},
		{"1b", 512, true},
		{"2b", 1024, true},
		{"1K", 1024, true},
		{"1k", 1024, true},
		{"1KiB", 1024, true},
		{"1kiB", 1024, true},
		{"1KB", 1000, true},
		{"1kB", 1000, true},
		{"3M", 3 << 20, true},
		{"3MiB", 3 << 20, true},
		{"3MB", 3_000_000, true},
		{"1G", 1 << 30, true},
		{"1GB", 1_000_000_000, true},
		{"1T", 1 << 40, true},
		{"1P", 1 << 50, true},
		{"1E", 1 << 60, true},
		{"1EB", 1_000_000_000_000_000_000, true},
		{"0Y", 0, true},
		// Too large for 64 bits: clamped, not rejected.
		{"8E", math.MaxInt64, true},
		{"1Z", math.MaxInt64, true},
		{"1YB", math.MaxInt64, true},
		{"1Q", math.MaxInt64, true},
		{"99999999999999999999", math.MaxInt64, true},

		{"", 0, false},
		{"x", 0, false},
		{"K", 0, false},
		{"-1", 0, false},
		{"+1", 0, false},
		{"1.5", 0, false},
		{"1x", 0, false},
		{"1Ki", 0, false},
		{"1KiBs", 0, false},
		{"1bB", 0, false},
		{"1 K", 0, false},
	}
	for _, tt := range tests {
		got, err := head.ParseSize(tt.in)
		if (err == nil) != tt.ok {
			t.Errorf("ParseSize(%q) error = %v, want ok = %v", tt.in, err, tt.ok)
			continue
		}
		if err != nil && !errors.Is(err, head.ErrInvalidSize) {
			t.Errorf("ParseSize(%q) error = %v, want ErrInvalidSize", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseSize(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestSelection(t *testing.T) {
	const five = "1\n2\n3\n4\n5\n"
	const fiveOpen = "1\n2\n3\n4\n5" // no trailing newline

	lines := func(n int64) head.Options { return head.Options{Count: n} }
	bytesOpt := func(n int64) head.Options { return head.Options{Count: n, Bytes: true} }
	butLines := func(n int64) head.Options { return head.Options{Count: n, FromEnd: true} }
	butBytes := func(n int64) head.Options { return head.Options{Count: n, Bytes: true, FromEnd: true} }

	tests := []struct {
		name  string
		opts  head.Options
		input string
		want  string
	}{
		{"default count of an empty input", lines(10), "", ""},
		{"fewer lines than asked", lines(10), five, five},
		{"first lines", lines(2), five, "1\n2\n"},
		{"first lines, unterminated", lines(5), fiveOpen, fiveOpen},
		{"first lines stop before unterminated", lines(4), fiveOpen, "1\n2\n3\n4\n"},
		{"zero lines", lines(0), five, ""},
		{"huge line count", lines(math.MaxInt64), five, five},
		{"blank lines count", lines(2), "\n\nx\n", "\n\n"},

		{"first bytes", bytesOpt(3), five, "1\n2"},
		{"zero bytes", bytesOpt(0), five, ""},
		{"more bytes than there are", bytesOpt(100), five, five},

		{"all but last lines", butLines(2), five, "1\n2\n3\n"},
		{"all but last lines, unterminated", butLines(2), fiveOpen, "1\n2\n3\n"},
		{"all but last one, unterminated", butLines(1), "a\nb\nc", "a\nb\n"},
		{"all but last zero lines", butLines(0), fiveOpen, fiveOpen},
		{"all but more lines than there are", butLines(9), five, ""},
		{"all but exactly all lines", butLines(5), five, ""},
		{"all but exactly all lines, unterminated", butLines(5), fiveOpen, ""},
		{"all but last of a single unterminated line", butLines(1), "abc", ""},

		{"all but last bytes", butBytes(3), five, "1\n2\n3\n4"},
		{"all but last zero bytes", butBytes(0), five, five},
		{"all but more bytes than there are", butBytes(100), five, ""},

		{"zero terminated", head.Options{Count: 2, ZeroTerminated: true}, "a\nb\x00c\x00d\x00", "a\nb\x00c\x00"},
		{"zero terminated, all but last", head.Options{Count: 1, FromEnd: true, ZeroTerminated: true}, "a\x00b\x00c", "a\x00b\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Standard input exercises the streaming paths, a file the
			// size-aware ones; both must agree.
			out, errb, err := run(tt.opts, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("stdin: err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdin: stdout = %q, want %q", out, tt.want)
			}

			out, errb, err = run(tt.opts, "", write(t, t.TempDir(), "f", tt.input))
			if err != nil || errb != "" {
				t.Fatalf("file: err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("file: stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestLargeInputs crosses the read-chunk boundary, where the windowed
// "all but the last" paths have to release data as they go.
func TestLargeInputs(t *testing.T) {
	var b strings.Builder
	for i := range 50000 {
		b.WriteString(strings.Repeat("y", i%37))
		b.WriteByte('\n')
	}
	input := b.String()
	all := strings.SplitAfter(input, "\n")
	all = all[:len(all)-1]

	tests := []struct {
		name string
		opts head.Options
		want string
	}{
		{"first 40000 lines", head.Options{Count: 40000}, strings.Join(all[:40000], "")},
		{"all but last 3", head.Options{Count: 3, FromEnd: true}, strings.Join(all[:len(all)-3], "")},
		{"all but last 30000", head.Options{Count: 30000, FromEnd: true}, strings.Join(all[:20000], "")},
		{"all but last 100000 bytes", head.Options{Count: 100000, Bytes: true, FromEnd: true}, input[:len(input)-100000]},
		{"first 100000 bytes", head.Options{Count: 100000, Bytes: true}, input[:100000]},
	}
	path := write(t, t.TempDir(), "big", input)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if out, _, err := run(tt.opts, input); err != nil || out != tt.want {
				t.Fatalf("stdin: got %d bytes, want %d (err = %v)", len(out), len(tt.want), err)
			}
			if out, _, err := run(tt.opts, "", path); err != nil || out != tt.want {
				t.Fatalf("file: got %d bytes, want %d (err = %v)", len(out), len(tt.want), err)
			}
		})
	}
}

func TestHeaders(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "A1\nA2\n")
	b := write(t, dir, "b", "B1")
	empty := write(t, dir, "empty", "")

	tests := []struct {
		name  string
		opts  head.Options
		stdin string
		args  []string
		want  string
	}{
		{"single file has no header", head.Options{Count: 1}, "", []string{a}, "A1\n"},
		{"verbose forces one", head.Options{Count: 1, Verbose: true}, "", []string{a}, "==> " + a + " <==\nA1\n"},
		{"verbose names stdin", head.Options{Count: 1, Verbose: true}, "in\n", nil, "==> standard input <==\nin\n"},
		{
			"several files", head.Options{Count: 1}, "in\n", []string{a, "-", b},
			"==> " + a + " <==\nA1\n\n==> standard input <==\nin\n\n==> " + b + " <==\nB1",
		},
		{
			"empty file still gets its header", head.Options{Count: 1}, "", []string{empty, a},
			"==> " + empty + " <==\n\n==> " + a + " <==\nA1\n",
		},
		{"quiet", head.Options{Count: 1, Quiet: true}, "", []string{a, b}, "A1\nB1"},
		{"quiet beats verbose", head.Options{Count: 1, Quiet: true, Verbose: true}, "", []string{a, b}, "A1\nB1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.stdin, tt.args...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestErrorsDoNotStopLaterOperands(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "A\n")
	b := write(t, dir, "b", "B\n")
	missing := filepath.Join(dir, "missing")

	out, errb, err := run(head.Options{Count: 10}, "", a, missing, dir, b)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	// As in GNU head, a file that cannot be opened gets no header, while a
	// directory opens fine and fails only when read.
	wantOut := "==> " + a + " <==\nA\n\n==> " + dir + " <==\n\n==> " + b + " <==\nB\n"
	if out != wantOut {
		t.Fatalf("stdout = %q, want %q", out, wantOut)
	}
	wantErr := "headx: cannot open '" + missing + "' for reading: No such file or directory\n" +
		"headx: error reading '" + dir + "': Is a directory\n"
	if errb != wantErr {
		t.Fatalf("stderr = %q, want %q", errb, wantErr)
	}
}

// TestLeavesRestOfSeekableInput checks that head consumes only what it prints
// from a seekable standard input, so a second reader continues after it.
func TestLeavesRestOfSeekableInput(t *testing.T) {
	f, err := os.Open(write(t, t.TempDir(), "f", "1\n2\n3\n4\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var out bytes.Buffer
	for range 2 {
		if err := head.New(head.Options{Count: 1}, core.Streams{In: f, Out: &out, Err: io.Discard}).Run(nil); err != nil {
			t.Fatal(err)
		}
	}
	rest, _ := io.ReadAll(f)
	if out.String() != "1\n2\n" || string(rest) != "3\n4\n" {
		t.Fatalf("printed %q, left %q", out.String(), rest)
	}
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	tests := []struct {
		name    string
		werr    error
		wantErr string
		failed  bool
	}{
		{"closed pipe is quiet", os.NewSyscallError("write", syscall.EPIPE), "", false},
		{"other errors are reported", errors.New("disk on fire"), "headx: write error: Disk on fire\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errb bytes.Buffer
			err := head.New(head.Options{Count: 1}, core.Streams{In: strings.NewReader("in\n"), Out: failWriter{tt.werr}, Err: &errb}).
				Run([]string{"-", "-"})
			if errors.Is(err, core.ErrFailed) != tt.failed {
				t.Fatalf("err = %v, want failed = %v", err, tt.failed)
			}
			if errb.String() != tt.wantErr {
				t.Fatalf("stderr = %q, want %q", errb.String(), tt.wantErr)
			}
		})
	}
}

func TestColoredHeaders(t *testing.T) {
	t.Setenv("GNUX_THEME", "")
	dir := t.TempDir()
	a, b := write(t, dir, "a", "1\n"), write(t, dir, "b", "2\n")
	out, _, err := run(head.Options{Count: 1, Color: true}, "", a, b)
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b[1;35m==> " + a + " <==\x1b[0m\n1\n\n\x1b[1;35m==> " + b + " <==\x1b[0m\n2\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}
