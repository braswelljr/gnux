// Package tee implements the engine behind teex, a GNU-compatible
// implementation of the GNU/UNIX tee command: copy standard input to standard
// output and to every FILE.
package tee

import (
	"errors"
	"io"
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "teex"

// bufSize is the read chunk. Each chunk is written out as soon as it is read,
// so tee never holds data back from a pipeline.
const bufSize = 64 << 10

// OutputError says what happens when writing to an output fails
// (--output-error=MODE, -p).
type OutputError int

const (
	// OutputErrorDefault is GNU's behavior without --output-error: a failing
	// output is diagnosed and dropped, and a closed pipe stops tee quietly,
	// as SIGPIPE would.
	OutputErrorDefault OutputError = iota
	// OutputErrorWarn diagnoses an error on any output, pipes included, and
	// carries on with the rest.
	OutputErrorWarn
	// OutputErrorWarnNoPipe diagnoses an error on any output that is not a
	// closed pipe, drops a closed pipe silently, and carries on (-p).
	OutputErrorWarnNoPipe
	// OutputErrorExit stops at the first error on any output.
	OutputErrorExit
	// OutputErrorExitNoPipe stops at the first error on an output that is not
	// a closed pipe; a closed pipe is dropped silently.
	OutputErrorExitNoPipe
)

// Options mirrors the GNU tee flag surface.
type Options struct {
	// Append (-a) appends to each FILE instead of truncating it.
	Append bool
	// IgnoreInterrupts (-i) is accepted for compatibility. gnux runs tools
	// in-process (in gnux shell and the playground as well), so it leaves
	// signal handling to the host and this field has no effect.
	IgnoreInterrupts bool
	// OutputError (--output-error, -p) selects the write-error behavior.
	OutputError OutputError
}

// Tee is the engine. Construct it with New.
type Tee struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds a tee engine reading from and writing to s.
func New(opts Options, s core.Streams) *Tee {
	return &Tee{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// output is one destination of the copy.
type output struct {
	name string // as it appears in diagnostics
	w    io.Writer
	file *os.File // nil for standard output
}

// Run copies standard input to standard output and to each operand. Each
// operand is a file name, "-" included, as in GNU tee 9. A FILE that cannot be
// opened is reported and the copy goes on to the others. The returned error is
// core.ErrFailed if anything failed, otherwise nil.
func (t *Tee) Run(args []string) error {
	flags := os.O_WRONLY | os.O_CREATE
	if t.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	outs := []*output{{name: "standard output", w: t.streams.Out}}
	for _, name := range args {
		f, err := os.OpenFile(name, flags, 0o666)
		if err != nil {
			t.diag.Errorf("%s: %s", name, core.ErrText(err))
			continue
		}
		outs = append(outs, &output{name: name, w: f, file: f})
	}

	t.copy(outs)

	for _, o := range outs {
		if o.file == nil {
			continue
		}
		if err := o.file.Close(); err != nil {
			t.diag.Errorf("%s: %s", o.name, core.ErrText(err))
		}
	}
	return t.diag.Result()
}

// copy moves standard input to every live output until input ends, every
// output is gone, or the error mode says to stop.
func (t *Tee) copy(outs []*output) {
	buf := make([]byte, bufSize)
	live := len(outs)
	for live > 0 {
		n, rerr := t.streams.In.Read(buf)
		if n > 0 {
			for _, o := range outs {
				if o.w == nil {
					continue
				}
				if _, err := o.w.Write(buf[:n]); err != nil {
					if !t.writeFailed(o, err) {
						return
					}
					o.w = nil
					live--
				}
			}
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				t.diag.Errorf("read error: %s", core.ErrText(rerr))
			}
			return
		}
	}
}

// writeFailed handles a write error on o according to the error mode. It
// returns false when tee must stop altogether.
func (t *Tee) writeFailed(o *output, err error) bool {
	pipe := core.IsClosedPipe(err)
	switch t.OutputError {
	case OutputErrorDefault:
		if pipe {
			return false
		}
		t.diag.Errorf("%s: %s", o.name, core.ErrText(err))
		return true
	case OutputErrorWarnNoPipe, OutputErrorExitNoPipe:
		if pipe {
			return true
		}
	}
	t.diag.Errorf("%s: %s", o.name, core.ErrText(err))
	return t.OutputError != OutputErrorExit && t.OutputError != OutputErrorExitNoPipe
}
