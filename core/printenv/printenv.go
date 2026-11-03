// Package printenv implements the engine behind printenvx, a GNU-compatible
// implementation of the GNU printenv command: print all or part of the
// environment.
package printenv

import (
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "printenvx"

// Options mirrors the GNU printenv flag surface.
type Options struct {
	// Null (-0) ends each output item with NUL instead of a newline.
	Null bool

	// Pretty, Color and Theme are gnux's terminal view, not GNU options:
	// with Pretty and Color, a listing of the whole environment has each
	// name in the theme's variable color and the '=' dimmed, and the list
	// separators of a *PATH variable dimmed (Theme is a built-in theme; ""
	// takes GNUX_THEME's). The text is GNU's; -0 is never colored.
	Pretty bool
	Color  bool
	Theme  string
}

// Printenv is the engine. Construct it with New.
type Printenv struct {
	Options

	// Environ lists the environment as NAME=value strings; it defaults to
	// os.Environ.
	Environ func() []string
	// Getenv reads GNUX_THEME for the view; nil means os.Getenv.
	Getenv func(string) string

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
	out := core.NewWriter(p.streams.Out)
	defer core.FlushOutput(out, core.NewDiag(Name, p.streams), &err)

	env := p.Environ()
	if len(args) == 0 {
		theme := p.theme()
		for _, kv := range env {
			_, _ = out.WriteString(theme.EnvEntry(kv, os.PathListSeparator) + end)
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

// theme is the view's colors for a listing, nil when uncolored.
func (p *Printenv) theme() *visual.Theme {
	if !p.Pretty || !p.Color || p.Null {
		return nil
	}
	getenv := p.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	return visual.Use(true, p.Theme, getenv, core.NewDiag(Name, p.streams).Warnf)
}
