// Package csplit implements the engine behind csplitx, a GNU-compatible
// implementation of the GNU/UNIX csplit command: split a file into sections
// determined by context lines.
package csplit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/bre"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "csplitx"

// Defaults GNU csplit uses when an option is not given.
const (
	DefaultPrefix = "xx"
	DefaultDigits = 2
)

// Options mirrors the GNU csplit flag surface. The zero value is plain
// csplit: files xx00, xx01, ..., each one's size printed, and every file
// removed again if anything fails.
type Options struct {
	// Prefix (-f) starts every file name: DefaultPrefix unless PrefixSet
	// (an empty prefix leaves just the numbers).
	Prefix    string
	PrefixSet bool
	// SuffixFormat (-b) is a printf format with one integer conversion, used
	// instead of Digits zero-padded digits.
	SuffixFormat string
	// Digits (-n) is the number of digits in a file number; zero means
	// DefaultDigits.
	Digits int
	// KeepFiles (-k) keeps the files already made when something fails.
	KeepFiles bool
	// Quiet (-s, -q) does not print the size of each file.
	Quiet bool
	// ElideEmpty (-z) removes empty files.
	ElideEmpty bool
	// SuppressMatched (--suppress-matched) leaves out the lines that matched
	// a pattern.
	SuppressMatched bool
}

// control is one pattern from the command line.
type control struct {
	arg           string // as written, for diagnostics
	re            *regexp.Regexp
	ignore        bool // %REGEXP%: skip the section rather than write it
	offset        int64
	lines         int64 // a line number pattern
	repeat        int64
	repeatForever bool
}

// Csplit is the engine. Construct it with New.
type Csplit struct {
	Options

	streams core.Streams
	diag    *core.Diag
	format  string

	in      *bufio.Reader
	eof     bool
	buf     [][]byte // lines read and not yet removed
	first   int64    // the line number of buf[0]
	current int64

	out     *bufio.Writer
	outFile *os.File
	outName string
	written int64
	created int
}

// New builds a csplit engine reading from and writing to s.
func New(opts Options, s core.Streams) *Csplit {
	return &Csplit{Options: opts, streams: s, diag: core.NewDiag(Name, s), first: 1}
}

// errFatal ends the run; the diagnostic has been written and, unless
// KeepFiles, the files made so far are removed.
var errFatal = errors.New("fatal")

// Run splits FILE (args[0]; "-" is standard input) at the PATTERNs that
// follow: a line number N, /REGEXP/[OFFSET], %REGEXP%[OFFSET] (skip the
// section), each optionally followed by {N} or {*} to repeat it. It returns
// a *core.UsageError for missing operands, core.ErrFailed when anything
// failed, and nil otherwise.
func (c *Csplit) Run(args []string) error {
	switch len(args) {
	case 0:
		return core.Usagef("missing operand")
	case 1:
		return core.Usagef("missing operand after '%s'", args[0])
	}
	if !c.PrefixSet {
		c.Prefix = DefaultPrefix
	}
	if c.Digits == 0 {
		c.Digits = DefaultDigits
	}
	if c.SuffixFormat != "" {
		f, err := suffixFormat(c.SuffixFormat)
		if err != nil {
			c.diag.Errorf("%s", err)
			return core.ErrFailed
		}
		c.format = f
	}

	r := c.streams.In
	if args[0] != "-" {
		f, err := os.Open(args[0])
		if err != nil {
			c.diag.Errorf("cannot open '%s' for reading: %s", args[0], core.ErrText(err))
			return core.ErrFailed
		}
		defer f.Close()
		r = f
	}
	if r == nil {
		r = strings.NewReader("")
	}
	c.in = bufio.NewReaderSize(r, 64*1024)

	controls, err := c.parse(args[1:])
	if err != nil {
		return core.ErrFailed
	}
	if err := c.split(controls); err != nil {
		_ = c.closeOutput(true)
		if !c.KeepFiles {
			for i := c.created - 1; i >= 0; i-- {
				name := c.filename(i)
				if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
					c.diag.Errorf("%s: %s", name, core.ErrText(err))
				}
			}
		}
		return core.ErrFailed
	}
	return c.diag.Result()
}

// parse reads the patterns, in GNU's words when one is wrong.
func (c *Csplit) parse(args []string) ([]*control, error) {
	var controls []*control
	var last int64
	fail := func(format string, a ...any) ([]*control, error) {
		c.diag.Errorf(format, a...)
		return nil, errFatal
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		p := &control{arg: arg}
		if arg != "" && (arg[0] == '/' || arg[0] == '%') {
			delim := arg[0]
			end := strings.LastIndexByte(arg[1:], delim)
			if end < 0 {
				return fail("%s: closing delimiter '%c' missing", arg, delim)
			}
			end++
			re, err := bre.Compile(arg[1:end])
			if err != nil {
				return fail("'%s': invalid regular expression: %s", arg, err)
			}
			p.re, p.ignore = re, delim == '%'
			if rest := arg[end+1:]; rest != "" {
				off, err := strconv.ParseInt(rest, 10, 64)
				if err != nil {
					return fail("'%s': integer expected after delimiter", arg)
				}
				p.offset = off
			}
		} else {
			n, err := strconv.ParseUint(arg, 10, 63)
			switch {
			case err != nil:
				return fail("'%s': invalid pattern", arg)
			case n == 0:
				return fail("%s: line number must be greater than zero", arg)
			case int64(n) < last:
				return fail("line number '%s' is smaller than preceding line number, %d", arg, last)
			case int64(n) == last:
				c.diag.Warnf("warning: line number '%s' is the same as preceding line number", arg)
			}
			last = int64(n)
			p.lines = int64(n)
		}
		if i+1 < len(args) && strings.HasPrefix(args[i+1], "{") {
			i++
			rep := args[i]
			if !strings.HasSuffix(rep, "}") {
				return fail("'%s': '}' is required in repeat count", rep)
			}
			body := rep[1 : len(rep)-1]
			if body == "*" {
				p.repeatForever = true
			} else {
				n, err := strconv.ParseUint(body, 10, 63)
				if err != nil {
					return fail("'%s': integer required between '{' and '}'", rep)
				}
				p.repeat = int64(n)
			}
		}
		controls = append(controls, p)
	}
	return controls, nil
}

// suffixFormat checks a -b format as GNU does and turns it into Go's.
func suffixFormat(f string) (string, error) {
	var b strings.Builder
	seen := false
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			b.WriteByte(f[i])
			continue
		}
		if i+1 < len(f) && f[i+1] == '%' {
			b.WriteString("%%")
			i++
			continue
		}
		if seen {
			return "", errors.New("too many % conversion specifications in suffix")
		}
		seen = true
		b.WriteByte('%')
		i++
		thousands, alt := false, false
		for ; i < len(f) && strings.IndexByte("-0'#", f[i]) >= 0; i++ {
			switch f[i] {
			case '\'':
				thousands = true // grouping is a no-op in the C locale
			case '#':
				alt = true
				b.WriteByte('#')
			default:
				b.WriteByte(f[i])
			}
		}
		for ; i < len(f) && (f[i] >= '0' && f[i] <= '9' || f[i] == '.'); i++ {
			b.WriteByte(f[i])
		}
		if i >= len(f) {
			return "", errors.New("missing conversion specifier in suffix")
		}
		ch := f[i]
		switch ch {
		case 'd', 'i', 'u':
			if alt {
				return "", fmt.Errorf("invalid flags in conversion specification: %%#%c", ch)
			}
			b.WriteByte('d')
		case 'o', 'x', 'X':
			if thousands {
				return "", fmt.Errorf("invalid flags in conversion specification: %%'%c", ch)
			}
			b.WriteByte(ch)
		default:
			if ch >= ' ' && ch < 0x7f {
				return "", fmt.Errorf("invalid conversion specifier in suffix: %c", ch)
			}
			return "", fmt.Errorf("invalid conversion specifier in suffix: \\%03o", ch)
		}
	}
	if !seen {
		return "", errors.New("missing % conversion specification in suffix")
	}
	return b.String(), nil
}

func (c *Csplit) filename(n int) string {
	if c.format != "" {
		return c.Prefix + fmt.Sprintf(c.format, n)
	}
	return c.Prefix + fmt.Sprintf("%0*d", c.Digits, n)
}

// split carries out the patterns, as GNU's split_file does.
func (c *Csplit) split(controls []*control) error {
	for _, p := range controls {
		for j := int64(0); p.repeatForever || j <= p.repeat; j++ {
			var err error
			done := false
			if p.re != nil {
				done, err = c.regexp(p, j)
			} else {
				err = c.lineCount(p, j)
			}
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
	if err := c.createOutput(); err != nil {
		return err
	}
	c.dumpRest()
	return c.closeOutput(false)
}

// Input buffer, after GNU's: lines are read on demand and kept until they
// are removed, so that a negative offset can reach back before a match.

func (c *Csplit) readLine() bool {
	if c.eof {
		return false
	}
	line, err := c.in.ReadBytes('\n')
	if len(line) > 0 {
		c.buf = append(c.buf, line)
	}
	if err != nil {
		c.eof = true
	}
	return len(line) > 0
}

// find returns line n, reading as far as needed, or nil.
func (c *Csplit) find(n int64) []byte {
	if n < c.first {
		return nil
	}
	for n-c.first >= int64(len(c.buf)) {
		if !c.readLine() {
			return nil
		}
	}
	return c.buf[n-c.first]
}

// firstLine is the number of the first line not yet removed, or 0 at the
// end of the input.
func (c *Csplit) firstLine() int64 {
	if len(c.buf) == 0 && !c.readLine() {
		return 0
	}
	return c.first
}

// remove takes the first line out of the buffer, or returns nil at the end.
func (c *Csplit) remove() []byte {
	if len(c.buf) == 0 && !c.readLine() {
		return nil
	}
	if c.current < c.first {
		c.current = c.first
	}
	line := c.buf[0]
	c.buf[0] = nil
	c.buf = c.buf[1:]
	c.first++
	return line
}

func (c *Csplit) noMoreLines() bool { return c.find(c.current+1) == nil }

func (c *Csplit) createOutput() error {
	name := c.filename(c.created)
	f, err := os.Create(name)
	if err != nil {
		c.diag.Errorf("%s: %s", name, core.ErrText(err))
		return errFatal
	}
	c.created++
	c.outFile, c.outName, c.written = f, name, 0
	c.out = bufio.NewWriter(f)
	return nil
}

// closeOutput closes the current file and prints its size; an empty file
// is removed under ElideEmpty. When failing is set the run is ending: the
// file is closed silently, to be removed with the others.
func (c *Csplit) closeOutput(failing bool) error {
	if c.outFile == nil {
		return nil
	}
	f, name := c.outFile, c.outName
	c.outFile = nil
	err := c.out.Flush()
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if failing {
		return nil
	}
	if err != nil {
		c.diag.Errorf("%s: %s", name, core.ErrText(err))
		return errFatal
	}
	if c.written == 0 && c.ElideEmpty {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			c.diag.Errorf("%s: %s", name, core.ErrText(err))
		}
		c.created--
		return nil
	}
	if !c.Quiet {
		fmt.Fprintf(c.streams.Out, "%d\n", c.written)
	}
	return nil
}

func (c *Csplit) save(line []byte) {
	_, _ = c.out.Write(line)
	c.written += int64(len(line))
}

func (c *Csplit) dumpRest() {
	for line := c.remove(); line != nil; line = c.remove() {
		c.save(line)
	}
}

func (c *Csplit) lineError(p *control, repetition int64) error {
	msg := fmt.Sprintf("'%d': line number out of range", p.lines)
	if repetition > 0 {
		msg += fmt.Sprintf(" on repetition %d", repetition)
	}
	c.diag.Errorf("%s", msg)
	return errFatal
}

// lineCount writes the lines before line lines*(repetition+1) to a file.
func (c *Csplit) lineCount(p *control, repetition int64) error {
	last := p.lines * (repetition + 1)
	if err := c.createOutput(); err != nil {
		return err
	}
	if c.noMoreLines() && c.SuppressMatched {
		return c.lineError(p, repetition)
	}
	n := c.firstLine()
	if n == 0 {
		return c.lineError(p, repetition)
	}
	for ; n < last; n++ {
		line := c.remove()
		if line == nil {
			return c.lineError(p, repetition)
		}
		c.save(line)
	}
	if err := c.closeOutput(false); err != nil {
		return err
	}
	if c.SuppressMatched {
		c.remove()
	}
	if c.noMoreLines() && !c.SuppressMatched {
		return c.lineError(p, repetition)
	}
	return nil
}

// regexp writes the lines up to the next match (moved by the offset) to a
// file, or skips them for %REGEXP%. It reports done when {*} reached the end
// of the input, which ends the run successfully.
func (c *Csplit) regexp(p *control, repetition int64) (done bool, err error) {
	if !p.ignore {
		if err := c.createOutput(); err != nil {
			return false, err
		}
	}
	for {
		c.current++
		line := c.find(c.current)
		if line == nil {
			if p.repeatForever {
				if !p.ignore {
					c.dumpRest()
					return true, c.closeOutput(false)
				}
				return true, nil
			}
			msg := fmt.Sprintf("'%s': match not found", p.arg)
			if repetition > 0 {
				msg += fmt.Sprintf(" on repetition %d", repetition)
			}
			c.diag.Errorf("%s", msg)
			if !p.ignore {
				c.dumpRest()
				_ = c.closeOutput(false)
			}
			return false, errFatal
		}
		if p.re.Match(bytes.TrimSuffix(line, []byte("\n"))) {
			break
		}
		if p.offset >= 0 {
			// Lines before the match are final; write them as we go.
			line = c.remove()
			if !p.ignore {
				c.save(line)
			}
		}
	}

	breakLine := c.current + p.offset
	first := c.firstLine()
	if first == 0 || first > breakLine {
		c.diag.Errorf("'%s': line number out of range", p.arg)
		return false, errFatal
	}
	for n := first; n < breakLine; n++ {
		line := c.remove()
		if line == nil {
			c.diag.Errorf("'%s': line number out of range", p.arg)
			return false, errFatal
		}
		if !p.ignore {
			c.save(line)
		}
	}
	if !p.ignore {
		if err := c.closeOutput(false); err != nil {
			return false, err
		}
	}
	if p.offset > 0 {
		c.current = breakLine
	}
	if c.SuppressMatched {
		c.remove()
	}
	return false, nil
}
