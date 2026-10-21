// Package dircolors implements the engine behind dircolorsx, a
// GNU-compatible implementation of the GNU/UNIX dircolors command: print
// shell commands that set LS_COLORS, the colors lsx --color uses.
package dircolors

import (
	"bufio"
	_ "embed" // the color database
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/fnmatch"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "dircolorsx"

// Database is GNU's default color database, which -p prints and which is
// read when no FILE is given. It is GNU coreutils 9.7's dircolors.hin, with
// its notice, in the squeezed form GNU's build gives it (no blank lines,
// single spaces): what GNU dircolors -p prints.
//
//go:embed database.txt
var Database string

// Shell is the syntax of the commands printed.
type Shell int

const (
	// Guess picks the syntax from $SHELL: C shell for csh and tcsh, Bourne
	// shell for anything else.
	Guess Shell = iota
	// Bourne prints LS_COLORS='...'; export LS_COLORS (-b).
	Bourne
	// C prints setenv LS_COLORS '...' (-c).
	C
)

// Options mirrors the GNU dircolors flag surface. The zero value prints the
// default database as commands for the shell in $SHELL.
type Options struct {
	// Shell (-b, -c) is the syntax to print.
	Shell Shell
	// PrintDatabase (-p) prints the default database instead.
	PrintDatabase bool
	// PrintLsColors (--print-ls-colors) prints each color as it looks, one
	// per line, instead of shell commands.
	PrintLsColors bool
}

// Dircolors is the engine. Construct it with New.
type Dircolors struct {
	Options

	// Getenv reads SHELL, TERM and COLORTERM. It defaults to os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag
}

// New builds a dircolors engine reading from and writing to s.
func New(opts Options, s core.Streams) *Dircolors {
	return &Dircolors{Options: opts, Getenv: os.Getenv, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the commands for FILE, or for the default database when there
// is no operand.
func (d *Dircolors) Run(args []string) error {
	if (d.PrintDatabase || d.PrintLsColors) && d.Shell != Guess {
		return core.Usagef("the options to output non shell syntax,\nand to select a shell syntax are mutually exclusive")
	}
	if d.PrintDatabase && d.PrintLsColors {
		return core.Usagef("options --print-database and --print-ls-colors are mutually exclusive")
	}
	if d.PrintDatabase && len(args) > 0 {
		return core.Usagef("extra operand %s\nfile operands cannot be combined with --print-database (-p)", quote(args[0]))
	}
	if len(args) > 1 {
		return core.Usagef("extra operand %s", quote(args[1]))
	}
	if d.PrintDatabase {
		if _, err := io.WriteString(d.streams.Out, Database); err != nil && !core.IsClosedPipe(err) {
			d.diag.Errorf("write error: %s", core.ErrText(err))
		}
		return d.diag.Result()
	}

	shell := d.Shell
	if shell == Guess && !d.PrintLsColors {
		sh := d.Getenv("SHELL")
		switch base := path.Base(strings.ReplaceAll(sh, `\`, "/")); {
		case sh == "":
			d.diag.Errorf("no SHELL environment variable, and no shell type option given")
			return d.diag.Result()
		case base == "csh" || base == "tcsh":
			shell = C
		default:
			shell = Bourne
		}
	}

	var colors strings.Builder
	var ok bool
	if len(args) == 0 {
		ok = d.parse(strings.NewReader(Database), "", &colors)
	} else {
		ok = d.parseFile(args[0], &colors)
	}
	if !ok {
		return core.ErrFailed
	}
	out := colors.String()
	if !d.PrintLsColors {
		if shell == Bourne {
			out = "LS_COLORS='" + out + "';\nexport LS_COLORS\n"
		} else {
			out = "setenv LS_COLORS '" + out + "'\n"
		}
	}
	if _, err := io.WriteString(d.streams.Out, out); err != nil && !core.IsClosedPipe(err) {
		d.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return d.diag.Result()
}

func (d *Dircolors) parseFile(name string, b *strings.Builder) bool {
	r := d.streams.In
	if name != "-" {
		f, err := fsys.Open(name)
		if err != nil {
			d.diag.Errorf("%s: %s", quotef(name), core.ErrText(err))
			return false
		}
		defer f.Close()
		r = f
	}
	return d.parse(r, name, b)
}

func quote(s string) string { return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale}) }

func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

// The keywords of the database, with the two-letter LS_COLORS names they
// stand for. Several are aliases from Slackware's dircolors.
var codes = map[string]string{
	"NORMAL": "no", "NORM": "no", "FILE": "fi", "RESET": "rs", "DIR": "di",
	"LNK": "ln", "LINK": "ln", "SYMLINK": "ln", "ORPHAN": "or", "MISSING": "mi",
	"FIFO": "pi", "PIPE": "pi", "SOCK": "so", "BLK": "bd", "BLOCK": "bd",
	"CHR": "cd", "CHAR": "cd", "DOOR": "do", "EXEC": "ex", "LEFT": "lc",
	"LEFTCODE": "lc", "RIGHT": "rc", "RIGHTCODE": "rc", "END": "ec",
	"ENDCODE": "ec", "SUID": "su", "SETUID": "su", "SGID": "sg",
	"SETGID": "sg", "STICKY": "st", "OTHER_WRITABLE": "ow", "OWR": "ow",
	"STICKY_OTHER_WRITABLE": "tw", "OWT": "tw", "CAPABILITY": "ca",
	"MULTIHARDLINK": "mh", "CLRTOEOL": "cl",
}

func space(c byte) bool { return c == ' ' || (c >= '\t' && c <= '\r') }

// splitLine returns a line's keyword and argument; an empty keyword for a
// blank line or a comment.
func splitLine(line string) (keyword, arg string, hasArg bool) {
	i := 0
	for i < len(line) && space(line[i]) {
		i++
	}
	if i == len(line) || line[i] == '#' {
		return "", "", false
	}
	start := i
	for i < len(line) && !space(line[i]) {
		i++
	}
	keyword = line[start:i]
	for i < len(line) && space(line[i]) {
		i++
	}
	if i == len(line) || line[i] == '#' {
		return keyword, "", false
	}
	arg = line[i:]
	if j := strings.IndexByte(arg, '#'); j >= 0 {
		arg = arg[:j]
	}
	return keyword, strings.TrimRightFunc(arg, func(r rune) bool { return r < 0x80 && space(byte(r)) }), true
}

// parse reads a database and appends its entries for this terminal to b.
// file is empty for the built-in database.
func (d *Dircolors) parse(r io.Reader, file string, b *strings.Builder) bool {
	term := d.Getenv("TERM")
	if term == "" {
		term = "none"
	}
	colorterm := d.Getenv("COLORTERM")
	const (
		termNo = iota
		termYes
		termSure
		global
	)
	state := global
	ok := true
	where := "<internal>"
	if file != "" {
		where = quotef(file)
	}
	br := bufio.NewReader(r)
	for n := 1; ; n++ {
		line, err := br.ReadString('\n')
		if line == "" {
			if err != nil && !errors.Is(err, io.EOF) {
				d.diag.Errorf("%s: read error: %s", where, core.ErrText(err))
				ok = false
			}
			break
		}
		if i := strings.IndexByte(line, 0); i >= 0 {
			line = line[:i] // a C string ends at its first NUL
		}
		keyword, arg, hasArg := splitLine(line)
		if keyword == "" {
			continue
		}
		if !hasArg {
			d.diag.Errorf("%s:%d: invalid line;  missing second token", where, n)
			ok = false
			continue
		}
		unrecognized := false
		upper := strings.ToUpper(keyword)
		switch upper {
		case "TERM", "COLORTERM":
			value := term
			if upper == "COLORTERM" {
				value = colorterm
			}
			if state != termSure {
				state = termNo
				if fnmatch.Match(arg, value, false) {
					state = termSure
				}
			}
		default:
			if state == termSure {
				state = termYes
			}
			if state == termNo {
				unrecognized = true
				break
			}
			switch {
			case keyword[0] == '.':
				d.entry(b, "*", keyword, arg)
			case keyword[0] == '*':
				d.entry(b, "", keyword, arg)
			case upper == "OPTIONS" || upper == "COLOR" || upper == "EIGHTBIT":
			default:
				if code, known := codes[upper]; known {
					d.entry(b, "", code, arg)
				} else {
					unrecognized = true
				}
			}
		}
		if unrecognized && (state == termSure || state == termYes) {
			d.diag.Errorf("%s:%s: unrecognized keyword %s", where, strconv.Itoa(n), keyword)
			ok = false
		}
		if err != nil {
			break
		}
	}
	return ok
}

// entry appends one KEY=VALUE: entry, or with --print-ls-colors one line
// showing the color.
func (d *Dircolors) entry(b *strings.Builder, prefix, item, arg string) {
	if d.PrintLsColors {
		b.WriteString("\x1b[" + arg + "m" + prefix + item + "\t" + arg + "\x1b[0m\n")
		return
	}
	b.WriteString(prefix)
	b.WriteString(escape(item))
	b.WriteByte('=')
	b.WriteString(escape(arg))
	b.WriteByte(':')
}

// escape quotes a value for the single-quoted shell string and for ls: a
// quote closes and reopens the string, and a ':' or '=' that is not already
// escaped by a backslash or a caret gets one.
func escape(s string) string {
	var b strings.Builder
	needBackslash := true
	for i := range len(s) {
		c := s[i]
		switch c {
		case '\'':
			b.WriteString(`'\'`)
			needBackslash = true
		case '\\', '^':
			needBackslash = !needBackslash
		case ':', '=':
			if needBackslash {
				b.WriteByte('\\')
			}
			needBackslash = true
		default:
			needBackslash = true
		}
		b.WriteByte(c)
	}
	return b.String()
}
