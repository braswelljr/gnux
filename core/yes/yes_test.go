package yes_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/yes"
)

// limitWriter accepts limit bytes and then fails every write with err, which
// is what bounds the otherwise endless engine in tests.
type limitWriter struct {
	buf    bytes.Buffer
	limit  int
	err    error
	writes int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	w.writes++
	room := w.limit - w.buf.Len()
	if len(p) <= room {
		return w.buf.Write(p)
	}
	w.buf.Write(p[:room])
	return room, w.err
}

func TestLine(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "y\n"},
		{"one operand", []string{"n"}, "n\n"},
		{"operands are joined by spaces", []string{"a", "b c", "d"}, "a b c d\n"},
		{"empty operand", []string{""}, "\n"},
		{"leading double dash is dropped", []string{"--", "a"}, "a\n"},
		{"only the first double dash is dropped", []string{"a", "--", "--", "b"}, "a -- b\n"},
		{"lone double dash", []string{"--"}, "y\n"},
		{"option-like operands are text", []string{"-n", "--foo"}, "-n --foo\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := yes.Line(tt.args); got != tt.want {
				t.Fatalf("Line(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunRepeatsUntilThePipeCloses(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		limit int
	}{
		{"default line", nil, 200_001},
		{"operands", []string{"hello", "world"}, 500_000},
		{"stops at once", []string{"x"}, 0},
		{"line longer than the buffer", []string{strings.Repeat("z", 100_000)}, 250_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &limitWriter{limit: tt.limit, err: io.ErrClosedPipe}
			var errb bytes.Buffer
			y := yes.New(core.Streams{In: strings.NewReader(""), Out: w, Err: &errb})
			if err := y.Run(tt.args); err != nil {
				t.Fatalf("Run = %v, want nil on a closed pipe", err)
			}
			if errb.Len() != 0 {
				t.Errorf("stderr = %q, want empty", errb.String())
			}

			line := yes.Line(tt.args)
			got := w.buf.String()
			if len(got) != tt.limit {
				t.Fatalf("wrote %d bytes, want %d", len(got), tt.limit)
			}
			// The output must be the line repeated, cut wherever the writer gave up.
			want := strings.Repeat(line, tt.limit/len(line)+1)[:tt.limit]
			if got != want {
				t.Fatalf("output is not %q repeated (starts %q)", line, got[:min(len(got), 40)])
			}
			// Batching: far fewer writes than lines for a short line.
			if lines := tt.limit / len(line); lines > 1000 && w.writes > lines/100 {
				t.Errorf("%d writes for %d lines: output is not batched", w.writes, lines)
			}
		})
	}
}

func TestRunStopsQuietlyOnARealClosedPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	// Hang up before yes starts, as a reader that exited would have.
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	var errb bytes.Buffer
	y := yes.New(core.Streams{In: strings.NewReader(""), Out: w, Err: &errb})
	if err := y.Run(nil); err != nil {
		t.Fatalf("Run = %v, want nil on a broken pipe", err)
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errb.String())
	}
}

func TestRunReportsOtherWriteErrors(t *testing.T) {
	w := &limitWriter{limit: 10, err: &fs.PathError{Op: "write", Path: "/dev/stdout", Err: errors.New("no space left on device")}}
	var errb bytes.Buffer
	y := yes.New(core.Streams{In: strings.NewReader(""), Out: w, Err: &errb})
	if err := y.Run([]string{"ab"}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("Run = %v, want ErrFailed", err)
	}
	if want := "yesx: standard output: No space left on device\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	if want := "ab\nab\nab\na"; w.buf.String() != want {
		t.Fatalf("stdout = %q, want %q", w.buf.String(), want)
	}
}
