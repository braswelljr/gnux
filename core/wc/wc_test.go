package wc_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/wc"
)

// run executes the engine against in-memory streams.
func run(opts wc.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = wc.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
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

// TestCounts checks each counter on its own. A lone counter of a lone input
// is printed without padding, so the output is just the number.
func TestCounts(t *testing.T) {
	l := wc.Options{Lines: true}
	w := wc.Options{Words: true}
	m := wc.Options{Chars: true}
	c := wc.Options{Bytes: true}
	L := wc.Options{MaxLineLength: true}

	tests := []struct {
		name  string
		opts  wc.Options
		input string
		want  string
	}{
		{"lines", l, "a\nb\nc\n", "3"},
		{"lines counts newlines, not unterminated lines", l, "a\nb", "1"},
		{"lines of nothing", l, "", "0"},

		{"words", w, "one two  three\n", "3"},
		{"words across lines and tabs", w, "a\tb\nc\r\nd\fe\vf", "6"},
		{"words with leading and trailing space", w, "  a  ", "1"},
		{"words of only whitespace", w, " \n\t ", "0"},
		{"words split on unicode whitespace", w, "a b c　d", "4"},
		{"words are not split by control characters", w, "a\x01b", "1"},
		{"words with multibyte letters", w, "héllo wörld\n", "2"},
		{"invalid bytes belong to words", w, "\xff\xfe a\n", "2"},

		{"chars ascii", m, "abc\n", "4"},
		{"chars count runes", m, "héllo wörld\n", "12"},
		{"chars wide", m, "日本語\n", "4"},
		{"chars skip invalid bytes", m, "\xff\xfe a\n", "3"},
		{"chars skip a truncated sequence at the end", m, "a\xe6\x97", "1"},

		{"bytes", c, "héllo\n", "7"},
		{"bytes of nothing", c, "", "0"},

		{"max line length", L, "ab\nabcd\nabc\n", "4"},
		{"max line length of an unterminated last line", L, "ab\nabcde", "5"},
		{"max line length expands tabs", L, "a\tb\n", "9"},
		{"max line length resets on carriage return", L, "abcdef\rxy\n", "6"},
		{"max line length counts columns, not bytes", L, "héllo\n", "5"},
		{"max line length gives wide characters two columns", L, "日本\n", "4"},
		{"max line length ignores control characters", L, "a\x01\x1bb\n", "2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("stdin: err = %v, stderr = %q", err, errb)
			}
			if out != tt.want+"\n" {
				t.Fatalf("stdin: stdout = %q, want %q", out, tt.want+"\n")
			}

			path := write(t, t.TempDir(), "f", tt.input)
			out, errb, err = run(tt.opts, "", path)
			if err != nil || errb != "" {
				t.Fatalf("file: err = %v, stderr = %q", err, errb)
			}
			if want := tt.want + " " + path + "\n"; out != want {
				t.Fatalf("file: stdout = %q, want %q", out, want)
			}
		})
	}
}

func TestLayout(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "one two\nthree\n")               // 2 lines, 3 words, 14 bytes
	b := write(t, dir, "b", "héllo")                          // 0 lines, 1 word, 5 chars, 6 bytes
	big := write(t, dir, "big", strings.Repeat("x y\n", 300)) // 300 lines, 600 words, 1200 bytes
	missing := filepath.Join(dir, "missing")

	all := wc.Options{Lines: true, Words: true, Chars: true, Bytes: true, MaxLineLength: true}

	tests := []struct {
		name    string
		opts    wc.Options
		stdin   string
		args    []string
		want    string
		wantErr string
	}{
		{name: "default counters on piped stdin use width 7", stdin: "one two\nthree\n", want: "      2       3      14\n"},
		{name: "default counters on a file size the columns", args: []string{a}, want: " 2  3 14 " + a + "\n"},
		{name: "dash is stdin and is named", stdin: "x\n", args: []string{"-"}, want: "      1       1       2 -\n"},
		{
			name: "several files get a total", args: []string{a, b},
			want: " 2  3 14 " + a + "\n 0  1  6 " + b + "\n 2  4 20 total\n",
		},
		{
			name: "width covers the combined size", args: []string{a, big},
			want: "   2    3   14 " + a + "\n 300  600 1200 " + big + "\n 302  603 1214 total\n",
		},
		{
			name: "a non-file input forces width 7", stdin: "x\n", args: []string{a, "-"},
			want: "      2       3      14 " + a + "\n      1       1       2 -\n      3       4      16 total\n",
		},
		{
			name: "fixed order whatever was asked", opts: all, args: []string{b},
			want: "0 1 5 6 5 " + b + "\n",
		},
		{
			name: "lines and bytes only", opts: wc.Options{Lines: true, Bytes: true}, args: []string{a},
			want: " 2 14 " + a + "\n",
		},
		{
			name: "single counter, several files", opts: wc.Options{Lines: true}, args: []string{a, big},
			want: "   2 " + a + "\n 300 " + big + "\n 302 total\n",
		},
		{
			name: "bytes alone", opts: wc.Options{Bytes: true}, args: []string{big},
			want: "1200 " + big + "\n",
		},
		{
			name: "max line length totals to the maximum", opts: wc.Options{MaxLineLength: true}, args: []string{a, b},
			want: " 7 " + a + "\n 5 " + b + "\n 7 total\n",
		},
		{
			name: "missing file is skipped and totals continue", args: []string{a, missing, b},
			want:    " 2  3 14 " + a + "\n 0  1  6 " + b + "\n 2  4 20 total\n",
			wantErr: "wcx: " + missing + ": No such file or directory\n",
		},
		{
			// An operand that cannot be examined adds nothing to the
			// width estimate, wherever it is (GNU 9).
			name: "missing first file", args: []string{missing, a},
			want:    " 2  3 14 " + a + "\n 2  3 14 total\n",
			wantErr: "wcx: " + missing + ": No such file or directory\n",
		},
		{
			name: "directory reports and prints zeros", args: []string{dir},
			want:    "      0       0       0 " + dir + "\n",
			wantErr: "wcx: " + dir + ": Is a directory\n",
		},

		{name: "total=always", opts: wc.Options{Total: wc.TotalAlways}, args: []string{a}, want: " 2  3 14 " + a + "\n 2  3 14 total\n"},
		{name: "total=never", opts: wc.Options{Total: wc.TotalNever}, args: []string{a, b}, want: " 2  3 14 " + a + "\n 0  1  6 " + b + "\n"},
		{name: "total=only", opts: wc.Options{Total: wc.TotalOnly}, args: []string{a, b}, want: "2 4 20\n"},
		{name: "total=only with one counter", opts: wc.Options{Total: wc.TotalOnly, Lines: true}, args: []string{a, big}, want: "302\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.stdin, tt.args...)
			if failed := errors.Is(err, core.ErrFailed); failed != (tt.wantErr != "") || (err != nil && !failed) {
				t.Fatalf("err = %v with stderr %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
			if errb != tt.wantErr {
				t.Fatalf("stderr = %q, want %q", errb, tt.wantErr)
			}
		})
	}
}

// TestStdinFile covers standard input redirected from a regular file: the
// size is knowable, so the width follows it and -c needs no read.
func TestStdinFile(t *testing.T) {
	path := write(t, t.TempDir(), "f", "one two\nthree\n")

	tests := []struct {
		name string
		opts wc.Options
		want string
	}{
		{"default", wc.Options{}, " 2  3 14\n"},
		{"bytes", wc.Options{Bytes: true}, "14\n"},
		{"lines and bytes", wc.Options{Lines: true, Bytes: true}, " 2 14\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			var out, errb bytes.Buffer
			if err := wc.New(tt.opts, core.Streams{In: f, Out: &out, Err: &errb}).Run(nil); err != nil {
				t.Fatalf("Run: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

// TestChunkBoundaries feeds input in awkward pieces: multi-byte characters
// split across reads must still count once, and totals must not depend on
// where the reads fall.
func TestChunkBoundaries(t *testing.T) {
	text := strings.Repeat("héllo wörld 日本語\n", 500)
	opts := wc.Options{Lines: true, Words: true, Chars: true, Bytes: true, MaxLineLength: true}
	const want = "    500    1500    8000   12000      18\n"

	t.Run("whole", func(t *testing.T) {
		var out bytes.Buffer
		if err := wc.New(opts, core.Streams{In: strings.NewReader(text), Out: &out, Err: &out}).Run(nil); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	})

	t.Run("one byte at a time", func(t *testing.T) {
		var out bytes.Buffer
		in := iotest.OneByteReader(strings.NewReader(text))
		if err := wc.New(opts, core.Streams{In: in, Out: &out, Err: &out}).Run(nil); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	})

	t.Run("data delivered together with EOF", func(t *testing.T) {
		var out bytes.Buffer
		in := iotest.DataErrReader(strings.NewReader(text))
		if err := wc.New(opts, core.Streams{In: in, Out: &out, Err: &out}).Run(nil); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	})

	t.Run("larger than the read buffer", func(t *testing.T) {
		big := strings.Repeat("é\n", 400_000) // 1.2 MB, with é straddling chunk ends
		path := write(t, t.TempDir(), "big", big)
		for _, o := range []wc.Options{{Lines: true}, {Chars: true}, {Words: true}, {Bytes: true}} {
			out, errb, err := run(o, "", path)
			if err != nil {
				t.Fatalf("%+v: %v (%s)", o, err, errb)
			}
			want := "400000 " + path + "\n"
			if o.Chars {
				want = "800000 " + path + "\n"
			}
			if o.Bytes {
				want = "1200000 " + path + "\n"
			}
			if out != want {
				t.Fatalf("%+v: stdout = %q, want %q", o, out, want)
			}
		}
	})
}

func TestReadErrorKeepsPartialCounts(t *testing.T) {
	var out, errb bytes.Buffer
	in := iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("a\nb\n")))
	err := wc.New(wc.Options{Lines: true}, core.Streams{In: in, Out: &out, Err: &errb}).Run(nil)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if out.String() != "0\n" {
		t.Fatalf("stdout = %q, want the partial count", out.String())
	}
	if want := "wcx: -: Timeout\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestFiles0From(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "one two\nthree\n")
	b := write(t, dir, "b", "x\n")
	missing := filepath.Join(dir, "missing")

	tests := []struct {
		name    string
		list    string
		opts    wc.Options
		want    string
		wantErr string
	}{
		{name: "terminated list", list: a + "\x00" + b + "\x00", want: "2 3 14 " + a + "\n1 1 2 " + b + "\n3 4 16 total\n"},
		{name: "unterminated last name", list: a + "\x00" + b, want: "2 3 14 " + a + "\n1 1 2 " + b + "\n3 4 16 total\n"},
		{name: "single name has no total", list: a + "\x00", want: "2 3 14 " + a + "\n"},
		{name: "empty list", list: "", want: ""},
		{name: "total only", list: a + "\x00" + b + "\x00", opts: wc.Options{Total: wc.TotalOnly, Lines: true}, want: "3\n"},
		{
			name: "missing entry", list: a + "\x00" + missing + "\x00",
			want: "2 3 14 " + a + "\n2 3 14 total\n", wantErr: "wcx: " + missing + ": No such file or directory\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Once with the list in a file, once on standard input.
			for _, from := range []string{write(t, dir, "list", tt.list), "-"} {
				opts := tt.opts
				opts.Files0From = from
				out, errb, err := run(opts, tt.list)
				if errors.Is(err, core.ErrFailed) != (tt.wantErr != "") {
					t.Fatalf("from %q: err = %v", from, err)
				}
				if out != tt.want || errb != tt.wantErr {
					t.Fatalf("from %q: stdout = %q, stderr = %q; want %q, %q", from, out, errb, tt.want, tt.wantErr)
				}
			}
		})
	}

	t.Run("empty name", func(t *testing.T) {
		list := write(t, dir, "list", a+"\x00\x00"+b+"\x00")
		out, errb, err := run(wc.Options{Files0From: list, Lines: true}, "")
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "2 " + a + "\n1 " + b + "\n3 total\n"; out != want {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
		if want := "wcx: " + list + ":2: invalid zero-length file name\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})

	t.Run("dash while names come from stdin", func(t *testing.T) {
		out, errb, err := run(wc.Options{Files0From: "-", Lines: true}, "-\x00"+a+"\x00")
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "2 " + a + "\n"; out != want {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
		if want := "wcx: when reading file names from stdin, no file name of '-' allowed\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})

	t.Run("unreadable list", func(t *testing.T) {
		_, errb, err := run(wc.Options{Files0From: missing}, "")
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "wcx: cannot open '" + missing + "' for reading: No such file or directory\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})

	t.Run("operands are a usage error", func(t *testing.T) {
		_, _, err := run(wc.Options{Files0From: a}, "", "extra")
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("err = %v, want a UsageError", err)
		}
		if want := "extra operand 'extra'\nfile operands cannot be combined with --files0-from"; usage.Msg != want {
			t.Fatalf("message = %q, want %q", usage.Msg, want)
		}
	})
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
		{"other errors are reported", errors.New("disk on fire"), "wcx: write error: Disk on fire\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errb bytes.Buffer
			err := wc.New(wc.Options{}, core.Streams{In: strings.NewReader("in\n"), Out: failWriter{tt.werr}, Err: &errb}).
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
