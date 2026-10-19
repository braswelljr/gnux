// Package join implements the engine behind joinx, a GNU-compatible
// implementation of the GNU/UNIX join command: join the lines of two files on
// a common field.
package join

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "joinx"

// OrderCheck says whether the inputs' sort order is verified.
type OrderCheck int

const (
	// CheckDefault warns, once per file, about disorder only once an
	// unpairable line has been seen, and fails at the end (GNU's default).
	CheckDefault OrderCheck = iota
	// CheckOrder (--check-order) fails at the first line out of order.
	CheckOrder
	// NoCheckOrder (--nocheck-order) never checks.
	NoCheckOrder
)

// Field is one entry of an output format (-o): File is 1 or 2 and Field a
// 1-based field number, or File 0 for the join field.
type Field struct {
	File  int
	Field int
}

// Options mirrors the GNU join flag surface. The zero value is plain join:
// fields separated by blanks, joined on the first field of each file, and
// only the paired lines printed.
type Options struct {
	// Unpaired1 and Unpaired2 (-a 1, -a 2) also print the lines of FILE1 or
	// FILE2 that have no match.
	Unpaired1, Unpaired2 bool
	// OnlyUnpaired (-v) suppresses the joined lines, leaving only the
	// unpaired lines selected by Unpaired1 and Unpaired2.
	OnlyUnpaired bool
	// Empty (-e EMPTY) replaces missing or empty output fields; it applies
	// only when EmptySet is true.
	Empty    string
	EmptySet bool
	// IgnoreCase (-i) compares the join fields without regard to ASCII case.
	IgnoreCase bool
	// Field1 and Field2 (-1, -2, -j) are the 1-based join fields of FILE1 and
	// FILE2; zero means the first field.
	Field1, Field2 int
	// Format (-o) lists the output fields; nil means the default output.
	Format []Field
	// AutoFormat (-o auto) prints, for every line, as many fields as the
	// first line of each file has.
	AutoFormat bool
	// Separator (-t CHAR) is the field separator for input and output; it
	// applies only when SeparatorSet is true. A newline makes the whole line
	// one field (GNU's -t '').
	Separator    byte
	SeparatorSet bool
	// Check selects the order check.
	Check OrderCheck
	// Header (--header) treats the first line of each file as a header,
	// printed first and never paired or order-checked.
	Header bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
}

// Join is the engine. Construct it with New.
type Join struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	eol     byte
	jf      [2]int // 0-based join fields
	auto    [2]int
	files   [2]*input

	seenUnpairable bool
	fatal          bool
}

// New builds a join engine reading from and writing to s.
func New(opts Options, s core.Streams) *Join {
	j := &Join{Options: opts, streams: s, diag: core.NewDiag(Name, s), eol: '\n'}
	if opts.ZeroTerminated {
		j.eol = 0
	}
	j.jf = [2]int{max(opts.Field1, 1) - 1, max(opts.Field2, 1) - 1}
	return j
}

// line is one input line split into fields.
type line struct {
	text   []byte
	fields [][]byte
}

// blank stands for the missing side of an unpaired line.
var blank = &line{}

// input is one of the two files.
type input struct {
	name   string
	r      *bufio.Reader
	closer io.Closer
	lineNo int
	prev   *line
	warned bool
}

// Run joins FILE1 and FILE2 (args); either may be "-" for standard input, but
// not both. The returned error is a *core.UsageError for a wrong number of
// operands, core.ErrFailed when a file failed or the input was out of order,
// otherwise nil.
func (j *Join) Run(args []string) error {
	switch len(args) {
	case 0:
		return core.Usagef("missing operand")
	case 1:
		return core.Usagef("missing operand after '%s'", args[0])
	case 2:
	default:
		return core.Usagef("extra operand '%s'", args[2])
	}
	if args[0] == "-" && args[1] == "-" {
		j.diag.Errorf("both files cannot be standard input")
		return j.diag.Result()
	}
	for i, name := range args {
		var r io.Reader
		var closer io.Closer
		if name == "-" {
			r = j.streams.In
			if r == nil {
				r = strings.NewReader("")
			}
		} else {
			f, err := fsys.Open(name)
			if err != nil {
				j.diag.Errorf("%s: %s", name, core.ErrText(err))
				j.close()
				return j.diag.Result()
			}
			r, closer = f, f
		}
		j.files[i] = &input{name: name, r: core.NewReader(r), closer: closer}
	}
	defer j.close()

	j.out = core.NewWriter(j.streams.Out)
	j.join()
	if err := j.out.Flush(); err != nil && !core.IsClosedPipe(err) {
		j.diag.Errorf("write error: %s", core.ErrText(err))
	}
	if !j.fatal && (j.files[0].warned || j.files[1].warned) {
		j.diag.Errorf("input is not in sorted order")
	}
	return j.diag.Result()
}

func (j *Join) close() {
	for _, f := range j.files {
		if f != nil && f.closer != nil {
			f.closer.Close()
		}
	}
}

// isBlank is GNU's field_sep: a blank or a newline.
func isBlank(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

// split breaks text into fields the way GNU's xfields does.
func (j *Join) split(text []byte) [][]byte {
	var fields [][]byte
	if len(text) == 0 {
		return nil
	}
	if !j.SeparatorSet {
		p := 0
		for p < len(text) && isBlank(text[p]) {
			p++
		}
		for p < len(text) {
			e := p + 1
			for e < len(text) && !isBlank(text[e]) {
				e++
			}
			fields = append(fields, text[p:e])
			for e < len(text) && isBlank(text[e]) {
				e++
			}
			p = e
		}
		return fields
	}
	if j.Separator == '\n' {
		return [][]byte{text}
	}
	return bytes.Split(text, []byte{j.Separator})
}

// getLine reads the next line of file which (0 or 1), checking its order
// against the previous one. ok is false at the end of the input, on a read
// error (diagnosed) or when --check-order found disorder.
func (j *Join) getLine(which int) (*line, bool) {
	if j.fatal {
		return nil, false
	}
	f := j.files[which]
	raw, err := f.r.ReadBytes(j.eol)
	if len(raw) > 0 && raw[len(raw)-1] == j.eol {
		raw = raw[:len(raw)-1]
	} else {
		if err != nil && !errors.Is(err, io.EOF) {
			j.diag.Errorf("%s: %s", f.name, core.ErrText(err))
			j.fatal = true
			return nil, false
		}
		if len(raw) == 0 {
			return nil, false
		}
	}
	l := &line{text: append([]byte(nil), raw...)}
	l.fields = j.split(l.text)
	f.lineNo++
	if f.prev != nil {
		j.checkOrder(which, f.prev, l)
	}
	f.prev = l
	if j.fatal {
		return nil, false
	}
	return l, true
}

// checkOrder is GNU's check_order.
func (j *Join) checkOrder(which int, prev, cur *line) {
	f := j.files[which]
	if j.Check == NoCheckOrder || (j.Check != CheckOrder && !j.seenUnpairable) || f.warned {
		return
	}
	if j.keycmp(prev, cur, j.jf[which], j.jf[which]) <= 0 {
		return
	}
	j.diag.Errorf("%s:%d: is not sorted: %s", f.name, f.lineNo, cur.text)
	if j.Check == CheckOrder {
		j.fatal = true
		return
	}
	f.warned = true
}

// keycmp compares the join fields of two lines, as GNU's keycmp does.
func (j *Join) keycmp(a, b *line, fa, fb int) int {
	var ka, kb []byte
	if fa < len(a.fields) {
		ka = a.fields[fa]
	}
	if fb < len(b.fields) {
		kb = b.fields[fb]
	}
	if len(ka) == 0 {
		if len(kb) == 0 {
			return 0
		}
		return -1
	}
	if len(kb) == 0 {
		return 1
	}
	if j.IgnoreCase {
		return compareFold(ka, kb)
	}
	return bytes.Compare(ka, kb)
}

// compareFold compares ASCII case-insensitively, like memcasecmp in the C
// locale.
func compareFold(a, b []byte) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		ca, cb := lower(a[i]), lower(b[i])
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// first reads the first line of a file into a fresh sequence.
func (j *Join) first(which int) []*line {
	if l, ok := j.getLine(which); ok {
		return []*line{l}
	}
	return nil
}

// join is GNU's join loop.
func (j *Join) join() {
	seq := [2][]*line{j.first(0), j.first(1)}
	if j.AutoFormat {
		for i := range seq {
			if len(seq[i]) > 0 {
				j.auto[i] = len(seq[i][0].fields)
			}
		}
	}

	if j.Header && (len(seq[0]) > 0 || len(seq[1]) > 0) {
		h := [2]*line{blank, blank}
		for i := range seq {
			if len(seq[i]) > 0 {
				h[i] = seq[i][0]
			}
		}
		j.prjoin(h[0], h[1])
		j.files[0].prev, j.files[1].prev = nil, nil
		for i := range seq {
			if len(seq[i]) > 0 {
				seq[i] = j.first(i)
			}
		}
	}

	for len(seq[0]) > 0 && len(seq[1]) > 0 && !j.fatal {
		diff := j.keycmp(seq[0][0], seq[1][0], j.jf[0], j.jf[1])
		if diff < 0 {
			if j.Unpaired1 {
				j.prjoin(seq[0][0], blank)
			}
			seq[0] = j.first(0)
			j.seenUnpairable = true
			continue
		}
		if diff > 0 {
			if j.Unpaired2 {
				j.prjoin(blank, seq[1][0])
			}
			seq[1] = j.first(1)
			j.seenUnpairable = true
			continue
		}

		// Collect the run of lines in each file that match the current line
		// of the other. The line that ends a run (if any) starts the next.
		eof1 := true
		for {
			l, ok := j.getLine(0)
			if !ok {
				break
			}
			seq[0] = append(seq[0], l)
			if j.keycmp(l, seq[1][0], j.jf[0], j.jf[1]) != 0 {
				eof1 = false
				break
			}
		}
		eof2 := true
		for {
			l, ok := j.getLine(1)
			if !ok {
				break
			}
			seq[1] = append(seq[1], l)
			if j.keycmp(seq[0][0], l, j.jf[0], j.jf[1]) != 0 {
				eof2 = false
				break
			}
		}
		group1, group2 := seq[0], seq[1]
		if !eof1 {
			group1 = group1[:len(group1)-1]
		}
		if !eof2 {
			group2 = group2[:len(group2)-1]
		}
		if !j.OnlyUnpaired {
			for _, a := range group1 {
				for _, b := range group2 {
					j.prjoin(a, b)
				}
			}
		}
		seq[0] = nextRun(seq[0], eof1)
		seq[1] = nextRun(seq[1], eof2)
	}

	// Print, or at least read to check the order of, what is left of either
	// file once the other has ended.
	check := j.Check != NoCheckOrder
	for i, unpaired := range []bool{j.Unpaired1, j.Unpaired2} {
		if j.fatal || len(seq[i]) == 0 || (!unpaired && !check) {
			continue
		}
		if unpaired {
			j.printUnpaired(i, seq[i][0])
		}
		for {
			l, ok := j.getLine(i)
			if !ok {
				break
			}
			if unpaired {
				j.printUnpaired(i, l)
			}
			if j.files[i].warned && !unpaired {
				break
			}
		}
	}
}

func nextRun(seq []*line, eof bool) []*line {
	if eof {
		return nil
	}
	return seq[len(seq)-1:]
}

func (j *Join) printUnpaired(which int, l *line) {
	if which == 0 {
		j.prjoin(l, blank)
	} else {
		j.prjoin(blank, l)
	}
}

func (j *Join) outSep() byte {
	if j.SeparatorSet {
		return j.Separator
	}
	return ' '
}

// prfield writes field n (0-based) of l, or the -e filler when it is missing
// or empty.
func (j *Join) prfield(n int, l *line) {
	if n < len(l.fields) && len(l.fields[n]) > 0 {
		_, _ = j.out.Write(l.fields[n])
		return
	}
	if j.EmptySet {
		_, _ = j.out.WriteString(j.Empty)
	}
}

// prfields writes the fields of l other than the join field.
func (j *Join) prfields(l *line, joinField, autocount int) {
	n := len(l.fields)
	if j.AutoFormat {
		n = autocount
	}
	sep := j.outSep()
	for i := 0; i < joinField && i < n; i++ {
		_ = j.out.WriteByte(sep)
		j.prfield(i, l)
	}
	for i := joinField + 1; i < n; i++ {
		_ = j.out.WriteByte(sep)
		j.prfield(i, l)
	}
}

// prjoin writes one output line for a pair (either side may be blank).
func (j *Join) prjoin(a, b *line) {
	sep := j.outSep()
	if len(j.Format) > 0 {
		for i, o := range j.Format {
			if i > 0 {
				_ = j.out.WriteByte(sep)
			}
			var l *line
			var field int
			switch o.File {
			case 0:
				if a == blank {
					l, field = b, j.jf[1]
				} else {
					l, field = a, j.jf[0]
				}
			case 1:
				l, field = a, o.Field-1
			default:
				l, field = b, o.Field-1
			}
			j.prfield(field, l)
		}
		_ = j.out.WriteByte(j.eol)
		return
	}
	if a == blank {
		j.prfield(j.jf[1], b)
	} else {
		j.prfield(j.jf[0], a)
	}
	j.prfields(a, j.jf[0], j.auto[0])
	j.prfields(b, j.jf[1], j.auto[1])
	_ = j.out.WriteByte(j.eol)
}
