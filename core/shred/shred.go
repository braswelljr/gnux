// Package shred implements the engine behind shredx, a GNU-compatible
// implementation of the GNU/UNIX shred command: overwrite a file to hide its
// contents, and optionally remove it.
package shred

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/human"
	"github.com/braswelljr/gnux/core/internal/quotearg"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "shredx"

// DefaultPasses is the number of overwrite passes when Options.Iterations
// is zero.
const DefaultPasses = 3

// Remove is how a file is removed after it is overwritten.
type Remove int

// The removal methods of -u and --remove.
const (
	// Keep leaves the file in place: the default.
	Keep Remove = iota
	// Unlink removes it directly (--remove=unlink).
	Unlink
	// Wipe renames it to ever shorter names of zeros first, to hide its
	// name too (--remove=wipe).
	Wipe
	// WipeSync is Wipe with the directory flushed to disk after each rename
	// (-u, --remove=wipesync).
	WipeSync
)

// Options mirrors the GNU shred flag surface. The zero value is plain shred:
// three random passes over each file, which is kept.
type Options struct {
	// Force (-f) changes permissions to allow writing if necessary.
	Force bool
	// Iterations (-n) is the number of overwrite passes: zero means
	// DefaultPasses, and a negative value none at all (-n 0).
	Iterations int
	// RandomSource (--random-source) is a file to take random bytes from,
	// instead of the system's generator.
	RandomSource string
	// Size (-s), when SizeSet, is how many bytes to overwrite instead of the
	// whole file.
	Size    int64
	SizeSet bool
	// Remove (-u, --remove) removes each file after overwriting it.
	Remove Remove
	// Verbose (-v) reports each pass, and the renaming and removal.
	Verbose bool
	// Exact (-x) does not round the size up to the next whole block.
	Exact bool
	// Zero (-z) adds a final pass of zeros, to hide the shredding.
	Zero bool
}

// Shred is the engine. Construct it with New.
type Shred struct {
	Options

	// Now is the clock that paces the progress reports of -v. Random is
	// where random bytes come from unless RandomSource names a file. They
	// are the injection points tests use.
	Now    func() time.Time
	Random io.Reader

	streams core.Streams
	diag    *core.Diag
	random  io.Reader
}

// New builds a shred engine whose diagnostics go to s.Err.
func New(opts Options, s core.Streams) *Shred {
	return &Shred{Options: opts, Now: time.Now, Random: rand.Reader, streams: s, diag: core.NewDiag(Name, s)}
}

// errRandomSource stops the run: the random source failed, and GNU exits.
var errRandomSource = errors.New("random source failed")

// Run shreds each FILE in turn; "-" shreds standard output. A file that
// cannot be shredded is reported and the rest are still shredded.
func (s *Shred) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing file operand")
	}
	s.random = s.Random
	if s.RandomSource != "" {
		f, err := fsys.Open(s.RandomSource)
		if err != nil {
			s.diag.Errorf("%s: %s", quotef(s.RandomSource), core.ErrText(err))
			return s.diag.Result()
		}
		defer f.Close()
		s.random = &sourceReader{r: f, name: s.RandomSource}
	}
	for _, name := range args {
		var err error
		if name == "-" {
			err = s.wipeStdout()
		} else {
			err = s.wipeFile(name)
		}
		if errors.Is(err, errRandomSource) {
			return core.ErrFailed
		}
	}
	return s.diag.Result()
}

// sourceReader reads a --random-source file, which must not run out.
type sourceReader struct {
	r    io.Reader
	name string
	err  error
}

func (r *sourceReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	n, err := io.ReadFull(r.r, p)
	if err != nil {
		r.err = err
	}
	return n, err
}

func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

// random fills p, reporting a failing random source as GNU does.
func (s *Shred) fill(p []byte) error {
	if _, err := io.ReadFull(s.random, p); err != nil {
		name := "getrandom"
		if s.RandomSource != "" {
			name = s.RandomSource
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			s.diag.Errorf("%s: end of file", quoteName(name))
		} else {
			s.diag.Errorf("%s: %s", quoteName(name), core.ErrText(err))
		}
		return errRandomSource
	}
	return nil
}

func quoteName(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscapeAlways})
}

// choose returns a random number below k.
func (s *Shred) choose(k int) (int, error) {
	var b [8]byte
	if err := s.fill(b[:]); err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(k)), nil
}

func (s *Shred) passes() int {
	switch {
	case s.Iterations < 0:
		return 0
	case s.Iterations == 0:
		return DefaultPasses
	}
	return s.Iterations
}

func (s *Shred) wipeFile(name string) error {
	q := quotef(name)
	f, err := fsys.OpenFile(name, os.O_WRONLY, 0)
	if err != nil && s.Force && errors.Is(err, os.ErrPermission) {
		if fsys.Chmod(name, 0o200) == nil {
			f, err = fsys.OpenFile(name, os.O_WRONLY, 0)
		}
	}
	if err != nil {
		s.diag.Errorf("%s: failed to open for writing: %s", q, core.ErrText(openError(name, err)))
		return nil
	}
	ok, werr := s.wipe(f, q)
	if cerr := f.Close(); cerr != nil {
		s.diag.Errorf("%s: failed to close: %s", q, core.ErrText(cerr))
		ok = false
	}
	if werr != nil {
		return werr
	}
	if ok && s.Remove != Keep {
		s.wipeName(name, q)
	}
	return nil
}

// openError is the error open(2) gives for a directory opened for writing,
// which Go reports differently on some systems.
func openError(name string, err error) error {
	if info, serr := fsys.Stat(name); serr == nil && info.IsDir() {
		return syscall.EISDIR
	}
	return err
}

func (s *Shred) wipeStdout() error {
	f, ok := s.streams.Out.(*os.File)
	if !ok {
		s.diag.Errorf("-: invalid file type")
		return nil
	}
	if appendOnly(f) {
		s.diag.Errorf("-: cannot shred append-only file descriptor")
		return nil
	}
	_, err := s.wipe(f, "-")
	return err
}

// wipe overwrites an open file: GNU's do_wipefd.
func (s *Shred) wipe(f *os.File, q string) (bool, error) {
	info, err := f.Stat()
	if err != nil {
		s.diag.Errorf("%s: fstat failed: %s", q, core.ErrText(err))
		return false, nil
	}
	m := info.Mode()
	if (m&os.ModeCharDevice != 0 && core.IsTerminal(f)) || m&os.ModeNamedPipe != 0 || m&os.ModeSocket != 0 {
		s.diag.Errorf("%s: invalid file type", q)
		return false, nil
	}

	n := s.passes()
	blk := sysstat.Of(info).BlockSize
	if blk <= 0 {
		blk = 512
	}
	size := int64(-1)
	var firstSize int64
	switch {
	case !s.SizeSet && m.IsRegular():
		size = info.Size()
		if !s.Exact {
			if size > 0 && size < blk {
				firstSize = size
			}
			if r := size % blk; r != 0 {
				size += blk - r
			}
		}
	case !s.SizeSet:
		// A device: its size is where the end is, if it says.
		if end, err := f.Seek(0, io.SeekEnd); err == nil && end > 0 {
			size = end
		}
	default:
		size = s.Size
		if m.IsRegular() && info.Size() < min(blk, size) {
			firstSize = info.Size()
		}
	}

	types, err := s.pattern(n)
	if err != nil {
		return false, err
	}
	total := n
	if s.Zero {
		total++
	}

	ok := true
	for {
		passSize, verbose := int64(0), s.Verbose
		switch {
		case firstSize != 0:
			// A file smaller than a block is first overwritten at its own
			// size, quietly, in case the file system keeps it in the inode.
			passSize, firstSize, verbose = firstSize, 0, false
		case size != 0:
			passSize, size = size, 0
		}
		if passSize == 0 {
			break
		}
		for i := range total {
			typ := 0
			if i < n {
				typ = types[i]
			}
			res, err := s.pass(f, info, q, &passSize, typ, i+1, total, verbose)
			if err != nil {
				return false, err
			}
			if res != 0 {
				ok = false
				if res < 0 {
					return false, nil
				}
			}
		}
	}
	if s.Remove != Keep {
		if err := f.Truncate(0); err != nil && m.IsRegular() {
			s.diag.Errorf("%s: error truncating: %s", q, core.ErrText(err))
			return false, nil
		}
	}
	return ok, nil
}

// pass is GNU's dopass: one pass of a pattern (or random data, for a type
// of -1) over the file. It returns 0 on success, 1 when some writes failed
// but the pass went on, and -1 when it had to stop.
func (s *Shred) pass(f *os.File, info os.FileInfo, q string, sizep *int64, typ, k, n int, verbose bool) (int, error) {
	size := *sizep
	outSize := int64(64 * 1024)
	if periodic(typ) {
		outSize = 60 * 1024
	}
	buf := make([]byte, (outSize+2)/3*3)

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		s.diag.Errorf("%s: cannot rewind: %s", q, core.ErrText(err))
		return -1, nil
	}
	name := "random"
	if typ >= 0 {
		lim := int64(len(buf))
		if size >= 0 && size < lim {
			lim = size
		}
		fillPattern(typ, buf[:lim])
		if lim >= 3 {
			name = fmt.Sprintf("%02x%02x%02x", buf[0], buf[1], buf[2])
		} else {
			full := make([]byte, 3)
			fillPattern(typ, full)
			name = fmt.Sprintf("%02x%02x%02x", full[0], full[1], full[2])
		}
	}
	var thresh time.Time
	previous := ""
	if verbose {
		s.diag.Warnf("%s: pass %d/%d (%s)...", q, k, n, name)
		thresh = s.Now().Add(5 * time.Second)
	}

	writeError := false
	var offset int64
	for {
		lim := outSize
		if size >= 0 && size-offset < outSize {
			if size < offset {
				break
			}
			lim = size - offset
			if lim == 0 {
				break
			}
		}
		if typ < 0 {
			if err := s.fill(buf[:lim]); err != nil {
				return -1, err
			}
		}
		written, err := f.Write(buf[:lim])
		if err != nil || int64(written) < lim {
			if size < 0 && (written == 0 || errors.Is(err, syscall.ENOSPC)) {
				// The end of a device whose size was not known.
				*sizep = offset + int64(written)
				break
			}
			if err == nil {
				err = io.ErrShortWrite
			}
			s.diag.Errorf("%s: error writing at offset %d: %s", q, offset+int64(written), core.ErrText(err))
			return -1, nil
		}
		offset += int64(written)
		done := offset == size

		if verbose {
			now := s.Now()
			if (done && previous != "") || !now.Before(thresh) {
				opts := human.Autoscale | human.SI | human.Base1024 | human.B
				off := human.Readable(uint64(offset), human.Floor|opts, 1, 1)
				if done || off != previous {
					if size < 0 {
						s.diag.Warnf("%s: pass %d/%d (%s)...%s", q, k, n, name, off)
					} else {
						percent := int64(100)
						if size != 0 {
							percent = offset * 100 / size
						}
						total := human.Readable(uint64(size), human.Ceiling|opts, 1, 1)
						if done {
							off = total
						}
						s.diag.Warnf("%s: pass %d/%d (%s)...%s/%s %d%%", q, k, n, name, off, total, percent)
					}
					previous = off
					thresh = now.Add(5 * time.Second)
					if !s.sync(f, q) {
						writeError = true
					}
				}
			}
		}
	}
	if !s.sync(f, q) {
		writeError = true
	}
	if writeError {
		return 1, nil
	}
	return 0, nil
}

// sync flushes the file to disk. A file that cannot be flushed (a pipe, a
// device that does not support it) is not an error.
func (s *Shred) sync(f *os.File, q string) bool {
	err := f.Sync()
	if err == nil || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EBADF) ||
		errors.Is(err, syscall.EROFS) || errors.Is(err, errors.ErrUnsupported) {
		return true
	}
	s.diag.Errorf("%s: fdatasync failed: %s", q, core.ErrText(err))
	return false
}

// fillPattern fills r with the 12-bit pattern of typ, repeated; a type with
// the 0x1000 bit set also has the first bit of every 512-byte sector flipped.
func fillPattern(typ int, r []byte) {
	bits := uint32(typ&0xfff) | uint32(typ&0xfff)<<12
	three := [3]byte{byte(bits >> 4), byte(bits >> 8), byte(bits)}
	for i := range r {
		r[i] = three[i%3]
	}
	if typ&0x1000 != 0 {
		for i := 0; i < len(r); i += 512 {
			r[i] ^= 0x80
		}
	}
}

func periodic(typ int) bool {
	if typ <= 0 {
		return false
	}
	var r [3]byte
	fillPattern(typ&0xfff, r[:])
	return r[0] != r[1] || r[0] != r[2]
}

// patterns is GNU's table of passes: a negative entry is that many random
// passes, a positive one is followed by that many fixed patterns, and 0
// starts again.
var patterns = []int{
	-2,
	2, 0x000, 0xFFF,
	2, 0x555, 0xAAA,
	-1,
	6, 0x249, 0x492, 0x6DB, 0x924, 0xB6D, 0xDB6,
	12, 0x111, 0x222, 0x333, 0x444, 0x666, 0x777, 0x888, 0x999, 0xBBB, 0xCCC, 0xDDD, 0xEEE,
	-1,
	8, 0x1000, 0x1249, 0x1492, 0x16DB, 0x1924, 0x1B6D, 0x1DB6, 0x1FFF,
	14, 0x1111, 0x1222, 0x1333, 0x1444, 0x1555, 0x1666, 0x1777, 0x1888, 0x1999, 0x1AAA, 0x1BBB, 0x1CCC, 0x1DDD, 0x1EEE,
	-1,
	0,
}

// pattern chooses num passes and shuffles them, with the random passes
// spread evenly among them, first and last included: GNU's genpattern.
// A random pass is -1.
func (s *Shred) pattern(num int) ([]int, error) {
	dest := make([]int, num)
	if num == 0 {
		return dest, nil
	}
	random, d, n, p := 0, 0, num, 0
	for {
		k := patterns[p]
		p++
		switch {
		case k == 0:
			p = 0
			continue
		case k < 0:
			k = -k
			if k >= n {
				random += n
				goto shuffle
			}
			random += k
			n -= k
		case k <= n:
			copy(dest[d:], patterns[p:p+k])
			p += k
			d += k
			n -= k
		case n < 2 || 3*n < k:
			random += n
			goto shuffle
		default:
			for n > 0 {
				c := 0
				if n != k {
					var err error
					if c, err = s.choose(k); err != nil {
						return nil, err
					}
				}
				if n == k || c < n {
					dest[d] = patterns[p]
					d++
					n--
				}
				p++
				k--
			}
			goto shuffle
		}
	}
shuffle:
	top := num - random
	random--
	accum := random
	for i := range num {
		if accum <= random {
			accum += num - 1
			dest[top] = dest[i]
			top++
			dest[i] = -1
		} else {
			c, err := s.choose(top - i)
			if err != nil {
				return nil, err
			}
			swap := i + c
			dest[i], dest[swap] = dest[swap], dest[i]
		}
		accum -= random
	}
	return dest, nil
}

// nameset is the alphabet of the names a file is renamed to before removal.
const nameset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_."

// incname turns name into the next name of the same length, as a number in
// base len(nameset); false when it was the last.
func incname(name []byte) bool {
	for i := len(name) - 1; i >= 0; i-- {
		j := strings.IndexByte(nameset, name[i])
		if j+1 < len(nameset) {
			name[i] = nameset[j+1]
			return true
		}
		name[i] = nameset[0]
	}
	return false
}

// wipeName removes the file, renaming it first to hide its name: GNU's
// wipename.
func (s *Shred) wipeName(oldname, q string) {
	if s.Verbose {
		s.diag.Warnf("%s: removing", q)
	}
	isSep := func(r rune) bool { return r < 0x80 && os.IsPathSeparator(uint8(r)) }
	trimmed := strings.TrimRightFunc(oldname, isSep)
	baseStart := strings.LastIndexFunc(trimmed, isSep) + 1
	dir := oldname[:baseStart]
	baseLen := len(trimmed) - baseStart
	syncDir := func() {
		if s.Remove == WipeSync {
			syncDirectory(dir)
		}
	}

	first := true
	if s.Remove != Unlink {
		for l := baseLen; l > 0; l-- {
			base := []byte(strings.Repeat("0", l))
			renamed := false
			for {
				newname := dir + string(base)
				if _, err := fsys.Lstat(newname); err == nil {
					if !incname(base) {
						break
					}
					continue
				}
				if fsys.Rename(oldname, newname) == nil {
					renamed = true
				}
				break
			}
			if !renamed {
				continue
			}
			newname := dir + string(base)
			syncDir()
			if s.Verbose {
				old := oldname
				if first {
					old = q
				}
				s.diag.Warnf("%s: renamed to %s", old, newname)
				first = false
			}
			oldname = newname
		}
	}
	if err := fsys.Remove(oldname); err != nil {
		s.diag.Errorf("%s: failed to remove: %s", q, core.ErrText(err))
		return
	}
	if s.Verbose {
		s.diag.Warnf("%s: removed", q)
	}
	syncDir()
}
