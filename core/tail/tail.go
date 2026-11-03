// Package tail implements the engine behind tailx, a GNU-compatible implementation of
// the GNU/UNIX tail command: print the last part of each file and, with
// --follow, keep printing what is appended to it.
package tail

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "tailx"

// DefaultLines is how many lines tail prints when no count is given.
const DefaultLines = 10

// DefaultSleepInterval is how long follow mode waits between polls.
const DefaultSleepInterval = time.Second

// bufSize is the read chunk, for both forward reads and the backward scan.
const bufSize = 64 << 10

// FollowMode selects how --follow tracks a file.
type FollowMode int

const (
	// FollowNone prints the tail and exits.
	FollowNone FollowMode = iota
	// FollowDescriptor keeps the file open and follows it even if it is
	// renamed or unlinked.
	FollowDescriptor
	// FollowName follows whatever file currently has the name, so it survives
	// log rotation.
	FollowName
)

// Options mirrors the GNU tail flag surface.
type Options struct {
	// Count is the number of lines (or bytes, with Bytes) selected by -n / -c.
	Count int64
	// Bytes (-c) makes Count a byte count instead of a line count.
	Bytes bool
	// FromStart (-n +NUM / -c +NUM) starts output at line or byte Count,
	// counted from 1, instead of printing the last Count.
	FromStart bool
	// Quiet (-q) never prints file-name headers; it wins over Verbose.
	Quiet bool
	// Verbose (-v) always prints file-name headers.
	Verbose bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
	// Follow (-f, --follow) keeps printing data appended to the files.
	Follow FollowMode
	// Retry (--retry) keeps trying to open a file that is, or becomes,
	// inaccessible.
	Retry bool
	// SleepInterval (-s) is the pause between follow polls. Zero means
	// DefaultSleepInterval.
	SleepInterval time.Duration
	// PID (--pid), when positive, ends follow mode once that process dies.
	PID int
	// Color is gnux's terminal view, not a GNU option: the file-name
	// headers in the heading color of Theme (a built-in theme; "" takes
	// GNUX_THEME's). The text is GNU's either way.
	Color bool
	Theme string
}

// Tail is the engine. Construct it with New.
type Tail struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	delim   byte
	headers bool
	last    *target       // whose data was printed last, to know when a header is due
	theme   *visual.Theme // the headers' colors; nil when uncolored
	printed bool          // a header has been written, so the next one needs a blank line
	lost    bool          // a followed file was given up on
	stopped bool          // standard output is gone
}

// target is one operand being followed.
type target struct {
	name    string      // path, as given
	display string      // name used in headers
	f       *os.File    // held open when following by descriptor
	info    os.FileInfo // identity of the file last read, when following by name
	pos     int64       // offset just past the data already printed
	stdin   bool        // f is standard input and not ours to close
	missing bool        // currently cannot be opened
	dead    bool        // given up on
}

// New builds a tail engine reading from and writing to s.
func New(opts Options, s core.Streams) *Tail {
	t := &Tail{
		Options: opts,
		streams: s,
		diag:    core.NewDiag(Name, s),
		delim:   '\n',
	}
	if opts.ZeroTerminated {
		t.delim = 0
	}
	t.theme = visual.Use(opts.Color, opts.Theme, os.Getenv, t.diag.Warnf)
	if t.SleepInterval <= 0 {
		t.SleepInterval = DefaultSleepInterval
	}
	return t
}

// Run is RunContext without a way to stop follow mode other than ending the
// process, which is what the command line wants.
func (t *Tail) Run(args []string) error {
	return t.RunContext(context.Background(), args)
}

// RunContext prints the tail of each operand; no operands, or "-", means
// standard input. With Follow set it then polls the files every SleepInterval
// until ctx is canceled, the --pid process dies or no file is left to follow.
// A failing operand is reported and the rest still run. The returned error is
// core.ErrFailed if any operand failed, otherwise nil.
func (t *Tail) RunContext(ctx context.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"-"}
	}
	following := t.Follow != FollowNone

	if t.Follow == FollowName {
		for _, a := range args {
			if a == "-" {
				t.diag.Errorf("cannot follow '-' by name")
				return t.diag.Result()
			}
		}
	}
	switch {
	case t.Retry && !following:
		t.diag.Warnf("warning: --retry ignored; --retry is useful only when following")
	case t.Retry && t.Follow == FollowDescriptor:
		t.diag.Warnf("warning: --retry only effective for the initial open")
	}
	if t.PID > 0 && !following {
		t.diag.Warnf("warning: PID ignored; --pid=PID is useful only when following")
	}

	t.out = core.NewWriter(t.streams.Out)
	t.headers = !t.Quiet && (t.Verbose || len(args) > 1)

	var targets []*target
	defer func() {
		for _, tg := range targets {
			if tg.f != nil && !tg.stdin {
				_ = tg.f.Close()
			}
		}
	}()

	for _, name := range args {
		if tg := t.initial(name); tg != nil {
			targets = append(targets, tg)
		}
		if t.stopped {
			return t.diag.Result()
		}
	}

	if following {
		t.follow(ctx, targets)
	}
	return t.diag.Result()
}

// initial prints the tail of one operand. In follow mode it returns the target
// to keep polling, or nil if the operand cannot be followed.
func (t *Tail) initial(name string) *target {
	following := t.Follow != FollowNone
	tg := &target{name: name, display: name}

	var r io.Reader
	if name == "-" {
		tg.display = "standard input"
		tg.stdin = true
		r = t.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		f, err := fsys.Open(name)
		if err != nil {
			t.diag.Errorf("cannot open '%s' for reading: %s", name, core.ErrText(err))
			if following && t.Retry {
				tg.missing = true
				return tg
			}
			t.lost = t.lost || following
			return nil
		}
		// Following by descriptor hands the file to the target, which
		// RunContext closes; every other path is done with it here.
		defer func() {
			if tg.f != f {
				_ = f.Close()
			}
		}()
		r = f
	}

	t.header(tg)

	var info os.FileInfo
	f, isFile := r.(*os.File)
	if isFile {
		var err error
		if info, err = f.Stat(); err != nil {
			isFile = false
		}
	}
	// Reading a directory fails differently on every platform; name it up
	// front so the diagnostic is the same everywhere.
	if isFile && info.IsDir() {
		t.flush()
		t.diag.Errorf("error reading '%s': Is a directory", tg.display)
		t.lost = t.lost || following
		return nil
	}

	err := t.tail(r)
	if !t.flush() {
		return nil
	}
	if err != nil {
		t.diag.Errorf("error reading '%s': %s", tg.display, core.ErrText(err))
		t.lost = t.lost || following
		return nil
	}

	// Only regular files are polled: their size says whether there is news
	// without a read that could block. Pipes and terminals were read to end
	// of input above and have nothing more to give.
	if !following || !isFile || !info.Mode().IsRegular() {
		return nil
	}
	pos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil
	}
	tg.pos = pos
	if t.Follow == FollowName {
		tg.info = info
	} else {
		tg.f = f
	}
	return tg
}

// tail writes the selected part of r to the output buffer, leaving r at its
// end.
func (t *Tail) tail(r io.Reader) error {
	f, base, size, seekable := regular(r)

	if t.FromStart {
		// +NUM names the first item to print; +0 and +1 both mean "all".
		skip := max(t.Count-1, 0)
		switch {
		case t.Bytes && seekable:
			if _, err := f.Seek(base+min(skip, size-base), io.SeekStart); err != nil {
				return err
			}
		case t.Bytes:
			if _, err := io.CopyN(io.Discard, r, skip); err != nil {
				return ignoreEOF(err)
			}
		default:
			return t.skipLines(r, skip)
		}
		_, err := io.Copy(t.out, r)
		return err
	}

	switch {
	case t.Count == 0:
		// Nothing to print, but follow mode needs the offset at the end.
		if seekable {
			_, err := f.Seek(0, io.SeekEnd)
			return err
		}
		_, err := io.Copy(io.Discard, r)
		return err
	case seekable:
		start := max(size-t.Count, base)
		if !t.Bytes {
			var err error
			if start, err = lastLines(f, base, size, t.Count, t.delim); err != nil {
				return err
			}
		}
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return err
		}
		_, err := io.Copy(t.out, f)
		return err
	case t.Bytes:
		return t.streamBytes(r, t.Count)
	default:
		return t.streamLines(r, t.Count)
	}
}

// regular reports whether r is a regular file with data past its current
// offset, in which case the tail can be found by seeking instead of reading
// everything. base is the current offset and size the file length. Files that
// claim to be empty (as /proc entries do) are left to the streaming path.
func regular(r io.Reader) (f *os.File, base, size int64, ok bool) {
	f, ok = r.(*os.File)
	if !ok {
		return nil, 0, 0, false
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, 0, 0, false
	}
	base, err = f.Seek(0, io.SeekCurrent)
	if err != nil || info.Size() <= base {
		return nil, 0, 0, false
	}
	return f, base, info.Size(), true
}

// lastLines returns the offset at which the last n lines of r[base:size]
// begin, scanning backward from the end so only the tail of a large file is
// read. A delimiter ending the file terminates the last line rather than
// starting an empty one.
func lastLines(r io.ReaderAt, base, size, n int64, delim byte) (int64, error) {
	buf := make([]byte, bufSize)
	pos := size
	for pos > base {
		chunk := min(int64(len(buf)), pos-base)
		b := buf[:chunk]
		if _, err := r.ReadAt(b, pos-chunk); err != nil {
			return 0, err
		}
		if pos == size && b[chunk-1] == delim && n < math.MaxInt64 {
			n++
		}
		pos -= chunk

		for i := len(b); ; {
			i = bytes.LastIndexByte(b[:i], delim)
			if i < 0 {
				break
			}
			if n--; n == 0 {
				return pos + int64(i) + 1, nil
			}
		}
	}
	return base, nil
}

// skipLines discards the first n lines of r and copies the rest.
func (t *Tail) skipLines(r io.Reader, n int64) error {
	buf := make([]byte, bufSize)
	for n > 0 {
		m, err := r.Read(buf)
		b := buf[:m]
		for n > 0 && len(b) > 0 {
			i := bytes.IndexByte(b, t.delim)
			if i < 0 {
				b = nil
				break
			}
			b = b[i+1:]
			n--
		}
		if n == 0 {
			if _, werr := t.out.Write(b); werr != nil {
				return werr
			}
		}
		if err != nil {
			return ignoreEOF(err)
		}
	}
	_, err := io.Copy(t.out, r)
	return err
}

// streamLines prints the last n lines of an input that cannot seek, holding
// only about that many lines in memory.
func (t *Tail) streamLines(r io.Reader, n int64) error {
	var (
		w     window
		held  int64 // delimiters currently in the window
		chunk = make([]byte, bufSize)
	)
	for {
		m, err := r.Read(chunk)
		if m > 0 {
			w.add(chunk[:m])
			held += int64(bytes.Count(chunk[:m], []byte{t.delim}))
			// n+1 delimiters always cover the last n lines, whether or not
			// the input ends with one; the exact cut is made at end of input.
			if held > n && held-n > 1 {
				extra := held - n - 1
				w.drop(nthIndex(w.bytes(), t.delim, extra) + 1)
				held -= extra
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
	}

	rest := w.bytes()
	start, err := lastLines(bytes.NewReader(rest), 0, int64(len(rest)), n, t.delim)
	if err != nil {
		return err
	}
	_, err = t.out.Write(rest[start:])
	return err
}

// streamBytes prints the last n bytes of an input that cannot seek.
func (t *Tail) streamBytes(r io.Reader, n int64) error {
	var w window
	chunk := make([]byte, bufSize)
	for {
		m, err := r.Read(chunk)
		if m > 0 {
			w.add(chunk[:m])
			if extra := int64(len(w.bytes())) - n; extra > 0 {
				w.drop(int(extra))
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
	}
	_, err := t.out.Write(w.bytes())
	return err
}

// follow polls targets until ctx is canceled, the watched process dies, no
// target is left, or standard output goes away.
func (t *Tail) follow(ctx context.Context, targets []*target) {
	timer := time.NewTimer(t.SleepInterval)
	defer timer.Stop()

	for {
		if !anyLive(targets) {
			// Operands that were never followable (a pipe on standard
			// input) end quietly; ones that failed are worth a word.
			if t.lost {
				t.diag.Errorf("no files remaining")
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		// Sample the process before the poll so that whatever it wrote just
		// before dying is still printed.
		exiting := t.PID > 0 && !processAlive(t.PID)
		for _, tg := range targets {
			if tg.dead {
				continue
			}
			t.poll(tg)
			if t.stopped {
				return
			}
		}
		if exiting {
			return
		}
		timer.Reset(t.SleepInterval)
	}
}

func anyLive(targets []*target) bool {
	for _, tg := range targets {
		if !tg.dead {
			return true
		}
	}
	return false
}

// poll prints whatever tg has gained since the last poll.
func (t *Tail) poll(tg *target) {
	if t.Follow == FollowName {
		t.pollName(tg)
		return
	}

	if tg.f == nil {
		// --retry: the initial open failed; keep trying.
		f, err := fsys.Open(tg.name)
		if err != nil {
			return
		}
		t.diag.Warnf("'%s' has appeared;  following new file", tg.name)
		tg.f, tg.missing, tg.pos = f, false, 0
	}
	info, err := tg.f.Stat()
	if err != nil {
		t.giveUp(tg, err)
		return
	}
	t.drain(tg, tg.f, info.Size())
}

// pollName follows by name. The file is reopened on every poll and never held
// open in between, so it can be renamed or deleted under us on every platform
// (an open handle blocks both on Windows) and a replacement is noticed by
// comparing identities.
func (t *Tail) pollName(tg *target) {
	f, err := fsys.Open(tg.name)
	var info os.FileInfo
	if err == nil {
		defer f.Close()
		info, err = f.Stat()
	}
	if err != nil {
		if busy(err) && !tg.missing {
			return // held by another process for now: look again next poll
		}
		if !tg.missing {
			t.diag.Warnf("'%s' has become inaccessible: %s", tg.name, core.ErrText(err))
			tg.missing = true
		}
		if !t.Retry {
			tg.dead, t.lost = true, true
		}
		return
	}

	switch {
	case tg.missing:
		t.diag.Warnf("'%s' has appeared;  following new file", tg.name)
		tg.pos = 0
	case !os.SameFile(tg.info, info):
		t.diag.Warnf("'%s' has been replaced;  following new file", tg.name)
		tg.pos = 0
	}
	tg.missing, tg.info = false, info

	if info.Mode().IsRegular() {
		t.drain(tg, f, info.Size())
	}
}

// drain prints f from tg.pos to its current size.
func (t *Tail) drain(tg *target, f *os.File, size int64) {
	if size < tg.pos {
		t.diag.Warnf("%s: file truncated", tg.name)
		tg.pos = 0
	}
	if size == tg.pos {
		return
	}
	if _, err := f.Seek(tg.pos, io.SeekStart); err != nil {
		t.giveUp(tg, err)
		return
	}

	t.header(tg)
	n, err := io.Copy(t.out, f)
	tg.pos += n
	if !t.flush() {
		return
	}
	if err != nil {
		t.giveUp(tg, err)
	}
}

// giveUp reports a read failure on a followed file and stops following it.
func (t *Tail) giveUp(tg *target, err error) {
	t.diag.Errorf("error reading '%s': %s", tg.display, core.ErrText(err))
	tg.dead, t.lost = true, true
}

// header announces tg before its data, when headers are on and the previous
// output belonged to another file.
func (t *Tail) header(tg *target) {
	if !t.headers || t.last == tg {
		return
	}
	if t.printed {
		_ = t.out.WriteByte('\n')
	}
	fmt.Fprintf(t.out, "%s\n", t.theme.Paint(visual.Heading, "==> "+tg.display+" <=="))
	t.printed, t.last = true, tg
}

// flush pushes buffered output to standard output. It returns false, and
// marks the run stopped, when standard output is gone. bufio.Writer errors are
// sticky, so a failed flush is how any earlier write error is recognized.
func (t *Tail) flush() bool {
	err := t.out.Flush()
	if err == nil {
		return true
	}
	if !t.stopped && !closedPipe(err) {
		t.diag.Errorf("write error: %s", core.ErrText(err))
	}
	t.stopped = true
	return false
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

func ignoreEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// closedPipe reports whether err means the reader of standard output went
// away (tail | true), which ends the run quietly rather than as a failure.
func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
