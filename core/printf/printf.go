// Package printf implements the engine behind printfx, a GNU-compatible
// implementation of the GNU/UNIX printf command: format and print the
// arguments under the control of a format string.
package printf

import (
	"bufio"
	"bytes"
	"errors"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "printfx"

// conversions are the conversion characters a directive may end with.
const conversions = "diouxXfFeEgGaAcsbq"

// Options mirrors the GNU printf flag surface. GNU printf takes no options
// other than --help and --version, so it is empty; it exists so that the
// engine has the same shape as every other.
type Options struct{}

// Printf is the engine. Construct it with New.
type Printf struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a printf engine writing to s.
func New(opts Options, s core.Streams) *Printf {
	return &Printf{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// errFatal ends the run after a diagnostic that stops GNU printf at once (an
// invalid directive or escape).
var errFatal = errors.New("fatal")

// Run formats args[1:] under the format args[0]. A leading "--" is skipped,
// as GNU printf does. The format is reused for as long as it consumes
// arguments and some remain; a missing argument reads as an empty string or
// zero. An argument that is not a valid number is diagnosed, printed as far as
// it was converted, and makes the run fail without stopping it.
func (p *Printf) Run(args []string) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	format, args := args[0], args[1:]

	// Write errors are sticky in a bufio.Writer, so the individual writes go
	// unchecked and finish reports the first one when it flushes.
	p.out = bufio.NewWriter(p.streams.Out)
	for {
		used, err := p.once(format, args)
		if err != nil {
			// \c ends output successfully; a fatal error has its diagnostic.
			return p.finish()
		}
		args = args[used:]
		if used == 0 || len(args) == 0 {
			if used == 0 && len(args) > 0 {
				p.diag.Warnf("warning: ignoring excess arguments, starting with %s", quote(args[0]))
			}
			return p.finish()
		}
	}
}

// errStop is returned by once when \c ends all output.
var errStop = errors.New("stop")

// finish flushes the output and returns the run's status.
func (p *Printf) finish() error {
	if err := p.out.Flush(); err != nil && !core.IsClosedPipe(err) {
		p.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return p.diag.Result()
}

// once writes the format a single time, consuming arguments as its
// directives need them, and reports how many it used. It returns errStop
// when \c ended all output and errFatal after a fatal diagnostic.
func (p *Printf) once(f string, args []string) (int, error) {
	used := 0
	next := func() (string, bool) {
		if used < len(args) {
			used++
			return args[used-1], true
		}
		return "", false
	}

	for i := 0; i < len(f); i++ {
		switch f[i] {
		case '\\':
			var buf bytes.Buffer
			n, stop, err := escape(&buf, f, i, false)
			_, _ = p.out.Write(buf.Bytes())
			if err != nil {
				p.diag.Errorf("%s", err)
				return used, errFatal
			}
			if stop {
				return used, errStop
			}
			i = n - 1
		case '%':
			n, err := p.directive(f, i, next)
			if err != nil {
				return used, err
			}
			i = n - 1
		default:
			_ = p.out.WriteByte(f[i])
		}
	}
	return used, nil
}

// spec is one parsed conversion specification.
type spec struct {
	flags   string
	width   int
	hasWid  bool
	prec    int
	hasPrec bool
	verb    byte
}

// directive handles the conversion specification starting at f[start] ('%')
// and returns the index just past it.
func (p *Printf) directive(f string, start int, next func() (string, bool)) (int, error) {
	i := start + 1
	if i < len(f) && f[i] == '%' {
		_ = p.out.WriteByte('%')
		return i + 1, nil
	}

	var sp spec
	for i < len(f) && strings.IndexByte("-+ #0'", f[i]) >= 0 {
		if f[i] != '\'' { // grouping has no effect in the C locale
			sp.flags += string(f[i])
		}
		i++
	}

	if i < len(f) && f[i] == '*' {
		arg, _ := next()
		w, ok := p.starValue(arg)
		if !ok {
			p.diag.Errorf("invalid field width: %s", quote(arg))
			return i, errFatal
		}
		if w < 0 {
			sp.flags += "-"
			w = -w
		}
		sp.width, sp.hasWid = w, true
		i++
	} else {
		j := i
		for i < len(f) && isDigit(f[i]) {
			i++
		}
		if i > j {
			w, err := strconv.Atoi(f[j:i])
			if err != nil {
				p.diag.Errorf("invalid field width: %s", quote(f[j:i]))
				return i, errFatal
			}
			sp.width, sp.hasWid = w, true
		}
	}

	if i < len(f) && f[i] == '.' {
		i++
		sp.hasPrec = true
		if i < len(f) && f[i] == '*' {
			arg, _ := next()
			pr, ok := p.starValue(arg)
			if !ok {
				p.diag.Errorf("invalid precision: %s", quote(arg))
				return i, errFatal
			}
			// A negative precision is taken as if it were omitted.
			sp.prec, sp.hasPrec = pr, pr >= 0
			i++
		} else {
			j := i
			for i < len(f) && isDigit(f[i]) {
				i++
			}
			if i > j {
				pr, err := strconv.Atoi(f[j:i])
				if err != nil {
					p.diag.Errorf("invalid precision: %s", quote(f[j:i]))
					return i, errFatal
				}
				sp.prec = pr
			}
		}
	}

	if i >= len(f) || strings.IndexByte(conversions, f[i]) < 0 {
		end := min(i+1, len(f))
		p.diag.Errorf("%s: invalid conversion specification", f[start:end])
		return end, errFatal
	}
	sp.verb = f[i]
	arg, _ := next()
	if err := p.convert(sp, arg); err != nil {
		return i + 1, err
	}
	return i + 1, nil
}

// starValue reads a '*' width or precision from its argument.
func (p *Printf) starValue(arg string) (int, bool) {
	v, ok := p.signed(arg)
	if !ok || v > int64(maxInt) || v < -int64(maxInt) {
		return 0, false
	}
	return int(v), true
}

const maxInt = int(^uint32(0) >> 1)

// convert formats one argument under sp and writes it.
func (p *Printf) convert(sp spec, arg string) error {
	switch sp.verb {
	case 'd', 'i':
		v, _ := p.signed(arg)
		_, _ = p.out.WriteString(formatSigned(sp, v))
	case 'o', 'u', 'x', 'X':
		v, _ := p.unsigned(arg)
		_, _ = p.out.WriteString(formatUnsigned(sp, v))
	case 'f', 'F', 'e', 'E', 'g', 'G', 'a', 'A':
		v, _ := p.float(arg)
		_, _ = p.out.WriteString(formatFloat(sp, v))
	case 'c':
		c := "\x00"
		if arg != "" {
			c = arg[:1]
		}
		_, _ = p.out.WriteString(pad(sp, c))
	case 's':
		s := arg
		if sp.hasPrec && sp.prec < len(s) {
			s = s[:sp.prec]
		}
		_, _ = p.out.WriteString(pad(sp, s))
	case 'b':
		var buf bytes.Buffer
		stop := false
		for i := 0; i < len(arg); i++ {
			if arg[i] != '\\' {
				buf.WriteByte(arg[i])
				continue
			}
			n, st, err := escape(&buf, arg, i, true)
			if err != nil {
				_, _ = p.out.Write(buf.Bytes())
				p.diag.Errorf("%s", err)
				return errFatal
			}
			if st {
				stop = true
				break
			}
			i = n - 1
		}
		s := buf.String()
		if sp.hasPrec && sp.prec < len(s) {
			s = s[:sp.prec]
		}
		_, _ = p.out.WriteString(pad(sp, s))
		if stop {
			return errStop
		}
	case 'q':
		_, _ = p.out.WriteString(pad(sp, ShellQuote(arg)))
	}
	return nil
}

// pad pads s with spaces to the field width, on the left unless the '-' flag
// is set. Widths count bytes, as C's printf does.
func pad(sp spec, s string) string {
	if !sp.hasWid || len(s) >= sp.width {
		return s
	}
	fill := strings.Repeat(" ", sp.width-len(s))
	if strings.Contains(sp.flags, "-") {
		return s + fill
	}
	return fill + s
}

// quote quotes an operand for a diagnostic, as GNU's quote() does.
func quote(s string) string {
	if strings.Contains(s, "'") {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
