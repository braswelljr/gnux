// Package nice implements the engine behind nicex, a GNU-compatible
// implementation of the GNU/UNIX nice command: run a command with an adjusted
// niceness (scheduling priority), or print the current niceness.
package nice

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "nicex"

// Exit statuses nice reserves, so a caller can tell nicex's own failures from
// the command's status.
const (
	// ExitCanceled means nicex itself failed.
	ExitCanceled = 125
	// ExitCannotInvoke means the command was found but could not be run.
	ExitCannotInvoke = 126
	// ExitNotFound means the command could not be found.
	ExitNotFound = 127
)

// DefaultAdjustment is added to the niceness when a command is given without
// an adjustment.
const DefaultAdjustment = 10

// The range of niceness values, from most to least favorable scheduling.
const (
	MinNiceness = -20
	MaxNiceness = 19
)

// ErrUnsupported is returned by the default GetNiceness and SetNiceness on a
// platform without Unix niceness (Windows, WebAssembly).
var ErrUnsupported = errors.New("operation not supported")

// Options mirrors the GNU nice flag surface.
type Options struct {
	// Adjustment (-n ADJ) is added to the current niceness. It is used only
	// when Adjust is set; otherwise a command runs at DefaultAdjustment.
	Adjustment int
	// Adjust records that an adjustment was given, so that -n 0 is told apart
	// from no -n at all.
	Adjust bool
}

// Nice is the engine. Construct it with New.
type Nice struct {
	Options

	// GetNiceness returns the niceness of the calling process, and
	// SetNiceness sets that of process pid. They default to the system calls
	// and are the injection points tests use.
	GetNiceness func() (int, error)
	SetNiceness func(pid, niceness int) error

	streams core.Streams
	diag    *core.Diag
}

// New builds a nice engine. The command it runs inherits the streams s.
func New(opts Options, s core.Streams) *Nice {
	return &Nice{
		Options:     opts,
		GetNiceness: getNiceness,
		SetNiceness: setNiceness,
		streams:     s,
		diag:        core.NewDiag(Name, s),
	}
}

// Run prints the current niceness when args is empty, or runs args as a
// command with the adjusted niceness. A niceness that cannot be set for lack
// of privilege (a negative adjustment, typically) is a warning and the
// command still runs, as in GNU nice. The result is nil or a
// *core.ExitError: the command's status, 125 when nicex failed, 126 or 127
// when the command could not be run.
func (n *Nice) Run(args []string) error {
	if len(args) == 0 {
		if n.Adjust {
			return core.Usagef("a command must be given with an adjustment")
		}
		cur, err := n.GetNiceness()
		if err != nil {
			return n.fail(ExitCanceled, "cannot get niceness: %s", core.ErrText(err))
		}
		if _, err := fmt.Fprintln(n.streams.Out, cur); err != nil {
			return n.fail(ExitCanceled, "write error: %s", core.ErrText(err))
		}
		return nil
	}

	adj := DefaultAdjustment
	if n.Adjust {
		adj = n.Adjustment
	}

	name := args[0]
	path, err := exec.LookPath(name)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		return n.cannotRun(name, err)
	}

	cur, err := n.GetNiceness()
	if err != nil && !errors.Is(err, ErrUnsupported) {
		return n.fail(ExitCanceled, "cannot get niceness: %s", core.ErrText(err))
	}
	target := min(max(cur+clamp(adj), MinNiceness), MaxNiceness)

	cmd := &exec.Cmd{
		Path:   path,
		Args:   args,
		Stdin:  n.streams.In,
		Stdout: n.streams.Out,
		Stderr: n.streams.Err,
	}
	if err := cmd.Start(); err != nil {
		return n.cannotRun(name, err)
	}
	// GNU nice changes its own niceness and then becomes the command. nicex
	// runs the command as a child, so the child's niceness is set as soon as
	// it has started.
	if err := n.SetNiceness(cmd.Process.Pid, target); err != nil {
		if !errors.Is(err, fs.ErrPermission) && !errors.Is(err, ErrUnsupported) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return n.fail(ExitCanceled, "cannot set niceness: %s", core.ErrText(err))
		}
		n.diag.Warnf("cannot set niceness: %s", core.ErrText(err))
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

// clamp limits an adjustment to the widest range that can make a difference,
// as GNU nice does, so that adding it to the niceness cannot overflow.
func clamp(adj int) int {
	return min(max(adj, 2*MinNiceness), 2*MaxNiceness+1)
}

// cannotRun reports why the command could not be started: 127 when it does
// not exist, 126 for anything else.
func (n *Nice) cannotRun(name string, err error) error {
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return n.fail(ExitNotFound, "'%s': No such file or directory", name)
	}
	return n.fail(ExitCannotInvoke, "'%s': %s", name, core.ErrText(err))
}

// fail writes a diagnostic and returns the exit status code.
func (n *Nice) fail(code int, format string, a ...any) error {
	n.diag.Errorf(format, a...)
	return &core.ExitError{Code: code}
}
