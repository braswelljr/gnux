package uniq_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uniq"
)

// run executes the engine against in-memory streams.
func run(opts uniq.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = uniq.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestFilter(t *testing.T) {
	const in = "a\na\nb\nc\nc\nc\nd\n"
	tests := []struct {
		name  string
		opts  uniq.Options
		input string
		want  string
	}{
		{"default", uniq.Options{}, in, "a\nb\nc\nd\n"},
		{"empty input", uniq.Options{}, "", ""},
		{"only adjacent lines", uniq.Options{}, "a\nb\na\n", "a\nb\na\n"},
		{"unterminated last line gets a newline", uniq.Options{}, "a\na", "a\n"},
		{"empty lines are lines", uniq.Options{}, "\n\nx\n", "\nx\n"},
		{"count", uniq.Options{Count: true}, in, "      2 a\n      1 b\n      3 c\n      1 d\n"},
		{"repeated", uniq.Options{Repeated: true}, in, "a\nc\n"},
		{"repeated with count", uniq.Options{Repeated: true, Count: true}, in, "      2 a\n      3 c\n"},
		{"unique", uniq.Options{Unique: true}, in, "b\nd\n"},
		{"unique with count", uniq.Options{Unique: true, Count: true}, in, "      1 b\n      1 d\n"},
		{"repeated and unique print nothing", uniq.Options{Repeated: true, Unique: true}, in, ""},
		{"all repeated", uniq.Options{AllRepeated: true}, in, "a\na\nc\nc\nc\n"},
		{"all repeated prepend", uniq.Options{AllRepeated: true, Delimit: uniq.DelimitPrepend}, in, "\na\na\n\nc\nc\nc\n"},
		{"all repeated separate", uniq.Options{AllRepeated: true, Delimit: uniq.DelimitSeparate}, in, "a\na\n\nc\nc\nc\n"},
		{"all repeated with nothing repeated", uniq.Options{AllRepeated: true, Delimit: uniq.DelimitSeparate}, "a\nb\n", ""},
		{"group separate", uniq.Options{Group: uniq.GroupSeparate}, "a\na\nb\n", "a\na\n\nb\n"},
		{"group prepend", uniq.Options{Group: uniq.GroupPrepend}, "a\na\nb\n", "\na\na\n\nb\n"},
		{"group append", uniq.Options{Group: uniq.GroupAppend}, "a\na\nb\n", "a\na\n\nb\n\n"},
		{"group both", uniq.Options{Group: uniq.GroupBoth}, "a\na\nb\n", "\na\na\n\nb\n\n"},
		{"group of nothing", uniq.Options{Group: uniq.GroupBoth}, "", ""},
		{"ignore case", uniq.Options{IgnoreCase: true}, "Apple\napple\nAPPLE\nb\n", "Apple\nb\n"},
		{"case matters by default", uniq.Options{}, "A\na\n", "A\na\n"},
		{"skip fields", uniq.Options{SkipFields: 1}, "1 x\n2 x\n3 y\n", "1 x\n3 y\n"},
		{"skip fields keeps leading blanks of the next field", uniq.Options{SkipFields: 1}, "a  x\nb x\n", "a  x\nb x\n"},
		{"skip more fields than there are", uniq.Options{SkipFields: 5}, "a b\nc d\n", "a b\n"},
		{"skip chars", uniq.Options{SkipChars: 2}, "aax\nbbx\nccy\n", "aax\nccy\n"},
		{"skip fields then chars, the blank included", uniq.Options{SkipFields: 1, SkipChars: 2}, "1 ax\n2 bx\n", "1 ax\n"},
		{"check chars", uniq.Options{CheckChars: 2, CheckCharsSet: true}, "abX\nabY\nacZ\n", "abX\nacZ\n"},
		{"check zero chars", uniq.Options{CheckChars: 0, CheckCharsSet: true}, "a\nb\nc\n", "a\n"},
		{"zero terminated", uniq.Options{ZeroTerminated: true}, "a\x00a\x00b\nc\x00", "a\x00b\nc\x00"},
		{"zero terminated count", uniq.Options{ZeroTerminated: true, Count: true}, "a\x00a", "      2 a\x00"},
		// With -D the anchor moves to each line, so a chain of near matches
		// stays one group, as in GNU.
		{"all repeated compares with the previous line", uniq.Options{AllRepeated: true, CheckChars: 1, CheckCharsSet: true}, "aa\nab\nb\n", "aa\nab\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts uniq.Options
		args []string
		want string
	}{
		{"extra operand", uniq.Options{}, []string{"a", "b", "c"}, "extra operand 'c'"},
		{"all repeated with count", uniq.Options{AllRepeated: true, Count: true}, nil,
			"printing all duplicated lines and repeat counts is meaningless"},
		{"group with count", uniq.Options{Group: uniq.GroupSeparate, Count: true}, nil,
			"--group is mutually exclusive with -c/-d/-D/-u"},
		{"group with repeated", uniq.Options{Group: uniq.GroupBoth, Repeated: true}, nil,
			"--group is mutually exclusive with -c/-d/-D/-u"},
		{"group with all repeated", uniq.Options{Group: uniq.GroupAppend, AllRepeated: true}, nil,
			"--group is mutually exclusive with -c/-d/-D/-u"},
		{"group with unique", uniq.Options{Group: uniq.GroupPrepend, Unique: true}, nil,
			"--group is mutually exclusive with -c/-d/-D/-u"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, "a\n", tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a UsageError", err)
			}
			if usage.Msg != tt.want {
				t.Fatalf("message = %q, want %q", usage.Msg, tt.want)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want nothing", out)
			}
		})
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(in, []byte("x\nx\ny\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("input file", func(t *testing.T) {
		out, _, err := run(uniq.Options{}, "ignored\n", in)
		if err != nil || out != "x\ny\n" {
			t.Fatalf("out = %q, err = %v", out, err)
		}
	})

	t.Run("dash is standard input", func(t *testing.T) {
		out, _, err := run(uniq.Options{}, "s\ns\n", "-")
		if err != nil || out != "s\n" {
			t.Fatalf("out = %q, err = %v", out, err)
		}
	})

	t.Run("output file", func(t *testing.T) {
		dst := filepath.Join(dir, "out.txt")
		if err := os.WriteFile(dst, []byte("old contents that are longer\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, _, err := run(uniq.Options{Count: true}, "", in, dst)
		if err != nil || out != "" {
			t.Fatalf("out = %q, err = %v", out, err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		if want := "      2 x\n      1 y\n"; string(got) != want {
			t.Fatalf("output file = %q, want %q", got, want)
		}
	})

	t.Run("dash output is standard output", func(t *testing.T) {
		out, _, err := run(uniq.Options{}, "", in, "-")
		if err != nil || out != "x\ny\n" {
			t.Fatalf("out = %q, err = %v", out, err)
		}
	})

	t.Run("missing input", func(t *testing.T) {
		missing := filepath.Join(dir, "missing")
		_, errb, err := run(uniq.Options{}, "", missing)
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "uniqx: " + missing + ": No such file or directory\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})

	t.Run("directory input", func(t *testing.T) {
		_, errb, err := run(uniq.Options{}, "", dir)
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "uniqx: error reading '" + dir + "': Is a directory\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})

	t.Run("output in a missing directory", func(t *testing.T) {
		dst := filepath.Join(dir, "nope", "out.txt")
		_, errb, err := run(uniq.Options{}, "", in, dst)
		if !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "uniqx: " + dst + ": No such file or directory\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	t.Run("closed pipe is quiet", func(t *testing.T) {
		var errb bytes.Buffer
		u := uniq.New(uniq.Options{}, core.Streams{In: strings.NewReader("a\n"), Out: failWriter{io.ErrClosedPipe}, Err: &errb})
		if err := u.Run(nil); err != nil || errb.String() != "" {
			t.Fatalf("err = %v, stderr = %q", err, errb.String())
		}
	})
	t.Run("other write errors are reported", func(t *testing.T) {
		var errb bytes.Buffer
		u := uniq.New(uniq.Options{}, core.Streams{In: strings.NewReader("a\n"), Out: failWriter{errors.New("disk full")}, Err: &errb})
		if err := u.Run(nil); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "uniqx: write error: Disk full\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
	})
}

func TestExample(t *testing.T) {
	// A library caller counts the distinct entries of a sorted list.
	var out bytes.Buffer
	u := uniq.New(uniq.Options{Count: true}, core.Streams{In: strings.NewReader("go\ngo\nrust\n"), Out: &out, Err: &out})
	if err := u.Run(nil); err != nil {
		t.Fatal(err)
	}
	if want := "      2 go\n      1 rust\n"; out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}
