// Package wc implements the engine behind wcx, a GNU-compatible implementation of the
// GNU/UNIX wc command: print line, word, character and byte counts for each
// file.
package wc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "wcx"

// bufSize is the read chunk. Counting newlines is a memchr-style scan, so a
// large buffer keeps the cost in the scan rather than in system calls.
const bufSize = 256 << 10

// pipeWidth is the column width used when an input's size cannot be known up
// front (a pipe, a terminal), as in GNU wc.
const pipeWidth = 7

// TotalMode says when the "total" line is printed (--total=WHEN).
type TotalMode int

const (
	// TotalAuto prints a total when there is more than one operand.
	TotalAuto TotalMode = iota
	// TotalAlways prints a total even for a single operand.
	TotalAlways
	// TotalOnly prints nothing but the total, without a label.
	TotalOnly
	// TotalNever never prints a total.
	TotalNever
)

// Options mirrors the GNU wc flag surface. With no counter selected, wc
// reports lines, words and bytes.
type Options struct {
	// Lines (-l) counts newline characters.
	Lines bool
	// Words (-w) counts whitespace-separated words.
	Words bool
	// Chars (-m) counts characters, decoding the input as UTF-8.
	Chars bool
	// Bytes (-c) counts bytes.
	Bytes bool
	// MaxLineLength (-L) reports the display width of the widest line.
	MaxLineLength bool
	// Files0From (--files0-from=F), when set, reads the operands as
	// NUL-terminated names from file F ("-" for standard input).
	Files0From string
	// Total (--total=WHEN) controls the summary line.
	Total TotalMode
}

// counts holds the tallies for one input, or the running total.
type counts struct {
	lines, words, chars, bytes, maxLen int64
}

func (c *counts) add(o counts) {
	c.lines += o.lines
	c.words += o.words
	c.chars += o.chars
	c.bytes += o.bytes
	c.maxLen = max(c.maxLen, o.maxLen)
}

// Wc is the engine. Construct it with New.
type Wc struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	width   int
	buf     []byte
}

// New builds a wc engine reading from and writing to s.
func New(opts Options, s core.Streams) *Wc {
	if !opts.Lines && !opts.Words && !opts.Chars && !opts.Bytes && !opts.MaxLineLength {
		opts.Lines, opts.Words, opts.Bytes = true, true, true
	}
	return &Wc{
		Options: opts,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run counts each operand; no operands, or "-", means standard input. A
// failing operand is reported and the rest still run. The returned error is
// core.ErrFailed if any operand failed, otherwise nil.
func (w *Wc) Run(args []string) error {
	w.out = core.NewWriter(w.streams.Out)

	// A nil name is standard input given implicitly, which prints no name.
	var names []*string
	switch {
	case w.Files0From != "":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'\nfile operands cannot be combined with --files0-from", args[0])
		}
		list, ok := w.readNames()
		if !ok {
			return w.diag.Result()
		}
		for i := range list {
			names = append(names, &list[i])
		}
	case len(args) == 0:
		names = []*string{nil}
	default:
		for i := range args {
			names = append(names, &args[i])
		}
	}

	w.width = w.numberWidth(names)

	var total counts
	for _, name := range names {
		c, ok := w.file(name)
		if !ok {
			continue
		}
		total.add(c)
		if w.Total != TotalOnly {
			w.print(c, name)
		}
		// Keep counts and diagnostics in operand order on a shared terminal.
		if !w.flush() {
			return w.diag.Result()
		}
	}

	switch w.Total {
	case TotalAuto:
		if len(names) > 1 {
			w.print(total, ptr("total"))
		}
	case TotalAlways:
		w.print(total, ptr("total"))
	case TotalOnly:
		w.print(total, nil)
	case TotalNever:
	}
	w.flush()
	return w.diag.Result()
}

func ptr(s string) *string { return &s }

// readNames loads the NUL-separated operand list named by --files0-from.
func (w *Wc) readNames() ([]string, bool) {
	var (
		data []byte
		err  error
	)
	fromStdin := w.Files0From == "-"
	if fromStdin {
		if w.streams.In != nil {
			data, err = io.ReadAll(w.streams.In)
		}
	} else {
		data, err = fsys.ReadFile(w.Files0From)
	}
	if err != nil {
		if fromStdin {
			w.diag.Errorf("-: read error: %s", core.ErrText(err))
		} else {
			w.diag.Errorf("cannot open '%s' for reading: %s", w.Files0From, core.ErrText(err))
		}
		return nil, false
	}

	// A terminating NUL ends the last name; it does not start an empty one.
	data = bytes.TrimSuffix(data, []byte{0})
	if len(data) == 0 {
		return nil, true
	}

	var names []string
	for i, name := range strings.Split(string(data), "\x00") {
		switch {
		case name == "":
			w.diag.Errorf("%s:%d: invalid zero-length file name", w.Files0From, i+1)
		case name == "-" && fromStdin:
			w.diag.Errorf("when reading file names from stdin, no file name of '-' allowed")
		default:
			names = append(names, name)
		}
	}
	return names, true
}

// stdinFile returns standard input as a file, if that is what it is.
func (w *Wc) stdinFile() (*os.File, bool) {
	f, ok := w.streams.In.(*os.File)
	return f, ok
}

// numberWidth picks the column width the way GNU wc does, before anything is
// read: wide enough for the combined size of the regular-file operands (no
// count can exceed it), at least pipeWidth as soon as one operand has no
// knowable size, and 1 when a lone counter of a lone operand cannot misalign.
func (w *Wc) numberWidth(names []*string) int {
	counters := 0
	for _, on := range []bool{w.Lines, w.Words, w.Chars, w.Bytes, w.MaxLineLength} {
		if on {
			counters++
		}
	}
	// GNU skips the size probe for --files0-from and --total=only too.
	if w.Files0From != "" || w.Total == TotalOnly || len(names) == 0 ||
		(len(names) == 1 && counters == 1) {
		return 1
	}

	minimum := 1
	var total int64
	for _, name := range names {
		info, err := w.stat(name)
		if err != nil {
			// An operand that cannot be read adds nothing: it is reported,
			// and the estimate covers the others.
			continue
		}
		if info == nil || !info.Mode().IsRegular() {
			minimum = pipeWidth
			continue
		}
		total += info.Size()
	}
	return max(minimum, len(strconv.FormatInt(total, 10)))
}

// stat describes an operand without opening it. A nil FileInfo with a nil
// error means an input that is not a file at all (an in-memory reader).
func (w *Wc) stat(name *string) (os.FileInfo, error) {
	if name == nil || *name == "-" {
		if f, ok := w.stdinFile(); ok {
			return f.Stat()
		}
		return nil, nil
	}
	return fsys.Stat(*name)
}

// file counts one operand. ok is false when there is nothing to print for it.
func (w *Wc) file(name *string) (c counts, ok bool) {
	var r io.Reader
	display := "-"
	if name != nil {
		display = *name
	}

	if name == nil || *name == "-" {
		r = w.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		f, err := fsys.Open(*name)
		if err != nil {
			w.diag.Errorf("%s: %s", display, core.ErrText(err))
			return c, false
		}
		defer f.Close()
		r = f
	}

	c, err := w.count(r)
	if err != nil {
		// GNU still prints the partial counts after a read error.
		w.diag.Errorf("%s: %s", display, core.ErrText(err))
	}
	return c, true
}

// count tallies r, choosing the cheapest strategy the selected counters allow.
func (w *Wc) count(r io.Reader) (counts, error) {
	var c counts

	f, isFile := r.(*os.File)
	if isFile {
		info, err := f.Stat()
		switch {
		case err != nil:
		case info.IsDir():
			// Reading a directory fails differently on every platform; name
			// it up front so the diagnostic is the same everywhere.
			return c, errIsDir
		case w.bytesOnly() && info.Mode().IsRegular():
			// -c alone needs no data at all: the size says it. Files that
			// claim to be empty (as /proc entries do) are read anyway.
			if pos, err := f.Seek(0, io.SeekCurrent); err == nil && info.Size() > pos {
				c.bytes = info.Size() - pos
				_, err = f.Seek(0, io.SeekEnd)
				return c, err
			}
		}
	}

	if w.buf == nil {
		w.buf = make([]byte, bufSize)
	}
	if w.Words || w.Chars || w.MaxLineLength {
		return w.countText(r)
	}
	return w.countLines(r)
}

func (w *Wc) bytesOnly() bool {
	return w.Bytes && !w.Lines && !w.Words && !w.Chars && !w.MaxLineLength
}

// countLines handles -l and -c without decoding: one newline scan per chunk.
func (w *Wc) countLines(r io.Reader) (counts, error) {
	var c counts
	newline := []byte{'\n'}
	for {
		n, err := r.Read(w.buf)
		c.bytes += int64(n)
		if w.Lines {
			c.lines += int64(bytes.Count(w.buf[:n], newline))
		}
		if err != nil {
			return c, ignoreEOF(err)
		}
	}
}

// countText is the full pass: it decodes UTF-8 to count characters, words and
// line widths. A multi-byte sequence split across two reads is carried over to
// the next chunk rather than miscounted.
func (w *Wc) countText(r io.Reader) (counts, error) {
	var (
		c       counts
		inWord  bool
		linePos int64
		carry   int
	)
	for {
		n, err := r.Read(w.buf[carry:])
		c.bytes += int64(n)
		data := w.buf[:carry+n]
		atEOF := err != nil

		i := 0
		for i < len(data) {
			b := data[i]
			ch, size := rune(b), 1
			if b >= utf8.RuneSelf {
				if !atEOF && !utf8.FullRune(data[i:]) {
					break
				}
				ch, size = utf8.DecodeRune(data[i:])
				if ch == utf8.RuneError && size == 1 {
					// An undecodable byte is not a character, but it is
					// not a separator either: it belongs to a word.
					if !inWord {
						c.words++
						inWord = true
					}
					i++
					continue
				}
			}
			i += size
			c.chars++

			switch ch {
			case '\n':
				c.lines++
				c.maxLen = max(c.maxLen, linePos)
				linePos = 0
				inWord = false
			case '\r', '\f':
				c.maxLen = max(c.maxLen, linePos)
				linePos = 0
				inWord = false
			case '\t':
				linePos += 8 - linePos%8
				inWord = false
			case ' ':
				linePos++
				inWord = false
			case '\v':
				inWord = false
			default:
				if ch >= utf8.RuneSelf && unicode.IsSpace(ch) {
					linePos += int64(runeWidth(ch))
					inWord = false
					break
				}
				if !inWord {
					c.words++
					inWord = true
				}
				linePos += int64(runeWidth(ch))
			}
		}
		carry = copy(w.buf, data[i:])

		if err != nil {
			c.maxLen = max(c.maxLen, linePos)
			return c, ignoreEOF(err)
		}
	}
}

// wide lists the East Asian wide and fullwidth blocks, which occupy two
// terminal columns. The standard library has no width table; these ranges
// cover the scripts and emoji that matter for -L.
var wide = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x1100, Hi: 0x115f, Stride: 1},
		{Lo: 0x2e80, Hi: 0x303e, Stride: 1},
		{Lo: 0x3041, Hi: 0xa4cf, Stride: 1},
		{Lo: 0xac00, Hi: 0xd7a3, Stride: 1},
		{Lo: 0xf900, Hi: 0xfaff, Stride: 1},
		{Lo: 0xfe30, Hi: 0xfe6f, Stride: 1},
		{Lo: 0xff00, Hi: 0xff60, Stride: 1},
		{Lo: 0xffe0, Hi: 0xffe6, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x1f300, Hi: 0x1f64f, Stride: 1},
		{Lo: 0x1f900, Hi: 0x1f9ff, Stride: 1},
		{Lo: 0x20000, Hi: 0x3fffd, Stride: 1},
	},
}

// runeWidth approximates wcwidth(3): how many terminal columns ch occupies.
func runeWidth(ch rune) int {
	switch {
	case ch < 0x20, ch >= 0x7f && ch < 0xa0:
		return 0
	case ch < utf8.RuneSelf:
		return 1
	case unicode.In(ch, unicode.Mn, unicode.Me, unicode.Cf):
		return 0
	case unicode.Is(wide, ch):
		return 2
	case unicode.IsPrint(ch), unicode.IsSpace(ch):
		return 1
	default:
		return 0
	}
}

// print writes one result line: the selected counts in GNU's fixed order,
// right-aligned to the common width, then the name if there is one.
func (w *Wc) print(c counts, name *string) {
	fields := []struct {
		on bool
		n  int64
	}{
		{w.Lines, c.lines},
		{w.Words, c.words},
		{w.Chars, c.chars},
		{w.Bytes, c.bytes},
		{w.MaxLineLength, c.maxLen},
	}
	first := true
	for _, f := range fields {
		if !f.on {
			continue
		}
		if !first {
			_ = w.out.WriteByte(' ')
		}
		first = false
		fmt.Fprintf(w.out, "%*d", w.width, f.n)
	}
	if name != nil {
		_ = w.out.WriteByte(' ')
		_, _ = w.out.WriteString(*name)
	}
	_ = w.out.WriteByte('\n')
}

// flush pushes buffered output to standard output, returning false when
// standard output is gone. bufio.Writer errors are sticky, so a failed flush
// is how any earlier write error is recognized.
func (w *Wc) flush() bool {
	err := w.out.Flush()
	if err == nil {
		return true
	}
	if !closedPipe(err) {
		w.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return false
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

func ignoreEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// closedPipe reports whether err means the reader of standard output went
// away (wc | true), which ends the run quietly rather than as a failure.
func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
