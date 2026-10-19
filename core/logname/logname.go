// Package logname implements the engine behind lognamex, a GNU-compatible
// implementation of the GNU logname command: print the name the user logged
// in as.
package logname

import (
	"fmt"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/tty"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "lognamex"

// Options mirrors the GNU logname flag surface, which has no options of its
// own. It is here so that logname has the same shape as every other engine.
type Options struct{}

// Logname is the engine. Construct it with New.
type Logname struct {
	Options

	// Login returns the login name of the session that stdin belongs to,
	// and false when there is none. It defaults to the platform's lookup.
	Login func(stdin any) (string, bool)

	streams core.Streams
	diag    *core.Diag
}

// New builds a logname engine reading the terminal of s.In and writing to s.
func New(opts Options, s core.Streams) *Logname {
	return &Logname{Options: opts, Login: login, streams: s, diag: core.NewDiag(Name, s)}
}

// Run prints the login name. logname takes no operands. It returns
// core.ErrFailed, after 'no login name', when there is none.
func (l *Logname) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	name, ok := l.Login(l.streams.In)
	if !ok || name == "" {
		l.diag.Errorf("no login name")
		return l.diag.Result()
	}
	if _, err := fmt.Fprintln(l.streams.Out, name); err != nil && !core.IsClosedPipe(err) {
		l.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return l.diag.Result()
}

// fromTerminal finds the login record of the terminal on stdin, the way
// getlogin(3) does when the kernel does not keep a login name.
func fromTerminal(stdin any) (string, bool) {
	name, ok := tty.Lookup(stdin)
	if !ok || name == "" {
		return "", false
	}
	line := strings.TrimPrefix(name, "/dev/")
	entries, err := utmp.Current()
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if e.Type == utmp.UserProcess && e.Line == line {
			if user := strings.TrimSpace(e.User); user != "" {
				return user, true
			}
		}
	}
	return "", false
}
