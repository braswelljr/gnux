// Package split implements the engine behind splitx, a GNU-compatible
// implementation of the GNU/UNIX split command: split a file into pieces.
package split

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "splitx"

// Mode is how the input is divided.
type Mode int

const (
	// Lines (-l) puts Count lines in each file; it is the default, with 1000
	// lines.
	Lines Mode = iota
	// Bytes (-b) puts Count bytes in each file.
	Bytes
	// LineBytes (-C) puts as many whole lines as fit in Count bytes in each
	// file, splitting only lines longer than that.
	LineBytes
	// ChunkBytes (-n N) makes Count files of (nearly) equal size.
	ChunkBytes
	// ChunkLines (-n l/N) makes Count files of nearly equal size without
	// splitting lines: each line goes where it starts.
	ChunkLines
	// RoundRobin (-n r/N) deals lines out to Count files in turn.
	RoundRobin
)

// Alphabet is the set of characters suffixes are made of.
type Alphabet int

const (
	// Alphabetic suffixes: aa, ab, ... (the default).
	Alphabetic Alphabet = iota
	// Numeric suffixes (-d): 00, 01, ...
	Numeric
	// Hex suffixes (-x): 00, 01, ... 0f, 10, ...
	Hex
)

var alphabets = [...]string{
	Alphabetic: "abcdefghijklmnopqrstuvwxyz",
	Numeric:    "0123456789",
	Hex:        "0123456789abcdef",
}

// DefaultLines is the number of lines per file when Count is zero in Lines
// mode.
const DefaultLines = 1000

// Options mirrors the GNU split flag surface. The zero value is plain split:
// 1000 lines per file, named xaa, xab, ...
type Options struct {
	Mode Mode
	// Count is the lines or bytes per file, or the number of files for the
	// chunk modes. Zero means DefaultLines in Lines mode.
	Count int64
	// Chunk (-n K/N) writes only chunk K (1-based) to standard output instead
	// of creating files; zero means create every file.
	Chunk int64
	// SuffixLength (-a) fixes the suffix length. Zero means 2, growing as
	// needed (zz is followed by zaaa) so the names still sort in order.
	SuffixLength int
	// AdditionalSuffix (--additional-suffix) is appended to every name.
	AdditionalSuffix string
	// Suffixes chooses alphabetic, numeric or hexadecimal suffixes, and
	// SuffixStart the first numeric or hexadecimal one ("" means 0).
	Suffixes    Alphabet
	SuffixStart string
	// ElideEmpty (-e) creates no empty files in the chunk modes.
	ElideEmpty bool
	// Separator (-t) is the line separator; a newline unless SeparatorSet.
	Separator    byte
	SeparatorSet bool
	// Verbose (--verbose) reports each file as it is created.
	Verbose bool
}

// Split is the engine. Construct it with New.
type Split struct {
	Options

	streams core.Streams
	diag    *core.Diag

	eol    byte
	input  os.FileInfo // the input's identity, when it is a file
	namer  *namer
	out    *os.File
	outBuf *bufio.Writer
	name   string
}

// New builds a split engine reading from and writing to s.
func New(opts Options, s core.Streams) *Split {
	return &Split{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// errFatal stops the run; the diagnostic has been written.
var errFatal = errors.New("fatal")

func (sp *Split) fail(format string, a ...any) error {
	sp.diag.Errorf(format, a...)
	return errFatal
}

// Run splits FILE (args[0], "-" or absent for standard input) into files
// named PREFIX (args[1], "x" by default) plus a suffix. It returns a
// *core.UsageError for a command-line mistake, core.ErrFailed when anything
// failed, and nil otherwise.
func (sp *Split) Run(args []string) error {
	if len(args) > 2 {
		return core.Usagef("extra operand '%s'", args[2])
	}
	infile, prefix := "-", "x"
	if len(args) > 0 {
		infile = args[0]
	}
	if len(args) > 1 {
		prefix = args[1]
	}
	if err := sp.setup(prefix); err != nil {
		return err
	}

	in := sp.streams.In
	var inFile *os.File
	if infile != "-" {
		f, err := os.Open(infile)
		if err != nil {
			sp.diag.Errorf("cannot open '%s' for reading: %s", infile, core.ErrText(err))
			return core.ErrFailed
		}
		defer f.Close()
		in, inFile = f, f
	} else if f, ok := sp.streams.In.(*os.File); ok {
		inFile = f
	}
	if inFile != nil {
		if info, err := inFile.Stat(); err == nil && info.Mode().IsRegular() {
			sp.input = info
		}
	}
	if in == nil {
		in = strings.NewReader("")
	}

	var err error
	switch sp.Mode {
	case Lines:
		err = sp.lines(in)
	case Bytes:
		err = sp.bytes(in)
	case LineBytes:
		err = sp.lineBytes(in)
	case RoundRobin:
		err = sp.roundRobin(in)
	case ChunkBytes, ChunkLines:
		err = sp.chunks(in, inFile, infile)
	}
	if cerr := sp.closeOut(); err == nil {
		err = cerr
	}
	if err != nil && !errors.Is(err, errFatal) {
		sp.diag.Errorf("%s: %s", infile, core.ErrText(err))
	}
	return sp.diag.Result()
}

// setup checks the options and prepares the file names.
func (sp *Split) setup(prefix string) error {
	if strings.ContainsAny(sp.AdditionalSuffix, `/`+string(filepath.Separator)) {
		return core.Usagef("invalid suffix '%s', contains directory separator", sp.AdditionalSuffix)
	}
	sp.eol = '\n'
	if sp.SeparatorSet {
		sp.eol = sp.Separator
	}
	if sp.Mode == Lines && sp.Count == 0 {
		sp.Count = DefaultLines
	}
	if sp.Count < 1 {
		return core.Usagef("invalid number of lines: '0'")
	}
	if sp.Chunk < 0 || sp.Chunk > sp.Count {
		return core.Usagef("invalid chunk number: '%d'", sp.Chunk)
	}

	alphabet := alphabets[sp.Suffixes]
	start := strings.TrimLeft(sp.SuffixStart, "0")
	if sp.SuffixStart != "" && start == "" {
		start = "0"
	}
	for _, c := range start {
		if !strings.ContainsRune(alphabet, c) {
			kind := "numerical"
			if sp.Suffixes == Hex {
				kind = "hexadecimal"
			}
			return core.Usagef("'%s': invalid start value for %s suffix", sp.SuffixStart, kind)
		}
	}

	// The suffix grows as needed unless its length is fixed, a start value
	// makes the names non-consecutive, or the number of files is known.
	auto := sp.SuffixStart == ""
	length := sp.SuffixLength
	if sp.Mode == ChunkBytes || sp.Mode == ChunkLines || sp.Mode == RoundRobin {
		end := sp.Count - 1
		if n, ok := parseIndex(start, alphabet); ok && n < sp.Count {
			end += n
		}
		needed := 0
		for {
			needed++
			end /= int64(len(alphabet))
			if end == 0 {
				break
			}
		}
		if length != 0 && length < needed {
			sp.diag.Errorf("the suffix length needs to be at least %d", needed)
			return core.ErrFailed
		}
		length = max(length, needed, 2)
		auto = false
	}
	if sp.SuffixLength != 0 {
		auto = false
	}
	if length == 0 {
		length = 2
	}
	if len(start) > length {
		return core.Usagef("numerical suffix start value is too large for the suffix length")
	}
	sp.namer = newNamer(prefix, sp.AdditionalSuffix, alphabet, length, start, auto)
	return nil
}

// parseIndex reads a start value as a number in the alphabet's base.
func parseIndex(s, alphabet string) (int64, bool) {
	var n int64
	for _, c := range s {
		i := strings.IndexRune(alphabet, c)
		if i < 0 || n > (1<<62)/int64(len(alphabet)) {
			return 0, false
		}
		n = n*int64(len(alphabet)) + int64(i)
	}
	return n, true
}

// openNext closes the current file and creates the next one.
func (sp *Split) openNext() error {
	if err := sp.closeOut(); err != nil {
		return err
	}
	name, ok := sp.namer.next()
	if !ok {
		return sp.fail("output file suffixes exhausted")
	}
	f, err := sp.create(name)
	if err != nil {
		return err
	}
	sp.out, sp.name = f, name
	sp.outBuf = bufio.NewWriterSize(f, 64*1024)
	return nil
}

// create makes one output file, refusing to overwrite the input.
func (sp *Split) create(name string) (*os.File, error) {
	if sp.Verbose {
		fmt.Fprintf(sp.streams.Out, "creating file '%s'\n", name)
	}
	if sp.input != nil {
		if info, err := os.Stat(name); err == nil && os.SameFile(info, sp.input) {
			return nil, sp.fail("'%s' would overwrite input; aborting", name)
		}
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return nil, sp.fail("%s: %s", name, core.ErrText(err))
	}
	return f, nil
}

// flushStdout flushes a chunk written to standard output, reporting a write
// error as the run's failure.
func (sp *Split) flushStdout(w *bufio.Writer, err *error) {
	if ferr := w.Flush(); ferr != nil && !core.IsClosedPipe(ferr) && *err == nil {
		*err = sp.fail("write error: %s", core.ErrText(ferr))
	}
}

func (sp *Split) write(p []byte) error {
	if _, err := sp.outBuf.Write(p); err != nil {
		return sp.fail("%s: %s", sp.name, core.ErrText(err))
	}
	return nil
}

func (sp *Split) closeOut() error {
	if sp.out == nil {
		return nil
	}
	f, name := sp.out, sp.name
	sp.out = nil
	err := sp.outBuf.Flush()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return sp.fail("%s: %s", name, core.ErrText(err))
	}
	return nil
}

// emptyFile creates the next file with nothing in it, unless ElideEmpty.
func (sp *Split) emptyFile() error {
	if sp.ElideEmpty {
		return nil
	}
	return sp.openNext()
}

// lines writes Count lines to each file.
func (sp *Split) lines(in io.Reader) error {
	r := bufio.NewReaderSize(in, 64*1024)
	n := sp.Count
	for {
		line, err := r.ReadSlice(sp.eol)
		if len(line) > 0 {
			if n == sp.Count {
				if oerr := sp.openNext(); oerr != nil {
					return oerr
				}
				n = 0
			}
			if werr := sp.write(line); werr != nil {
				return werr
			}
			if line[len(line)-1] == sp.eol && !errors.Is(err, bufio.ErrBufferFull) {
				n++
			}
		}
		switch {
		case err == nil || errors.Is(err, bufio.ErrBufferFull):
		case err == io.EOF:
			return nil
		default:
			return err
		}
	}
}

// bytes writes Count bytes to each file.
func (sp *Split) bytes(in io.Reader) error {
	buf := make([]byte, 64*1024)
	left := int64(0)
	for {
		n, err := in.Read(buf)
		p := buf[:n]
		for len(p) > 0 {
			if left == 0 {
				if oerr := sp.openNext(); oerr != nil {
					return oerr
				}
				left = sp.Count
			}
			k := min(int64(len(p)), left)
			if werr := sp.write(p[:k]); werr != nil {
				return werr
			}
			p, left = p[k:], left-k
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// lineBytes writes as many whole lines as fit in Count bytes to each file;
// a line longer than that is split across files.
func (sp *Split) lineBytes(in io.Reader) error {
	r := bufio.NewReaderSize(in, 64*1024)
	used := int64(0)
	open := false
	for {
		line, err := r.ReadBytes(sp.eol)
		for len(line) > 0 {
			room := sp.Count - used
			switch {
			case open && int64(len(line)) <= room:
			case open && used > 0:
				open = false
				continue
			default:
				if oerr := sp.openNext(); oerr != nil {
					return oerr
				}
				open, used, room = true, 0, sp.Count
			}
			k := min(int64(len(line)), room)
			if werr := sp.write(line[:k]); werr != nil {
				return werr
			}
			used += k
			line = line[k:]
			if used == sp.Count {
				open = false
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// roundRobin deals lines out to Count files in turn, or with Chunk writes
// every Count-th line, starting with line Chunk, to standard output.
func (sp *Split) roundRobin(in io.Reader) (err error) {
	r := bufio.NewReaderSize(in, 64*1024)
	var stdout *bufio.Writer
	var files []*os.File
	var bufs []*bufio.Writer
	var names []string
	if sp.Chunk > 0 {
		stdout = bufio.NewWriter(sp.streams.Out)
		defer sp.flushStdout(stdout, &err)
	} else {
		for range sp.Count {
			name, ok := sp.namer.next()
			if !ok {
				return sp.fail("output file suffixes exhausted")
			}
			names = append(names, name)
		}
		files = make([]*os.File, sp.Count)
		bufs = make([]*bufio.Writer, sp.Count)
		// On an early return the run has already failed: just close.
		defer func() {
			for _, f := range files {
				if f != nil {
					f.Close()
				}
			}
		}()
	}
	i := int64(0)
	for {
		line, err := r.ReadBytes(sp.eol)
		if len(line) > 0 {
			if stdout != nil {
				if i == sp.Chunk-1 {
					_, _ = stdout.Write(line)
				}
			} else {
				if files[i] == nil {
					f, cerr := sp.create(names[i])
					if cerr != nil {
						return cerr
					}
					files[i], bufs[i] = f, bufio.NewWriter(f)
				}
				if _, werr := bufs[i].Write(line); werr != nil {
					return sp.fail("%s: %s", names[i], core.ErrText(werr))
				}
			}
			i = (i + 1) % sp.Count
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	created := make([]bool, len(files))
	for i, f := range files {
		if f == nil {
			continue
		}
		created[i], files[i] = true, nil
		ferr := bufs[i].Flush()
		if cerr := f.Close(); ferr == nil {
			ferr = cerr
		}
		if ferr != nil {
			return sp.fail("%s: %s", names[i], core.ErrText(ferr))
		}
	}
	if stdout == nil && !sp.ElideEmpty {
		for i := range files {
			if !created[i] {
				nf, err := sp.create(names[i])
				if err != nil {
					return err
				}
				nf.Close()
			}
		}
	}
	return nil
}

// chunks makes Count files of nearly equal size: the first size%Count are a
// byte longer, as in GNU. In ChunkLines mode a line goes to the chunk it
// starts in. A pipe is first copied to a temporary file, as GNU does, so
// its size is known without holding it in memory.
func (sp *Split) chunks(in io.Reader, inFile *os.File, infile string) (err error) {
	var src io.ReaderAt
	var size int64
	if sp.input != nil && inFile != nil {
		src, size = inFile, sp.input.Size()
	} else {
		tmp, err := os.CreateTemp("", "splitx-*")
		if err != nil {
			return sp.fail("%s: cannot determine file size: %s", infile, core.ErrText(err))
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()
		if size, err = io.Copy(tmp, in); err != nil {
			return err
		}
		src = tmp
	}

	n := sp.Count
	q, rem := size/n, size%n
	start := func(c int64) int64 { return c*q + min(c, rem) } // c is 0-based

	if sp.Mode == ChunkBytes {
		for c := range n {
			if sp.Chunk > 0 && c != sp.Chunk-1 {
				continue
			}
			sec := io.NewSectionReader(src, start(c), start(c+1)-start(c))
			if sp.Chunk > 0 {
				_, err := io.Copy(sp.streams.Out, sec)
				return err
			}
			if sec.Size() == 0 {
				if err := sp.emptyFile(); err != nil {
					return err
				}
				continue
			}
			if err := sp.openNext(); err != nil {
				return err
			}
			if _, err := io.Copy(sp.outBuf, sec); err != nil {
				return sp.fail("%s: %s", sp.name, core.ErrText(err))
			}
		}
		return nil
	}

	// ChunkLines: the chunk a byte offset falls in.
	chunkOf := func(off int64) int64 {
		if big := rem * (q + 1); off < big {
			return off / (q + 1)
		} else if q > 0 {
			return rem + (off-big)/q
		}
		return n - 1
	}
	r := bufio.NewReaderSize(io.NewSectionReader(src, 0, size), 64*1024)
	var stdout *bufio.Writer
	if sp.Chunk > 0 {
		stdout = bufio.NewWriter(sp.streams.Out)
		defer sp.flushStdout(stdout, &err)
	}
	current := int64(-1) // the chunk whose file is open
	off := int64(0)
	for {
		line, err := r.ReadBytes(sp.eol)
		if len(line) > 0 {
			c := chunkOf(off)
			off += int64(len(line))
			if stdout != nil {
				if c == sp.Chunk-1 {
					_, _ = stdout.Write(line)
				} else if c > sp.Chunk-1 {
					return nil
				}
			} else {
				for current < c {
					current++
					if current < c {
						if eerr := sp.emptyFile(); eerr != nil {
							return eerr
						}
						continue
					}
					if oerr := sp.openNext(); oerr != nil {
						return oerr
					}
				}
				if werr := sp.write(line); werr != nil {
					return werr
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if stdout == nil {
		for current++; current < n; current++ {
			if err := sp.emptyFile(); err != nil {
				return err
			}
		}
	}
	return nil
}

// namer generates output file names as GNU split does.
type namer struct {
	prefix, addsuf string
	alphabet       string
	idx            []int
	auto           bool
	started        bool
}

func newNamer(prefix, addsuf, alphabet string, length int, start string, auto bool) *namer {
	nm := &namer{prefix: prefix, addsuf: addsuf, alphabet: alphabet, idx: make([]int, length), auto: auto}
	for i, c := range start {
		nm.idx[length-len(start)+i] = strings.IndexRune(alphabet, c)
	}
	return nm
}

// next returns the next name, or false when the suffixes are exhausted.
// With auto, a suffix about to start with the alphabet's last character is
// instead widened: the prefix absorbs that character and the suffix gets one
// longer, so xyz is followed by xzaaa and the names still sort in order.
func (nm *namer) next() (string, bool) {
	if !nm.started {
		nm.started = true
		return nm.String(), true
	}
	last := len(nm.alphabet) - 1
	for i := len(nm.idx) - 1; i >= 0; i-- {
		nm.idx[i]++
		if nm.auto && i == 0 && nm.idx[0] == last {
			nm.prefix += string(nm.alphabet[last])
			nm.idx = make([]int, len(nm.idx)+1)
			return nm.String(), true
		}
		if nm.idx[i] <= last {
			return nm.String(), true
		}
		nm.idx[i] = 0
	}
	return "", false
}

func (nm *namer) String() string {
	var b bytes.Buffer
	b.WriteString(nm.prefix)
	for _, i := range nm.idx {
		b.WriteByte(nm.alphabet[i])
	}
	b.WriteString(nm.addsuf)
	return b.String()
}
