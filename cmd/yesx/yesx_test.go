package yesx

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/yes"
)

// closingWriter accepts limit bytes, then behaves like a pipe whose reader has
// gone away. It is what lets the endless command return in a test.
type closingWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *closingWriter) Write(p []byte) (int, error) {
	room := w.limit - w.buf.Len()
	if len(p) <= room {
		return w.buf.Write(p)
	}
	w.buf.Write(p[:room])
	return room, io.ErrClosedPipe
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		line string
	}{
		{"default", []string{}, "y\n"},
		{"operands", []string{"a", "b"}, "a b\n"},
		{"option-like arguments are text", []string{"-n", "x"}, "-n x\n"},
		{"double dash ends options", []string{"--", "-v"}, "-v\n"},
		{"--help with operands is text", []string{"--help", "me"}, "--help me\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const limit = 4096
			out := &closingWriter{limit: limit}
			var errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			want := strings.Repeat(tt.line, limit/len(tt.line)+1)[:limit]
			if got := out.buf.String(); got != want {
				t.Errorf("stdout is not %q repeated (starts %q)", tt.line, got[:min(len(got), 40)])
			}
			if errb.Len() != 0 {
				t.Errorf("stderr = %q, want empty", errb.String())
			}
		})
	}
}

func TestLoneVersion(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"--version"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.String(), yes.Name+" ") {
		t.Fatalf("stdout = %q, want a version line for %s", out.String(), yes.Name)
	}
}
