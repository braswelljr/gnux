// Package hostname implements the engine behind hostnamex, a GNU-compatible
// implementation of the GNU coreutils hostname command: print the name of
// the current host system.
package hostname

import (
	"errors"
	"fmt"
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "hostnamex"

// errNotSupported refuses setting the name, which gnux does not do.
var errNotSupported = errors.New("operation not supported")

// Options mirrors the GNU hostname flag surface, which is empty.
type Options struct{}

// Hostname is the engine. Construct it with New.
type Hostname struct {
	Options

	// Lookup returns the host name; it defaults to os.Hostname.
	Lookup func() (string, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a hostname engine writing to s.
func New(opts Options, s core.Streams) *Hostname {
	return &Hostname{
		Options: opts,
		Lookup:  os.Hostname,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the host name. A NAME operand asks to set it, which GNU
// hostname does with sethostname(2); gnux never changes system settings, so
// it refuses with "Operation not supported".
func (h *Hostname) Run(args []string) error {
	switch len(args) {
	case 0:
	case 1:
		h.diag.Errorf("cannot set name to '%s': %s", args[0], core.ErrText(errNotSupported))
		return h.diag.Result()
	default:
		return core.Usagef("extra operand '%s'", args[1])
	}
	name, err := h.Lookup()
	if err != nil {
		h.diag.Errorf("cannot determine hostname: %s", core.ErrText(err))
		return h.diag.Result()
	}
	fmt.Fprintln(h.streams.Out, name)
	return nil
}
