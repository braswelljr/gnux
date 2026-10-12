package cut_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cut"
)

// run executes the engine against in-memory streams.
func run(opts cut.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = cut.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func bytesOf(list string) cut.Options  { return cut.Options{Mode: cut.ModeBytes, List: list} }
func fieldsOf(list string) cut.Options { return cut.Options{Mode: cut.ModeFields, List: list} }

func TestBytes(t *testing.T) {
	tests := []struct {
		name  string
		opts  cut.Options
		input string
		want  string
	}{
		{"one position", bytesOf("2"), "abcdef\n", "b\n"},
		{"list", bytesOf("1,3,5"), "abcdef\n", "ace\n"},
		{"range", bytesOf("2-4"), "abcdef\n", "bcd\n"},
		{"open end", bytesOf("4-"), "abcdef\n", "def\n"},
		{"open start", bytesOf("-2"), "abcdef\n", "ab\n"},
		{"order and repeats do not matter", bytesOf("3,1,1-2"), "abcdef\n", "abc\n"},
		{"blank separated list", bytesOf("1 3"), "abc\n", "ac\n"},
		{"past the end", bytesOf("5-9"), "abc\n", "\n"},
		{"each line", bytesOf("1"), "ab\ncd\n", "a\nc\n"},
		{"unterminated last line gets a newline", bytesOf("1"), "ab", "a\n"},
		{"empty line", bytesOf("1"), "\n", "\n"},
		{"bytes, not characters", bytesOf("1-2"), "héllo\n", "h\xc3\n"},
		{"chars are bytes, as in GNU", cut.Options{Mode: cut.ModeChars, List: "2"}, "abc\n", "b\n"},
		{"complement", cut.Options{Mode: cut.ModeBytes, List: "2-3", Complement: true}, "abcdef\n", "adef\n"},
		{"complement of an open range", cut.Options{Mode: cut.ModeBytes, List: "3-", Complement: true}, "abcdef\n", "ab\n"},
		{"output delimiter between ranges", cut.Options{Mode: cut.ModeBytes, List: "1-2,4-5", OutputDelimiter: ":", OutputDelimiterSet: true}, "abcdef\n", "ab:de\n"},
		{"adjacent ranges stay apart", cut.Options{Mode: cut.ModeBytes, List: "1,2", OutputDelimiter: ":", OutputDelimiterSet: true}, "abc\n", "a:b\n"},
		{"overlapping ranges merge", cut.Options{Mode: cut.ModeBytes, List: "1-3,2-4", OutputDelimiter: ":", OutputDelimiterSet: true}, "abcdef\n", "abcd\n"},
		{"no delimiter before an absent range", cut.Options{Mode: cut.ModeBytes, List: "1,9", OutputDelimiter: ":", OutputDelimiterSet: true}, "abc\n", "a\n"},
		{"zero terminated", cut.Options{Mode: cut.ModeBytes, List: "1", ZeroTerminated: true}, "ab\x00cd", "a\x00c\x00"},
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

func TestFields(t *testing.T) {
	tests := []struct {
		name  string
		opts  cut.Options
		input string
		want  string
	}{
		{"tab by default", fieldsOf("2"), "a\tb\tc\n", "b\n"},
		{"several", fieldsOf("1,3"), "a\tb\tc\n", "a\tc\n"},
		{"range", fieldsOf("2-"), "a\tb\tc\n", "b\tc\n"},
		{"delimiter", cut.Options{Mode: cut.ModeFields, List: "1,3", Delimiter: ":"}, "root:x:0:0\n", "root:0\n"},
		{"empty fields count", cut.Options{Mode: cut.ModeFields, List: "3", Delimiter: ","}, "a,,c\n", "c\n"},
		{"trailing empty field", cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: ","}, "a,\n", "\n"},
		{"field past the end prints an empty line", cut.Options{Mode: cut.ModeFields, List: "5", Delimiter: ","}, "a,b\n", "\n"},
		{"no delimiter prints the whole line", cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: ","}, "plain\n", "plain\n"},
		{"only delimited", cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: ",", OnlyDelimited: true}, "plain\na,b\n", "a\n"},
		{"output delimiter", cut.Options{Mode: cut.ModeFields, List: "1,2", Delimiter: ":", OutputDelimiter: " | ", OutputDelimiterSet: true}, "a:b:c\n", "a | b\n"},
		{"empty output delimiter", cut.Options{Mode: cut.ModeFields, List: "1-", Delimiter: ":", OutputDelimiterSet: true}, "a:b:c\n", "abc\n"},
		{"complement", cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: ",", Complement: true}, "a,b,c\n", "a,c\n"},
		{"nul delimiter", cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: "\x00"}, "a\x00b\n", "b\n"},
		{"unterminated last line", cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: ","}, "a,b", "a\n"},
		{"zero terminated", cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: ",", ZeroTerminated: true}, "a,b\nc\x00d,e\x00", "b\nc\x00e\x00"},
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
		opts cut.Options
		want string
	}{
		{"no list", cut.Options{}, "you must specify a list of bytes, characters, or fields"},
		{"zero field", fieldsOf("0"), "fields are numbered from 1"},
		{"zero byte", bytesOf("1,0"), "byte/character positions are numbered from 1"},
		{"decreasing", fieldsOf("3-1"), "invalid decreasing range"},
		{"lone dash", fieldsOf("-"), "invalid range with no endpoint: -"},
		{"double range", fieldsOf("1-2-3"), "invalid field range"},
		{"double byte range", bytesOf("1-2-3"), "invalid byte or character range"},
		{"letters", fieldsOf("1,x"), "invalid field value 'x'"},
		{"letters after digits", bytesOf("2a"), "invalid byte/character position 'a'"},
		{"empty list", fieldsOf(""), "missing list of fields"},
		{"only commas", bytesOf(","), "missing list of byte/character positions"},
		{"too large", fieldsOf("99999999999999999999"), "field number '99999999999999999999' is too large"},
		{"long delimiter", cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: "ab"}, "the delimiter must be a single character"},
		{"multibyte delimiter", cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: "é"}, "the delimiter must be a single character"},
		{"delimiter without fields", cut.Options{Mode: cut.ModeBytes, List: "1", Delimiter: ","}, "an input delimiter may be specified only when operating on fields"},
		{"only delimited without fields", cut.Options{Mode: cut.ModeChars, List: "1", OnlyDelimited: true}, "suppressing non-delimited lines makes sense\n\tonly when operating on fields"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, "a\n")
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

func TestOperands(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(a, []byte("1,2\n3,4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	opts := cut.Options{Mode: cut.ModeFields, List: "2", Delimiter: ","}

	out, errb, err := run(opts, "x,stdin\n", a, missing, "-", dir)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "2\n4\nstdin\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	want := "cutx: " + missing + ": No such file or directory\n" +
		"cutx: " + dir + ": Is a directory\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	opts := cut.Options{Mode: cut.ModeBytes, List: "1"}
	t.Run("closed pipe is quiet", func(t *testing.T) {
		var errb bytes.Buffer
		c := cut.New(opts, core.Streams{In: strings.NewReader("a\n"), Out: failWriter{io.ErrClosedPipe}, Err: &errb})
		if err := c.Run(nil); err != nil || errb.String() != "" {
			t.Fatalf("err = %v, stderr = %q", err, errb.String())
		}
	})
	t.Run("other write errors are reported", func(t *testing.T) {
		var errb bytes.Buffer
		c := cut.New(opts, core.Streams{In: strings.NewReader("a\n"), Out: failWriter{errors.New("disk full")}, Err: &errb})
		if err := c.Run(nil); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("err = %v, want ErrFailed", err)
		}
		if want := "cutx: write error: Disk full\n"; errb.String() != want {
			t.Fatalf("stderr = %q, want %q", errb.String(), want)
		}
	})
}
