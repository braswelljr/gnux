// Package dd implements the engine behind ddx, a GNU-compatible
// implementation of the GNU/UNIX dd command: copy a file block by block,
// converting it on the way, and report what was copied.
package dd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "ddx"

// defaultBlockSize is GNU dd's input and output block size.
const defaultBlockSize = 512

// StatusLevel says what dd reports on standard error (status=LEVEL).
type StatusLevel int

const (
	// StatusDefault prints the record counts and the transfer statistics.
	StatusDefault StatusLevel = iota
	// StatusNone prints nothing but error messages.
	StatusNone
	// StatusNoXfer prints the record counts only.
	StatusNoXfer
	// StatusProgress is accepted; gnux prints the final statistics only.
	StatusProgress
)

// Options mirrors GNU dd's operands. The zero value copies standard input to
// standard output in 512-byte blocks.
type Options struct {
	// Input (if=FILE) and Output (of=FILE); empty means the standard streams.
	Input, Output string

	// BlockSize (bs=N) sets both block sizes and writes each input block as
	// read; InputBlockSize (ibs=N) and OutputBlockSize (obs=N) set one each.
	// Zero means 512.
	BlockSize, InputBlockSize, OutputBlockSize int64
	// ConversionBlockSize (cbs=N) is the record length for block/unblock.
	ConversionBlockSize int64

	// Skip (skip=N) input blocks, and Seek (seek=N) output blocks, before
	// copying; Count (count=N) copies only N input blocks when HasCount.
	Skip, Seek, Count int64
	HasCount          bool

	// Status (status=LEVEL) is what is reported on standard error.
	Status StatusLevel

	// Conversions (conv=...).
	NoTrunc, NoError, Sync, Fsync, Fdatasync, Excl, NoCreat bool
	LCase, UCase, Swab, Block, Unblock                      bool

	// Input flags (iflag=...): FullBlock accumulates full input blocks,
	// CountBytes and SkipBytes read count= and skip= as bytes.
	FullBlock, CountBytes, SkipBytes bool
	// Output flags (oflag=...): SeekBytes reads seek= as bytes, Append
	// appends to the output.
	SeekBytes, Append bool
}

// Dd is the engine. Construct it with New.
type Dd struct {
	Options

	// Now is the clock the transfer statistics are timed with; it defaults
	// to time.Now.
	Now func() time.Time

	streams core.Streams
	diag    *core.Diag
}

// New builds a dd engine reading from and writing to s.
func New(opts Options, s core.Streams) *Dd {
	return &Dd{Options: opts, Now: time.Now, streams: s, diag: core.NewDiag(Name, s)}
}

// stats are the counts reported at the end.
type stats struct {
	inFull, inPart, outFull, outPart, truncated, bytes int64
}

// Run copies the input to the output. Operands (KEY=VALUE), when given,
// replace the engine's Options, as on the command line. The returned error
// is core.ErrFailed if anything failed, otherwise nil.
func (d *Dd) Run(args []string) error {
	if len(args) > 0 {
		o, err := Parse(args)
		var perr *ParseError
		if errors.As(err, &perr) {
			d.diag.Errorf("%s", perr.Msg)
			return core.ErrFailed
		}
		if err != nil {
			return err
		}
		d.Options = o
	}

	ibs, obs := d.InputBlockSize, d.OutputBlockSize
	if ibs == 0 {
		ibs = defaultBlockSize
	}
	if obs == 0 {
		obs = defaultBlockSize
	}
	if d.BlockSize > 0 {
		ibs, obs = d.BlockSize, d.BlockSize
	}

	in, inName, closeIn, ok := d.openInput()
	if !ok {
		return d.diag.Result()
	}
	defer closeIn()
	out, outName, closeOut, ok := d.openOutput(obs)
	if !ok {
		return d.diag.Result()
	}
	defer closeOut()

	start := d.Now()
	c := &copier{
		d: d, in: in, out: out, inName: inName, outName: outName,
		ibs: ibs, obs: obs,
		direct: d.BlockSize > 0 && !d.Block && !d.Unblock,
	}
	d.skip(in, inName, ibs)
	quiet := c.run()
	if quiet {
		return d.diag.Result()
	}
	if (d.Fsync || d.Fdatasync) && !c.failed {
		if f, ok := out.(*os.File); ok {
			if err := f.Sync(); err != nil && !errors.Is(err, os.ErrInvalid) {
				d.diag.Errorf("fsync failed for '%s': %s", outName, core.ErrText(err))
			}
		}
	}
	d.report(c.st, d.Now().Sub(start))
	return d.diag.Result()
}

func (d *Dd) openInput() (io.Reader, string, func(), bool) {
	if d.Input == "" {
		return d.streams.In, "standard input", func() {}, true
	}
	f, err := os.Open(d.Input)
	if err != nil {
		d.diag.Errorf("failed to open '%s': %s", d.Input, core.ErrText(err))
		return nil, "", nil, false
	}
	return f, d.Input, func() { f.Close() }, true
}

func (d *Dd) openOutput(obs int64) (io.Writer, string, func(), bool) {
	seek := d.Seek
	if !d.SeekBytes {
		seek *= obs
	}
	if d.Output == "" {
		out := d.streams.Out
		d.seekOutput(out, "standard output", seek)
		return out, "standard output", func() {}, true
	}

	flags := os.O_WRONLY
	if !d.NoCreat {
		flags |= os.O_CREATE
	}
	if d.Excl {
		flags |= os.O_EXCL
	}
	if d.Append {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(d.Output, flags, 0o666)
	if err != nil {
		d.diag.Errorf("failed to open '%s': %s", d.Output, core.ErrText(err))
		return nil, "", nil, false
	}
	if !d.NoTrunc && !d.Append {
		if info, err := f.Stat(); err == nil && info.Mode().IsRegular() {
			if err := f.Truncate(seek); err != nil {
				d.diag.Errorf("failed to truncate to %d bytes in output file '%s': %s", seek, d.Output, core.ErrText(err))
				f.Close()
				return nil, "", nil, false
			}
		}
	}
	if !d.Append {
		d.seekOutput(f, d.Output, seek)
	}
	return f, d.Output, func() { f.Close() }, true
}

// seekOutput moves past n bytes of the output, writing NUL bytes where the
// output cannot seek (a pipe), as GNU dd does.
func (d *Dd) seekOutput(w io.Writer, name string, n int64) {
	if n == 0 {
		return
	}
	if s, ok := w.(io.Seeker); ok {
		if _, err := s.Seek(n, io.SeekCurrent); err == nil {
			return
		}
	}
	if _, err := io.CopyN(w, zeroReader{}, n); err != nil {
		d.diag.Errorf("'%s': cannot seek: %s", name, core.ErrText(err))
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// skip moves past the skip= amount of input: by seeking when the input can,
// otherwise by reading and discarding.
func (d *Dd) skip(in io.Reader, name string, ibs int64) {
	n := d.Skip
	if !d.SkipBytes {
		n *= ibs
	}
	if n == 0 {
		return
	}
	if f, ok := in.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.Mode().IsRegular() {
			if _, err := f.Seek(n, io.SeekCurrent); err == nil {
				if n > info.Size() {
					d.diag.Warnf("'%s': cannot skip to specified offset", name)
				}
				return
			}
		}
	}
	if got, err := io.CopyN(io.Discard, in, n); got < n {
		if err != nil && err != io.EOF {
			d.diag.Errorf("'%s': cannot skip: %s", name, core.ErrText(err))
			return
		}
		d.diag.Warnf("'%s': cannot skip to specified offset", name)
	}
}

type copier struct {
	d               *Dd
	in              io.Reader
	out             io.Writer
	inName, outName string
	ibs, obs        int64
	direct          bool

	st     stats
	obuf   []byte
	col    int64 // block/unblock: bytes in the current record
	rec    []byte
	failed bool
}

// run copies until the input or the count runs out. It reports true when
// the output pipe closed, which ends dd quietly.
func (c *copier) run() (quiet bool) {
	buf := make([]byte, c.ibs)
	remaining := c.d.Count
	for {
		want := c.ibs
		if c.d.HasCount {
			if c.d.CountBytes {
				if remaining <= 0 {
					break
				}
				want = min(want, remaining)
			} else if c.st.inFull+c.st.inPart >= c.d.Count {
				break
			}
		}

		n, err := c.read(buf[:want])
		if n > 0 {
			if int64(n) == c.ibs {
				c.st.inFull++
			} else {
				c.st.inPart++
			}
			remaining -= int64(n)
		}
		data := buf[:n]
		if n > 0 && int64(n) < c.ibs && c.d.Sync {
			pad := byte(0)
			if c.d.Block || c.d.Unblock {
				pad = ' '
			}
			for i := n; i < len(buf); i++ {
				buf[i] = pad
			}
			data = buf
		}
		if len(data) > 0 {
			if closed, ok := c.write(c.convert(data)); closed {
				return true
			} else if !ok {
				break
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			c.d.diag.Errorf("error reading '%s': %s", c.inName, core.ErrText(err))
			c.failed = true
			if !c.d.NoError || !c.skipBad(n) {
				break
			}
		} else if n == 0 {
			break
		}
	}
	if closed, _ := c.flush(); closed {
		return true
	}
	return false
}

// read fills p with one input block: a single read, or with
// iflag=fullblock as many reads as it takes.
func (c *copier) read(p []byte) (int, error) {
	if !c.d.FullBlock {
		for {
			n, err := c.in.Read(p)
			if n > 0 || err != nil {
				return n, err
			}
		}
	}
	n, err := io.ReadFull(c.in, p)
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}

// skipBad moves past the rest of a block that failed to read, for
// conv=noerror; it reports whether reading can go on.
func (c *copier) skipBad(n int) bool {
	s, ok := c.in.(io.Seeker)
	if !ok {
		return false
	}
	pos, err := s.Seek(c.ibs-int64(n), io.SeekCurrent)
	if err != nil {
		return false
	}
	if f, ok := c.in.(*os.File); ok {
		if info, err := f.Stat(); err == nil && pos >= info.Size() {
			return false
		}
	}
	return true
}

// convert applies conv=swab, lcase/ucase and block/unblock, in GNU's order.
func (c *copier) convert(p []byte) []byte {
	if c.d.Swab {
		for i := 0; i+1 < len(p); i += 2 {
			p[i], p[i+1] = p[i+1], p[i]
		}
	}
	switch {
	case c.d.LCase:
		p = bytes.ToLower(p)
	case c.d.UCase:
		p = bytes.ToUpper(p)
	}
	cbs := c.d.ConversionBlockSize
	switch {
	case c.d.Block && cbs > 0:
		var out []byte
		for _, b := range p {
			if b == '\n' {
				out = append(out, bytes.Repeat([]byte{' '}, int(cbs-min(c.col, cbs)))...)
				c.col = 0
				continue
			}
			if c.col < cbs {
				out = append(out, b)
			} else if c.col == cbs {
				c.st.truncated++
			}
			c.col++
		}
		return out
	case c.d.Unblock && cbs > 0:
		var out []byte
		for _, b := range p {
			c.rec = append(c.rec, b)
			if int64(len(c.rec)) == cbs {
				out = append(out, bytes.TrimRight(c.rec, " ")...)
				out = append(out, '\n')
				c.rec = c.rec[:0]
			}
		}
		return out
	}
	return p
}

// write sends converted data to the output: as one record when bs= was
// given, otherwise through an obs-sized buffer.
func (c *copier) write(p []byte) (closed, ok bool) {
	if c.direct {
		return c.emit(p)
	}
	c.obuf = append(c.obuf, p...)
	for int64(len(c.obuf)) >= c.obs {
		if closed, ok := c.emit(c.obuf[:c.obs]); !ok {
			return closed, false
		}
		c.obuf = c.obuf[c.obs:]
	}
	return false, true
}

// flush writes what block/unblock and the output buffer still hold.
func (c *copier) flush() (closed, ok bool) {
	cbs := c.d.ConversionBlockSize
	var tail []byte
	switch {
	case c.d.Block && cbs > 0 && c.col > 0:
		tail = bytes.Repeat([]byte{' '}, int(cbs-min(c.col, cbs)))
	case c.d.Unblock && cbs > 0 && len(c.rec) > 0:
		tail = append(bytes.TrimRight(c.rec, " "), '\n')
	}
	if len(tail) > 0 && !c.failed {
		if closed, ok := c.write(tail); !ok {
			return closed, false
		}
	}
	if len(c.obuf) > 0 {
		return c.emit(c.obuf)
	}
	return false, true
}

// emit writes one output record and counts it.
func (c *copier) emit(p []byte) (closed, ok bool) {
	if len(p) == 0 {
		return false, true
	}
	n, err := c.out.Write(p)
	c.st.bytes += int64(n)
	if err != nil {
		if core.IsClosedPipe(err) {
			return true, false
		}
		c.d.diag.Errorf("error writing '%s': %s", c.outName, core.ErrText(err))
		c.failed = true
		if n > 0 {
			c.st.outPart++
		}
		return false, false
	}
	if int64(n) == c.obs {
		c.st.outFull++
	} else {
		c.st.outPart++
	}
	return false, true
}

// report writes the statistics GNU dd prints on standard error.
func (d *Dd) report(st stats, elapsed time.Duration) {
	if d.Status == StatusNone {
		return
	}
	w := d.streams.Err
	fmt.Fprintf(w, "%d+%d records in\n%d+%d records out\n", st.inFull, st.inPart, st.outFull, st.outPart)
	if st.truncated > 0 {
		fmt.Fprintf(w, "%d truncated record%s\n", st.truncated, core.Plural(int(st.truncated)))
	}
	if d.Status == StatusNoXfer {
		return
	}

	si, iec := human(float64(st.bytes), 1000, siUnits), human(float64(st.bytes), 1024, iecUnits)
	switch {
	case st.bytes < 1000:
		fmt.Fprintf(w, "%d byte%s copied", st.bytes, core.Plural(int(st.bytes)))
	case st.bytes < 1024:
		fmt.Fprintf(w, "%d bytes (%s) copied", st.bytes, si)
	default:
		fmt.Fprintf(w, "%d bytes (%s, %s) copied", st.bytes, si, iec)
	}
	secs := elapsed.Seconds()
	rate := "Infinity B"
	if secs > 0 {
		rate = human(float64(st.bytes)/secs, 1000, siUnits)
	}
	fmt.Fprintf(w, ", %g s, %s/s\n", secs, rate)
}

var (
	siUnits  = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	iecUnits = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

// human renders v with the largest unit that keeps it at or above 1: one
// decimal below 10, a whole number otherwise ("5.1 MB", "852 MB", "512 B").
func human(v float64, base float64, units []string) string {
	u := 0
	for v >= base && u < len(units)-1 {
		v /= base
		u++
	}
	if u > 0 && math.Round(v*10)/10 < 10 {
		return fmt.Sprintf("%.1f %s", math.Round(v*10)/10, units[u])
	}
	return fmt.Sprintf("%.0f %s", math.Round(v), units[u])
}
