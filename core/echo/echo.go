// Package echo implements the engine behind echox, a GNU-compatible implementation of
// the GNU/UNIX echo command: write the operands, separated by single spaces and
// followed by a newline, to standard output.
package echo

import (
	"bytes"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "echox"

// Options mirrors the GNU echo flag surface.
type Options struct {
	// NoNewline (-n) omits the trailing newline.
	NoNewline bool
	// Escapes (-e) interprets backslash escapes in the operands. When false
	// (-E, the default) operands are written verbatim.
	Escapes bool
}

// Echo is the engine. Construct it with New.
type Echo struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds an echo engine writing to s.
func New(opts Options, s core.Streams) *Echo {
	return &Echo{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Parse splits an echo command line into its options and the operands to
// print, the way GNU echo does rather than the way getopt would: a leading
// argument is an option only when it is a '-' followed solely by the letters
// n, e and E. The first argument that is anything else (including "-", "--"
// and "-x") ends option parsing and is printed like every argument after it.
func Parse(args []string) (Options, []string) {
	var o Options
	for len(args) > 0 && isOptionCluster(args[0]) {
		for _, c := range args[0][1:] {
			switch c {
			case 'n':
				o.NoNewline = true
			case 'e':
				o.Escapes = true
			case 'E':
				o.Escapes = false
			}
		}
		args = args[1:]
	}
	return o, args
}

// isOptionCluster reports whether arg is "-" followed by one or more of n/e/E.
func isOptionCluster(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' {
		return false
	}
	for i := 1; i < len(arg); i++ {
		switch arg[i] {
		case 'n', 'e', 'E':
		default:
			return false
		}
	}
	return true
}

// Run writes the operands. The whole line is assembled first and written with
// a single call, so concurrent writers to the same pipe do not interleave
// inside it.
func (e *Echo) Run(operands []string) error {
	var buf bytes.Buffer
	stopped := false
	for i, op := range operands {
		if i > 0 {
			buf.WriteByte(' ')
		}
		if !e.Escapes {
			buf.WriteString(op)
			continue
		}
		if stopped = expand(&buf, op); stopped {
			break
		}
	}
	// \c suppresses everything after it, the trailing newline included.
	if !stopped && !e.NoNewline {
		buf.WriteByte('\n')
	}

	// A reader that has gone away is not an error worth reporting: GNU echo
	// is ended silently by SIGPIPE in the same situation.
	if _, err := e.streams.Out.Write(buf.Bytes()); err != nil && !core.IsClosedPipe(err) {
		e.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return e.diag.Result()
}

// expand appends s to buf with backslash escapes interpreted. It reports true
// when a \c was met, which ends all output.
func expand(buf *bytes.Buffer, s string) (stop bool) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		// A backslash that ends the string is printed as is.
		if c != '\\' || i+1 == len(s) {
			buf.WriteByte(c)
			continue
		}
		i++
		switch c = s[i]; c {
		case 'a':
			buf.WriteByte('\a')
		case 'b':
			buf.WriteByte('\b')
		case 'c':
			return true
		case 'e':
			buf.WriteByte(0x1b)
		case 'f':
			buf.WriteByte('\f')
		case 'n':
			buf.WriteByte('\n')
		case 'r':
			buf.WriteByte('\r')
		case 't':
			buf.WriteByte('\t')
		case 'v':
			buf.WriteByte('\v')
		case '\\':
			buf.WriteByte('\\')
		case 'x':
			// \xHH: one or two hex digits; with none, "\x" is literal.
			v, n := digits(s[i+1:], 16, 2)
			if n == 0 {
				buf.WriteString(`\x`)
				continue
			}
			buf.WriteByte(v)
			i += n
		case '0':
			// \0NNN: up to three octal digits after the zero.
			v, n := digits(s[i+1:], 8, 3)
			buf.WriteByte(v)
			i += n
		case '1', '2', '3', '4', '5', '6', '7':
			// \NNN: GNU also accepts the form without the leading zero.
			v, n := digits(s[i:], 8, 3)
			buf.WriteByte(v)
			i += n - 1
		default:
			// Unknown escape: keep the backslash and the character.
			buf.WriteByte('\\')
			buf.WriteByte(c)
		}
	}
	return false
}

// digits reads up to limit leading digits of s in the given base (8 or 16) and
// returns their value, truncated to a byte as C's putchar would, and how many
// characters were consumed.
func digits(s string, base, limit int) (value byte, n int) {
	v := 0
	for n < limit && n < len(s) {
		d := digit(s[n])
		if d < 0 || d >= base {
			break
		}
		v = v*base + d
		n++
	}
	return byte(v), n
}

// digit returns the numeric value of a hex digit, or -1.
func digit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
