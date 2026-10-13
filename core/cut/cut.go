// Package cut implements the engine behind cutx, a GNU-compatible
// implementation of the GNU/UNIX cut command: remove sections from each line
// of files.
package cut

import (
	"bufio"
	"errors"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "cutx"

// Mode says what the list selects (-b, -c or -f).
type Mode int

const (
	// ModeNone means no list was given, which Run refuses.
	ModeNone Mode = iota
	// ModeBytes (-b) selects byte positions.
	ModeBytes
	// ModeChars (-c) selects character positions. As in GNU, a character is
	// a byte.
	ModeChars
	// ModeFields (-f) selects fields separated by the delimiter.
	ModeFields
)

// Options mirrors the GNU cut flag surface. A list is required, so the zero
// value is refused with GNU's usage error, as plain cut is.
type Options struct {
	// Mode selects what List counts.
	Mode Mode
	// List is the LIST argument: positions or ranges (N, N-, N-M, -M)
	// separated by commas or blanks.
	List string
	// Delimiter (-d) is the one-byte field delimiter; empty means TAB. Pass
	// "\x00" for NUL, which is what GNU makes of -d ''.
	Delimiter string
	// OnlyDelimited (-s) drops lines that contain no delimiter.
	OnlyDelimited bool
	// Complement (--complement) selects everything the list does not.
	Complement bool
	// OutputDelimiter (--output-delimiter) is written between the selected
	// fields or ranges; it applies only when OutputDelimiterSet is true, so
	// that an empty delimiter can be asked for.
	OutputDelimiter string
	// OutputDelimiterSet records that --output-delimiter was given.
	OutputDelimiterSet bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
}

// span is one selected range, 1-based and inclusive.
type span struct{ lo, hi int64 }

// Cut is the engine. Construct it with New.
type Cut struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	spans   []span
	delim   byte // field delimiter
	eol     byte // line delimiter
	outSep  string
}

// New builds a cut engine reading from and writing to s.
func New(opts Options, s core.Streams) *Cut {
	return &Cut{
		Options: opts,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run cuts each operand; no operands, or "-", means standard input. Invalid
// options are a *core.UsageError; a failing operand is reported and the rest
// still run, and the result is then core.ErrFailed.
func (c *Cut) Run(args []string) error {
	if err := c.prepare(); err != nil {
		return err
	}
	if len(args) == 0 {
		args = []string{"-"}
	}
	c.out = core.NewWriter(c.streams.Out)
	for _, name := range args {
		if !c.file(name) {
			break
		}
	}
	return c.diag.Result()
}

// prepare validates the options in GNU's order and parses the list.
func (c *Cut) prepare() error {
	if c.Delimiter != "" && len(c.Delimiter) != 1 {
		return core.Usagef("the delimiter must be a single character")
	}
	if c.Mode != ModeNone {
		spans, err := parseList(c.List, c.Mode == ModeFields)
		if err != nil {
			return err
		}
		c.spans = spans
	}
	switch {
	case c.Mode == ModeNone:
		return core.Usagef("you must specify a list of bytes, characters, or fields")
	case c.Delimiter != "" && c.Mode != ModeFields:
		return core.Usagef("an input delimiter may be specified only when operating on fields")
	case c.OnlyDelimited && c.Mode != ModeFields:
		return core.Usagef("suppressing non-delimited lines makes sense\n\tonly when operating on fields")
	}

	if c.Complement {
		c.spans = complement(c.spans)
	}
	c.delim = '\t'
	if c.Delimiter != "" {
		c.delim = c.Delimiter[0]
	}
	c.eol = '\n'
	if c.ZeroTerminated {
		c.eol = 0
	}
	switch {
	case c.OutputDelimiterSet:
		c.outSep = c.OutputDelimiter
	case c.Mode == ModeFields:
		c.outSep = string(c.delim)
	}
	return nil
}

// parseList turns LIST into sorted, merged spans, with GNU's diagnostics.
func parseList(list string, fields bool) ([]span, error) {
	numbered := "byte/character positions are numbered from 1"
	invalidRange := "invalid byte or character range"
	invalidValue := "invalid byte/character position '%s'"
	tooLarge := "byte/character offset '%s' is too large"
	missing := "missing list of byte/character positions"
	if fields {
		numbered = "fields are numbered from 1"
		invalidRange = "invalid field range"
		invalidValue = "invalid field value '%s'"
		tooLarge = "field number '%s' is too large"
		missing = "missing list of fields"
	}

	var spans []span
	items := strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
	for _, item := range items {
		if i := strings.IndexFunc(item, func(r rune) bool { return r != '-' && (r < '0' || r > '9') }); i >= 0 {
			return nil, core.Usagef(invalidValue, item[i:])
		}
		number := func(s string) (int64, error) {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, core.Usagef(tooLarge, s)
			}
			if n == 0 {
				return 0, core.Usagef("%s", numbered)
			}
			return n, nil
		}

		lo, hi, isRange := strings.Cut(item, "-")
		var sp span
		var err error
		switch {
		case !isRange:
			if sp.lo, err = number(lo); err != nil {
				return nil, err
			}
			sp.hi = sp.lo
		case lo == "" && hi == "":
			return nil, core.Usagef("invalid range with no endpoint: -")
		case strings.Contains(hi, "-"):
			return nil, core.Usagef("%s", invalidRange)
		default:
			sp.lo, sp.hi = 1, math.MaxInt64
			if lo != "" {
				if sp.lo, err = number(lo); err != nil {
					return nil, err
				}
			}
			if hi != "" {
				if sp.hi, err = number(hi); err != nil {
					return nil, err
				}
			}
			if sp.hi < sp.lo {
				return nil, core.Usagef("invalid decreasing range")
			}
		}
		spans = append(spans, sp)
	}
	if len(spans) == 0 {
		return nil, core.Usagef("%s", missing)
	}

	// Sort, then merge overlapping spans. Adjacent ones stay apart, which is
	// where --output-delimiter goes for -b and -c, as in GNU.
	slices.SortFunc(spans, func(a, b span) int {
		switch {
		case a.lo < b.lo:
			return -1
		case a.lo > b.lo:
			return 1
		}
		return 0
	})
	merged := spans[:1]
	for _, sp := range spans[1:] {
		last := &merged[len(merged)-1]
		if sp.lo <= last.hi {
			last.hi = max(last.hi, sp.hi)
			continue
		}
		merged = append(merged, sp)
	}
	return merged, nil
}

// complement returns the spans of [1, ∞) not covered by spans.
func complement(spans []span) []span {
	var out []span
	next := int64(1)
	for _, sp := range spans {
		if sp.lo > next {
			out = append(out, span{next, sp.lo - 1})
		}
		if sp.hi == math.MaxInt64 {
			return out
		}
		next = sp.hi + 1
	}
	return append(out, span{next, math.MaxInt64})
}

// file cuts one operand. It returns false when standard output is gone.
func (c *Cut) file(name string) bool {
	var r io.Reader
	if name == "-" {
		r = c.streams.In
		if r == nil {
			r = strings.NewReader("")
		}
	} else {
		f, err := os.Open(name)
		if err != nil {
			c.diag.Errorf("%s: %s", name, core.ErrText(err))
			return true
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			c.diag.Errorf("%s: %s", name, core.ErrText(errIsDir))
			return true
		}
		r = f
	}

	err := c.lines(core.NewReader(r))
	if werr := c.out.Flush(); werr != nil {
		if !core.IsClosedPipe(werr) {
			c.diag.Errorf("write error: %s", core.ErrText(werr))
		}
		return false
	}
	if err != nil {
		c.diag.Errorf("%s: %s", name, core.ErrText(err))
	}
	return true
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

// lines cuts every line of r. Each output line ends with the line
// delimiter, even when the input's last line did not.
func (c *Cut) lines(r *bufio.Reader) error {
	for {
		line, err := r.ReadBytes(c.eol)
		if len(line) > 0 {
			if line[len(line)-1] == c.eol {
				line = line[:len(line)-1]
			}
			if c.Mode == ModeFields {
				c.fields(line)
			} else {
				c.bytes(line)
			}
			core.FlushIfIdle(c.out, r)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// bytes writes the selected byte positions of line.
func (c *Cut) bytes(line []byte) {
	n := int64(len(line))
	printed := false
	for _, sp := range c.spans {
		if sp.lo > n {
			break
		}
		if printed && c.OutputDelimiterSet {
			_, _ = c.out.WriteString(c.outSep)
		}
		_, _ = c.out.Write(line[sp.lo-1 : min(sp.hi, n)])
		printed = true
	}
	_ = c.out.WriteByte(c.eol)
}

// fields writes the selected fields of line, or the whole line when it
// holds no delimiter (unless -s).
func (c *Cut) fields(line []byte) {
	if !slices.Contains(line, c.delim) {
		if !c.OnlyDelimited {
			_, _ = c.out.Write(line)
			_ = c.out.WriteByte(c.eol)
		}
		return
	}

	printed := false
	field := int64(1)
	s := 0
	for start := 0; start <= len(line); field++ {
		end := start
		for end < len(line) && line[end] != c.delim {
			end++
		}
		for s < len(c.spans) && c.spans[s].hi < field {
			s++
		}
		if s == len(c.spans) {
			break
		}
		if c.spans[s].lo <= field {
			if printed {
				_, _ = c.out.WriteString(c.outSep)
			}
			_, _ = c.out.Write(line[start:end])
			printed = true
		}
		start = end + 1
	}
	_ = c.out.WriteByte(c.eol)
}
