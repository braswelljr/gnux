// Package timeout implements the engine behind timeoutx, a GNU-compatible
// implementation of the GNU/UNIX timeout command: run a command, and signal
// it if it is still running after a time limit.
package timeout

import (
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "timeoutx"

// Exit statuses timeout reserves, so a caller can tell a timeout, and
// timeoutx's own failures, from the command's status.
const (
	// ExitTimedOut means the command was still running at the limit.
	ExitTimedOut = 124
	// ExitCanceled means timeoutx itself failed.
	ExitCanceled = 125
	// ExitCannotInvoke means the command was found but could not be run.
	ExitCannotInvoke = 126
	// ExitNotFound means the command could not be found.
	ExitNotFound = 127
)

// Options mirrors the GNU timeout flag surface. The time limit itself is the
// first operand.
type Options struct {
	// Signal (-s SIG) is sent at the limit, by name or number. Empty means TERM.
	Signal string
	// KillAfter (-k DURATION), when positive, sends KILL if the command is
	// still running that long after the first signal.
	KillAfter time.Duration
	// PreserveStatus (--preserve-status) exits with the command's status even
	// when it timed out, instead of 124.
	PreserveStatus bool
	// Foreground (--foreground) signals only the command, not the processes
	// it started, and leaves it in timeoutx's process group so it can read
	// from the terminal.
	Foreground bool
	// Verbose (-v) reports each signal sent on the error stream.
	Verbose bool
}

// Timeout is the engine. Construct it with New.
type Timeout struct {
	Options

	// IsTerminal reports whether a stream is a terminal. It defaults to
	// core.IsTerminal; a command reading from a terminal is kept in the
	// terminal's process group, as --foreground would.
	IsTerminal func(any) bool

	streams core.Streams
	diag    *core.Diag
}

// New builds a timeout engine. The command it runs inherits the streams s.
func New(opts Options, s core.Streams) *Timeout {
	return &Timeout{
		Options:    opts,
		IsTerminal: core.IsTerminal,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// ParseDuration parses a GNU timeout interval: a non-negative decimal or
// hexadecimal floating-point number with an optional unit suffix, s
// (seconds, the default), m (minutes), h (hours) or d (days). 0 means no
// limit, as does inf; a value too large for a time.Duration is capped.
func ParseDuration(s string) (time.Duration, error) {
	invalid := core.Usagef("invalid time interval '%s'", s)
	num, unit := s, time.Second
	if n := len(s); n > 0 {
		switch s[n-1] {
		case 's':
			num = s[:n-1]
		case 'm':
			num, unit = s[:n-1], time.Minute
		case 'h':
			num, unit = s[:n-1], time.Hour
		case 'd':
			num, unit = s[:n-1], 24*time.Hour
		}
	}
	if num == "" || num[0] == '-' {
		return 0, invalid
	}
	// strtod reads a hexadecimal number without a binary exponent; Go's
	// parser insists on one.
	if h := strings.ToLower(strings.TrimPrefix(num, "+")); strings.HasPrefix(h, "0x") && !strings.Contains(h, "p") {
		num += "p0"
	}
	f, err := strconv.ParseFloat(num, 64)
	if (err != nil && !errors.Is(err, strconv.ErrRange)) || math.IsNaN(f) || f < 0 {
		return 0, invalid
	}
	if f == 0 || math.IsInf(f, 1) {
		return 0, nil
	}
	d := f * float64(unit)
	switch {
	case d >= math.MaxInt64:
		return time.Duration(math.MaxInt64), nil
	case d < 1:
		// A positive interval shorter than a nanosecond still times out.
		return 1, nil
	}
	return time.Duration(d), nil
}

// Run treats args[0] as the time limit and runs the rest as a command. The
// result is nil or a *core.ExitError: the command's status, 124 when it
// timed out, 125 when timeoutx failed, 126 or 127 when the command could not
// be run.
func (t *Timeout) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	limit, err := ParseDuration(args[0])
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return core.Usagef("missing operand after '%s'", args[0])
	}

	sig := kill.TermSignal
	if t.Signal != "" {
		n, ok := kill.ParseSignal(t.Signal)
		if !ok {
			return t.fail(ExitCanceled, "'%s': invalid signal", t.Signal)
		}
		sig = n
	}

	name := args[1]
	path, err := exec.LookPath(name)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		return t.cannotRun(name, err)
	}
	// The command's error stream and timeoutx's own diagnostics share one
	// writer; unless it is a file, the command's output is copied into it
	// from another goroutine, so the writes are serialized.
	errw := t.streams.Err
	if _, isFile := errw.(*os.File); !isFile {
		errw = &lockedWriter{w: errw}
		t.diag = core.NewDiag(Name, core.Streams{In: t.streams.In, Out: t.streams.Out, Err: errw})
	}
	// Without --foreground the command leads a process group of its own, so
	// the signal reaches everything it started. A command reading from the
	// terminal stays in the terminal's group, or it could not read.
	group := !t.Foreground && !t.IsTerminal(t.streams.In)
	cmd := &exec.Cmd{
		Path:        path,
		Args:        args[1:],
		Stdin:       t.streams.In,
		Stdout:      t.streams.Out,
		Stderr:      errw,
		SysProcAttr: procAttr(group),
	}
	if err := cmd.Start(); err != nil {
		return t.cannotRun(name, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	received, stopForwarding := forwardSignals()
	defer stopForwarding()

	var expired, killAfter <-chan time.Time
	if limit > 0 {
		timer := time.NewTimer(limit)
		defer timer.Stop()
		expired = timer.C
	}

	timedOut, killed := false, false
	signal := func(s int) {
		if t.Verbose {
			t.diag.Warnf("sending signal %s to command '%s'", signalName(s), name)
		}
		_ = kill.SignalProcess(cmd.Process, s, group)
	}

	for {
		select {
		case err := <-done:
			return t.result(err, timedOut, killed)
		case <-expired:
			expired = nil
			timedOut = true
			signal(sig)
			if sig == kill.KillSignal {
				killed = true
				continue
			}
			// A stopped command must be continued to act on the signal.
			if cont, ok := kill.ParseSignal("CONT"); ok && sig != cont {
				_ = kill.SignalProcess(cmd.Process, cont, group)
			}
			if t.KillAfter > 0 {
				timer := time.NewTimer(t.KillAfter)
				defer timer.Stop()
				killAfter = timer.C
			}
		case <-killAfter:
			killAfter = nil
			killed = true
			signal(kill.KillSignal)
		case s := <-received:
			// A signal sent to timeoutx is passed on to the command.
			_ = kill.SignalProcess(cmd.Process, s, group)
		}
	}
}

// result turns the command's end into timeoutx's exit status. A command that
// timed out and had to be killed with KILL ends with 128+9, as GNU timeout
// does when it signals its whole process group, itself included.
func (t *Timeout) result(err error, timedOut, killed bool) error {
	status := 0
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		status = kill.ExitStatus(exitErr.ProcessState)
	default:
		return t.fail(ExitCanceled, "%s", core.ErrText(err))
	}
	if timedOut && !t.PreserveStatus {
		status = ExitTimedOut
		if killed && !t.Foreground {
			status = 128 + kill.KillSignal
		}
	}
	if status == 0 {
		return nil
	}
	return &core.ExitError{Code: status}
}

// cannotRun reports why the command could not be started: 127 when it does
// not exist, 126 for anything else.
func (t *Timeout) cannotRun(name string, err error) error {
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return t.fail(ExitNotFound, "failed to run command '%s': No such file or directory", name)
	}
	return t.fail(ExitCannotInvoke, "failed to run command '%s': %s", name, core.ErrText(err))
}

// fail writes a diagnostic and returns the exit status code.
func (t *Timeout) fail(code int, format string, a ...any) error {
	t.diag.Errorf(format, a...)
	return &core.ExitError{Code: code}
}

// signalName returns a signal's name for -v, or its number when it has none.
func signalName(sig int) string {
	if name, ok := kill.SignalName(sig); ok {
		return name
	}
	return strconv.Itoa(sig)
}

// lockedWriter serializes writes to w.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
