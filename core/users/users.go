// Package users implements the engine behind usersx, a GNU-compatible
// implementation of the GNU users command: print the user names of the users
// logged in. The records are the ones whox reads.
package users

import (
	"bufio"
	"slices"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "usersx"

// Options mirrors the GNU users flag surface, which has no options of its
// own. It is here so that users has the same shape as every other engine.
type Options struct{}

// Record is one entry of the login records: a user name and whether the
// entry is a user's login session (rather than a boot, a run level or a
// process that has ended).
type Record struct {
	User  string
	Login bool
}

// Users is the engine. Construct it with New.
type Users struct {
	Options

	// Current returns the records of the users logged in now, without the
	// sessions whose process has ended; ReadFile reads a utmp-format file.
	// They default to the platform's own records.
	Current  func() ([]Record, error)
	ReadFile func(path string) ([]Record, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a users engine writing to s.
func New(opts Options, s core.Streams) *Users {
	return &Users{
		Options:  opts,
		Current:  func() ([]Record, error) { return records(utmp.Current()) },
		ReadFile: func(path string) ([]Record, error) { return records(utmp.ReadFile(path)) },
		streams:  s,
		diag:     core.NewDiag(Name, s),
	}
}

func records(entries []utmp.Entry, err error) ([]Record, error) {
	out := make([]Record, 0, len(entries))
	for _, e := range entries {
		out = append(out, Record{User: e.User, Login: e.Type == utmp.UserProcess})
	}
	return out, err
}

// Run prints the names of the users logged in, sorted and separated by
// spaces, one name per session, from the system's records or from FILE (a
// utmp file, args[0]). With nobody logged in it prints nothing. It returns
// core.ErrFailed when the records cannot be read.
func (u *Users) Run(args []string) (err error) {
	var recs []Record
	switch len(args) {
	case 0:
		recs, err = u.Current()
		if err != nil {
			u.diag.Errorf("cannot read the login records: %s", core.ErrText(err))
			return core.ErrFailed
		}
	case 1:
		recs, err = u.ReadFile(args[0])
		if err != nil {
			u.diag.Errorf("%s: %s", args[0], core.ErrText(err))
			return core.ErrFailed
		}
	default:
		return core.Usagef("extra operand '%s'", args[1])
	}

	var names []string
	for _, r := range recs {
		if name := strings.TrimSpace(r.User); r.Login && name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	slices.Sort(names)

	out := core.NewWriter(u.streams.Out)
	defer core.FlushOutput(out, u.diag, &err)
	write(out, names)
	return nil
}

func write(out *bufio.Writer, names []string) {
	_, _ = out.WriteString(strings.Join(names, " "))
	_ = out.WriteByte('\n')
}
