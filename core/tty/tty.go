// Package tty implements the engine behind ttyx, a GNU-compatible
// implementation of the GNU/UNIX tty command: print the file name of the
// terminal connected to standard input.
package tty

import (
	"fmt"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "ttyx"

// Exit statuses, as GNU tty defines them.
const (
	// ExitNotTTY means standard input is not a terminal.
	ExitNotTTY = 1
	// ExitUsage means the command line was wrong.
	ExitUsage = 2
	// ExitWrite means the name could not be written.
	ExitWrite = 3
)

// Options mirrors the GNU tty flag surface.
type Options struct {
	// Silent (-s) prints nothing; only the exit status tells.
	Silent bool
}

// Tty is the engine. Construct it with New.
type Tty struct {
	Options

	// Terminal reports whether a stream is a terminal and, if so, its file
	// name. It defaults to the platform's lookup and is the injection point
	// for tests.
	Terminal func(stream any) (name string, ok bool)

	streams core.Streams
	diag    *core.Diag
}

// New builds a tty engine for the streams s.
func New(opts Options, s core.Streams) *Tty {
	return &Tty{Options: opts, Terminal: Lookup, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the name of the terminal on standard input, or "not a tty". It
// returns nil when standard input is a terminal, a *core.ExitError with
// ExitNotTTY when it is not, and with ExitWrite when the output failed.
func (t *Tty) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	name, ok := t.Terminal(t.streams.In)
	if t.Silent {
		if !ok {
			return &core.ExitError{Code: ExitNotTTY}
		}
		return nil
	}
	line := "not a tty"
	if ok {
		line = name
	}
	if _, err := fmt.Fprintln(t.streams.Out, line); err != nil {
		t.diag.Errorf("write error: %s", core.ErrText(err))
		return &core.ExitError{Code: ExitWrite}
	}
	if !ok {
		return &core.ExitError{Code: ExitNotTTY}
	}
	return nil
}
