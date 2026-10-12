// Package nohup implements the engine behind nohupx, a GNU-compatible
// implementation of the GNU/UNIX nohup command: run a command immune to
// hangups, with output to a file when it would otherwise go to a terminal.
package nohup

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "nohupx"

// OutFile is the file standard output is appended to when it is a terminal.
const OutFile = "nohup.out"

// Exit statuses nohup reserves, so a caller can tell nohupx's own failures
// from the command's status.
const (
	// ExitCanceled means nohupx itself failed.
	ExitCanceled = 125
	// ExitCannotInvoke means the command was found but could not be run.
	ExitCannotInvoke = 126
	// ExitNotFound means the command could not be found.
	ExitNotFound = 127
)

// Options is empty: GNU nohup has no options besides --help and --version.
type Options struct{}

// Nohup is the engine. Construct it with New.
type Nohup struct {
	Options

	// IsTerminal reports whether a stream is a terminal. It defaults to
	// core.IsTerminal and decides which streams are redirected.
	IsTerminal func(any) bool
	// Getenv reads HOME, where nohup.out goes when the current directory is
	// not writable. It defaults to os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag
}

// New builds a nohup engine for the streams s, which the command inherits
// except where a terminal is replaced.
func New(opts Options, s core.Streams) *Nohup {
	return &Nohup{
		Options:    opts,
		IsTerminal: core.IsTerminal,
		Getenv:     os.Getenv,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// Run runs args as a command that ignores the hangup signal. A terminal on
// standard input is replaced by an unreadable /dev/null; a terminal on
// standard output by nohup.out, appended to, in the current directory or
// else in $HOME; a terminal on standard error by standard output. The result
// is nil or a *core.ExitError: the command's status, 125 when nohupx failed,
// 126 or 127 when the command could not be run.
func (n *Nohup) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	in, out, errw := n.streams.In, n.streams.Out, n.streams.Err
	ignoringInput := n.IsTerminal(in)
	redirectingOut := n.IsTerminal(out)
	redirectingErr := n.IsTerminal(errw)

	if ignoringInput {
		// Write-only, as GNU opens it: a command that reads gets an error
		// rather than waiting on a terminal that may be gone.
		f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return n.fail(ExitCanceled, "failed to render standard input unusable: %s", core.ErrText(err))
		}
		defer f.Close()
		in = f
		if !redirectingOut && !redirectingErr {
			n.diag.Warnf("ignoring input")
		}
	}

	if redirectingOut {
		f, name, err := n.openOut()
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
		if ignoringInput {
			n.diag.Warnf("ignoring input and appending output to '%s'", name)
		} else {
			n.diag.Warnf("appending output to '%s'", name)
		}
	}

	if redirectingErr {
		if !redirectingOut {
			if ignoringInput {
				n.diag.Warnf("ignoring input and redirecting stderr to stdout")
			} else {
				n.diag.Warnf("redirecting stderr to stdout")
			}
		}
		errw = out
	}

	name := args[0]
	path, err := exec.LookPath(name)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		return n.cannotRun(name, err)
	}

	restore := ignoreHangup()
	defer restore()

	cmd := &exec.Cmd{Path: path, Args: args, Stdin: in, Stdout: out, Stderr: errw, SysProcAttr: hangupAttr()}
	if err := cmd.Start(); err != nil {
		return n.cannotRun(name, err)
	}
	err = cmd.Wait()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exitErr):
		return &core.ExitError{Code: kill.ExitStatus(exitErr.ProcessState)}
	default:
		return n.fail(ExitCanceled, "%s", core.ErrText(err))
	}
}

// openOut opens nohup.out for appending, readable and writable only by the
// owner, in the current directory or, failing that, in $HOME. It returns the
// name the file was opened under.
func (n *Nohup) openOut() (io.WriteCloser, string, error) {
	const flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	f, err := os.OpenFile(OutFile, flags, 0o600)
	if err == nil {
		return f, OutFile, nil
	}
	inHome := ""
	if home := n.Getenv("HOME"); home != "" {
		inHome = filepath.Join(home, OutFile)
		f, err2 := os.OpenFile(inHome, flags, 0o600)
		if err2 == nil {
			return f, inHome, nil
		}
		n.diag.Errorf("failed to open '%s': %s", OutFile, core.ErrText(err))
		n.diag.Errorf("failed to open '%s': %s", inHome, core.ErrText(err2))
		return nil, "", &core.ExitError{Code: ExitCanceled}
	}
	return nil, "", n.fail(ExitCanceled, "failed to open '%s': %s", OutFile, core.ErrText(err))
}

// cannotRun reports why the command could not be started: 127 when it does
// not exist, 126 for anything else.
func (n *Nohup) cannotRun(name string, err error) error {
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return n.fail(ExitNotFound, "failed to run command '%s': No such file or directory", name)
	}
	return n.fail(ExitCannotInvoke, "failed to run command '%s': %s", name, core.ErrText(err))
}

// fail writes a diagnostic and returns the exit status code.
func (n *Nohup) fail(code int, format string, a ...any) error {
	n.diag.Errorf(format, a...)
	return &core.ExitError{Code: code}
}
