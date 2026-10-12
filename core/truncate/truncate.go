// Package truncate implements the engine behind truncatex, a GNU-compatible
// implementation of the GNU/UNIX truncate command: shrink or extend the size
// of each file to a given size.
package truncate

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "truncatex"

// Options mirrors the GNU truncate flag surface.
type Options struct {
	// NoCreate (-c) does not create files that do not exist.
	NoCreate bool
	// IOBlocks (-o) treats SIZE as a number of I/O blocks instead of bytes.
	IOBlocks bool
	// Reference (-r) bases the size on that of the named file.
	Reference string
	// Size (-s) is the size or adjustment: [+-<>/%]NUMBER[SUFFIX].
	Size string
}

// op is how a size is applied to a file's current size.
type op byte

const (
	opSet     op = 0
	opExtend  op = '+'
	opReduce  op = '-'
	opAtMost  op = '<'
	opAtLeast op = '>'
	opRoundDn op = '/'
	opRoundUp op = '%'
)

// Truncate is the engine. Construct it with New.
type Truncate struct {
	Options

	diag *core.Diag
}

// New builds a truncate engine writing diagnostics to s.
func New(opts Options, s core.Streams) *Truncate {
	return &Truncate{Options: opts, diag: core.NewDiag(Name, s)}
}

// Run resizes each FILE in args. A failing file is reported and the rest are
// still processed; the returned error is core.ErrFailed if any failed.
func (t *Truncate) Run(args []string) error {
	haveSize := t.Size != ""
	if !haveSize && t.Reference == "" {
		return core.Usagef("you must specify either '--size' or '--reference'")
	}
	if t.IOBlocks && !haveSize {
		return core.Usagef("'--io-blocks' was specified but '--size' was not")
	}

	// With -r alone the size is the reference's: zero added to it.
	how, n := opExtend, int64(0)
	if haveSize {
		var err error
		how, n, err = parseSize(t.Size)
		if err != nil {
			return core.Usagef("Invalid number: '%s'", t.Size)
		}
		if t.Reference != "" && how == opSet {
			return core.Usagef("you must specify a relative '--size' with '--reference'")
		}
		if (how == opRoundDn || how == opRoundUp) && n == 0 {
			return core.Usagef("division by zero")
		}
	}
	if len(args) == 0 {
		return core.Usagef("missing file operand")
	}

	refSize := int64(-1)
	if t.Reference != "" {
		info, err := os.Stat(t.Reference)
		if err != nil {
			t.diag.Errorf("cannot stat '%s': %s", t.Reference, core.ErrText(err))
			return core.ErrFailed
		}
		refSize = info.Size()
	}

	for _, path := range args {
		t.resize(path, how, n, refSize)
	}
	return t.diag.Result()
}

// resize applies the size to one file.
func (t *Truncate) resize(path string, how op, n, refSize int64) {
	flags := os.O_WRONLY
	if !t.NoCreate {
		flags |= os.O_CREATE
	}
	f, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		if t.NoCreate && errors.Is(err, os.ErrNotExist) {
			return
		}
		t.diag.Errorf("cannot open '%s' for writing: %s", path, core.ErrText(err))
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		t.diag.Errorf("cannot fstat '%s': %s", path, core.ErrText(err))
		return
	}

	size := n
	if t.IOBlocks {
		bs := blockSize(info)
		if size > math.MaxInt64/bs {
			t.diag.Errorf("overflow in %d * %d byte blocks for file '%s'", size, bs, path)
			return
		}
		size *= bs
	}

	base := info.Size()
	if refSize >= 0 {
		base = refSize
	}
	target, err := apply(how, base, size)
	if err != nil {
		t.diag.Errorf("overflow extending size of file '%s'", path)
		return
	}
	if how != opSet && !info.Mode().IsRegular() && refSize < 0 {
		// GNU only knows the size of a regular file; others are left as is.
		t.diag.Errorf("cannot get the size of '%s': %s", path, "Invalid argument")
		return
	}
	if err := f.Truncate(target); err != nil {
		t.diag.Errorf("failed to truncate '%s' at %d bytes: %s", path, target, core.ErrText(err))
	}
}

// apply computes the new size from the current one.
func apply(how op, cur, n int64) (int64, error) {
	switch how {
	case opExtend:
		if cur > math.MaxInt64-n {
			return 0, errOverflow
		}
		return cur + n, nil
	case opReduce:
		return max(cur-n, 0), nil
	case opAtMost:
		return min(cur, n), nil
	case opAtLeast:
		return max(cur, n), nil
	case opRoundDn:
		return cur / n * n, nil
	case opRoundUp:
		r := (cur + n - 1) / n
		if cur > math.MaxInt64-n+1 || r > math.MaxInt64/n {
			return 0, errOverflow
		}
		return r * n, nil
	}
	return n, nil
}

var errOverflow = errors.New("overflow")

// parseSize splits "[+-<>/%]NUMBER[SUFFIX]" into its operator and count.
func parseSize(s string) (op, int64, error) {
	how := opSet
	if s != "" {
		switch c := s[0]; c {
		case '+', '-', '<', '>', '/', '%':
			how = op(c)
			s = s[1:]
		}
	}
	n, err := head.ParseSize(s)
	if err != nil {
		return 0, 0, fmt.Errorf("parse size: %w", err)
	}
	return how, n, nil
}
