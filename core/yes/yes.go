// Package yes implements the engine behind yesx, a GNU-compatible implementation of the
// GNU/UNIX yes command: print a line over and over until the reader goes away.
package yes

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "yesx"

// bufferSize is the least amount of output handed to each write. Batching many
// copies of the line per write is what makes yes fast: one system call moves
// tens of thousands of lines instead of one.
const bufferSize = 64 * 1024

// Yes is the engine. Construct it with New.
type Yes struct {
	streams core.Streams
	diag    *core.Diag
}

// New builds a yes engine writing to s.
func New(s core.Streams) *Yes {
	return &Yes{streams: s, diag: core.NewDiag(Name, s)}
}

// Line returns the line yes repeats for the given arguments: the operands
// joined by single spaces, or "y" when there are none, plus a newline. As with
// getopt, the first "--" marks the end of options and is not an operand.
func Line(args []string) string {
	operands := make([]string, 0, len(args))
	seenDashes := false
	for _, a := range args {
		if a == "--" && !seenDashes {
			seenDashes = true
			continue
		}
		operands = append(operands, a)
	}
	if len(operands) == 0 {
		return "y\n"
	}
	return strings.Join(operands, " ") + "\n"
}

// Run writes the line until a write fails; it never returns otherwise. A
// reader that closed the pipe is the normal way for yes to end, so that is not
// an error. Any other failure is reported and yields core.ErrFailed.
func (y *Yes) Run(args []string) error {
	line := Line(args)
	// Only whole lines go in the buffer, so every write ends on a line
	// boundary. A line longer than the buffer is simply written by itself.
	buf := bytes.Repeat([]byte(line), max(1, bufferSize/len(line)))

	for {
		if _, err := y.streams.Out.Write(buf); err != nil {
			if errors.Is(err, io.ErrClosedPipe) || isBrokenPipe(err) {
				return nil
			}
			y.diag.Errorf("standard output: %s", core.ErrText(err))
			return y.diag.Result()
		}
	}
}
