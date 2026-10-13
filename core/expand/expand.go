// Package expand implements the engine behind expandx, a GNU-compatible
// implementation of the GNU/UNIX expand command: convert tabs to spaces. Its
// tab-stop list (TabStops) and multi-file input (Input) are shared with
// unexpand.
package expand

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "expandx"

// Options mirrors the GNU expand flag surface. The zero value is plain
// expand: every tab converted, tab stops every 8 columns.
type Options struct {
	// Initial (-i) converts only the tabs before the first non-blank of each
	// line.
	Initial bool
	// Tabs (-t) are tab-stop lists in GNU syntax, one per -t option, applied
	// in order: "4" (every 4 columns), "4,8,12" (those columns), and a final
	// "/N" (every multiple of N after the last stop) or "+N" (every N columns
	// after it).
	Tabs []string
}

// Expand is the engine. Construct it with New.
type Expand struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds an expand engine reading from and writing to s.
func New(opts Options, s core.Streams) *Expand {
	return &Expand{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Build parses specs into a finalized TabStops, writing any error to diag.
func Build(specs []string, diag *core.Diag) (*TabStops, bool) {
	t := &TabStops{}
	for _, spec := range specs {
		if !t.Parse(spec, diag) {
			return nil, false
		}
	}
	if !t.Finalize(diag) {
		return nil, false
	}
	return t, true
}

// Run converts the operands, read as one stream; no operands, or "-", means
// standard input. An invalid tab-stop list stops the run before any input is
// read. The returned error is core.ErrFailed if anything failed, otherwise
// nil.
func (e *Expand) Run(args []string) error {
	tabs, ok := Build(e.Tabs, e.diag)
	if !ok {
		return core.ErrFailed
	}
	in := NewInput(args, e.streams, e.diag)
	defer in.Close()
	out := core.NewWriter(e.streams.Out)

	var c byte
	var err error
	for {
		convert := true
		var col uint64
		idx := 0
		for {
			c, err = in.ReadByte()
			if err != nil {
				return e.finish(out)
			}
			if convert {
				switch c {
				case '\t':
					next, last := tabs.Next(col, &idx)
					if last {
						next = col + 1
					}
					if next < col {
						e.diag.Errorf("input line is too long")
						_ = out.Flush()
						return core.ErrFailed
					}
					for col++; col < next; col++ {
						_ = out.WriteByte(' ')
					}
					c = ' '
				case '\b':
					if col > 0 {
						col--
					}
					if idx > 0 {
						idx--
					}
				default:
					col++
					if col == 0 {
						e.diag.Errorf("input line is too long")
						_ = out.Flush()
						return core.ErrFailed
					}
				}
				convert = convert && (!e.Initial || c == ' ' || c == '\t')
			}
			if werr := out.WriteByte(c); werr != nil {
				return e.writeFailed(werr)
			}
			if c == '\n' {
				if in.Idle() {
					_ = out.Flush()
				}
				break
			}
		}
	}
}

func (e *Expand) finish(out *bufio.Writer) error {
	if err := out.Flush(); err != nil {
		return e.writeFailed(err)
	}
	return e.diag.Result()
}

func (e *Expand) writeFailed(err error) error {
	if closedPipe(err) {
		return e.diag.Result()
	}
	e.diag.Errorf("write error: %s", core.ErrText(err))
	return core.ErrFailed
}

// Input reads a list of operands as one stream of bytes, as GNU expand and
// unexpand do: a line may continue from one file into the next. An operand
// that cannot be opened or read is reported and skipped.
type Input struct {
	names   []string
	streams core.Streams
	diag    *core.Diag

	cur  *bufio.Reader
	file *os.File
	name string
}

// NewInput returns an Input over names; no names, or "-", means standard
// input.
func NewInput(names []string, s core.Streams, diag *core.Diag) *Input {
	if len(names) == 0 {
		names = []string{"-"}
	}
	return &Input{names: names, streams: s, diag: diag}
}

// ReadByte returns the next byte of the stream, or io.EOF after the last
// operand.
func (in *Input) ReadByte() (byte, error) {
	for {
		if in.cur == nil && !in.next() {
			return 0, io.EOF
		}
		c, err := in.cur.ReadByte()
		if err == nil {
			return c, nil
		}
		if !errors.Is(err, io.EOF) {
			in.diag.Errorf("%s: %s", in.name, core.ErrText(err))
		}
		in.closeCurrent()
	}
}

// Idle reports that no more input is buffered: the next ReadByte will wait
// for whoever produces the input (see core.FlushIfIdle).
func (in *Input) Idle() bool {
	return in.cur == nil || in.cur.Buffered() == 0
}

// next opens the next readable operand, reporting the ones that fail.
func (in *Input) next() bool {
	for len(in.names) > 0 {
		name := in.names[0]
		in.names = in.names[1:]
		if name == "-" {
			r := in.streams.In
			if r == nil {
				r = strings.NewReader("")
			}
			in.cur, in.name = core.NewReader(r), name
			return true
		}
		f, err := os.Open(name)
		if err != nil {
			in.diag.Errorf("%s: %s", name, core.ErrText(err))
			continue
		}
		if info, err := f.Stat(); err == nil && info.IsDir() {
			in.diag.Errorf("%s: %s", name, core.ErrText(errIsDir))
			f.Close()
			continue
		}
		in.cur, in.file, in.name = core.NewReader(f), f, name
		return true
	}
	return false
}

func (in *Input) closeCurrent() {
	if in.file != nil {
		in.file.Close()
	}
	in.cur, in.file = nil, nil
}

// Close releases the operand being read, if any.
func (in *Input) Close() {
	in.closeCurrent()
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
