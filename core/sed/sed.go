// Package sed implements the engine behind sedx, a GNU-compatible
// implementation of the GNU/UNIX sed command: a stream editor that filters
// and transforms text one line at a time under the control of a script.
//
// The script is compiled first (parse.go, with its regular expressions
// translated for Go in regex.go) and only then run against the input
// (exec.go, reading and writing through io.go), so that a mistake anywhere
// in it is reported before a line is read or a file is touched.
package sed

import (
	"io"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sedx"

// DefaultLineLength is the width the l command wraps its output at.
const DefaultLineLength = 70

// Exit statuses beyond 0 and 1, as GNU sed documents them.
const (
	// ExitBadInput: an input file could not be opened.
	ExitBadInput = 2
	// ExitIO: an I/O error while the script ran; sed stopped at once.
	ExitIO = 4
)

// Script is one piece of the script: the argument of one -e or -f. The
// pieces are joined in order, each ending where a newline would.
type Script struct {
	// Text is the script text (-e), or, when File is set, the name of the
	// file that holds it (-f; "-" is standard input).
	Text string
	File bool
}

// Options mirrors the GNU sed flag surface. The zero value is plain sed:
// the script is the first operand, regular expressions are basic ones, every
// line is printed after the script has run on it, and nothing but standard
// output is written.
type Options struct {
	// Scripts (-e, -f) is the script, in command-line order. When it is
	// empty the first operand given to Run is the script.
	Scripts []Script
	// Quiet (-n) prints a line only when the script asks for it.
	Quiet bool
	// Extended (-E, -r) reads the script's regular expressions as POSIX
	// extended ones.
	Extended bool
	// InPlace (-i) writes each file's output back over the file instead of
	// to standard output; it implies Separate.
	InPlace bool
	// Suffix (-iSUFFIX) keeps the original of a file edited in place under
	// its name with Suffix appended, or under Suffix with every '*' replaced
	// by the file's base name. Empty keeps no backup.
	Suffix string
	// FollowSymlinks (--follow-symlinks) edits the file a symbolic link
	// points to in place, instead of replacing the link with a regular file.
	FollowSymlinks bool
	// Separate (-s) treats the input files one by one: line numbers restart
	// and $ is the last line of each.
	Separate bool
	// NullData (-z) separates lines by NUL characters instead of newlines.
	NullData bool
	// Unbuffered (-u) flushes the output after every line.
	Unbuffered bool
	// LineLength (-l) is where the l command wraps a line: zero means
	// DefaultLineLength, and a negative value or 1 never wraps.
	LineLength int
	// Posix (--posix) follows POSIX where GNU sed chooses otherwise at the
	// end of the input: n and N do not print the pattern space when there
	// is no next line. The GNU extensions of the script language stay on.
	Posix bool
	// Sandbox (--sandbox) refuses a script that uses r, R, w or W.
	Sandbox bool
}

// source is one loaded piece of the script.
type source struct {
	text string
	file string // the file it came from; empty for an expression
}

// Sed is the engine. Construct it with New.
type Sed struct {
	Options

	// Getenv looks up an environment variable; it is consulted for
	// POSIXLY_CORRECT, which acts as Posix. It defaults to os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag
}

// New builds a sed engine reading from and writing to s.
func New(opts Options, s core.Streams) *Sed {
	return &Sed{Options: opts, Getenv: os.Getenv, streams: s, diag: core.NewDiag(Name, s)}
}

// Run compiles the script and runs it over the files named by args ("-" or
// no file is standard input). When Options.Scripts is empty, args[0] is the
// script. It returns a *core.UsageError when there is no script,
// core.ErrFailed for a mistake in the script, a *core.ExitError carrying
// ExitBadInput when an input file could not be opened, ExitIO after an I/O
// error, or the status a q or Q command asked for, and nil otherwise.
func (e *Sed) Run(args []string) error {
	scripts := e.Scripts
	if len(scripts) == 0 {
		if len(args) == 0 {
			return core.Usagef("no script specified")
		}
		scripts = []Script{{Text: args[0]}}
		args = args[1:]
	}
	in := e.streams.In
	if in == nil {
		in = strings.NewReader("")
	}
	stdin := core.NewReader(in)

	sources := make([]source, 0, len(scripts))
	for _, s := range scripts {
		if !s.File {
			sources = append(sources, source{text: s.Text})
			continue
		}
		var text []byte
		var err error
		if s.Text == "-" {
			text, err = io.ReadAll(stdin)
		} else {
			text, err = os.ReadFile(s.Text)
		}
		if err != nil {
			e.diag.Errorf("couldn't open file %s: %s", s.Text, core.ErrText(err))
			return &core.ExitError{Code: ExitIO}
		}
		sources = append(sources, source{text: string(text), file: s.Text})
	}

	prog, err := parse(sources, e.Extended, e.Sandbox)
	if err != nil {
		e.diag.Errorf("%s", err)
		return core.ErrFailed
	}
	if e.InPlace && len(args) == 0 {
		e.diag.Errorf("no input files")
		return core.ErrFailed
	}
	if len(args) == 0 {
		args = []string{"-"}
	}
	return newMachine(e, prog, args, stdin).run()
}
