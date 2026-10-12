// Package printenv implements the engine behind printenvx, a GNU-compatible
// implementation of the GNU printenv command: print all or part of the
// environment.
package printenv

import (
	"bufio"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "printenvx"

// Options mirrors the GNU printenv flag surface.
type Options struct {
	// Null (-0) ends each output item with NUL instead of a newline.
	Null bool
}

// Printenv is the engine. Construct it with New.
type Printenv struct {
	Options

	// Environ lists the environment as NAME=value strings; it defaults to
	// os.Environ.
	Environ func() []string

	streams core.Streams
}

// New builds a printenv engine writing to s.
func New(opts Options, s core.Streams) *Printenv {
	return &Printenv{Options: opts, Environ: os.Environ, streams: s}
}

// Run prints the whole environment with no operands, or the value of each
// named variable. A variable that is not set prints nothing and makes the
// result core.ErrFailed (status 1), with no diagnostic, as in GNU printenv.
func (p *Printenv) Run(args []string) (err error) {
	end := "\n"
	if p.Null {
		end = "\x00"
	}
	out := bufio.NewWriter(p.streams.Out)
	defer core.FlushOutput(out, core.NewDiag(Name, p.streams), &err)

	env := p.Environ()
	if len(args) == 0 {
		for _, kv := range env {
			_, _ = out.WriteString(kv + end)
		}
		return nil
	}

	missing := false
	for _, name := range args {
		found := false
		// A name with '=' in it can never be a variable's name.
		if !strings.Contains(name, "=") {
			for _, kv := range env {
				if value, ok := strings.CutPrefix(kv, name+"="); ok {
					_, _ = out.WriteString(value + end)
					found = true
				}
			}
		}
		if !found {
			missing = true
		}
	}
	if missing {
		return core.ErrFailed
	}
	return nil
}
