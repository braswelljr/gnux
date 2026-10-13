// Package who implements the engine behind whox, a GNU-compatible
// implementation of the GNU/UNIX who command: show who is logged on.
package who

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/sysstat"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/tty"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "whox"

// Options mirrors the GNU who flag surface. The zero value is plain who:
// the users logged in, name, line, time and host.
type Options struct {
	// All (-a) is -b -d -l -p -r -t -u -T --all at once.
	All bool
	// Boot (-b) shows the time of the last system boot.
	Boot bool
	// Dead (-d) shows dead processes.
	Dead bool
	// Heading (-H) prints a line of column headings.
	Heading bool
	// Login (-l) shows system login processes.
	Login bool
	// Lookup (--lookup) canonicalizes host names via DNS.
	Lookup bool
	// MyLine (-m, or two operands as in 'who am i') shows only the terminal
	// on standard input.
	MyLine bool
	// Process (-p) shows active processes spawned by init.
	Process bool
	// Count (-q) lists the login names and their number.
	Count bool
	// RunLevel (-r) shows the current run level.
	RunLevel bool
	// Short (-s) prints only name, line and time (the default with no
	// selection).
	Short bool
	// Time (-t) shows the last system clock change.
	Time bool
	// Mesg (-T, -w) adds the user's message status: + writable, - not, ?
	// unknown.
	Mesg bool
	// Users (-u) lists the users logged in, with idle time and PID.
	Users bool
}

// Who is the engine. Construct it with New.
type Who struct {
	Options

	// Now returns the current time, for idle times.
	Now func() time.Time
	// Location is the zone times are shown in; nil means the local zone.
	Location *time.Location

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	current  func() ([]utmp.Entry, error)
	readFile func(string) ([]utmp.Entry, error)
	ttyName  func(any) (string, bool)
	stat     func(string) (os.FileInfo, error)

	needUsers, needBoot, needDead, needLogin, needInit, needRunLevel, needClock bool
	includeIdle, includeExit, includeMesg                                       bool
}

// New builds a who engine reading standard input's terminal from s.
func New(opts Options, s core.Streams) *Who {
	return &Who{
		Options:  opts,
		Now:      time.Now,
		streams:  s,
		diag:     core.NewDiag(Name, s),
		current:  utmp.Current,
		readFile: utmp.ReadFile,
		ttyName:  tty.Lookup,
		stat:     os.Stat,
	}
}

// timeFormat is the C locale's "%b %e %H:%M".
const (
	timeLayout = "Jan _2 15:04"
	timeWidth  = 12
)

// Run shows the selected records of the system (or of FILE, args[0], a utmp
// file). Two operands, as in 'who am i', imply -m. It returns
// core.ErrFailed when the records could not be read.
func (w *Who) Run(args []string) (err error) {
	var entries []utmp.Entry
	switch len(args) {
	case 0, 2:
		if len(args) == 2 {
			w.MyLine = true
		}
		entries, err = w.current()
		if err != nil {
			w.diag.Errorf("cannot read the login records: %s", core.ErrText(err))
			return core.ErrFailed
		}
	case 1:
		entries, err = w.readFile(args[0])
		if err != nil {
			w.diag.Errorf("%s: %s", args[0], core.ErrText(err))
			return core.ErrFailed
		}
	default:
		return core.Usagef("extra operand '%s'", args[2])
	}
	w.resolve()
	w.out = core.NewWriter(w.streams.Out)
	defer core.FlushOutput(w.out, w.diag, &err)
	if w.Count {
		w.count(entries)
		return nil
	}
	w.scan(entries)
	return nil
}

// resolve turns the options into what to show, as GNU who does.
func (w *Who) resolve() {
	o := w.Options
	assumptions := true
	if o.All {
		w.needBoot, w.needDead, w.needLogin, w.needInit, w.needRunLevel, w.needClock, w.needUsers = true, true, true, true, true, true, true
		w.includeMesg, w.includeIdle, w.includeExit = true, true, true
		assumptions = false
	}
	set := func(on bool, need *bool, idle, exit bool) {
		if on {
			*need = true
			w.includeIdle = w.includeIdle || idle
			w.includeExit = w.includeExit || exit
			assumptions = false
		}
	}
	set(o.Boot, &w.needBoot, false, false)
	set(o.Dead, &w.needDead, true, true)
	set(o.Login, &w.needLogin, true, false)
	set(o.Process, &w.needInit, false, false)
	set(o.RunLevel, &w.needRunLevel, true, false)
	set(o.Time, &w.needClock, false, false)
	set(o.Users, &w.needUsers, true, false)
	w.includeMesg = w.includeMesg || o.Mesg
	if assumptions {
		w.needUsers = true
		w.Short = true
	}
	if w.includeExit {
		w.Short = false
	}
}

func (w *Who) count(entries []utmp.Entry) {
	n := 0
	for _, e := range entries {
		if e.Type == utmp.UserProcess && e.User != "" {
			if n > 0 {
				_ = w.out.WriteByte(' ')
			}
			_, _ = w.out.WriteString(strings.TrimSpace(e.User))
			n++
		}
	}
	fmt.Fprintf(w.out, "\n# users=%d\n", n)
}

func (w *Who) scan(entries []utmp.Entry) {
	if w.Heading {
		w.line("NAME", ' ', "LINE", "TIME", "IDLE", "PID", "COMMENT", "EXIT")
	}
	myLine := ""
	if w.MyLine {
		name, ok := w.ttyName(w.streams.In)
		if !ok || name == "" {
			return
		}
		myLine = strings.TrimPrefix(name, "/dev/")
	}
	var boot time.Time
	for _, e := range entries {
		if myLine == "" || strings.HasSuffix(e.Line, myLine) {
			switch {
			case w.needUsers && e.Type == utmp.UserProcess && e.User != "":
				w.user(e, boot)
			case w.needRunLevel && e.Type == utmp.RunLevel:
				last, curr := e.PID/256, e.PID%256
				comment := ""
				if last >= ' ' && last < 0x7f {
					if last == 'N' {
						last = 'S'
					}
					comment = "last=" + string(rune(last))
				}
				w.line("", ' ', "run-level "+string(rune(curr)), w.when(e.Time), "", "", comment, "")
			case w.needBoot && e.Type == utmp.BootTime:
				w.line("", ' ', "system boot", w.when(e.Time), "", "", "", "")
			case w.needClock && e.Type == utmp.NewTime:
				w.line("", ' ', "clock change", w.when(e.Time), "", "", "", "")
			case w.needInit && e.Type == utmp.InitProcess:
				w.line("", ' ', e.Line, w.when(e.Time), "", pid(e), "id="+e.ID, "")
			case w.needLogin && e.Type == utmp.LoginProcess:
				w.line("LOGIN", ' ', e.Line, w.when(e.Time), "", pid(e), "id="+e.ID, "")
			case w.needDead && e.Type == utmp.DeadProcess:
				w.line("", ' ', e.Line, w.when(e.Time), "", pid(e), "id="+e.ID,
					fmt.Sprintf("term=%d exit=%d", e.Termination, e.Exit))
			}
		}
		if e.Type == utmp.BootTime {
			boot = e.Time
		}
	}
}

func pid(e utmp.Entry) string {
	if e.PID <= 0 {
		return ""
	}
	return strconv.Itoa(e.PID)
}

func (w *Who) when(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	loc := w.Location
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format(timeLayout)
}

// user prints a logged-in user, with message status and idle time from the
// terminal device.
func (w *Who) user(e utmp.Entry, boot time.Time) {
	line := e.Line
	if i := strings.IndexByte(line, ' '); i >= 0 {
		line = line[i+1:]
	}
	dev := line
	if !filepath.IsAbs(dev) {
		dev = "/dev/" + dev
	}
	mesg, idle := byte('?'), "  ?"
	if info, err := w.stat(dev); err == nil && dev != "/dev/" {
		mesg = '-'
		if info.Mode().Perm()&0o020 != 0 {
			mesg = '+'
		}
		atime := sysstat.Of(info).Atime
		if atime.IsZero() {
			atime = info.ModTime()
		}
		idle = w.idle(atime, boot)
	}
	host := ""
	if e.Host != "" {
		h, display, hasDisplay := strings.Cut(e.Host, ":")
		if h != "" && w.Lookup {
			if canon, err := net.LookupCNAME(h); err == nil && canon != "" {
				h = strings.TrimSuffix(canon, ".")
			}
		}
		if hasDisplay {
			host = "(" + h + ":" + display + ")"
		} else {
			host = "(" + h + ")"
		}
	}
	w.line(e.User, mesg, e.Line, w.when(e.Time), idle, pid(e), host, "")
}

// idle is how long a terminal has been idle: "  .  " under a minute, HH:MM
// under a day, " old " otherwise.
func (w *Who) idle(when, boot time.Time) string {
	now := w.Now()
	if boot.Before(when) && !when.After(now) {
		secs := int(now.Sub(when) / time.Second)
		if secs < 24*60*60 {
			if secs < 60 {
				return "  .  "
			}
			return fmt.Sprintf("%02d:%02d", secs/3600, secs%3600/60)
		}
	}
	return " old "
}

// line prints one line in GNU's columns, without trailing blanks.
func (w *Who) line(user string, state byte, line, when, idle, pid, comment, exit string) {
	var b strings.Builder
	fmt.Fprintf(&b, "%-8s", user)
	if w.includeMesg {
		b.WriteByte(' ')
		b.WriteByte(state)
	}
	fmt.Fprintf(&b, " %-12s %-*s", line, timeWidth, when)
	if w.includeIdle && !w.Short {
		fmt.Fprintf(&b, " %-6s", idle)
	}
	if !w.Short {
		fmt.Fprintf(&b, " %10s", pid)
	}
	fmt.Fprintf(&b, " %-8s", comment)
	if w.includeExit {
		fmt.Fprintf(&b, " %-12s", exit)
	}
	_, _ = w.out.WriteString(strings.TrimRight(b.String(), " "))
	_ = w.out.WriteByte('\n')
}
