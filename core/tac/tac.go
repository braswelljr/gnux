// Package tac implements the engine behind tacx, a GNU-compatible
// implementation of the GNU/UNIX tac command: concatenate and print files in
// reverse, last record first.
package tac

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "tacx"

// Options mirrors the GNU tac flag surface. The zero value is plain tac:
// lines, separated by a newline that ends each one, printed last first.
type Options struct {
	// Before (-b) attaches the separator to the start of the record that
	// follows it rather than to the end of the one before.
	Before bool
	// Regex (-r) treats Separator as a regular expression.
	Regex bool
	// Separator (-s) is the record separator; empty means a newline, unless
	// SeparatorSet says it was given explicitly (an empty separator is an
	// error, as in GNU).
	Separator string
	// SeparatorSet records that -s was given.
	SeparatorSet bool
}

// Tac is the engine. Construct it with New.
type Tac struct {
	Options

	streams core.Streams
	diag    *core.Diag
	re      *regexp.Regexp
}

// New builds a tac engine reading from and writing to s.
func New(opts Options, s core.Streams) *Tac {
	if !opts.SeparatorSet {
		opts.Separator = "\n"
	}
	return &Tac{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints each operand reversed, in order; no operands, or "-", means
// standard input. A failing operand is reported and the rest still run. The
// returned error is core.ErrFailed if anything failed, otherwise nil.
func (t *Tac) Run(args []string) error {
	if t.Separator == "" {
		t.diag.Errorf("separator cannot be empty")
		return core.ErrFailed
	}
	if t.Regex {
		re, err := regexp.Compile(t.Separator)
		if err != nil {
			t.diag.Errorf("%s", err.Error())
			return core.ErrFailed
		}
		re.Longest()
		t.re = re
	}
	if len(args) == 0 {
		args = []string{"-"}
	}

	out := core.NewWriter(t.streams.Out)
	for _, name := range args {
		data, ok := t.read(name)
		if !ok {
			continue
		}
		t.reverse(out, data)
		if err := out.Flush(); err != nil {
			if !closedPipe(err) {
				t.diag.Errorf("write error: %s", core.ErrText(err))
			}
			return core.ErrFailed
		}
	}
	return t.diag.Result()
}

// read returns the whole of one operand.
func (t *Tac) read(name string) ([]byte, bool) {
	var r io.Reader
	if name == "-" {
		r = t.streams.In
		if r == nil {
			return nil, true
		}
	} else {
		f, err := os.Open(name)
		if err != nil {
			t.diag.Errorf("failed to open '%s' for reading: %s", name, core.ErrText(err))
			return nil, false
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			t.diag.Errorf("%s: read error: %s", name, core.ErrText(errIsDir))
			return nil, false
		}
		r = f
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.diag.Errorf("%s: read error: %s", name, core.ErrText(err))
		return nil, false
	}
	return data, true
}

// separators returns the [start, end) of each separator in data, in order.
// A fixed separator is searched from the end backward, as GNU does, so
// overlapping occurrences resolve the same way.
func (t *Tac) separators(data []byte) [][2]int {
	var found [][2]int
	if t.re != nil {
		for _, m := range t.re.FindAllIndex(data, -1) {
			if m[1] > m[0] {
				found = append(found, [2]int{m[0], m[1]})
			}
		}
		return found
	}
	sep := []byte(t.Separator)
	end := len(data)
	for {
		i := bytes.LastIndex(data[:end], sep)
		if i < 0 {
			break
		}
		found = append(found, [2]int{i, i + len(sep)})
		end = i
	}
	for i, j := 0, len(found)-1; i < j; i, j = i+1, j-1 {
		found[i], found[j] = found[j], found[i]
	}
	return found
}

// reverse writes data's records last first. A record ends with its
// separator, or with -b starts with it.
func (t *Tac) reverse(w *bufio.Writer, data []byte) {
	seps := t.separators(data)
	bounds := make([]int, 0, len(seps)+2)
	bounds = append(bounds, 0)
	for _, s := range seps {
		if t.Before {
			bounds = append(bounds, s[0])
		} else {
			bounds = append(bounds, s[1])
		}
	}
	bounds = append(bounds, len(data))
	for i := len(bounds) - 1; i > 0; i-- {
		_, _ = w.Write(data[bounds[i-1]:bounds[i]])
	}
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
