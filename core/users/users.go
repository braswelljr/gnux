// Package users implements the engine behind usersx, a GNU-compatible
// implementation of the GNU users command: print the user names of the users
// logged in. The records are the ones whox reads.
package users

import (
	"bufio"
	"os"
	"slices"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "usersx"

// Options holds gnux's terminal view only: GNU users has no options of
// its own, and the zero value is its output.
type Options struct {
	// Pretty, Color and Theme are gnux's terminal view: with Pretty and
	// Color, the caller's own name is in the theme's "yours" color and the
	// others in its "others" color (Theme is a built-in theme; "" takes
	// GNUX_THEME's). The text is GNU's.
	Pretty bool
	Color  bool
	Theme  string
}

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
	// Self is the caller's login name, which the view emphasizes; nil asks
	// the system. Getenv reads GNUX_THEME for the view; nil is os.Getenv.
	Self   func() string
	Getenv func(string) string

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
	write(out, u.paint(names))
	return nil
}

func write(out *bufio.Writer, names []string) {
	_, _ = out.WriteString(strings.Join(names, " "))
	_ = out.WriteByte('\n')
}

// paint is gnux's terminal view (BR-13): each name in the color of whose
// it is, the caller's or someone else's.
func (u *Users) paint(names []string) []string {
	if !u.Pretty || !u.Color {
		return names
	}
	getenv := u.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	self := ""
	if u.Self != nil {
		self = u.Self()
	} else {
		self = utmp.Self(getenv)
	}
	theme := visual.Use(true, u.Theme, getenv, u.diag.Warnf)
	painted := make([]string, len(names))
	for i, n := range names {
		key := visual.Others
		if n == self {
			key = visual.You
		}
		painted[i] = theme.Paint(key, n)
	}
	return painted
}
