// Package pwd implements the engine behind pwdx, a GNU-compatible implementation of the
// GNU/UNIX pwd command: print the name of the current working directory.
package pwd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "pwdx"

// Options mirrors the GNU pwd flag surface.
type Options struct {
	// Logical (-L) prints $PWD when it names the current directory, keeping any
	// symlinks the shell walked through. When false (-P, the default, as in GNU
	// pwd) every symlink is resolved.
	Logical bool
}

// Pwd is the engine. Construct it with New.
type Pwd struct {
	Options

	// Getenv and Getwd are the injection points tests use to control the
	// environment; they default to os.Getenv and fsys.Getwd.
	Getenv func(string) string
	Getwd  func() (string, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a pwd engine writing to s.
func New(opts Options, s core.Streams) *Pwd {
	return &Pwd{
		Options: opts,
		Getenv:  os.Getenv,
		Getwd:   fsys.Getwd,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the working directory. pwd takes no operands; any given are
// ignored, as GNU pwd does.
func (p *Pwd) Run(_ []string) error {
	dir, err := p.Getwd()
	if err != nil {
		p.diag.Errorf("cannot get current directory: %s", core.ErrText(err))
		return p.diag.Result()
	}

	if p.Logical {
		if env := p.Getenv("PWD"); logicalOK(env, dir) {
			dir = env
		}
	} else if resolved, err := fsys.EvalSymlinks(dir); err == nil {
		dir = resolved
	}

	fmt.Fprintln(p.streams.Out, dir)
	return nil
}

// logicalOK reports whether env is an acceptable logical name for dir: an
// absolute path with no "." or ".." components that refers to the same
// directory. This is the POSIX rule for trusting $PWD.
func logicalOK(env, dir string) bool {
	if env == "" || !filepath.IsAbs(env) || filepath.Clean(env) != env {
		return false
	}
	a, err := fsys.Stat(env)
	if err != nil {
		return false
	}
	b, err := fsys.Stat(dir)
	if err != nil {
		return false
	}
	return os.SameFile(a, b)
}
