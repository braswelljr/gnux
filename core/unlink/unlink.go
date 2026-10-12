// Package unlink implements the engine behind unlinkx, a GNU-compatible
// implementation of the GNU unlink command: remove one file with the
// unlink(2) system call, and nothing more.
package unlink

import (
	"errors"
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "unlinkx"

// errIsDir is unlink(2)'s refusal of a directory, in GNU's words, whatever
// the platform's own text would be.
var errIsDir = errors.New("is a directory")

// Options mirrors the GNU unlink flag surface, which is empty: unlink takes
// exactly one operand and no options.
type Options struct{}

// Unlink is the engine. Construct it with New.
type Unlink struct {
	Options

	// RemoveFn removes the file once it is known not to be a directory; it
	// defaults to os.Remove.
	RemoveFn func(name string) error

	streams core.Streams
	diag    *core.Diag
}

// New builds an unlink engine writing to s.
func New(opts Options, s core.Streams) *Unlink {
	return &Unlink{
		Options:  opts,
		RemoveFn: os.Remove,
		streams:  s,
		diag:     core.NewDiag(Name, s),
	}
}

// Run removes the single file named by args[0]. A directory is refused, even
// an empty one: that is rmdir's job.
func (u *Unlink) Run(args []string) error {
	switch len(args) {
	case 0:
		return core.Usagef("missing operand")
	case 1:
	default:
		return core.Usagef("extra operand '%s'", args[1])
	}
	name := args[0]
	info, err := os.Lstat(name)
	switch {
	case err != nil:
	case info.IsDir():
		err = errIsDir
	default:
		err = u.RemoveFn(name)
	}
	if err != nil {
		u.diag.Errorf("cannot unlink '%s': %s", name, core.ErrText(err))
	}
	return u.diag.Result()
}
