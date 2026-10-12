// Package kill implements the engine behind killx, a GNU-compatible
// implementation of the GNU/UNIX kill command: send a signal to processes, or
// list the signal names and numbers.
//
// It also holds the signal table and the process helpers other engines that
// run or signal a command share (core/timeout): ParseSignal, SignalName, Send
// and ExitStatus.
package kill

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "killx"

// Signal describes one signal: its number on this platform, its name without
// the SIG prefix and the description strsignal(3) gives it.
type Signal struct {
	Number      int
	Name        string
	Description string
}

// ErrUnsupported is returned by Send for a signal or target the platform
// cannot deliver (on Windows, anything but terminating a single process).
var ErrUnsupported = errors.New("operation not supported")

// Signals returns the platform's signal table, ordered by number.
func Signals() []Signal {
	return append([]Signal(nil), signals...)
}

// ParseSignal resolves a signal given by name or number, as kill -s and
// timeout -s accept it: a name with or without the SIG prefix, in any case
// (TERM, sigterm), or a number, 0 included (it sends nothing and only checks
// that the process exists).
func ParseSignal(spec string) (int, bool) {
	if spec == "" {
		return 0, false
	}
	if spec[0] >= '0' && spec[0] <= '9' {
		n, err := strconv.Atoi(spec)
		if err != nil {
			return 0, false
		}
		if n == 0 {
			return 0, true
		}
		if _, ok := SignalName(n); !ok {
			return 0, false
		}
		return n, true
	}
	name := strings.ToUpper(spec)
	for _, try := range []string{name, strings.TrimPrefix(name, "SIG")} {
		for _, s := range signals {
			if s.Name == try {
				return s.Number, true
			}
		}
	}
	return 0, false
}

// SignalName returns the name of signal n, without the SIG prefix.
func SignalName(n int) (string, bool) {
	for _, s := range signals {
		if s.Number == n {
			return s.Name, true
		}
	}
	return "", false
}

// Mode selects what killx does with its operands.
type Mode int

const (
	// ModeSend sends the signal to each PID operand (the default).
	ModeSend Mode = iota
	// ModeList (-l) prints signal names, or converts each operand between a
	// name and a number.
	ModeList
	// ModeTable (-L, -t) prints a table of numbers, names and descriptions.
	ModeTable
)

// Options mirrors the GNU kill flag surface.
type Options struct {
	// Signal (-s SIG, -n NUM, -SIG) is the signal to send, by name or number.
	// Empty means TERM.
	Signal string
	// Mode selects sending, listing (-l) or the table (-L).
	Mode Mode
}

// Kill is the engine. Construct it with New.
type Kill struct {
	Options

	// SendFn delivers signal sig to pid. It defaults to Send, and is the
	// injection point tests use so that no real process is signaled.
	SendFn func(pid, sig int) error

	streams core.Streams
	diag    *core.Diag
}

// New builds a kill engine writing to s.
func New(opts Options, s core.Streams) *Kill {
	return &Kill{
		Options: opts,
		SendFn:  Send,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run sends the signal to every PID operand, or lists signals. A PID that
// cannot be signaled is reported and the rest are still tried. The returned
// error is core.ErrFailed if any operand failed.
func (k *Kill) Run(args []string) error {
	switch k.Mode {
	case ModeList:
		return k.list(args, false)
	case ModeTable:
		return k.list(args, true)
	}

	sig := TermSignal
	if k.Signal != "" {
		n, ok := ParseSignal(k.Signal)
		if !ok {
			k.diag.Errorf("'%s': invalid signal", k.Signal)
			return core.ErrFailed
		}
		sig = n
	}
	if len(args) == 0 {
		return core.Usagef("no process ID specified")
	}

	for _, a := range args {
		pid, err := strconv.Atoi(a)
		if err != nil {
			k.diag.Errorf("'%s': invalid process id", a)
			continue
		}
		if err := k.SendFn(pid, sig); err != nil {
			k.diag.Errorf("%s: %s", a, core.ErrText(err))
		}
	}
	return k.diag.Result()
}

// list prints every signal, or converts each operand: a number (or an exit
// status above 128) to its name, a name to its number. With table, each line
// is "NUMBER NAME DESCRIPTION".
func (k *Kill) list(args []string, table bool) error {
	numWidth, nameWidth := len(strconv.Itoa(signals[len(signals)-1].Number)), 0
	for _, s := range signals {
		nameWidth = max(nameWidth, len(s.Name))
	}
	var b strings.Builder
	row := func(s Signal) {
		if table {
			fmt.Fprintf(&b, "%*d %-*s %s\n", numWidth, s.Number, nameWidth, s.Name, s.Description)
			return
		}
		fmt.Fprintln(&b, s.Name)
	}

	if len(args) == 0 {
		for _, s := range signals {
			row(s)
		}
	}
	for _, a := range args {
		s, isNumber, ok := lookupOperand(a)
		switch {
		case !ok:
			k.diag.Errorf("'%s': invalid signal", a)
		case table:
			row(s)
		case isNumber:
			fmt.Fprintln(&b, s.Name)
		default:
			fmt.Fprintln(&b, s.Number)
		}
	}
	if _, err := k.streams.Out.Write([]byte(b.String())); err != nil {
		k.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return k.diag.Result()
}

// lookupOperand resolves a -l operand. A number above the largest signal is
// taken as an exit status, 128 plus the signal, as GNU kill does.
func lookupOperand(a string) (Signal, bool, bool) {
	if a != "" && a[0] >= '0' && a[0] <= '9' {
		n, err := strconv.Atoi(a)
		if err != nil {
			return Signal{}, true, false
		}
		if n > signals[len(signals)-1].Number {
			n &= 127
		}
		for _, s := range signals {
			if s.Number == n {
				return s, true, true
			}
		}
		return Signal{}, true, false
	}
	n, ok := ParseSignal(a)
	if !ok || n == 0 {
		return Signal{}, false, false
	}
	for _, s := range signals {
		if s.Number == n {
			return s, false, true
		}
	}
	return Signal{}, false, false
}

// ExitStatus converts a finished process's state to the status a shell
// reports: its exit code, or 128 plus N when signal N killed it (Unix).
func ExitStatus(state *os.ProcessState) int {
	return exitStatus(state)
}

// Send delivers signal sig to pid. On Unix a negative pid names a process
// group and 0 the caller's group, as kill(2) defines. Elsewhere only a single
// process can be signaled: 0 checks that it exists, KILL and TERM terminate
// it, and anything else is ErrUnsupported.
func Send(pid, sig int) error {
	return send(pid, sig)
}

// SignalProcess delivers signal sig to a process this program started. With
// group set (Unix), the signal goes to the process group the process leads.
func SignalProcess(p *os.Process, sig int, group bool) error {
	return signalProcess(p, sig, group)
}

// TermSignal and KillSignal are the numbers of TERM (the default signal) and
// KILL on this platform.
var (
	TermSignal = mustSignal("TERM")
	KillSignal = mustSignal("KILL")
)

func mustSignal(name string) int {
	n, ok := ParseSignal(name)
	if !ok {
		panic("kill: no signal " + name)
	}
	return n
}
