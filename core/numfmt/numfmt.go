// Package numfmt implements the engine behind numfmtx, a GNU-compatible
// implementation of the GNU/UNIX numfmt command: convert numbers to and from
// human-readable strings such as 1.5K, 2.0Mi or 977k.
package numfmt

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "numfmtx"

// Scale is a way of writing a number with a unit suffix.
type Scale int

// The scales of --from and --to.
const (
	// ScaleNone takes and writes plain numbers.
	ScaleNone Scale = iota
	// ScaleAuto (--from only) reads 1K as 1000 and 1Ki as 1024.
	ScaleAuto
	// ScaleSI uses powers of 1000: 1k, 1M, ...
	ScaleSI
	// ScaleIEC uses powers of 1024 with a single letter: 1K, 1M, ...
	ScaleIEC
	// ScaleIECI uses powers of 1024 with an 'i': 1Ki, 1Mi, ...
	ScaleIECI
)

// Round is how a value is rounded to the digits that are printed.
type Round int

// The rounding methods of --round. The zero value is GNU's default.
const (
	RoundFromZero Round = iota
	RoundUp
	RoundDown
	RoundTowardsZero
	RoundNearest
)

// Invalid is what happens to input that is not a number numfmt can convert.
type Invalid int

// The modes of --invalid. The zero value is GNU's default.
const (
	// InvalidAbort reports the input and stops with status 2.
	InvalidAbort Invalid = iota
	// InvalidFail reports it, prints it unchanged and goes on; status 2.
	InvalidFail
	// InvalidWarn reports it, prints it unchanged and goes on; status 0.
	InvalidWarn
	// InvalidIgnore prints it unchanged without a word; status 0.
	InvalidIgnore
)

// ExitConversion is the status of a run in which some input could not be
// converted.
const ExitConversion = 2

// Options mirrors the GNU numfmt flag surface. The zero value is plain
// numfmt: numbers are checked and printed as they are.
type Options struct {
	// From (--from) is how input numbers are written; To (--to) how the
	// output is. ScaleAuto is not valid for To.
	From, To Scale
	// FromUnit (--from-unit) and ToUnit (--to-unit) are the sizes of the
	// input and output units; zero means 1.
	FromUnit, ToUnit uint64
	// Round (--round) is the rounding method.
	Round Round
	// Padding (--padding) pads each number to that width, on the left when
	// positive and on the right when negative. Zero pads a number that
	// starts after blanks, or is not in the first field, to the field's
	// original width, unless a Delimiter is set.
	Padding int64
	// Suffix (--suffix) is appended to output numbers and taken off input
	// numbers that end with it.
	Suffix string
	// Grouping (--grouping) groups the digits of a plain number as the
	// locale does; gnux uses the C locale, which does not group.
	Grouping bool
	// Delimiter (-d), when DelimiterSet, separates fields instead of blanks.
	Delimiter    byte
	DelimiterSet bool
	// Fields (--field) is the list of fields to convert, as cut takes it
	// ("1,3-5", "2-", "-" for all). Empty means the first field.
	Fields string
	// Debug (--debug) warns about input and options that may not do what
	// was meant.
	Debug bool
	// Header (--header) is the number of input lines printed unchanged.
	Header uint64
	// Format (--format) is a printf-style format with one %f directive.
	Format string
	// Invalid (--invalid) is what to do with input that cannot be converted.
	Invalid Invalid
	// ZeroTerminated (-z) ends lines with NUL rather than newline.
	ZeroTerminated bool
}

// Numfmt is the engine. Construct it with New.
type Numfmt struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	fields []span
	// What --format and --padding make of the output.
	prefix, suffixText string
	padding            int64
	zeroPad            int
	precision          int // -1: the input's own
	autoPad            bool
	exitOnError        bool
}

type span struct{ lo, hi uint64 }

// New builds a numfmt engine reading from and writing to s.
func New(opts Options, s core.Streams) *Numfmt {
	return &Numfmt{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// errAbort stops a run at the first invalid number, in the default
// --invalid=abort mode; the diagnostic is already written.
var errAbort = errors.New("abort")

// Run converts each operand, or each line of standard input when there are
// none.
func (n *Numfmt) Run(args []string) (err error) {
	if n.To == ScaleAuto {
		return core.Usagef("invalid argument 'auto' for '--to'\nValid arguments are:\n  - 'none'\n  - 'si'\n  - 'iec'\n  - 'iec-i'")
	}
	if n.Fields != "" {
		if n.fields, err = parseFields(n.Fields); err != nil {
			return err
		}
	}
	if n.Format != "" && n.Grouping {
		n.diag.Errorf("--grouping cannot be combined with --format")
		return n.diag.Result()
	}
	n.padding = n.Padding
	n.precision = -1
	if n.Debug && n.From == ScaleNone && n.To == ScaleNone && !n.Grouping && n.Padding == 0 && n.Format == "" {
		n.diag.Warnf("no conversion option specified")
	}
	if n.Format != "" {
		if msg := n.parseFormat(n.Format); msg != "" {
			n.diag.Errorf("%s", msg)
			return n.diag.Result()
		}
	}
	if n.Grouping {
		if n.To != ScaleNone {
			n.diag.Errorf("grouping cannot be combined with --to")
			return n.diag.Result()
		}
		if n.Debug {
			n.diag.Warnf("grouping has no effect in this locale")
		}
	}
	n.autoPad = n.padding == 0 && !n.DelimiterSet
	n.exitOnError = n.Invalid == InvalidAbort

	n.out = core.NewWriter(n.streams.Out)
	defer core.FlushOutput(n.out, n.diag, &err)

	valid := true
	if len(args) > 0 {
		if n.Debug && n.Header > 0 {
			n.diag.Warnf("--header ignored with command-line input")
		}
		for _, a := range args {
			ok, aborted := n.line([]byte(a), true)
			if aborted {
				return n.finish(false, true)
			}
			valid = valid && ok
		}
		return n.finish(valid, false)
	}

	delim := byte('\n')
	if n.ZeroTerminated {
		delim = 0
	}
	in := core.NewReader(n.streams.In)
	for h := n.Header; h > 0; h-- {
		line, rerr := in.ReadBytes(delim)
		if len(line) == 0 {
			if rerr != nil && !errors.Is(rerr, io.EOF) {
				n.diag.Errorf("error reading input: %s", core.ErrText(rerr))
				return n.diag.Result()
			}
			break
		}
		_, _ = n.out.Write(line)
	}
	for {
		line, rerr := in.ReadBytes(delim)
		if len(line) > 0 {
			newline := line[len(line)-1] == delim
			if newline {
				line = line[:len(line)-1]
			}
			ok, aborted := n.line(line, newline)
			if aborted {
				return n.finish(false, true)
			}
			valid = valid && ok
			core.FlushIfIdle(n.out, in)
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				n.diag.Errorf("error reading input: %s", core.ErrText(rerr))
				return n.diag.Result()
			}
			break
		}
	}
	return n.finish(valid, false)
}

func (n *Numfmt) finish(valid, aborted bool) error {
	if aborted {
		return &core.ExitError{Code: ExitConversion}
	}
	if n.Debug && !valid {
		n.diag.Warnf("failed to convert some of the input numbers")
	}
	if !valid && n.Invalid != InvalidWarn && n.Invalid != InvalidIgnore {
		return &core.ExitError{Code: ExitConversion}
	}
	return nil
}

// line converts the fields of one line. It reports whether every converted
// field was valid, and whether the run must stop (--invalid=abort).
func (n *Numfmt) line(line []byte, newline bool) (valid, aborted bool) {
	valid = true
	// A C string ends at its first NUL, and so does the line GNU sees.
	if i := bytes.IndexByte(line, 0); i >= 0 && !n.ZeroTerminated {
		line = line[:i]
	}
	for field := uint64(1); ; field++ {
		end := n.fieldEnd(line)
		text := string(line[:end])
		ok, err := n.field(text, field)
		if err != nil {
			return false, true
		}
		valid = valid && ok
		if end == len(line) {
			break
		}
		sep := byte(' ')
		if n.DelimiterSet {
			sep = n.Delimiter
		}
		_ = n.out.WriteByte(sep)
		line = line[end+1:]
	}
	if newline {
		if n.ZeroTerminated {
			_ = n.out.WriteByte(0)
		} else {
			_ = n.out.WriteByte('\n')
		}
	}
	return valid, false
}

func blank(c byte) bool { return c == ' ' || c == '\t' }

// fieldEnd is where the field at the start of line ends: at the delimiter,
// or, with none, after the blanks that lead it and the run of other
// characters that follows them.
func (n *Numfmt) fieldEnd(line []byte) int {
	i := 0
	if n.DelimiterSet {
		for i < len(line) && line[i] != n.Delimiter {
			i++
		}
		return i
	}
	for i < len(line) && (blank(line[i]) || line[i] == '\n') {
		i++
	}
	for i < len(line) && !blank(line[i]) && line[i] != '\n' {
		i++
	}
	return i
}

func (n *Numfmt) included(field uint64) bool {
	if n.fields == nil {
		return field == 1
	}
	for _, s := range n.fields {
		if s.lo <= field && field <= s.hi {
			return true
		}
	}
	return false
}

// field converts one field and writes it, or writes it unchanged when it
// is not to be converted or cannot be.
func (n *Numfmt) field(text string, field uint64) (bool, error) {
	if !n.included(field) {
		_, _ = n.out.WriteString(text)
		return true, nil
	}
	// A --suffix at the very end is taken off, even if the number turns out
	// to be invalid and the field is printed as it came.
	if n.Suffix != "" && len(text) > len(n.Suffix) && strings.HasSuffix(text, n.Suffix) {
		text = text[:len(text)-len(n.Suffix)]
	}
	p := strings.TrimLeft(text, " \t")
	if n.autoPad {
		n.padding = 0
		if len(p) < len(text) || field > 1 {
			n.padding = int64(len(text))
		}
	}

	val, precision, ok := n.parseHuman(p)
	if ok {
		if n.FromUnit > 1 || n.ToUnit > 1 {
			val = mul(val, fromUint(max(n.FromUnit, 1)))
			val = quo(val, fromUint(max(n.ToUnit, 1)))
		}
		var out string
		out, ok = n.format(val, precision)
		if ok {
			n.write(out)
			return true, nil
		}
	}
	if n.exitOnError {
		return false, errAbort
	}
	_, _ = n.out.WriteString(text)
	return false, nil
}

// invalid reports a conversion failure as --invalid says to.
func (n *Numfmt) invalid(format string, a ...any) {
	if n.Invalid != InvalidIgnore {
		n.diag.Warnf(format, a...)
	}
}

// write writes a converted number with the --format text around it and the
// padding it calls for.
func (n *Numfmt) write(num string) {
	_, _ = n.out.WriteString(n.prefix)
	width := int64(utf8.RuneCountInString(num))
	pad := int64(0)
	switch {
	case n.padding < 0 && n.padding < -width:
		pad = n.padding + width
	case n.padding > 0 && width < n.padding:
		pad = n.padding - width
	}
	for p := pad; p > 0; p-- {
		_ = n.out.WriteByte(' ')
	}
	_, _ = n.out.WriteString(num)
	for p := pad; p < 0; p++ {
		_ = n.out.WriteByte(' ')
	}
	_, _ = n.out.WriteString(n.suffixText)
}

// quote is gnulib's quote(): the text in single quotes.
func quote(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale})
}
