// Package whoami implements the engine behind whoamix, a GNU-compatible
// implementation of the GNU whoami command: print the effective user name.
package whoami

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "whoamix"

// Options mirrors the GNU whoami flag surface, which is empty.
type Options struct{}

// Whoami is the engine. Construct it with New.
type Whoami struct {
	Options

	// Username returns the effective user's name, and UID the effective user
	// ID reported when there is none; they default to os/user and os.Geteuid.
	Username func() (string, error)
	UID      func() int

	streams core.Streams
	diag    *core.Diag
}

// New builds a whoami engine writing to s.
func New(opts Options, s core.Streams) *Whoami {
	return &Whoami{
		Options:  opts,
		Username: currentUsername,
		UID:      os.Geteuid,
		streams:  s,
		diag:     core.NewDiag(Name, s),
	}
}

// Run prints the user name. whoami takes no operands.
func (w *Whoami) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	name, err := w.Username()
	if err != nil || name == "" {
		w.diag.Errorf("cannot find name for user ID %d", w.UID())
		return w.diag.Result()
	}
	// Windows reports DOMAIN\user; whoami prints the user. A Unix user name
	// cannot contain a backslash, so this changes nothing there.
	if i := strings.LastIndexByte(name, '\\'); i >= 0 {
		name = name[i+1:]
	}
	fmt.Fprintln(w.streams.Out, name)
	return nil
}

func currentUsername() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}
