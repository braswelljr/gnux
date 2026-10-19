// Package arch implements the engine behind archx, a GNU-compatible
// implementation of the GNU arch command: print the machine hardware name,
// as uname -m does.
package arch

import (
	"fmt"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uname"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "archx"

// Options mirrors the GNU arch flag surface, which has no options of its
// own. It is here so that arch has the same shape as every other engine.
type Options struct{}

// Arch is the engine. Construct it with New.
type Arch struct {
	Options

	// System reads the system information; it defaults to uname.System.
	System func() (uname.Info, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds an arch engine writing to s.
func New(opts Options, s core.Streams) *Arch {
	return &Arch{Options: opts, System: uname.System, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the machine hardware name. arch takes no operands.
func (a *Arch) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	info, err := a.System()
	if err != nil {
		a.diag.Errorf("cannot get system name: %s", core.ErrText(err))
		return a.diag.Result()
	}
	machine := info.Machine
	if machine == "" {
		machine = "unknown"
	}
	if _, err := fmt.Fprintln(a.streams.Out, machine); err != nil && !core.IsClosedPipe(err) {
		a.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return a.diag.Result()
}
