// Package shuf implements the engine behind shufx, a GNU-compatible
// implementation of the GNU/UNIX shuf command: write a random permutation of
// the input lines, of the operands (-e) or of a range of numbers (-i).
package shuf

import (
	"bufio"
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "shufx"

// Options mirrors the GNU shuf flag surface. The zero value shuffles the
// lines of one FILE (or standard input) and writes them all.
type Options struct {
	// Echo (-e) treats each operand as an input line.
	Echo bool
	// InputRange (-i LO-HI) treats each number LO through HI as an input line.
	// Empty means no range.
	InputRange string
	// HeadCount (-n COUNT) writes at most COUNT lines. Negative means no limit.
	// A Options{} value has HeadCount 0, which New reads as "no limit" unless
	// HeadCountSet is true.
	HeadCount int64
	// HeadCountSet records that -n was given, so that -n 0 is distinguishable
	// from no -n at all.
	HeadCountSet bool
	// Output (-o FILE) writes to FILE instead of standard output. The file is
	// opened only once the input has been read, so it may be an input.
	Output string
	// RandomSource (--random-source=FILE) takes the random bytes from FILE.
	RandomSource string
	// Repeat (-r) lets output lines repeat; with no -n it writes forever.
	Repeat bool
	// ZeroTerminated (-z) uses NUL, not newline, as the line delimiter.
	ZeroTerminated bool
}

// Shuf is the engine. Construct it with New.
type Shuf struct {
	Options

	// Rand returns a uniformly distributed integer in [0, n). It is the
	// injection point tests use for a deterministic order; when nil, New's
	// default (a cryptographically seeded generator, or --random-source) is
	// used.
	Rand func(n uint64) (uint64, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a shuf engine reading from and writing to s.
func New(opts Options, s core.Streams) *Shuf {
	return &Shuf{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// errRandom wraps a failure to read the random source, already diagnosed.
var errRandom = errors.New("random source failed")

// Run shuffles the input selected by the options. A failure is diagnosed and
// returned as core.ErrFailed; a command-line mistake as a *core.UsageError.
func (sh *Shuf) Run(args []string) error {
	if sh.Echo && sh.InputRange != "" {
		return core.Usagef("cannot combine -e and -i options")
	}

	var lines [][]byte
	var lo, hi uint64
	ranged := false
	switch {
	case sh.InputRange != "":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'", args[0])
		}
		var ok bool
		lo, hi, ok = parseRange(sh.InputRange)
		if !ok {
			sh.diag.Errorf("invalid input range: '%s'", sh.InputRange)
			return core.ErrFailed
		}
		ranged = true
	case sh.Echo:
		for _, a := range args {
			lines = append(lines, []byte(a))
		}
	default:
		if len(args) > 1 {
			return core.Usagef("extra operand '%s'", args[1])
		}
		name := "-"
		if len(args) == 1 {
			name = args[0]
		}
		data, err := sh.read(name)
		if err != nil {
			sh.diag.Errorf("%s: %s", name, core.ErrText(err))
			return core.ErrFailed
		}
		lines = split(data, sh.delim())
	}

	random, closeRandom, err := sh.random()
	if err != nil {
		return err
	}
	defer closeRandom()

	limit := uint64(math.MaxUint64)
	if sh.HeadCountSet && sh.HeadCount >= 0 {
		limit = uint64(sh.HeadCount)
	}

	n := uint64(len(lines))
	if ranged {
		n = hi - lo + 1
		if hi < lo {
			n = 0
		}
	}

	out := sh.streams.Out
	if sh.Output != "" {
		f, err := fsys.OpenFile(sh.Output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			sh.diag.Errorf("%s: %s", sh.Output, core.ErrText(err))
			return core.ErrFailed
		}
		defer f.Close()
		out = f
	}
	w := core.NewWriter(out)

	item := func(i uint64) []byte {
		if ranged {
			return strconv.AppendUint(nil, lo+i, 10)
		}
		return lines[i]
	}

	err = sh.emit(w, n, limit, random, item)
	if ferr := w.Flush(); err == nil {
		err = ferr
	}
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errRandom):
		return core.ErrFailed
	case errors.Is(err, errNoLines):
		sh.diag.Errorf("no lines to repeat")
		return core.ErrFailed
	case core.IsClosedPipe(err):
		return nil
	default:
		name := "standard output"
		if sh.Output != "" {
			name = sh.Output
		}
		sh.diag.Errorf("write error: %s: %s", name, core.ErrText(err))
		return core.ErrFailed
	}
}

var errNoLines = errors.New("no lines to repeat")

// emit writes the permutation (or, with -r, the repeated sample) of n items.
func (sh *Shuf) emit(w *bufio.Writer, n, limit uint64, random func(uint64) (uint64, error), item func(uint64) []byte) error {
	delim := sh.delim()
	put := func(b []byte) error {
		if _, err := w.Write(b); err != nil {
			return err
		}
		return w.WriteByte(delim)
	}

	if sh.Repeat {
		if limit == 0 {
			return nil
		}
		if n == 0 {
			return errNoLines
		}
		for written := uint64(0); written < limit; written++ {
			i, err := random(n)
			if err != nil {
				return err
			}
			if err := put(item(i)); err != nil {
				return err
			}
		}
		return nil
	}

	count := min(n, limit)
	if count == 0 {
		return nil
	}
	// A partial Fisher-Yates shuffle over an index permutation.
	at, swap := permutation(n, count)
	for i := uint64(0); i < count; i++ {
		r, err := random(n - i)
		if err != nil {
			return err
		}
		j := i + r
		vj := at(j)
		swap(i, j)
		if err := put(item(vj)); err != nil {
			return err
		}
	}
	return nil
}

// permutation returns the identity permutation of n indexes as a pair of
// accessors, stored to suit a shuffle that outputs count of them. Outputting
// most of the indexes touches most of them, so they sit in a slice. Outputting
// a few of many (a small -n over a huge -i range) keeps only the indexes that
// were swapped, in a map: the rest still map to themselves.
func permutation(n, count uint64) (at func(uint64) uint64, swap func(i, j uint64)) {
	if count >= n/2 {
		perm := make([]uint64, n)
		for i := range perm {
			perm[i] = uint64(i)
		}
		return func(i uint64) uint64 { return perm[i] },
			func(i, j uint64) { perm[i], perm[j] = perm[j], perm[i] }
	}
	perm := map[uint64]uint64{}
	at = func(i uint64) uint64 {
		if v, ok := perm[i]; ok {
			return v
		}
		return i
	}
	return at, func(i, j uint64) { perm[i], perm[j] = at(j), at(i) }
}

func (sh *Shuf) delim() byte {
	if sh.ZeroTerminated {
		return 0
	}
	return '\n'
}

func (sh *Shuf) read(name string) ([]byte, error) {
	if name == "-" {
		return io.ReadAll(sh.streams.In)
	}
	return fsys.ReadFile(name)
}

// split cuts data into lines; a final line without a delimiter still counts.
func split(data []byte, delim byte) [][]byte {
	var lines [][]byte
	for len(data) > 0 {
		i := bytes.IndexByte(data, delim)
		if i < 0 {
			lines = append(lines, data)
			break
		}
		lines = append(lines, data[:i])
		data = data[i+1:]
	}
	return lines
}

// parseRange parses LO-HI, two unsigned decimal numbers. HI may be LO-1, an
// empty range.
func parseRange(s string) (lo, hi uint64, ok bool) {
	a, b, found := strings.Cut(s, "-")
	if !found {
		return 0, 0, false
	}
	lo, err1 := strconv.ParseUint(a, 10, 64)
	hi, err2 := strconv.ParseUint(b, 10, 64)
	if err1 != nil || err2 != nil || (hi < lo && hi+1 != lo) {
		return 0, 0, false
	}
	return lo, hi, true
}

// random returns the generator to draw from: the injected one, the bytes of
// --random-source, or a cryptographically seeded PCG.
func (sh *Shuf) random() (draw func(uint64) (uint64, error), closeFn func(), err error) {
	noClose := func() {}
	if sh.Rand != nil {
		return sh.Rand, noClose, nil
	}
	if sh.RandomSource != "" {
		// Read lazily, since the source may be endless (/dev/urandom), and
		// closed by Run: an open file cannot be removed on Windows.
		f, err := fsys.Open(sh.RandomSource)
		if err != nil {
			sh.diag.Errorf("%s: %s", sh.RandomSource, core.ErrText(err))
			return nil, noClose, core.ErrFailed
		}
		br := bufio.NewReader(f)
		name := sh.RandomSource
		return func(n uint64) (uint64, error) {
			v, err := uniform(br, n)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					sh.diag.Errorf("'%s': end of file", name)
				} else {
					sh.diag.Errorf("'%s': %s", name, core.ErrText(err))
				}
				return 0, errRandom
			}
			return v, nil
		}, func() { _ = f.Close() }, nil
	}
	var seed [16]byte
	_, _ = crand.Read(seed[:])
	r := rand.New(rand.NewPCG(binary.LittleEndian.Uint64(seed[:8]), binary.LittleEndian.Uint64(seed[8:])))
	return func(n uint64) (uint64, error) { return r.Uint64N(n), nil }, noClose, nil
}

// uniform draws a number in [0, n) from the bytes of r by rejection sampling,
// reading only as many bytes as n needs, so a given source always yields the
// same permutation.
func uniform(r io.Reader, n uint64) (uint64, error) {
	if n <= 1 {
		return 0, nil
	}
	size := 1
	for size < 8 && n-1 > (uint64(1)<<(8*size))-1 {
		size++
	}
	// maxOK is the largest drawn value kept: values above the last whole
	// multiple of n in the drawn space are rejected, so every result is
	// equally likely.
	var maxOK uint64
	if size < 8 {
		space := uint64(1) << (8 * size)
		maxOK = space - space%n - 1
	} else {
		maxOK = math.MaxUint64 - (math.MaxUint64%n+1)%n
	}
	buf := make([]byte, size)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		var v uint64
		for _, b := range buf {
			v = v<<8 | uint64(b)
		}
		if v <= maxOK {
			return v % n, nil
		}
	}
}

// ParseCount parses a -n COUNT the way GNU shuf does, returning a usage error
// in its words when COUNT is not a non-negative integer.
func ParseCount(s string) (int64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return math.MaxInt64, nil
		}
		return 0, fmt.Errorf("invalid line count: '%s'", s)
	}
	if v > math.MaxInt64 {
		return math.MaxInt64, nil
	}
	return int64(v), nil
}
