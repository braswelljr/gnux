// Package run starts the command a tool was asked to run for the user: what
// xargs builds from its input, and what find runs for -exec. A command named
// after a gnux tool runs in-process, so a script behaves the same where the
// GNU programs are not installed; anything else is a program found on PATH.
package run

import (
	"errors"
	"io/fs"
	"os/exec"

	"github.com/braswelljr/gnux/core"
)

// ToolFunc runs the gnux tool called name, if there is one. It has the shape
// of cmd.RunTool and of the gnux shell's ExecFunc.
type ToolFunc = func(name string, args []string, s core.Streams) (status int, found bool)

// Result is how a command that was started ended.
type Result struct {
	// Status is the command's exit status, or -1 when a signal ended it.
	Status int
	// Signaled reports that a signal, not an exit, ended the command.
	Signaled bool
	// Signal is the number of that signal, or 0 where the platform does not
	// say which it was.
	Signal int
}

// Command runs argv with the given streams, in directory dir when it is not
// empty. tool, when not nil, is asked first and runs argv[0] if it names a
// gnux tool; dir does not apply to a tool, which runs in this process.
//
// The error is non-nil only when the command could not be started; it matches
// fs.ErrNotExist when no such program was found. A command that ran and
// failed is a Result with a nonzero Status and a nil error.
func Command(tool ToolFunc, argv []string, dir string, s core.Streams) (Result, error) {
	if len(argv) == 0 {
		return Result{}, fs.ErrNotExist
	}
	if tool != nil {
		if status, found := tool(argv[0], argv[1:], s); found {
			return Result{Status: status}, nil
		}
	}

	path, err := exec.LookPath(argv[0])
	if err != nil {
		// A name with no match on PATH is "not found" whatever the platform
		// calls it; a match that cannot be executed keeps its own error.
		if errors.Is(err, exec.ErrNotFound) {
			return Result{}, fs.ErrNotExist
		}
		return Result{}, err
	}
	cmd := &exec.Cmd{Path: path, Args: argv, Dir: dir, Stdin: s.In, Stdout: s.Out, Stderr: s.Err}
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}

	err = cmd.Wait()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return Result{}, nil
	case errors.As(err, &exitErr):
		// ExitCode is -1 for a process that did not exit: a signal ended it.
		code := exitErr.ExitCode()
		if code < 0 {
			return Result{Status: code, Signaled: true, Signal: signalOf(exitErr.ProcessState)}, nil
		}
		return Result{Status: code}, nil
	default:
		// The command ran, but relaying its streams failed.
		return Result{Status: 1}, nil
	}
}
