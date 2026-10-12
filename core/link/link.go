// Package link implements the engine behind linkx, a GNU-compatible
// implementation of the GNU link command: make a hard link with the link(2)
// system call, and nothing more.
package link

import (
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "linkx"

// Options mirrors the GNU link flag surface, which is empty: link takes
// exactly two operands and no options.
type Options struct{}

// Link is the engine. Construct it with New.
type Link struct {
	Options

	// LinkFn creates the hard link; it defaults to os.Link.
	LinkFn func(oldname, newname string) error

	streams core.Streams
	diag    *core.Diag
}

// New builds a link engine writing to s.
func New(opts Options, s core.Streams) *Link {
	return &Link{
		Options: opts,
		LinkFn:  os.Link,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run creates the hard link args[1] to the existing file args[0].
func (l *Link) Run(args []string) error {
	switch len(args) {
	case 0:
		return core.Usagef("missing operand")
	case 1:
		return core.Usagef("missing operand after '%s'", args[0])
	case 2:
	default:
		return core.Usagef("extra operand '%s'", args[2])
	}
	if err := l.LinkFn(args[0], args[1]); err != nil {
		l.diag.Errorf("cannot create link '%s' to '%s': %s", args[1], args[0], core.ErrText(err))
	}
	return l.diag.Result()
}
