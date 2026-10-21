// Package pinky implements the engine behind pinkyx, a GNU-compatible
// implementation of the GNU/UNIX pinky command: a lightweight finger, which
// shows who is logged in, or what the system knows about given users.
package pinky

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
	"github.com/braswelljr/gnux/core/internal/sysstat"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "pinkyx"

// Options mirrors the GNU pinky flag surface. The zero value is plain
// pinky: the short format, with every column and the heading.
type Options struct {
	// Long (-l) prints what is known about each USER instead of who is
	// logged in; at least one USER is then needed.
	Long bool
	// NoHomeAndShell (-b) omits the home directory and shell in -l.
	NoHomeAndShell bool
	// NoProject (-h) omits the user's ~/.project file in -l.
	NoProject bool
	// NoPlan (-p) omits the user's ~/.plan file in -l.
	NoPlan bool
	// NoHeading (-f) omits the column heading in the short format.
	NoHeading bool
	// NoFullName (-w, also -i and -q) omits the user's full name.
	NoFullName bool
	// NoWhere (-i, also -q) omits where the user logged in from.
	NoWhere bool
	// NoIdle (-q) omits the idle time.
	NoIdle bool
	// Lookup (--lookup) canonicalizes host names through DNS.
	Lookup bool
}

// Account is what pinky shows about a user.
type Account struct {
	Username string
	Gecos    string // the full name: the GECOS field up to its first comma
	Home     string
	Shell    string
}

// Pinky is the engine. Construct it with New.
type Pinky struct {
	Options

	// Now is the clock idle times are measured against, Location the zone
	// login times are shown in (nil: local), Current the source of the
	// login records and LookupUser the user database. They are the
	// injection points tests use.
	Now        func() time.Time
	Location   *time.Location
	Current    func() ([]utmp.Entry, error)
	LookupUser func(name string) (*Account, error)
	Stat       func(name string) (os.FileInfo, error)

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a pinky engine writing to s.
func New(opts Options, s core.Streams) *Pinky {
	return &Pinky{
		Options:    opts,
		Now:        time.Now,
		Current:    utmp.Current,
		LookupUser: lookupUser,
		Stat:       fsys.Stat,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// Run prints the logged-in users (only those named, if any) or, with Long,
// each named user's details.
func (p *Pinky) Run(args []string) (err error) {
	if p.Long && len(args) == 0 {
		return core.Usagef("no username specified; at least one must be specified when using -l")
	}
	p.out = core.NewWriter(p.streams.Out)
	defer core.FlushOutput(p.out, p.diag, &err)
	if p.Long {
		for _, name := range args {
			p.long(name)
		}
		return nil
	}
	entries, rerr := p.Current()
	if rerr != nil {
		where := utmp.DefaultFile
		if where == "" {
			where = "the login records"
		}
		p.diag.Errorf("%s: %s", quotearg.Quote(where, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"}), core.ErrText(rerr))
		return p.diag.Result()
	}
	if !p.NoHeading {
		p.heading()
	}
	for _, e := range entries {
		if e.Type != utmp.UserProcess {
			continue
		}
		if len(args) > 0 && !contains(args, e.User) {
			continue
		}
		p.entry(e)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// The C locale's "%b %e %H:%M".
const (
	timeLayout = "Jan _2 15:04"
	timeWidth  = 12
)

func (p *Pinky) heading() {
	fmt.Fprintf(p.out, "%-8s", "Login")
	if !p.NoFullName {
		fmt.Fprintf(p.out, " %-19s", "Name")
	}
	fmt.Fprintf(p.out, " %-9s", " TTY")
	if !p.NoIdle {
		fmt.Fprintf(p.out, " %-6s", "Idle")
	}
	fmt.Fprintf(p.out, " %-*s", timeWidth, "When")
	if !p.NoWhere {
		fmt.Fprintf(p.out, " %s", "Where")
	}
	_ = p.out.WriteByte('\n')
}

// fullName expands each '&' of a GECOS name into the user name, its first
// letter capitalized, as finger does.
func fullName(a *Account) string {
	if !strings.Contains(a.Gecos, "&") {
		return a.Gecos
	}
	u := a.Username
	if u != "" && u[0] >= 'a' && u[0] <= 'z' {
		u = string(u[0]-'a'+'A') + u[1:]
	}
	return strings.ReplaceAll(a.Gecos, "&", u)
}

// padded writes s left-aligned in eight columns, or whole when longer.
func (p *Pinky) padded(s string) {
	if len(s) < 8 {
		fmt.Fprintf(p.out, "%-8s", s)
	} else {
		_, _ = p.out.WriteString(s)
	}
}

func (p *Pinky) entry(e utmp.Entry) {
	line := e.Line
	if i := strings.IndexByte(line, ' '); i >= 0 {
		line = line[i+1:]
	}
	dev := line
	if !filepath.IsAbs(dev) {
		dev = "/dev/" + dev
	}
	mesg := byte('?')
	var lastChange time.Time
	if info, err := p.Stat(dev); err == nil {
		mesg = '*'
		if info.Mode().Perm()&0o020 != 0 {
			mesg = ' '
		}
		lastChange = sysstat.Of(info).Atime
		if lastChange.IsZero() {
			lastChange = info.ModTime()
		}
	}

	p.padded(e.User)
	if !p.NoFullName {
		if a, err := p.LookupUser(e.User); err != nil || a == nil {
			fmt.Fprintf(p.out, " %19s", "        ???")
		} else {
			name := fullName(a)
			if len(name) > 19 {
				name = name[:19]
			}
			fmt.Fprintf(p.out, " %-19s", name)
		}
	}
	_ = p.out.WriteByte(' ')
	_ = p.out.WriteByte(mesg)
	p.padded(e.Line)
	if !p.NoIdle {
		idle := "?????"
		if !lastChange.IsZero() {
			idle = p.idle(lastChange)
		}
		fmt.Fprintf(p.out, " %-6s", idle)
	}
	loc := p.Location
	if loc == nil {
		loc = time.Local
	}
	fmt.Fprintf(p.out, " %s", e.Time.In(loc).Format(timeLayout))
	if !p.NoWhere && e.Host != "" {
		host, display, hasDisplay := strings.Cut(e.Host, ":")
		if host != "" && p.Lookup {
			if canon, err := net.LookupCNAME(host); err == nil && canon != "" {
				host = strings.TrimSuffix(canon, ".")
			}
		}
		_ = p.out.WriteByte(' ')
		_, _ = p.out.WriteString(host)
		if hasDisplay {
			_ = p.out.WriteByte(':')
			_, _ = p.out.WriteString(display)
		}
	}
	_ = p.out.WriteByte('\n')
}

// idle is how long ago the terminal was last used: blank under a minute,
// HH:MM under a day, and a number of days after that.
func (p *Pinky) idle(when time.Time) string {
	secs := int64(p.Now().Sub(when) / time.Second)
	switch {
	case secs < 60:
		return "     "
	case secs < 24*60*60:
		return fmt.Sprintf("%02d:%02d", secs/3600, secs%3600/60)
	}
	return fmt.Sprintf("%dd", secs/(24*60*60))
}

// long prints what is known about one user.
func (p *Pinky) long(name string) {
	fmt.Fprintf(p.out, "Login name: %-28sIn real life: ", name)
	a, err := p.LookupUser(name)
	if err != nil || a == nil {
		_, _ = p.out.WriteString(" ???\n")
		return
	}
	fmt.Fprintf(p.out, " %s\n", fullName(a))
	if !p.NoHomeAndShell {
		fmt.Fprintf(p.out, "Directory: %-29sShell:  %s\n", a.Home, a.Shell)
	}
	if !p.NoProject {
		p.file(a.Home+"/.project", "Project: ")
	}
	if !p.NoPlan {
		p.file(a.Home+"/.plan", "Plan:\n")
	}
	_ = p.out.WriteByte('\n')
}

// file copies a user's ~/.project or ~/.plan, under its label, when it can
// be read.
func (p *Pinky) file(path, label string) {
	f, err := fsys.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = p.out.WriteString(label)
	_, _ = io.Copy(p.out, f)
}

// lookupUser reads the user database: the name and home directory from the
// system, the shell from /etc/passwd where it is kept there.
func lookupUser(name string) (*Account, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return nil, err
	}
	return &Account{Username: u.Username, Gecos: u.Name, Home: u.HomeDir, Shell: shellOf(u.Username)}, nil
}
