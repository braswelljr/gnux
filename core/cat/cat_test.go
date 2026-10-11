package cat_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cat"
)

// run executes the engine against in-memory streams.
func run(opts cat.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = cat.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
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

func TestTransforms(t *testing.T) {
	all := cat.Options{ShowEnds: true, ShowTabs: true, ShowNonprinting: true}
	tests := []struct {
		name  string
		opts  cat.Options
		input string
		want  string
	}{
		{"plain", cat.Options{}, "a\nb\n", "a\nb\n"},
		{"plain keeps binary", cat.Options{}, "\x00\xff\r\n", "\x00\xff\r\n"},
		{"plain without trailing newline", cat.Options{}, "a\nb", "a\nb"},
		{"empty", cat.Options{Number: true}, "", ""},

		{"number", cat.Options{Number: true}, "a\n\nb\n", "     1\ta\n     2\t\n     3\tb\n"},
		{"number without trailing newline", cat.Options{Number: true}, "a\nb", "     1\ta\n     2\tb"},
		{"number nonblank", cat.Options{NumberNonblank: true}, "a\n\nb\n", "     1\ta\n\n     2\tb\n"},
		{"nonblank overrides number", cat.Options{Number: true, NumberNonblank: true}, "a\n\nb\n", "     1\ta\n\n     2\tb\n"},

		{"squeeze", cat.Options{SqueezeBlank: true}, "a\n\n\n\nb\n\n", "a\n\nb\n\n"},
		{"squeeze leading", cat.Options{SqueezeBlank: true}, "\n\n\na\n", "\na\n"},
		{"squeeze then number", cat.Options{SqueezeBlank: true, Number: true}, "a\n\n\nb\n", "     1\ta\n     2\t\n     3\tb\n"},

		{"show ends", cat.Options{ShowEnds: true}, "a\n\nb", "a$\n$\nb"},
		{"show ends marks CRLF", cat.Options{ShowEnds: true}, "a\r\nb\rc\r", "a^M$\nb\rc\r"},
		{"show ends with number nonblank", cat.Options{ShowEnds: true, NumberNonblank: true}, "a\n\n", "     1\ta$\n$\n"},
		{"show tabs", cat.Options{ShowTabs: true}, "a\tb\n", "a^Ib\n"},

		{"nonprinting controls", cat.Options{ShowNonprinting: true}, "\x00\x01\x1b\x7f\r\n", "^@^A^[^?^M\n"},
		{"nonprinting leaves tab and newline", cat.Options{ShowNonprinting: true}, "a\tb\n", "a\tb\n"},
		{"nonprinting high bytes", cat.Options{ShowNonprinting: true}, "\x80\x89\xa0\xe9\xff\n", "M-^@M-^IM- M-iM-^?\n"},
		{"nonprinting utf8", cat.Options{ShowNonprinting: true}, "é\n", "M-CM-)\n"},
		{"show all", all, "a\tb\r\n\x01", "a^Ib^M$\n^A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each case runs against standard input and against a file.
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

func TestMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "one\ntwo\n\n")
	b := write(t, dir, "b", "\nthree")
	c := write(t, dir, "c", "four\n")

	tests := []struct {
		name  string
		opts  cat.Options
		stdin string
		args  []string
		want  string
	}{
		{"concatenates", cat.Options{}, "", []string{a, b, c}, "one\ntwo\n\n\nthreefour\n"},
		{"dash is stdin", cat.Options{}, "in\n", []string{a, "-", c}, "one\ntwo\n\nin\nfour\n"},
		{"numbering continues", cat.Options{Number: true}, "", []string{a, c}, "     1\tone\n     2\ttwo\n     3\t\n     4\tfour\n"},
		{"unterminated line joins the next file", cat.Options{Number: true}, "", []string{b, c}, "     1\t\n     2\tthreefour\n"},
		{"squeeze spans files", cat.Options{SqueezeBlank: true}, "", []string{a, b}, "one\ntwo\n\nthree"},
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

	for _, opts := range []cat.Options{{}, {Number: true}} {
		out, errb, err := run(opts, "", a, missing, dir, b)
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		want := "A\nB\n"
		if opts.Number {
			want = "     1\tA\n     2\tB\n"
		}
		if out != want {
			t.Fatalf("stdout = %q, want %q", out, want)
		}
		wantErr := "catx: " + missing + ": No such file or directory\n" +
			"catx: " + dir + ": Is a directory\n"
		if errb != wantErr {
			t.Fatalf("stderr = %q, want %q", errb, wantErr)
		}
	}
}

func TestInputIsOutput(t *testing.T) {
	path := write(t, t.TempDir(), "f", "data\n")
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	var errb bytes.Buffer
	err = cat.New(cat.Options{}, core.Streams{In: strings.NewReader(""), Out: out, Err: &errb}).Run([]string{path})
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "catx: " + path + ": input file is output file\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	if got, _ := os.ReadFile(path); string(got) != "data\n" {
		t.Fatalf("file = %q, want it untouched", got)
	}
}

func TestLargeInputCrossesBuffers(t *testing.T) {
	line := strings.Repeat("x", 99) + "\n"
	input := strings.Repeat(line, 3000) // ~300 KiB, several read chunks

	out, _, err := run(cat.Options{}, input)
	if err != nil || out != input {
		t.Fatalf("plain copy altered %d bytes of input (err = %v)", len(input), err)
	}

	out, _, err = run(cat.Options{Number: true}, input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, "  3000\t"+line) || strings.Count(out, "\n") != 3000 {
		t.Fatalf("numbered output ends %q", out[len(out)-20:])
	}
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	path := write(t, t.TempDir(), "f", "data\n")

	tests := []struct {
		name    string
		werr    error
		wantErr string
		failed  bool
	}{
		{"closed pipe is quiet", os.NewSyscallError("write", syscall.EPIPE), "", false},
		{"other errors are reported", errors.New("disk on fire"), "catx: write error: Disk on fire\n", true},
	}
	for _, tt := range tests {
		for _, opts := range []cat.Options{{}, {Number: true}} {
			t.Run(tt.name, func(t *testing.T) {
				var errb bytes.Buffer
				err := cat.New(opts, core.Streams{In: strings.NewReader("in\n"), Out: failWriter{tt.werr}, Err: &errb}).
					Run([]string{path, "-"})
				if errors.Is(err, core.ErrFailed) != tt.failed {
					t.Fatalf("err = %v, want failed = %v", err, tt.failed)
				}
				if errb.String() != tt.wantErr {
					t.Fatalf("stderr = %q, want %q", errb.String(), tt.wantErr)
				}
			})
		}
	}
}
