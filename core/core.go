// Package core holds the plumbing every gnux engine shares: the standard
// streams, GNU-style diagnostics, exit-status errors and confirmation prompts.
// The engines themselves live in sub-packages, one per tool (core/rm, core/cp, …).
package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unicode"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/mattn/go-isatty"
)

// ErrFailed reports that a tool ran but one or more operations failed. It is
// returned after all diagnostics have already been written to the error stream,
// so callers should exit with status 1 without printing it again.
var ErrFailed = errors.New("command failed")

// ExitError asks for a specific, silent exit status. Tools whose status carries
// meaning beyond 0/1 (ls exits 2 on serious trouble, env 125–127) return it once
// their diagnostics are written.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// UsageError is a command-line mistake (missing operand, bad flag value). The
// caller prints it prefixed with the tool name, followed by the standard
// "Try '<tool> --help' for more information." hint.
type UsageError struct {
	Msg string
	// Code is the exit status, when it is not the tool's usual one for a
	// usage error (zero). GNU exits 1 for an invalid option argument read
	// with argmatch even in a tool, such as ls, whose usage errors exit 2.
	Code int
}

func (e *UsageError) Error() string {
	return e.Msg
}

// Usagef builds a UsageError from a format string.
func Usagef(format string, a ...any) error {
	return &UsageError{Msg: fmt.Sprintf(format, a...)}
}

// Streams bundles the three standard streams a tool reads and writes. Passing
// them explicitly keeps every engine testable against in-memory buffers, and
// lets an application embedding an engine capture or silence its output. All
// three fields must be set: use Stdio for the process's own streams, or
// Discard and override what you need.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// Stdio returns the process's standard input, output and error.
func Stdio() Streams {
	return Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
}

// Discard returns streams that read as empty and drop everything written, for
// callers that only care about an engine's side effects and returned error.
func Discard() Streams {
	return Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
}

// IsTerminal reports whether v (a stream) is an *os.File attached to a terminal.
func IsTerminal(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// Diag writes a tool's diagnostics the way GNU does - "<prog>: <message>" on the
// error stream - and counts errors so the engine can report its exit status. It
// is safe for concurrent use.
type Diag struct {
	prog string

	mu       sync.Mutex // guards out/err writes and errCount
	out, err io.Writer
	errCount int
}

// NewDiag builds a Diag for the tool named prog writing to s.
func NewDiag(prog string, s Streams) *Diag {
	return &Diag{prog: prog, out: s.Out, err: s.Err}
}

// Prog returns the tool name diagnostics are prefixed with.
func (d *Diag) Prog() string {
	return d.prog
}

// Errorf writes an error diagnostic and counts it toward the exit status.
func (d *Diag) Errorf(format string, a ...any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Fprintf(d.err, d.prog+": "+format+"\n", a...)
	d.errCount++
}

// Warnf writes a diagnostic that does not affect the exit status.
func (d *Diag) Warnf(format string, a ...any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Fprintf(d.err, d.prog+": "+format+"\n", a...)
}

// Printf writes one line of regular output (e.g. --verbose reports) to stdout.
func (d *Diag) Printf(format string, a ...any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fmt.Fprintf(d.out, format+"\n", a...)
}

// ChildErr returns the error stream for a child process that runs while
// these diagnostics may still be written (nicex's warnings, timeoutx
// --verbose). A file is returned as it is: the child writes to it directly,
// and keeps seeing a terminal there. Any other writer is wrapped so that the
// child's output, which os/exec copies in from a goroutine of its own, takes
// turns with the diagnostics instead of racing them.
func (d *Diag) ChildErr() io.Writer {
	if f, ok := d.err.(*os.File); ok {
		return f
	}
	return diagWriter{d}
}

// diagWriter writes to a Diag's error stream under its lock.
type diagWriter struct{ d *Diag }

func (w diagWriter) Write(p []byte) (int, error) {
	w.d.mu.Lock()
	defer w.d.mu.Unlock()
	return w.d.err.Write(p)
}

// Failed reports whether any error has been recorded.
func (d *Diag) Failed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.errCount > 0
}

// Result returns ErrFailed if any error was recorded, otherwise nil.
func (d *Diag) Result() error {
	if d.Failed() {
		return ErrFailed
	}
	return nil
}

// Prompter asks yes/no questions on the error stream and reads the answer from
// the input stream, as rm -i, cp -i and friends do.
type Prompter struct {
	// Fn, when non-nil, is consulted instead of reading from the input stream.
	// It is the injection point tests use to answer confirmation prompts.
	Fn func(question string) bool

	prog   string
	mu     sync.Mutex
	reader *bufio.Reader
	err    io.Writer
}

// NewPrompter builds a Prompter for the tool named prog.
func NewPrompter(prog string, s Streams) *Prompter {
	return &Prompter{prog: prog, reader: bufio.NewReader(s.In), err: s.Err}
}

// Ask poses question and reports whether the answer was affirmative (it starts
// with 'y' or 'Y'). End of input counts as "no".
func (p *Prompter) Ask(question string) bool {
	if p.Fn != nil {
		return p.Fn(question)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.err, "%s: %s ", p.prog, question)

	line, err := p.reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	line = strings.TrimSpace(line)
	return line != "" && (line[0] == 'y' || line[0] == 'Y')
}

// ErrText renders an error the way GNU tools do: the bare, capitalized syscall
// message (e.g. "Permission denied", "Directory not empty") with the path and
// operation stripped. The common cases are normalized so the text is identical
// on every platform.
func ErrText(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, syscall.ENOTEMPTY):
		// Checked before fs.ErrExist, which Go also matches ENOTEMPTY against.
		return "Directory not empty"
	case errors.Is(err, fs.ErrNotExist):
		return "No such file or directory"
	case errors.Is(err, fs.ErrPermission):
		return "Permission denied"
	case errors.Is(err, fs.ErrExist):
		return "File exists"
	}

	var perr *fs.PathError
	if errors.As(err, &perr) {
		err = perr.Err
	}
	var lerr *os.LinkError
	if errors.As(err, &lerr) {
		err = lerr.Err
	}
	msg := err.Error()
	if msg == "" {
		return msg
	}
	runes := []rune(msg)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// Describe returns the GNU noun for a filesystem entry (e.g. "regular empty
// file"), as used in prompts and diagnostics.
func Describe(info os.FileInfo) string {
	m := info.Mode()
	switch {
	case m&os.ModeSymlink != 0:
		return "symbolic link"
	case m.IsDir():
		return "directory"
	case m&os.ModeNamedPipe != 0:
		return "fifo"
	case m&os.ModeSocket != 0:
		return "socket"
	case m&os.ModeDevice != 0:
		if m&os.ModeCharDevice != 0 {
			return "character special file"
		}
		return "block special file"
	case m.IsRegular():
		if info.Size() == 0 {
			return "regular empty file"
		}
		return "regular file"
	default:
		return "file"
	}
}

// IsRoot reports whether path resolves to a filesystem root (e.g. "/" or "C:\").
func IsRoot(path string) bool {
	abs, err := fsys.Abs(path)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	return filepath.Dir(abs) == abs
}

// Plural returns "s" unless n is 1, for building count-aware messages.
func Plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
