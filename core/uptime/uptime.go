// Package uptime implements the engine behind uptimex, a GNU-compatible
// implementation of the GNU/UNIX uptime command: tell how long the system
// has been running.
package uptime

import (
	"fmt"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "uptimex"

// Options is empty: GNU uptime has no options besides --help and --version.
type Options struct{}

// Uptime is the engine. Construct it with New.
type Uptime struct {
	Options

	// Now returns the current time; it is the injection point for tests.
	Now func() time.Time
	// Location is the zone the clock is shown in; nil means the local zone.
	Location *time.Location

	streams core.Streams
	diag    *core.Diag

	current  func() ([]utmp.Entry, error)
	readFile func(string) ([]utmp.Entry, error)
	loadAvg  func() ([]float64, error)
}

// New builds an uptime engine writing to s.
func New(opts Options, s core.Streams) *Uptime {
	return &Uptime{
		Options:  opts,
		Now:      time.Now,
		streams:  s,
		diag:     core.NewDiag(Name, s),
		current:  utmp.Current,
		readFile: utmp.ReadFile,
		loadAvg:  utmp.LoadAvg,
	}
}

// Run prints the current time, how long the system has been up, the number
// of users logged in and the load averages, from the system's records or
// from FILE (args[0]), a utmp file. It returns core.ErrFailed when the
// records or the boot time could not be read.
func (u *Uptime) Run(args []string) (err error) {
	if len(args) > 1 {
		return core.Usagef("extra operand '%s'", args[1])
	}
	var entries []utmp.Entry
	if len(args) == 1 {
		entries, err = u.readFile(args[0])
		if err != nil {
			u.diag.Errorf("%s: %s", args[0], core.ErrText(err))
			entries = nil
		}
	} else {
		entries, err = u.current()
		if err != nil {
			u.diag.Errorf("cannot read the login records: %s", core.ErrText(err))
		}
	}

	users := 0
	var boot time.Time
	for _, e := range entries {
		if e.Type == utmp.UserProcess && e.User != "" {
			users++
		}
		if e.Type == utmp.BootTime {
			boot = e.Time
		}
	}

	out := core.NewWriter(u.streams.Out)
	defer core.FlushOutput(out, u.diag, &err)
	loc := u.Location
	if loc == nil {
		loc = time.Local
	}
	now := u.Now()
	if boot.IsZero() {
		_ = out.Flush()
		u.diag.Errorf("couldn't get boot time")
	}
	fmt.Fprintf(out, " %s  ", now.In(loc).Format("15:04:05"))

	up := int64(now.Sub(boot) / time.Second)
	switch {
	case boot.IsZero() || up < 0:
		_, _ = out.WriteString("up ???? days ??:??,  ")
		if !boot.IsZero() {
			u.diag.Errorf("couldn't get boot time")
		}
	default:
		days, hours, mins := up/86400, up%86400/3600, up%3600/60
		switch {
		case days == 1:
			fmt.Fprintf(out, "up %d day %2d:%02d,  ", days, hours, mins)
		case days > 1:
			fmt.Fprintf(out, "up %d days %2d:%02d,  ", days, hours, mins)
		default:
			fmt.Fprintf(out, "up %2d:%02d,  ", hours, mins)
		}
	}
	if users == 1 {
		_, _ = out.WriteString("1 user")
	} else {
		fmt.Fprintf(out, "%d users", users)
	}
	if avg, err := u.loadAvg(); err == nil && len(avg) > 0 {
		fmt.Fprintf(out, ",  load average: %.2f", avg[0])
		for _, a := range avg[1:] {
			fmt.Fprintf(out, ", %.2f", a)
		}
	}
	_ = out.WriteByte('\n')
	return u.diag.Result()
}
