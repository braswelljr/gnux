// Package stty implements the engine behind sttyx, a GNU-compatible
// implementation of the GNU/UNIX stty command: print or change the settings
// of a terminal.
package stty

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sttyx"

// Options mirrors the GNU stty flag surface. The zero value is plain stty:
// print the settings of standard input that differ from sane ones.
type Options struct {
	// All (-a) prints every setting, in a form people read.
	All bool
	// Save (-g) prints every setting in a form stty reads back.
	Save bool
	// File (-F) is the device to use instead of standard input.
	File string
}

// Stty is the engine. Construct it with New.
type Stty struct {
	Options

	// Getenv reads COLUMNS, for the width of the output when standard
	// output is not a terminal. It defaults to os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag
	out     io.Writer

	device   string // what diagnostics call the terminal
	fd       uintptr
	maxCol   int
	col      int
	lastIn   int64 // speeds asked for, -1 when not
	lastOut  int64
	drain    bool
	setAttrs bool
}

// New builds a stty engine for the terminal on s.In, writing to s.Out.
func New(opts Options, s core.Streams) *Stty {
	return &Stty{Options: opts, Getenv: os.Getenv, streams: s, diag: core.NewDiag(Name, s), out: s.Out}
}

// errUsage carries a usage error out of the setting parser.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// fatal is a failure GNU reports and exits on, with status 1.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

func quote(s string) string { return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale}) }
func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

// Run applies the settings in turn, each a word of the command line
// ("-echo", "intr", "^C", ...). With none it prints the terminal's.
// "drain" and "-drain" only say when to apply the others.
func (s *Stty) Run(settings []string) (err error) {
	noargs := true
	for _, a := range settings {
		if a != "drain" && a != "-drain" {
			noargs = false
		}
	}
	if s.All && s.Save {
		return s.report(&fatal{"the options for verbose and stty-readable output styles are\nmutually exclusive"})
	}
	if !noargs && (s.All || s.Save) {
		return s.report(&fatal{"when specifying an output style, modes may not be set"})
	}
	s.device = "standard input"
	if s.File != "" {
		s.device = s.File
	}

	if !noargs && !s.All && !s.Save {
		var check termios
		s.lastIn, s.lastOut = -1, -1
		if err := s.apply(true, settings, &check); err != nil {
			return s.report(err)
		}
	}

	var f *os.File
	if s.File != "" {
		f, err = openDevice(s.File)
		if err != nil {
			return s.report(&fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))})
		}
		defer f.Close()
	} else {
		var ok bool
		if f, ok = s.streams.In.(*os.File); !ok {
			return s.report(&fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(errNotTTY))})
		}
	}
	s.fd = f.Fd()

	mode, err := getAttr(s.fd)
	if err != nil {
		return s.report(&fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))})
	}
	if s.All || s.Save || noargs {
		s.maxCol, s.col = s.screenColumns(), 0
		switch {
		case s.All:
			s.displayAll(&mode)
		case s.Save:
			s.displayRecoverable(&mode)
		default:
			s.displayChanged(&mode)
		}
		return nil
	}

	s.lastIn, s.lastOut = -1, -1
	s.drain = true
	if err := s.apply(false, settings, &mode); err != nil {
		return s.report(err)
	}
	if !s.setAttrs {
		return nil
	}
	if err := setAttr(s.fd, &mode, s.drain); err != nil {
		return s.report(&fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))})
	}
	got, err := getAttr(s.fd)
	if err != nil {
		return s.report(&fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))})
	}
	if !equalModes(&mode, &got) {
		return s.report(&fatal{fmt.Sprintf("%s: unable to perform all requested operations", quotef(s.device))})
	}
	return nil
}

// report turns a parse or apply error into the tool's result.
func (s *Stty) report(err error) error {
	var u *usageError
	var f *fatal
	switch {
	case errors.As(err, &u):
		return core.Usagef("%s", u.msg)
	case errors.As(err, &f):
		s.diag.Errorf("%s", f.msg)
		return core.ErrFailed
	}
	return err
}

// errNotTTY is ENOTTY's text, for a standard input that is not a file.
var errNotTTY = errors.New("Inappropriate ioctl for device") //nolint:revive,staticcheck // strerror(ENOTTY)

// openDevice opens a -F device without waiting for a modem's carrier, then
// makes it blocking again, as GNU does.
func openDevice(name string) (*os.File, error) {
	f, err := fsys.OpenFile(name, os.O_RDONLY|nonblock, 0)
	if err == nil {
		clearNonblock(f)
	}
	return f, err
}

// wrapf prints one item of a listing, wrapping before it when it would not
// fit in the line and separating it from the last one with a space.
func (s *Stty) wrapf(format string, a ...any) {
	buf := fmt.Sprintf(format, a...)
	if s.col > 0 {
		if s.maxCol-s.col <= len(buf) {
			fmt.Fprint(s.out, "\n")
			s.col = 0
		} else {
			fmt.Fprint(s.out, " ")
			s.col++
		}
	}
	fmt.Fprint(s.out, buf)
	s.col += len(buf)
}

func (s *Stty) newline() {
	fmt.Fprint(s.out, "\n")
	s.col = 0
}

// screenColumns is the width listings are wrapped at: the width of the
// terminal on standard output, else $COLUMNS, else 80.
func (s *Stty) screenColumns() int {
	if f, ok := s.streams.Out.(*os.File); ok {
		if _, cols, err := getWinsize(f.Fd()); err == nil && cols > 0 {
			return cols
		}
	}
	if n, ok := cInteger(s.Getenv("COLUMNS")); ok && n > 0 && n <= math.MaxInt32 {
		return int(n)
	}
	return 80
}

// cInteger reads a number as strtol with base 0 does: 0x for hexadecimal,
// a leading 0 for octal, decimal otherwise, and nothing after it.
func cInteger(s string) (int64, bool) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	neg := false
	if t != "" && (t[0] == '+' || t[0] == '-') {
		neg = t[0] == '-'
		t = t[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	if t == "" || strings.ContainsAny(t, "_+-") {
		return 0, false
	}
	n, err := strconv.ParseInt(t, base, 64)
	if err != nil {
		return 0, false
	}
	if neg {
		n = -n
	}
	return n, true
}

// integerArg reads a number for min, time, rows, cols and line: base 0 as
// in C, with an optional b (512) or B (1024) multiplier.
func integerArg(s string, maxval uint64) (uint64, error) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	mult := uint64(1)
	switch {
	case strings.HasSuffix(t, "b"):
		mult, t = 512, t[:len(t)-1]
	case strings.HasSuffix(t, "B"):
		mult, t = 1024, t[:len(t)-1]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	bad := &fatal{"invalid integer argument: " + quote(s)}
	if t == "" || strings.ContainsAny(t, "_+-") {
		return 0, bad
	}
	n, err := strconv.ParseUint(t, base, 64)
	if errors.Is(err, strconv.ErrRange) || (err == nil && (n > math.MaxUint64/mult || n*mult > maxval)) {
		return 0, &fatal{"invalid integer argument: " + quote(s) + ": Numerical result out of range"}
	}
	if err != nil {
		return 0, bad
	}
	return n * mult, nil
}

// The kinds of mode, and the flags of the mode table, as in GNU stty.
type kind int

const (
	control kind = iota
	input
	output
	local
	combination
)

const (
	saneSet   = 1  // set by 'sane'
	saneUnset = 2  // cleared by 'sane'
	rev       = 4  // can be turned off with a leading '-'
	omit      = 8  // not shown
	noSetAttr = 16 // set with an ioctl of its own, not tcsetattr
)

type modeInfo struct {
	name  string
	kind  kind
	flags int
	bits  uint64
	mask  uint64
}

type controlInfo struct {
	name  string
	sane  byte
	index int
}

type speedInfo struct {
	name  string
	speed uint64 // the system's speed_t
	value uint64 // in baud
}

// flag returns the termios field a mode of kind k lives in.
func (t *termios) flag(k kind) *uint64 {
	switch k {
	case control:
		return &t.cflag
	case input:
		return &t.iflag
	case output:
		return &t.oflag
	case local:
		return &t.lflag
	}
	return nil
}

// equalModes reports whether the terminal took the settings. kernelState
// are local flags the kernel sets by itself (PENDIN on the BSDs, raised when
// canonical mode comes back with input pending), which say nothing about
// whether a setting was refused.
func equalModes(a, b *termios) bool {
	return a.iflag == b.iflag && a.oflag == b.oflag && a.cflag == b.cflag && a.lflag&^kernelState == b.lflag&^kernelState &&
		a.line == b.line && a.cc == b.cc && cfgetispeed(a) == cfgetispeed(b) && cfgetospeed(a) == cfgetospeed(b)
}

func findControl(name string) (controlInfo, bool) {
	for _, c := range controls {
		if c.name == name {
			return c, true
		}
	}
	return controlInfo{}, false
}

func stringToBaud(s string) (uint64, bool) {
	for _, sp := range speeds {
		if sp.name == s {
			return sp.speed, true
		}
	}
	return 0, false
}

func baudToValue(speed uint64) uint64 {
	for _, sp := range speeds {
		if sp.speed == speed {
			return sp.value
		}
	}
	return 0
}

// apply goes through the settings, changing mode. With checking, it only
// validates them, on a zeroed mode, before the terminal is opened.
func (s *Stty) apply(checking bool, settings []string, mode *termios) error {
	argument := func(k int, arg string) error {
		if k == len(settings)-1 {
			return &usageError{"missing argument to " + quote(arg)}
		}
		return nil
	}
	for k := 0; k < len(settings); k++ {
		arg := settings[k]
		reversed := false
		if strings.HasPrefix(arg, "-") {
			arg, reversed = arg[1:], true
		}
		if arg == "drain" {
			s.drain = !reversed
			continue
		}
		matched, notSetAttr := false, false
		for i := range modes {
			if modes[i].name != arg {
				continue
			}
			if modes[i].flags&noSetAttr == 0 {
				matched = setMode(&modes[i], reversed, mode)
				s.setAttrs = true
			} else {
				matched, notSetAttr = true, true
			}
			break
		}
		if !matched && reversed {
			return &usageError{"invalid argument " + quote("-"+arg)}
		}
		if !matched {
			if c, ok := findControl(arg); ok {
				if err := argument(k, arg); err != nil {
					return err
				}
				k++
				if err := setControlChar(c, settings[k], mode); err != nil {
					return err
				}
				s.setAttrs = true
				continue
			}
		}
		if matched && !notSetAttr {
			continue
		}
		switch {
		case arg == "ispeed" || arg == "ospeed":
			if err := argument(k, arg); err != nil {
				return err
			}
			k++
			speed, ok := stringToBaud(settings[k])
			if !ok {
				return &usageError{"invalid " + arg + " " + quote(settings[k])}
			}
			if err := s.setSpeed(arg == "ispeed", arg == "ospeed", speed, settings[k], mode); err != nil {
				return err
			}
			if !checking {
				s.setAttrs = true
			}
		case arg == "extproc" && notSetAttr:
			if checking {
				continue
			}
			if err := setExtproc(s.fd, !reversed); err != nil {
				return &fatal{fmt.Sprintf("%s: error setting %s: %s", quotef(s.device), quote(arg), core.ErrText(err))}
			}
		case arg == "rows" || arg == "cols" || arg == "columns":
			if err := argument(k, arg); err != nil {
				return err
			}
			k++
			n, err := integerArg(settings[k], math.MaxInt32)
			if err != nil {
				return err
			}
			if checking {
				continue
			}
			if arg == "rows" {
				err = s.setWindowSize(int(n), -1)
			} else {
				err = s.setWindowSize(-1, int(n))
			}
			if err != nil {
				return err
			}
		case arg == "size":
			if checking {
				continue
			}
			s.maxCol, s.col = s.screenColumns(), 0
			if err := s.displayWindowSize(false); err != nil {
				return err
			}
		case arg == "line" && hasLine:
			if err := argument(k, arg); err != nil {
				return err
			}
			k++
			n, err := integerArg(settings[k], math.MaxUint64)
			if err != nil {
				return err
			}
			if n > 255 {
				s.diag.Errorf("invalid line discipline %s: Value too large for defined data type", quote(settings[k]))
			} else {
				mode.line = byte(n)
			}
			s.setAttrs = true
		case arg == "speed":
			if checking {
				continue
			}
			s.maxCol = s.screenColumns()
			s.displaySpeed(mode, false)
		default:
			if speed, ok := stringToBaud(arg); ok {
				if err := s.setSpeed(true, true, speed, arg, mode); err != nil {
					return err
				}
				if !checking {
					s.setAttrs = true
				}
				continue
			}
			if !recoverMode(arg, mode) {
				return &usageError{"invalid argument " + quote(arg)}
			}
			s.setAttrs = true
		}
	}
	if checking && s.lastIn != -1 && s.lastOut != -1 {
		if cfgetispeed(mode) != uint64(s.lastIn) || cfgetospeed(mode) != uint64(s.lastOut) {
			return &fatal{fmt.Sprintf("asymmetric input (%d), output (%d) speeds not supported",
				baudToValue(uint64(s.lastIn)), baudToValue(uint64(s.lastOut)))}
		}
	}
	return nil
}

func (s *Stty) setSpeed(in, out bool, speed uint64, arg string, mode *termios) error {
	q := quotearg.Quote(arg, quotearg.Options{Style: quotearg.ShellEscapeAlways})
	if in {
		s.lastIn = int64(speed)
		if cfsetispeed(mode, speed) != nil {
			return &fatal{"unsupported ispeed " + q}
		}
	}
	if out {
		s.lastOut = int64(speed)
		if cfsetospeed(mode, speed) != nil {
			return &fatal{"unsupported ospeed " + q}
		}
	}
	return nil
}

// setMode applies one mode; it returns false for a '-' the mode does not
// take.
func setMode(info *modeInfo, reversed bool, m *termios) bool {
	if reversed && info.flags&rev == 0 {
		return false
	}
	if p := m.flag(info.kind); p != nil {
		if reversed {
			*p &^= info.mask | info.bits
		} else {
			*p = *p&^info.mask | info.bits
		}
		return true
	}
	switch info.name {
	case "evenp", "parity":
		if reversed {
			m.cflag = m.cflag&^(c.PARENB|c.CSIZE) | c.CS8
		} else {
			m.cflag = m.cflag&^(c.PARODD|c.CSIZE) | c.PARENB | c.CS7
		}
	case "oddp":
		if reversed {
			m.cflag = m.cflag&^(c.PARENB|c.CSIZE) | c.CS8
		} else {
			m.cflag = m.cflag&^c.CSIZE | c.CS7 | c.PARODD | c.PARENB
		}
	case "nl":
		if reversed {
			m.iflag = (m.iflag | c.ICRNL) &^ (c.INLCR | c.IGNCR)
			m.oflag = (m.oflag | c.ONLCR) &^ (c.OCRNL | c.ONLRET)
		} else {
			m.iflag &^= c.ICRNL
			m.oflag &^= c.ONLCR
		}
	case "ek":
		m.cc[c.VERASE] = 127
		m.cc[c.VKILL] = 'U' & 0x1f
	case "sane":
		saneMode(m)
	case "cbreak":
		if reversed {
			m.lflag |= c.ICANON
		} else {
			m.lflag &^= c.ICANON
		}
	case "pass8":
		if reversed {
			m.cflag = m.cflag&^c.CSIZE | c.CS7 | c.PARENB
			m.iflag |= c.ISTRIP
		} else {
			m.cflag = m.cflag&^(c.PARENB|c.CSIZE) | c.CS8
			m.iflag &^= c.ISTRIP
		}
	case "litout":
		if reversed {
			m.cflag = m.cflag&^c.CSIZE | c.CS7 | c.PARENB
			m.iflag |= c.ISTRIP
			m.oflag |= c.OPOST
		} else {
			m.cflag = m.cflag&^(c.PARENB|c.CSIZE) | c.CS8
			m.iflag &^= c.ISTRIP
			m.oflag &^= c.OPOST
		}
	case "raw", "cooked":
		if (info.name == "raw") == reversed {
			m.iflag |= c.BRKINT | c.IGNPAR | c.ISTRIP | c.ICRNL | c.IXON
			m.oflag |= c.OPOST
			m.lflag |= c.ISIG | c.ICANON
		} else {
			m.iflag = 0
			m.oflag &^= c.OPOST
			m.lflag &^= c.ISIG | c.ICANON | c.XCASE
			m.cc[c.VMIN] = 1
			m.cc[c.VTIME] = 0
		}
	case "decctlq":
		if reversed {
			m.iflag |= c.IXANY
		} else {
			m.iflag &^= c.IXANY
		}
	case "tabs":
		switch {
		case c.TABDLY != 0 && reversed:
			m.oflag = m.oflag&^c.TABDLY | c.TAB3
		case c.TABDLY != 0:
			m.oflag = m.oflag&^c.TABDLY | c.TAB0
		case reversed:
			m.oflag |= c.OXTABS
		default:
			m.oflag &^= c.OXTABS
		}
	case "lcase", "LCASE":
		if reversed {
			m.lflag &^= c.XCASE
			m.iflag &^= c.IUCLC
			m.oflag &^= c.OLCUC
		} else {
			m.lflag |= c.XCASE
			m.iflag |= c.IUCLC
			m.oflag |= c.OLCUC
		}
	case "crt":
		m.lflag |= c.ECHOE | c.ECHOCTL | c.ECHOKE
	case "dec":
		m.cc[c.VINTR] = 3
		m.cc[c.VERASE] = 127
		m.cc[c.VKILL] = 21
		m.lflag |= c.ECHOE | c.ECHOCTL | c.ECHOKE
		m.iflag &^= c.IXANY
	}
	return true
}

func setControlChar(info controlInfo, arg string, m *termios) error {
	var value uint64
	switch {
	case info.name == "min" || info.name == "time":
		n, err := integerArg(arg, 255)
		if err != nil {
			return err
		}
		value = n
	case len(arg) <= 1:
		if arg != "" {
			value = uint64(arg[0])
		}
	case arg == "^-" || arg == "undef":
		value = uint64(vdisable)
	case arg[0] == '^':
		if arg[1] == '?' {
			value = 127
		} else {
			value = uint64(arg[1]) &^ 0o140
		}
	default:
		n, err := integerArg(arg, 255)
		if err != nil {
			return err
		}
		value = n
	}
	m.cc[info.index] = byte(value)
	return nil
}

func saneMode(m *termios) {
	for _, ci := range controls {
		m.cc[ci.index] = ci.sane
	}
	for i := range modes {
		info := &modes[i]
		if info.flags&noSetAttr != 0 {
			continue
		}
		p := m.flag(info.kind)
		switch {
		case info.flags&saneSet != 0:
			*p = *p&^info.mask | info.bits
		case info.flags&saneUnset != 0:
			*p &^= info.mask | info.bits
		}
	}
}

// visible is how a control character is shown: ^C, ^?, M-a, <undef>.
func visible(ch byte) string {
	if ch == vdisable {
		return "<undef>"
	}
	switch {
	case ch < 32:
		return "^" + string(rune(ch+64))
	case ch < 127:
		return string(rune(ch))
	case ch == 127:
		return "^?"
	case ch >= 128+32:
		if ch < 128+127 {
			return "M-" + string(rune(ch-128))
		}
		return "M-^?"
	}
	return "M-^" + string(rune(ch-128+64))
}

// shownControl reports whether a control character is listed: flush is
// discard's old name, and swtch is not listed where it is susp.
func shownControl(ci controlInfo) bool {
	if ci.name == "flush" {
		return false
	}
	if ci.name == "swtch" && swtchIsSusp {
		return false
	}
	return true
}

func (s *Stty) displaySpeed(m *termios, fancy bool) {
	in, out := cfgetispeed(m), cfgetospeed(m)
	switch {
	case (in == 0 || in == out) && fancy:
		s.wrapf("speed %d baud;", baudToValue(out))
	case in == 0 || in == out:
		s.wrapf("%d\n", baudToValue(out))
	case fancy:
		s.wrapf("ispeed %d baud; ospeed %d baud;", baudToValue(in), baudToValue(out))
	default:
		s.wrapf("%d %d\n", baudToValue(in), baudToValue(out))
	}
	if !fancy {
		s.col = 0
	}
}

func (s *Stty) displayWindowSize(fancy bool) error {
	rows, cols, err := getWinsize(s.fd)
	if err != nil {
		if !isEINVAL(err) {
			return &fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))}
		}
		if !fancy {
			return &fatal{fmt.Sprintf("%s: no size information for this device", quotef(s.device))}
		}
		return nil
	}
	if fancy {
		s.wrapf("rows %d; columns %d;", rows, cols)
	} else {
		s.wrapf("%d %d\n", rows, cols)
		s.col = 0
	}
	return nil
}

func (s *Stty) setWindowSize(rows, cols int) error {
	r, c, err := getWinsize(s.fd)
	if err != nil {
		if !isEINVAL(err) {
			return &fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))}
		}
		r, c = 0, 0
	}
	if rows >= 0 {
		r = rows
	}
	if cols >= 0 {
		c = cols
	}
	if err := setWinsize(s.fd, r, c); err != nil {
		return &fatal{fmt.Sprintf("%s: %s", quotef(s.device), core.ErrText(err))}
	}
	return nil
}

func (s *Stty) displayChanged(m *termios) {
	s.displaySpeed(m, true)
	if hasLine {
		s.wrapf("line = %d;", m.line)
	}
	s.newline()

	empty := true
	for _, ci := range controls {
		if ci.name == "min" {
			break
		}
		if m.cc[ci.index] == ci.sane || !shownControl(ci) {
			continue
		}
		empty = false
		s.wrapf("%s = %s;", ci.name, visible(m.cc[ci.index]))
	}
	if m.lflag&c.ICANON == 0 {
		s.wrapf("min = %d; time = %d;\n", m.cc[c.VMIN], m.cc[c.VTIME])
	} else if !empty {
		fmt.Fprint(s.out, "\n")
	}
	s.col = 0

	empty = true
	prev := control
	for i := range modes {
		info := &modes[i]
		if info.flags&omit != 0 {
			continue
		}
		if info.kind != prev {
			if !empty {
				s.newline()
				empty = true
			}
			prev = info.kind
		}
		mask := info.mask
		if mask == 0 {
			mask = info.bits
		}
		if *m.flag(info.kind)&mask == info.bits {
			if info.flags&saneUnset != 0 {
				s.wrapf("%s", info.name)
				empty = false
			}
		} else if info.flags&(saneSet|rev) == saneSet|rev {
			s.wrapf("-%s", info.name)
			empty = false
		}
	}
	if !empty {
		fmt.Fprint(s.out, "\n")
	}
	s.col = 0
}

func (s *Stty) displayAll(m *termios) {
	s.displaySpeed(m, true)
	_ = s.displayWindowSize(true)
	if hasLine {
		s.wrapf("line = %d;", m.line)
	}
	s.newline()
	for _, ci := range controls {
		if ci.name == "min" {
			break
		}
		if !shownControl(ci) {
			continue
		}
		s.wrapf("%s = %s;", ci.name, visible(m.cc[ci.index]))
	}
	s.wrapf("min = %d; time = %d;", m.cc[c.VMIN], m.cc[c.VTIME])
	if s.col != 0 {
		fmt.Fprint(s.out, "\n")
	}
	s.col = 0
	prev := control
	for i := range modes {
		info := &modes[i]
		if info.flags&omit != 0 {
			continue
		}
		if info.kind != prev {
			s.newline()
			prev = info.kind
		}
		mask := info.mask
		if mask == 0 {
			mask = info.bits
		}
		switch {
		case *m.flag(info.kind)&mask == info.bits:
			s.wrapf("%s", info.name)
		case info.flags&rev != 0:
			s.wrapf("-%s", info.name)
		}
	}
	s.newline()
}

func (s *Stty) displayRecoverable(m *termios) {
	var b strings.Builder
	fmt.Fprintf(&b, "%x:%x:%x:%x", m.iflag, m.oflag, m.cflag, m.lflag)
	for i := range ncc {
		fmt.Fprintf(&b, ":%x", m.cc[i])
	}
	b.WriteByte('\n')
	_, _ = io.WriteString(s.out, b.String())
}

// recoverMode reads back what -g prints.
func recoverMode(arg string, m *termios) bool {
	parts := strings.Split(arg, ":")
	if len(parts) != 4+ncc {
		return false
	}
	var flags [4]uint64
	for i := range 4 {
		v, err := strconv.ParseUint(parts[i], 16, flagBits)
		if err != nil || parts[i] == "" {
			return false
		}
		flags[i] = v
	}
	var cc [32]byte
	for i := range ncc {
		v, err := strconv.ParseUint(parts[4+i], 16, 8)
		if err != nil || parts[4+i] == "" {
			return false
		}
		cc[i] = byte(v)
	}
	m.iflag, m.oflag, m.cflag, m.lflag = flags[0], flags[1], flags[2], flags[3]
	m.cc = cc
	return true
}
