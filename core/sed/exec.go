package sed

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
	"strconv"

	"github.com/braswelljr/gnux/core"
)

// action is how a run of the script over one pattern space ended.
type action uint8

const (
	actPrint   action = iota // the end of the script: print, then the next cycle
	actDelete                // the next cycle, without printing
	actRestart               // D: run the script again on what is left, reading nothing
	actQuit                  // q or Q: stop
)

// appended is text waiting for the end of the cycle: what a, r and R queue.
type appended struct {
	text []byte
	file string // r: the file to copy; empty for text
}

// abort is what the machine panics with to stop at once; run recovers it. A
// script can fail deep inside a command (a write error, a file that cannot
// be edited), and GNU sed stops right there too.
type abort struct {
	err error // what Run returns; nil for a quiet stop (a closed pipe)
}

// machine runs a compiled script.
type machine struct {
	e    *Sed
	diag *core.Diag
	prog []command

	quiet    bool
	posix    bool
	separate bool
	delim    byte
	width    int // the l command's line length; at most 1 never wraps

	// Input (io.go).
	args     []string
	stdin    *bufio.Reader
	files    *bufio.Reader // reused for every input file
	cur      *bufio.Reader // the current input, nil between files
	curFile  *os.File
	name     string // the current input's name, as F prints it
	atEnd    bool   // the current input has been read to its end
	lineNo   int64
	badInput bool
	readers  map[string]*lineReader // the files of R commands

	// Output (io.go).
	stdout  *output
	stderr  *output
	out     *output // where the script's output goes: stdout, or the file edited in place
	outputs map[string]*output
	edit    *edit

	ps, hold    []byte // the pattern and hold spaces
	spare       []byte // working space, swapped with ps by s and y
	chomped     bool   // the pattern space's line ended with a newline
	holdChomped bool
	replaced    bool           // an s has succeeded since the last line was read (t, T)
	last        *regexp.Regexp // the last regular expression used
	queue       []appended
	scratch     []byte
	status      int // the exit status of q or Q
}

func newMachine(e *Sed, prog program, args []string, stdin *bufio.Reader) *machine {
	m := &machine{
		e:           e,
		diag:        e.diag,
		prog:        prog.commands,
		quiet:       e.Quiet || prog.quiet,
		posix:       e.Posix || (e.Getenv != nil && e.Getenv("POSIXLY_CORRECT") != ""),
		separate:    e.Separate || e.InPlace,
		delim:       '\n',
		width:       e.LineLength,
		args:        args,
		stdin:       stdin,
		holdChomped: true,
	}
	if e.NullData {
		m.delim = 0
	}
	if m.width == 0 {
		m.width = DefaultLineLength
	}
	// A terminal shows each line as it is produced, as C's line buffering
	// would.
	m.stdout = &output{m: m, w: core.NewWriter(e.streams.Out), name: "stdout", sync: e.Unbuffered || core.IsTerminal(e.streams.Out)}
	m.out = m.stdout
	return m
}

// run executes the script over the whole input and returns Run's result.
func (m *machine) run() (err error) {
	defer func() {
		r := recover()
		m.cleanup()
		if r == nil {
			return
		}
		a, ok := r.(abort)
		if !ok {
			panic(r)
		}
		err = a.err
	}()
	m.openOutputs()

	restart := false
cycles:
	for restart || m.readLine(false) {
		restart = false
		switch m.execute() {
		case actPrint:
			if !m.quiet {
				m.out.line(m.ps, m.chomped)
			}
		case actRestart:
			m.flushQueue()
			restart = true
		case actQuit:
			break cycles
		}
	}
	m.finish()

	switch {
	case m.status != 0:
		return &core.ExitError{Code: m.status}
	case m.badInput:
		return &core.ExitError{Code: ExitBadInput}
	}
	return nil
}

// fatalf reports an error that ends the run with status ExitIO.
func (m *machine) fatalf(format string, a ...any) {
	m.diag.Errorf(format, a...)
	panic(abort{&core.ExitError{Code: ExitIO}})
}

// regex returns the regular expression a command uses: its own, or the last
// one used when the script gave an empty one.
func (m *machine) regex(re *regexp.Regexp) *regexp.Regexp {
	if re == nil {
		if m.last == nil {
			m.diag.Errorf("no previous regular expression")
			panic(abort{core.ErrFailed})
		}
		return m.last
	}
	m.last = re
	return re
}

// matches reports whether one address selects the current line.
func (m *machine) matches(a *address) bool {
	switch a.kind {
	case addrLine:
		return m.lineNo == a.n
	case addrStep:
		return m.lineNo >= a.n && (m.lineNo-a.n)%a.step == 0
	case addrLast:
		return m.lastLine()
	case addrRegex:
		return m.regex(a.re).Match(m.ps)
	}
	return true
}

// selected reports whether a command's addresses select the current line,
// before any negation, and moves its range along.
func (m *machine) selected(c *command) bool {
	if c.a2.kind == addrNone {
		return m.matches(&c.a1)
	}
	if !c.active {
		if c.a1.kind == addrLine && c.a1.n == 0 {
			// 0,/regex/: the range is open before the first line, so
			// the regex can end it on that very line.
			if c.started {
				return false
			}
			c.started = true
		} else {
			if !m.matches(&c.a1) {
				return false
			}
			// A second address at or before this line selects only it.
			switch c.a2.kind {
			case addrLine:
				if c.a2.n <= m.lineNo {
					return true
				}
				c.end = c.a2.n
			case addrPlus:
				if c.a2.n == 0 {
					return true
				}
				c.end = m.lineNo + c.a2.n
			case addrMultiple:
				if c.a2.n == 0 || m.lineNo%c.a2.n == 0 {
					return true
				}
				c.end = (m.lineNo/c.a2.n + 1) * c.a2.n
			}
			c.active = true
			return true
		}
		c.active = true
	}
	switch c.a2.kind {
	case addrLine, addrPlus, addrMultiple:
		if m.lineNo >= c.end {
			c.active = false
		}
		return m.lineNo <= c.end
	}
	if m.matches(&c.a2) {
		c.active = false
	}
	return true
}

// execute runs the script once over the pattern space.
func (m *machine) execute() action {
	prog := m.prog
	for pc := 0; pc < len(prog); {
		c := &prog[pc]
		pc++
		if (c.a1.kind != addrNone || c.negate) && m.selected(c) == c.negate {
			if c.name == '{' {
				pc = c.jump
			}
			continue
		}
		switch c.name {
		case '{':
			// The block's commands follow.
		case '=':
			m.scratch = append(strconv.AppendInt(m.scratch[:0], m.lineNo, 10), '\n')
			m.out.text(m.scratch)
		case 'a':
			m.queue = append(m.queue, appended{text: c.text})
		case 'i':
			m.out.text(c.text)
		case 'c':
			// A range is replaced as a whole: the text is printed at its
			// end only.
			if c.a2.kind == addrNone || !c.active {
				m.out.text(c.text)
			}
			return actDelete
		case 'b':
			pc = c.jump
		case 't':
			if m.replaced {
				m.replaced = false
				pc = c.jump
			}
		case 'T':
			if m.replaced {
				m.replaced = false
			} else {
				pc = c.jump
			}
		case 'd':
			return actDelete
		case 'D':
			i := bytes.IndexByte(m.ps, m.delim)
			if i < 0 {
				return actDelete
			}
			m.ps = m.ps[:copy(m.ps, m.ps[i+1:])]
			return actRestart
		case 'g':
			m.ps = append(m.ps[:0], m.hold...)
			m.chomped = m.holdChomped
		case 'G':
			m.ps = append(append(m.ps, m.delim), m.hold...)
		case 'h':
			m.hold = append(m.hold[:0], m.ps...)
			m.holdChomped = m.chomped
		case 'H':
			m.hold = append(append(m.hold, m.delim), m.ps...)
		case 'x':
			m.ps, m.hold = m.hold, m.ps
			m.chomped, m.holdChomped = m.holdChomped, m.chomped
		case 'n', 'N':
			if m.noNextLine() {
				// GNU sed prints the pattern space before it stops;
				// POSIX has it stop without printing.
				if m.posix {
					return actDelete
				}
				return actPrint
			}
			if c.name == 'n' && !m.quiet {
				m.out.line(m.ps, m.chomped)
			}
			m.readLine(c.name == 'N')
		case 'p':
			m.out.line(m.ps, m.chomped)
		case 'P':
			if i := bytes.IndexByte(m.ps, m.delim); i >= 0 {
				m.out.line(m.ps[:i], true)
			} else {
				m.out.line(m.ps, m.chomped)
			}
		case 'l':
			width := m.width
			if c.num >= 0 {
				width = c.num
			}
			m.list(width)
		case 'q':
			if !m.quiet {
				m.out.line(m.ps, m.chomped)
			}
			m.flushQueue()
			m.status = c.num
			return actQuit
		case 'Q':
			m.status = c.num
			return actQuit
		case 'r':
			m.queue = append(m.queue, appended{file: c.file})
		case 'R':
			if line := m.readFileLine(c.file); len(line) > 0 {
				m.queue = append(m.queue, appended{text: line})
			}
		case 's':
			if !m.substitute(c) {
				break
			}
			m.replaced = true
			if c.print {
				m.out.line(m.ps, m.chomped)
			}
			if c.out != nil {
				c.out.line(m.ps, m.chomped)
			}
		case 'w':
			c.out.line(m.ps, m.chomped)
		case 'W':
			if i := bytes.IndexByte(m.ps, m.delim); i >= 0 {
				c.out.line(m.ps[:i], true)
			} else {
				c.out.line(m.ps, m.chomped)
			}
		case 'y':
			m.ps, m.spare = c.tr.apply(m.ps, m.spare)
		case 'z':
			m.ps = m.ps[:0]
		case 'F':
			m.scratch = append(append(m.scratch[:0], m.name...), '\n')
			m.out.text(m.scratch)
		}
	}
	return actPrint
}

// substitute runs an s command on the pattern space and reports whether it
// replaced anything.
func (m *machine) substitute(c *command) bool {
	re := m.regex(c.re)
	ps := m.ps
	// The matches come from the regexp package's own scan, which keeps the
	// context a match needs (^ only at the start, a word boundary) and
	// skips an empty match right after another match, as sed does: a
	// "s/x*/-/g" turns "abc" into "-a-b-c-" and "baaac" under "s/a*/x/g"
	// into "xbxcx".
	var one [1][]int
	var matches [][]int
	if c.global || c.nth > 1 {
		n := -1
		if !c.global {
			n = c.nth
		}
		matches = re.FindAllSubmatchIndex(ps, n)
		if len(matches) < c.nth {
			return false
		}
		matches = matches[c.nth-1:]
	} else {
		loc := re.FindSubmatchIndex(ps)
		if loc == nil {
			return false
		}
		one[0] = loc
		matches = one[:]
	}
	dst := m.spare[:0]
	prev := 0
	for _, loc := range matches {
		dst = append(dst, ps[prev:loc[0]]...)
		dst = c.repl.expand(dst, ps, loc)
		prev = loc[1]
	}
	dst = append(dst, ps[prev:]...)
	m.ps, m.spare = dst, ps
	return true
}

// list writes the pattern space the way the l command shows it: unambiguous,
// with escapes for what does not print, wrapped at width with a backslash
// and ended with a dollar sign.
func (m *machine) list(width int) {
	buf := m.scratch[:0]
	col := 0
	var piece [4]byte
	for _, b := range m.ps {
		p := piece[:0]
		switch b {
		case '\\':
			p = append(p, '\\', '\\')
		case '\a':
			p = append(p, '\\', 'a')
		case '\b':
			p = append(p, '\\', 'b')
		case '\f':
			p = append(p, '\\', 'f')
		case '\n':
			p = append(p, '\\', 'n')
		case '\r':
			p = append(p, '\\', 'r')
		case '\t':
			p = append(p, '\\', 't')
		case '\v':
			p = append(p, '\\', 'v')
		default:
			if b >= ' ' && b < 0x7f {
				p = append(p, b)
			} else {
				p = append(p, '\\', '0'+b>>6, '0'+b>>3&7, '0'+b&7)
			}
		}
		// The last column is kept for the backslash.
		if width > 1 && col+len(p) > width-1 {
			buf = append(buf, '\\', '\n')
			col = 0
		}
		buf = append(buf, p...)
		col += len(p)
	}
	buf = append(buf, '$', '\n')
	m.scratch = buf
	m.out.text(buf)
}
