// Package head implements the engine behind headx, a GNU-compatible implementation of
// the GNU/UNIX head command: print the first part of each file.
package head

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "headx"

// DefaultLines is how many lines head prints when no count is given.
const DefaultLines = 10

// bufSize is the read chunk; large enough that line counting is dominated by
// bytes.IndexByte rather than by system calls.
const bufSize = 64 << 10

// Options mirrors the GNU head flag surface.
type Options struct {
	// Count is the number of lines (or bytes, with Bytes) selected by -n / -c.
	Count int64
	// Bytes (-c) makes Count a byte count instead of a line count.
	Bytes bool
	// FromEnd (-n -NUM / -c -NUM) prints everything except the last Count
	// lines or bytes, instead of the first Count.
	FromEnd bool
	// Quiet (-q) never prints file-name headers; it wins over Verbose.
	Quiet bool
	// Verbose (-v) always prints file-name headers.
	Verbose bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
	// Color is gnux's terminal view, not a GNU option: the file-name
	// headers in the heading color of Theme (a built-in theme; "" takes
	// GNUX_THEME's). The text is GNU's either way.
	Color bool
	Theme string
}

// Head is the engine. Construct it with New.
type Head struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	delim   byte
	theme   *visual.Theme // the headers' colors; nil when uncolored
	printed bool          // a header has been written, so the next one needs a blank line
}

// New builds a head engine reading from and writing to s.
func New(opts Options, s core.Streams) *Head {
	h := &Head{
		Options: opts,
		streams: s,
		diag:    core.NewDiag(Name, s),
		delim:   '\n',
	}
	if opts.ZeroTerminated {
		h.delim = 0
	}
	h.theme = visual.Use(opts.Color, opts.Theme, os.Getenv, h.diag.Warnf)
	return h
}

// Run prints the selected part of each operand; no operands, or "-", means
// standard input. A failing operand is reported and the rest still run. The
// returned error is core.ErrFailed if any operand failed, otherwise nil.
func (h *Head) Run(args []string) error {
	if len(args) == 0 {
		args = []string{"-"}
	}
	h.out = core.NewWriter(h.streams.Out)
	headers := !h.Quiet && (h.Verbose || len(args) > 1)

	for _, name := range args {
		if !h.file(name, headers) {
			break
		}
	}
	return h.diag.Result()
}

// file processes one operand. It returns false when standard output is gone
// and there is no point in continuing.
func (h *Head) file(name string, headers bool) bool {
	var (
		r       io.Reader
		display = name
	)
	if name == "-" {
		display = "standard input"
		r = h.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		f, err := fsys.Open(name)
		if err != nil {
			h.diag.Errorf("cannot open '%s' for reading: %s", name, core.ErrText(err))
			return true
		}
		defer f.Close()
		r = f
	}

	if headers {
		if h.printed {
			_ = h.out.WriteByte('\n')
		}
		fmt.Fprintf(h.out, "%s\n", h.theme.Paint(visual.Heading, "==> "+display+" <=="))
		h.printed = true
	}

	err := h.copy(r)

	// bufio.Writer errors are sticky, so a failed flush identifies a write
	// error; whatever copy returned otherwise came from reading.
	if werr := h.out.Flush(); werr != nil {
		if !closedPipe(werr) {
			h.diag.Errorf("write error: %s", core.ErrText(werr))
		}
		return false
	}
	if err != nil {
		h.diag.Errorf("error reading '%s': %s", display, core.ErrText(err))
	}
	return true
}

// copy writes the selected part of r to the output buffer.
func (h *Head) copy(r io.Reader) error {
	// Reading a directory fails differently on every platform; name it up
	// front so the diagnostic is the same everywhere.
	if f, ok := r.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.IsDir() {
			return errIsDir
		}
	}

	switch {
	case !h.FromEnd && h.Bytes:
		_, err := io.CopyN(h.out, r, h.Count)
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	case !h.FromEnd:
		return h.firstLines(r, h.Count)
	case h.Count == 0:
		_, err := io.Copy(h.out, r)
		return err
	case h.Bytes:
		return h.elideBytes(r, h.Count)
	default:
		return h.elideLines(r, h.Count)
	}
}

// firstLines copies the first n lines of r.
func (h *Head) firstLines(r io.Reader, n int64) error {
	buf := make([]byte, bufSize)
	for n > 0 {
		m, err := r.Read(buf)
		end := 0
		for n > 0 {
			i := bytes.IndexByte(buf[end:m], h.delim)
			if i < 0 {
				end = m
				break
			}
			end += i + 1
			n--
		}
		if _, werr := h.out.Write(buf[:end]); werr != nil {
			return werr
		}
		// One write for each read, so the lines of a slow producer are
		// passed on as they come.
		if werr := h.out.Flush(); werr != nil {
			return werr
		}
		if n == 0 {
			// Hand back what was read past the last line, so a following
			// reader of the same (seekable) input resumes right after it.
			if s, ok := r.(io.Seeker); ok && end < m {
				_, _ = s.Seek(int64(end-m), io.SeekCurrent)
			}
			return nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
	return nil
}

// elideLines copies r except its last n lines (n > 0). Only the trailing n
// lines are ever held in memory. A final line without a delimiter still counts
// as a line.
func (h *Head) elideLines(r io.Reader, n int64) error {
	var (
		w     window
		held  int64 // delimiters currently in the window
		chunk = make([]byte, bufSize)
	)
	for {
		m, err := r.Read(chunk)
		if m > 0 {
			w.add(chunk[:m])
			held += int64(bytes.Count(chunk[:m], []byte{h.delim}))
			if held > n {
				cut := nthIndex(w.bytes(), h.delim, held-n) + 1
				if _, werr := h.out.Write(w.bytes()[:cut]); werr != nil {
					return werr
				}
				w.drop(cut)
				held = n
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
	}

	// An unterminated last line makes the window hold n+1 lines: release one.
	rest := w.bytes()
	if held == n && len(rest) > 0 && rest[len(rest)-1] != h.delim {
		cut := bytes.IndexByte(rest, h.delim) + 1
		_, werr := h.out.Write(rest[:cut])
		return werr
	}
	return nil
}

// elideBytes copies r except its last n bytes (n > 0).
func (h *Head) elideBytes(r io.Reader, n int64) error {
	// A regular file's length is known, so nothing needs to be buffered.
	if f, ok := r.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.Mode().IsRegular() {
			if pos, err := f.Seek(0, io.SeekCurrent); err == nil {
				if keep := info.Size() - pos - n; keep > 0 {
					_, err := io.CopyN(h.out, f, keep)
					if errors.Is(err, io.EOF) {
						return nil
					}
					return err
				}
				return nil
			}
		}
	}

	var w window
	chunk := make([]byte, bufSize)
	for {
		m, err := r.Read(chunk)
		if m > 0 {
			w.add(chunk[:m])
			if extra := int64(len(w.bytes())) - n; extra > 0 {
				if _, werr := h.out.Write(w.bytes()[:extra]); werr != nil {
					return werr
				}
				w.drop(int(extra))
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// window is a FIFO byte buffer: data is appended at the back and released from
// the front. Released space is reclaimed lazily so a long stream costs
// amortized constant work per byte rather than a copy per chunk.
type window struct {
	buf []byte
	off int
}

func (w *window) bytes() []byte { return w.buf[w.off:] }

func (w *window) add(p []byte) { w.buf = append(w.buf, p...) }

func (w *window) drop(n int) {
	w.off += n
	if w.off > len(w.buf)/2 {
		w.buf = append(w.buf[:0], w.buf[w.off:]...)
		w.off = 0
	}
}

// nthIndex returns the index of the nth (1-based) occurrence of c in b, or -1.
func nthIndex(b []byte, c byte, n int64) int {
	pos := 0
	for {
		i := bytes.IndexByte(b[pos:], c)
		if i < 0 {
			return -1
		}
		pos += i
		n--
		if n == 0 {
			return pos
		}
		pos++
	}
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

// closedPipe reports whether err means the reader of standard output went
// away (head | true), which ends the run quietly rather than as a failure.
func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
