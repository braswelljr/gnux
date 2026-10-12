// Package fold implements the engine behind foldx, a GNU-compatible
// implementation of the GNU/UNIX fold command: wrap each input line to fit in
// a specified width.
package fold

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "foldx"

// DefaultWidth is the line width used when Options.Width is zero.
const DefaultWidth = 80

// tabWidth is the distance between tab stops when counting columns.
const tabWidth = 8

// Options mirrors the GNU fold flag surface. The zero value is plain fold:
// lines wrapped at 80 columns, counting display columns.
type Options struct {
	// Bytes (-b) counts bytes rather than columns, so a tab, backspace or
	// carriage return takes one position like any other byte.
	Bytes bool
	// Spaces (-s) breaks a long line after its last blank within the width,
	// rather than exactly at the width.
	Spaces bool
	// Width (-w) is the maximum line width; zero means DefaultWidth.
	Width int
}

// Fold is the engine. Construct it with New.
type Fold struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a fold engine reading from and writing to s.
func New(opts Options, s core.Streams) *Fold {
	if opts.Width <= 0 {
		opts.Width = DefaultWidth
	}
	return &Fold{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run folds each operand in turn; no operands, or "-", means standard input.
// A failing operand is reported and the rest still run. The returned error is
// core.ErrFailed if any operand failed, otherwise nil.
func (f *Fold) Run(args []string) error {
	f.out = bufio.NewWriter(f.streams.Out)
	if len(args) == 0 {
		args = []string{"-"}
	}
	for _, name := range args {
		if !f.file(name) {
			break
		}
	}
	return f.diag.Result()
}

// file folds one operand. It returns false when output can no longer be
// written, which ends the run.
func (f *Fold) file(name string) bool {
	var r io.Reader
	if name == "-" {
		r = f.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		file, err := os.Open(name)
		if err != nil {
			f.diag.Errorf("%s: %s", name, core.ErrText(err))
			return true
		}
		defer file.Close()
		if info, err := file.Stat(); err == nil && info.IsDir() {
			f.diag.Errorf("%s: %s", name, core.ErrText(errIsDir))
			return true
		}
		r = file
	}

	err := f.fold(bufio.NewReader(r))
	if werr := f.out.Flush(); werr != nil {
		if !closedPipe(werr) {
			f.diag.Errorf("write error: %s", core.ErrText(werr))
		}
		return false
	}
	if err != nil {
		f.diag.Errorf("%s: %s", name, core.ErrText(err))
	}
	return true
}

// column returns the column after c is written at column col.
func (f *Fold) column(col int, c byte) int {
	if f.Bytes {
		return col + 1
	}
	switch c {
	case '\b':
		if col > 0 {
			return col - 1
		}
		return 0
	case '\r':
		return 0
	case '\t':
		return col + tabWidth - col%tabWidth
	}
	return col + 1
}

// fold copies r to the output, breaking lines wider than the width. It is a
// port of GNU fold's fold_file, byte for byte.
func (f *Fold) fold(r *bufio.Reader) error {
	var line []byte
	col := 0
	for {
		c, err := r.ReadByte()
		if err != nil {
			if len(line) > 0 {
				_, _ = f.out.Write(line)
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if c == '\n' {
			line = append(line, c)
			_, _ = f.out.Write(line)
			line, col = line[:0], 0
			continue
		}

	rescan:
		col = f.column(col, c)
		if col > f.Width {
			// With -s, break after the last blank seen on this line.
			if f.Spaces {
				if end := lastBlank(line); end >= 0 {
					end++
					_, _ = f.out.Write(line[:end])
					_ = f.out.WriteByte('\n')
					line = append(line[:0], line[end:]...)
					col = 0
					for _, b := range line {
						col = f.column(col, b)
					}
					goto rescan
				}
			}
			// A single character wider than the line still goes out alone.
			if len(line) == 0 {
				line = append(line, c)
				continue
			}
			line = append(line, '\n')
			_, _ = f.out.Write(line)
			line, col = line[:0], 0
			goto rescan
		}
		line = append(line, c)
	}
}

// lastBlank returns the index of the last space or tab in line, or -1.
func lastBlank(line []byte) int {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ' ' || line[i] == '\t' {
			return i
		}
	}
	return -1
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

// closedPipe reports whether err means the reader of the output has gone,
// which ends the run quietly.
func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
