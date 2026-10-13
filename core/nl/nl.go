// Package nl implements the engine behind nlx, a GNU-compatible
// implementation of the GNU/UNIX nl command: number the lines of files.
package nl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/bre"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "nlx"

// Defaults GNU nl uses when an option is not given.
const (
	DefaultWidth     = 6
	DefaultSeparator = "\t"
	DefaultDelimiter = `\:`
)

// Options mirrors the GNU nl flag surface. The zero value is plain nl:
// non-empty body lines numbered from 1, right-aligned in 6 columns and
// followed by a tab.
type Options struct {
	// BodyStyle, HeaderStyle and FooterStyle (-b, -h, -f) choose which lines
	// of each section are numbered: "a" all, "t" non-empty ones, "n" none,
	// or "pBRE" those matching the basic regular expression BRE. Empty means
	// GNU's default: "t" for the body, "n" for the header and footer.
	BodyStyle, HeaderStyle, FooterStyle string
	// Delimiter (-d) is the section delimiter, "\:" unless DelimiterSet. One
	// character replaces only the first of the two, as in POSIX; an empty
	// delimiter turns sections off.
	Delimiter    string
	DelimiterSet bool
	// StartingLine (-v) is the first line number, 1 unless StartingLineSet.
	StartingLine    int64
	StartingLineSet bool
	// Increment (-i) is added after each numbered line, 1 unless
	// IncrementSet.
	Increment    int64
	IncrementSet bool
	// JoinBlank (-l) numbers only every JoinBlank-th of consecutive empty
	// lines under style "a"; zero means 1.
	JoinBlank int
	// Format (-n) is "ln" (left-aligned), "rn" (right-aligned, the default
	// when empty) or "rz" (right-aligned with leading zeros).
	Format string
	// NoRenumber (-p) keeps counting across logical pages instead of
	// starting again at each header.
	NoRenumber bool
	// Separator (-s) follows each number; a tab unless SeparatorSet.
	Separator    string
	SeparatorSet bool
	// Width (-w) is the width of the number column; zero means DefaultWidth.
	Width int
}

// section is the part of a logical page a line belongs to.
type section int

const (
	header section = iota
	body
	footer
)

// style decides whether a line is numbered.
type style struct {
	kind byte // 'a', 't', 'n' or 'p'
	re   *regexp.Regexp
}

// Nl is the engine. Construct it with New.
type Nl struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	styles     [3]style
	delims     [3]string // the header, body and footer delimiter lines
	blankFmt   string    // what an unnumbered line starts with
	current    section
	lineNo     int64
	overflow   bool
	blankLines int
}

// New builds an nl engine reading from and writing to s.
func New(opts Options, s core.Streams) *Nl {
	return &Nl{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// setup checks the options and fills in the defaults. It writes each
// mistake in GNU's words and reports whether all was well.
func (n *Nl) setup() bool {
	ok := true
	for i, s := range []struct {
		text, def, name string
	}{
		{n.HeaderStyle, "n", "header"},
		{n.BodyStyle, "t", "body"},
		{n.FooterStyle, "n", "footer"},
	} {
		text := s.text
		if text == "" {
			text = s.def
		}
		st, err := parseStyle(text)
		if err != nil {
			var e *bre.Error
			if errors.As(err, &e) {
				n.diag.Errorf("%s", e.Msg)
			} else {
				n.diag.Errorf("invalid %s numbering style: '%s'", s.name, text)
			}
			ok = false
		}
		n.styles[i] = st
	}
	switch n.Format {
	case "", "ln", "rn", "rz":
	default:
		n.diag.Errorf("invalid line numbering format: '%s'", n.Format)
		ok = false
	}
	if n.Width <= 0 {
		n.Width = DefaultWidth
	}
	if n.JoinBlank <= 0 {
		n.JoinBlank = 1
	}
	if !n.StartingLineSet {
		n.StartingLine = 1
	}
	if !n.IncrementSet {
		n.Increment = 1
	}
	if !n.SeparatorSet {
		n.Separator = DefaultSeparator
	}
	delim := DefaultDelimiter
	if n.DelimiterSet {
		switch len(n.Delimiter) {
		case 1:
			delim = n.Delimiter + delim[1:]
		default:
			delim = n.Delimiter
		}
	}
	if delim != "" {
		n.delims = [3]string{strings.Repeat(delim, 3), strings.Repeat(delim, 2), delim}
	}
	n.blankFmt = strings.Repeat(" ", n.Width+len(n.Separator))
	n.lineNo = n.StartingLine
	n.current = body
	return ok
}

func parseStyle(s string) (style, error) {
	switch {
	case s == "a" || s == "t" || s == "n":
		return style{kind: s[0]}, nil
	case strings.HasPrefix(s, "p"):
		re, err := bre.Compile(s[1:])
		if err != nil {
			return style{}, err
		}
		return style{kind: 'p', re: re}, nil
	}
	return style{}, errors.New("invalid style")
}

// Run numbers the operands, read as one stream; no operands, or "-", means
// standard input. A missing file is reported and the rest still run. The
// returned error is core.ErrFailed if anything failed, otherwise nil.
func (n *Nl) Run(args []string) error {
	if !n.setup() {
		return core.ErrFailed
	}
	if len(args) == 0 {
		args = []string{"-"}
	}
	n.out = core.NewWriter(n.streams.Out)
	for _, name := range args {
		if !n.file(name) {
			break
		}
	}
	if err := n.out.Flush(); err != nil && !core.IsClosedPipe(err) {
		n.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return n.diag.Result()
}

// file numbers one operand. It returns false when the run must stop.
func (n *Nl) file(name string) bool {
	r := n.streams.In
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			n.diag.Errorf("%s: %s", name, core.ErrText(err))
			return true
		}
		defer f.Close()
		r = f
	}
	if r == nil {
		return true
	}
	br := core.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			if !n.line(bytes.TrimSuffix(line, []byte("\n"))) {
				return false
			}
			core.FlushIfIdle(n.out, br)
		}
		if err == io.EOF {
			return true
		}
		if err != nil {
			n.diag.Errorf("%s: %s", name, core.ErrText(err))
			return true
		}
	}
}

// line handles one line, without its newline. It returns false when the
// line number overflowed.
func (n *Nl) line(text []byte) bool {
	if n.delims[2] != "" {
		switch string(text) {
		case n.delims[header]:
			n.current = header
			if !n.NoRenumber {
				n.lineNo = n.StartingLine
				n.overflow = false
			}
			_ = n.out.WriteByte('\n')
			return true
		case n.delims[body]:
			n.current = body
			_ = n.out.WriteByte('\n')
			return true
		case n.delims[footer]:
			n.current = footer
			_ = n.out.WriteByte('\n')
			return true
		}
	}

	number := false
	switch st := n.styles[n.current]; st.kind {
	case 'a':
		if n.JoinBlank > 1 && len(text) == 0 {
			n.blankLines++
			if n.blankLines == n.JoinBlank {
				number, n.blankLines = true, 0
			}
		} else {
			number, n.blankLines = true, 0
		}
	case 't':
		number = len(text) > 0
	case 'p':
		number = st.re.Match(text)
	}

	if number {
		if n.overflow {
			n.diag.Errorf("line number overflow")
			return false
		}
		switch n.Format {
		case "ln":
			fmt.Fprintf(n.out, "%-*d", n.Width, n.lineNo)
		case "rz":
			fmt.Fprintf(n.out, "%0*d", n.Width, n.lineNo)
		default:
			_, _ = n.out.Write(core.AppendInt(n.out.AvailableBuffer(), n.lineNo, n.Width))
		}
		_, _ = n.out.WriteString(n.Separator)
		n.lineNo, n.overflow = add(n.lineNo, n.Increment)
	} else {
		_, _ = n.out.WriteString(n.blankFmt)
	}
	_, _ = n.out.Write(text)
	_ = n.out.WriteByte('\n')
	return true
}

// add returns a+b and whether that overflowed.
func add(a, b int64) (int64, bool) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return a, true
	}
	return a + b, false
}
