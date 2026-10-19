// Package cat implements the engine behind catx, a GNU-compatible implementation of the
// GNU/UNIX cat command: concatenate files to standard output, optionally
// numbering lines and making invisible characters visible.
package cat

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "catx"

// Options mirrors the GNU cat flag surface. The compound flags (-A, -e, -t)
// are expanded by the command adapter; -u has no effect and no field.
type Options struct {
	// Number (-n) numbers every output line.
	Number bool
	// NumberNonblank (-b) numbers only non-empty lines; it overrides Number.
	NumberNonblank bool
	// SqueezeBlank (-s) collapses runs of empty lines into one.
	SqueezeBlank bool
	// ShowEnds (-E) prints $ at the end of each line.
	ShowEnds bool
	// ShowTabs (-T) prints TAB as ^I.
	ShowTabs bool
	// ShowNonprinting (-v) prints control and non-ASCII bytes in ^X / M-
	// notation, leaving newline and TAB alone.
	ShowNonprinting bool
}

// Cat is the engine. Construct it with New.
type Cat struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	// Line state deliberately outlives a single operand: numbering continues
	// and blank-line squeezing spans file boundaries, as in GNU cat.
	lineNo    int64
	lineStart bool // the next byte begins a line
	blanks    int  // consecutive empty lines just written
	pendingCR bool // a '\r' held back to see whether '\n' follows (-E)
}

// New builds a cat engine reading from and writing to s.
func New(opts Options, s core.Streams) *Cat {
	return &Cat{
		Options:   opts,
		streams:   s,
		diag:      core.NewDiag(Name, s),
		lineStart: true,
	}
}

// Run copies each operand to standard output; no operands, or "-", means
// standard input. A failing operand is reported and the rest still run. The
// returned error is core.ErrFailed if any operand failed, otherwise nil.
func (c *Cat) Run(args []string) error {
	if len(args) == 0 {
		args = []string{"-"}
	}
	c.out = core.NewWriter(c.streams.Out)

	for _, name := range args {
		if !c.file(name) {
			break
		}
	}
	return c.diag.Result()
}

// transforming reports whether any flag requires looking at the data.
func (c *Cat) transforming() bool {
	return c.Number || c.NumberNonblank || c.SqueezeBlank ||
		c.ShowEnds || c.ShowTabs || c.ShowNonprinting
}

// file copies one operand. It returns false when standard output is gone and
// there is no point in continuing.
func (c *Cat) file(name string) bool {
	var r io.Reader
	if name == "-" {
		r = c.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		f, err := fsys.Open(name)
		if err != nil {
			c.diag.Errorf("%s: %s", name, core.ErrText(err))
			return true
		}
		defer f.Close()
		r = f
	}

	if f, ok := r.(*os.File); ok {
		if info, err := f.Stat(); err == nil {
			// Reading a directory fails differently on every platform; name
			// it up front so the diagnostic is the same everywhere.
			if info.IsDir() {
				c.diag.Errorf("%s: Is a directory", name)
				return true
			}
			if c.feedsItself(info) {
				c.diag.Errorf("%s: input file is output file", name)
				return true
			}
		}
	}

	var rerr, werr error
	if c.transforming() {
		rerr = c.transform(r)
		werr = c.out.Flush()
	} else {
		// Unbuffered and untouched: io.Copy can hand two files to the kernel
		// (sendfile, copy_file_range) instead of moving the bytes itself.
		rerr, werr = plainCopy(c.streams.Out, r)
	}

	if werr != nil {
		if !closedPipe(werr) {
			c.diag.Errorf("write error: %s", core.ErrText(werr))
		}
		return false
	}
	if rerr != nil {
		c.diag.Errorf("%s: %s", name, core.ErrText(rerr))
	}
	return true
}

// feedsItself reports whether a non-empty regular input is also where
// standard output goes (catx f >> f), which would never reach end of file.
// An input already truncated by the shell (catx f > f) is empty and harmless.
func (c *Cat) feedsItself(in os.FileInfo) bool {
	out, ok := c.streams.Out.(*os.File)
	if !ok || !in.Mode().IsRegular() || in.Size() == 0 {
		return false
	}
	outInfo, err := out.Stat()
	return err == nil && outInfo.Mode().IsRegular() && os.SameFile(in, outInfo)
}

// errReader and errWriter tag which side of a copy failed, since io.Copy
// returns a single error.
type errReader struct {
	r   io.Reader
	err error
}

func (e *errReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		e.err = err
	}
	return n, err
}

// plainCopy copies r to w, reporting read and write failures separately.
func plainCopy(w io.Writer, r io.Reader) (rerr, werr error) {
	// A bare *os.File keeps io.Copy's kernel fast paths, and a file read
	// error is recognizable by its "read" operation.
	if f, ok := r.(*os.File); ok {
		_, err := io.Copy(w, f)
		var perr *os.PathError
		if errors.As(err, &perr) && perr.Op == "read" {
			return err, nil
		}
		return nil, err
	}
	er := &errReader{r: r}
	_, err := io.Copy(w, er)
	if er.err != nil {
		return er.err, nil
	}
	return nil, err
}

// transform copies r through the line-numbering and character-revealing
// filters. A write failure surfaces when the caller flushes.
func (c *Cat) transform(r io.Reader) error {
	buf := make([]byte, core.BufferSize)
	for {
		n, err := r.Read(buf)
		for _, b := range buf[:n] {
			c.put(b)
		}
		// One write for each read: a slow producer is answered as it goes,
		// and a fast one fills the whole buffer anyway. Reading stops once
		// output has failed; the error stays with the writer for the
		// caller's flush to report.
		if werr := c.out.Flush(); werr != nil {
			return nil
		}
		if err != nil {
			// A carriage return ending the file has no newline to pair with.
			if c.pendingCR {
				c.pendingCR = false
				_ = c.out.WriteByte('\r')
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// put writes one input byte in its transformed form. Write errors are sticky
// in the buffered writer, so they are left for the flush to report.
func (c *Cat) put(b byte) {
	w := c.out

	if c.pendingCR {
		c.pendingCR = false
		if b == '\n' {
			_, _ = w.WriteString("^M")
		} else {
			_ = w.WriteByte('\r')
		}
	}

	if c.lineStart {
		if b == '\n' {
			c.blanks++
			if c.SqueezeBlank && c.blanks > 1 {
				return
			}
			if c.Number && !c.NumberNonblank {
				c.number()
			}
			c.endLine()
			return
		}
		c.blanks = 0
		if c.Number || c.NumberNonblank {
			c.number()
		}
		c.lineStart = false
	}

	switch {
	case b == '\n':
		c.endLine()
		c.lineStart = true
	case b == '\t':
		if c.ShowTabs {
			_, _ = w.WriteString("^I")
		} else {
			_ = w.WriteByte(b)
		}
	case c.ShowNonprinting:
		c.reveal(b)
	case b == '\r' && c.ShowEnds:
		// Without -v, -E still marks a CR that is part of a CRLF line
		// ending (as ^M$); whether this one is depends on the next byte.
		c.pendingCR = true
	default:
		_ = w.WriteByte(b)
	}
}

func (c *Cat) number() {
	c.lineNo++
	_, _ = c.out.Write(core.AppendInt(c.out.AvailableBuffer(), c.lineNo, 6))
	_ = c.out.WriteByte('\t')
}

func (c *Cat) endLine() {
	if c.ShowEnds {
		_ = c.out.WriteByte('$')
	}
	_ = c.out.WriteByte('\n')
}

// reveal writes b in cat -v notation: ^X for control characters, ^? for DEL,
// and an M- prefix for bytes with the high bit set.
func (c *Cat) reveal(b byte) {
	w := c.out
	if b >= 0x80 {
		_, _ = w.WriteString("M-")
		b -= 0x80
	}
	switch {
	case b < 0x20:
		_ = w.WriteByte('^')
		_ = w.WriteByte(b + 0x40)
	case b == 0x7f:
		_, _ = w.WriteString("^?")
	default:
		_ = w.WriteByte(b)
	}
}

// closedPipe reports whether err means the reader of standard output went
// away (cat | true), which ends the run quietly rather than as a failure.
func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
