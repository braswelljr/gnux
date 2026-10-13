// Package comm implements the engine behind commx, a GNU-compatible
// implementation of the GNU/UNIX comm command: compare two sorted files line
// by line.
package comm

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "commx"

// OrderCheck says whether the inputs' sort order is verified.
type OrderCheck int

const (
	// CheckDefault warns, once per file, about disorder only once a line has
	// been found in one file alone, and then fails at the end (GNU's default).
	CheckDefault OrderCheck = iota
	// CheckOrder (--check-order) fails at the first line out of order.
	CheckOrder
	// NoCheckOrder (--nocheck-order) never checks.
	NoCheckOrder
)

// Options mirrors the GNU comm flag surface. The zero value is plain comm:
// three columns separated by TABs.
type Options struct {
	// Suppress1, Suppress2 and Suppress3 (-1, -2, -3) hide the lines unique
	// to FILE1, unique to FILE2, and common to both.
	Suppress1, Suppress2, Suppress3 bool
	// Check selects the order check.
	Check OrderCheck
	// OutputDelimiter (--output-delimiter=STR) separates the columns; it
	// applies only when OutputDelimiterSet is true. An empty string writes
	// a NUL byte, as in GNU.
	OutputDelimiter string
	// OutputDelimiterSet records that --output-delimiter was given.
	OutputDelimiterSet bool
	// Total (--total) ends the output with a summary line of the counts.
	Total bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
}

// Comm is the engine. Construct it with New.
type Comm struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	eol     byte
	sep     string
}

// New builds a comm engine reading from and writing to s.
func New(opts Options, s core.Streams) *Comm {
	c := &Comm{Options: opts, streams: s, diag: core.NewDiag(Name, s), eol: '\n', sep: "\t"}
	if opts.ZeroTerminated {
		c.eol = 0
	}
	if opts.OutputDelimiterSet {
		c.sep = opts.OutputDelimiter
		if c.sep == "" {
			c.sep = "\x00"
		}
	}
	return c
}

// input is one of the two files being compared.
type input struct {
	name     string
	r        *bufio.Reader
	line     []byte // current line, without its delimiter
	ok       bool   // line is valid
	prev     []byte
	hasPrev  bool
	warned   bool
	closer   io.Closer
	readFail bool
}

// Run compares FILE1 and FILE2 (args); either may be "-" for standard input.
// The returned error is a *core.UsageError for a wrong number of operands,
// core.ErrFailed when a file failed or the input was out of order, otherwise
// nil.
func (c *Comm) Run(args []string) error {
	switch len(args) {
	case 0:
		return core.Usagef("missing operand")
	case 1:
		return core.Usagef("missing operand after '%s'", args[0])
	case 2:
	default:
		return core.Usagef("extra operand '%s'", args[2])
	}

	var in [2]*input
	for i, name := range args {
		var r io.Reader
		var closer io.Closer
		if name == "-" {
			r = c.streams.In
			if r == nil {
				r = strings.NewReader("")
			}
		} else {
			f, err := os.Open(name)
			if err != nil {
				c.diag.Errorf("%s: %s", name, core.ErrText(err))
				if in[0] != nil && in[0].closer != nil {
					in[0].closer.Close()
				}
				return c.diag.Result()
			}
			r, closer = f, f
		}
		in[i] = &input{name: name, r: core.NewReader(r), closer: closer}
	}
	defer func() {
		for _, f := range in {
			if f.closer != nil {
				f.closer.Close()
			}
		}
	}()

	c.out = core.NewWriter(c.streams.Out)
	err := c.compare(in)
	if ferr := c.out.Flush(); ferr != nil && !core.IsClosedPipe(ferr) {
		c.diag.Errorf("write error: %s", core.ErrText(ferr))
	}
	if err != nil {
		return err
	}
	if in[0].warned || in[1].warned {
		c.diag.Errorf("input is not in sorted order")
	}
	return c.diag.Result()
}

// errDisorder stops the comparison under --check-order; the diagnostic has
// already been written.
var errDisorder = errors.New("disorder")

func (c *Comm) compare(in [2]*input) error {
	var total [3]uint64
	seenUnpairable := false

	advance := func(i int) error {
		f := in[i]
		if f.ok {
			f.prev = append(f.prev[:0], f.line...)
			f.hasPrev = true
		}
		f.ok = c.readLine(f)
		if !f.ok || !f.hasPrev || c.Check == NoCheckOrder || f.warned {
			return nil
		}
		if c.Check != CheckOrder && !seenUnpairable {
			return nil
		}
		if bytes.Compare(f.prev, f.line) > 0 {
			c.diag.Errorf("file %d is not in sorted order", i+1)
			if c.Check == CheckOrder {
				return errDisorder
			}
			f.warned = true
		}
		return nil
	}

	for i := range in {
		if err := advance(i); err != nil {
			return core.ErrFailed
		}
	}
	for in[0].ok || in[1].ok {
		var order int
		switch {
		case !in[0].ok:
			order = 1
		case !in[1].ok:
			order = -1
		default:
			order = bytes.Compare(in[0].line, in[1].line)
		}

		switch {
		case order == 0:
			total[2]++
			if !c.Suppress3 {
				if !c.Suppress1 {
					_, _ = c.out.WriteString(c.sep)
				}
				if !c.Suppress2 {
					_, _ = c.out.WriteString(c.sep)
				}
				c.writeLine(in[1].line)
			}
		case order < 0:
			seenUnpairable = true
			total[0]++
			if !c.Suppress1 {
				c.writeLine(in[0].line)
			}
		default:
			seenUnpairable = true
			total[1]++
			if !c.Suppress2 {
				if !c.Suppress1 {
					_, _ = c.out.WriteString(c.sep)
				}
				c.writeLine(in[1].line)
			}
		}

		// Step the file the line came from, or both when they matched; file 1
		// first, as GNU does, so the disorder diagnostics come in its order.
		if order <= 0 {
			if err := advance(0); err != nil {
				return core.ErrFailed
			}
		}
		if order >= 0 {
			if err := advance(1); err != nil {
				return core.ErrFailed
			}
		}
	}
	for _, f := range in {
		if f.readFail {
			return core.ErrFailed
		}
	}

	if c.Total {
		for i, n := range total {
			_, _ = c.out.WriteString(strconv.FormatUint(n, 10))
			if i < 2 {
				_, _ = c.out.WriteString(c.sep)
			}
		}
		_, _ = c.out.WriteString(c.sep)
		_, _ = c.out.WriteString("total")
		_ = c.out.WriteByte(c.eol)
	}
	return nil
}

func (c *Comm) writeLine(line []byte) {
	_, _ = c.out.Write(line)
	_ = c.out.WriteByte(c.eol)
}

// readLine loads the next line of f into f.line. It reports false at the end
// of the input or on a read error, which is diagnosed.
func (c *Comm) readLine(f *input) bool {
	line, err := f.r.ReadBytes(c.eol)
	if len(line) > 0 && line[len(line)-1] == c.eol {
		f.line = append(f.line[:0], line[:len(line)-1]...)
		return true
	}
	if err != nil && !errors.Is(err, io.EOF) {
		c.diag.Errorf("%s: %s", f.name, core.ErrText(err))
		f.readFail = true
		return false
	}
	f.line = append(f.line[:0], line...)
	return len(line) > 0
}
