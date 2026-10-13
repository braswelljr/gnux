// Package xargs implements the engine behind xargsx, a GNU-compatible
// implementation of the xargs command: build command lines from the items
// read on standard input and run them.
package xargs

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "xargsx"

// Exit statuses xargs reserves, so a caller can tell what went wrong with the
// commands it ran from a failure of xargsx itself (status 1).
const (
	// ExitCommandFailed means a command exited with a nonzero status.
	ExitCommandFailed = 123
	// ExitAborted means a command exited with status 255, which stops xargs.
	ExitAborted = 124
	// ExitSignaled means a signal ended a command.
	ExitSignaled = 125
	// ExitCannotRun means the command was found but could not be run.
	ExitCannotRun = 126
	// ExitNotFound means the command could not be found.
	ExitNotFound = 127
)

// DefaultMaxChars is the size a command line is filled to when MaxChars is
// not set, as in GNU xargs: 128 KiB, or the system's limit where that is
// smaller (see Options.MaxChars).
const DefaultMaxChars = 128 << 10

// Unlimited is the MaxProcs value of "-P 0": as many commands at once as
// possible, which xargsx bounds at MaxParallel.
const Unlimited = -1

// MaxParallel is how many commands run at once with MaxProcs set to
// Unlimited. GNU xargs starts processes until the system refuses one; a fixed
// bound keeps an in-process tool from being started thousands of times over.
const MaxParallel = 128

// Options mirrors the GNU xargs flag surface. The zero value is plain xargs:
// items separated by blanks and newlines, with quotes and backslash special,
// as many items per command as fit, one command at a time.
type Options struct {
	// Null (-0) reads items terminated by NUL; quotes and backslash are
	// literal. It wins over Delimiter.
	Null bool
	// Delimiter (-d DELIM) reads items terminated by one character, given as
	// ParseDelimiter reads it; quotes and backslash are literal.
	Delimiter string
	// ArgFile (-a FILE) reads the items from FILE and leaves standard input
	// to the commands. "-" is standard input.
	ArgFile string
	// EOF (-E EOF-STR) is the item that ends the input. It has no effect
	// with Null or Delimiter.
	EOF string

	// MaxArgs (-n N) is the most items passed to one command; 0 is no limit.
	MaxArgs int
	// MaxLines (-L N) is the most non-blank input lines passed to one
	// command; 0 is no limit. It implies Exit.
	MaxLines int
	// MaxChars (-s N) is the size limit of a command line, counting the
	// command, every argument and one terminating NUL for each; 0 is
	// DefaultMaxChars. A value over what the system takes is lowered to it
	// with a warning.
	MaxChars int
	// Exit (-x) fails when a command line asked for with MaxArgs or MaxLines
	// does not fit MaxChars, instead of running a shorter one.
	Exit bool
	// Replace (-I REPLACE-STR) runs the command once per input line, with
	// every REPLACE-STR in the command and its initial arguments replaced by
	// the line. It implies Exit, and MaxArgs and MaxLines are ignored.
	Replace string

	// NoRunIfEmpty (-r) does not run the command when the input has no
	// items; without it the command runs once.
	NoRunIfEmpty bool
	// Verbose (-t) prints each command line on the error stream before
	// running it.
	Verbose bool
	// Interactive (-p) prints each command line and runs it only on an
	// affirmative answer. It implies Verbose.
	Interactive bool
	// MaxProcs (-P N) is how many commands run at once: 0 and 1 mean one at
	// a time, Unlimited means MaxParallel.
	MaxProcs int
	// ShowLimits (--show-limits) prints the size limits in effect on the
	// error stream before reading the input.
	ShowLimits bool

	// OpenTTY (-o) and ProcessSlotVar (--process-slot-var=VAR) are GNU
	// options xargsx does not implement: setting either makes Run fail with
	// a usage error instead of running anything.
	OpenTTY        bool
	ProcessSlotVar string
}

// Xargs is the engine. Construct it with New.
type Xargs struct {
	Options

	// Exec runs the gnux tool called name, if there is one; found is false
	// for any other name, which is then run as a program found on PATH. The
	// xargsx command sets it to the tool registry; nil runs every command as
	// a program. Tests use it to record the command lines. With MaxProcs
	// over 1 it is called from several goroutines at once.
	Exec func(name string, args []string, s core.Streams) (status int, found bool)

	// PromptFn answers the -p prompt for the command line it is given. It is
	// the injection point tests use; nil reads a line from the terminal.
	PromptFn func(command string) bool

	// Environ supplies the environment whose size counts against the
	// system's command-line limit. It defaults to os.Environ.
	Environ func() []string

	streams core.Streams
}

// New builds an xargs engine using the streams s: items are read from s.In
// (or from Options.ArgFile), and the commands write to s.Out and s.Err.
func New(opts Options, s core.Streams) *Xargs {
	return &Xargs{Options: opts, Environ: os.Environ, streams: s}
}

// Run reads the items and runs args, the command and its initial arguments
// ("echo" when empty), with them appended. The result is nil when every
// command succeeded, a *core.ExitError carrying one of the Exit statuses when
// a command failed or could not be run, core.ErrFailed for a failure of
// xargsx itself and a usage error for an option that cannot be honored.
func (x *Xargs) Run(args []string) error {
	switch {
	case x.OpenTTY:
		return core.Usagef("the --open-tty (-o) option is not supported")
	case x.ProcessSlotVar != "":
		return core.Usagef("the --process-slot-var option is not supported")
	}

	r := newRunner(x)
	defer r.close()

	in := &input{max: 0, eofStr: x.EOF, wholeLine: x.Replace != "", onNUL: func() {
		r.diag.Warnf("WARNING: a NUL character occurred in the input.  It cannot be passed through " +
			"in the argument list.  Did you mean to use the --null option?")
	}}
	switch {
	case x.Null:
		in.delimited, in.eofStr = true, ""
	case x.Delimiter != "":
		delim, err := ParseDelimiter(x.Delimiter)
		if err != nil {
			r.diag.Errorf("%v", err)
			return core.ErrFailed
		}
		in.delimited, in.delim, in.eofStr = true, delim, ""
	}

	envSize := 0
	for _, v := range x.Environ() {
		envSize += len(v) + 1
	}
	// GNU leaves 2048 bytes of headroom under the system's limit, after the
	// environment, and never goes below POSIX's minimum.
	usable := max(systemArgMax-2048-envSize, posixArgMax)
	limit := min(DefaultMaxChars, usable)
	if x.MaxChars > 0 {
		limit = x.MaxChars
		if limit > usable {
			r.diag.Warnf("warning: value %d for -s option is too large, using %d instead", limit, usable)
			limit = usable
		}
	}
	in.max = limit - 1

	command := args
	if len(command) == 0 {
		command = []string{"echo"}
	}
	if x.ShowLimits {
		x.showLimits(r.streams.Err, command[0], envSize, usable, limit)
	}
	base := 0
	for _, a := range command {
		base += len(a) + 1
	}
	if x.Replace == "" && base > limit {
		r.diag.Errorf("cannot fit single argument within argument list size limit")
		return core.ErrFailed
	}

	src := x.streams.In
	if x.ArgFile != "" && x.ArgFile != "-" {
		f, err := os.Open(x.ArgFile)
		if err != nil {
			r.diag.Errorf("Cannot open input file '%s': %s", x.ArgFile, core.ErrText(err))
			return core.ErrFailed
		}
		defer f.Close()
		src = f
		// The items do not come from standard input, so the commands may
		// read it.
		r.inheritStdin()
	}
	in.r = core.NewReader(src)

	if x.Replace != "" {
		return x.runReplace(r, in, command, limit)
	}
	return x.runBatches(r, in, command, base, limit)
}

// posixArgMax is the smallest command-line size POSIX lets a system have.
const posixArgMax = 4096

// runBatches appends items to the command until a limit is reached (-n, -L,
// or the size limit), runs it, and starts over.
func (x *Xargs) runBatches(r *runner, in *input, command []string, base, limit int) error {
	exit := x.Exit || x.MaxLines > 0
	bounded := x.MaxArgs > 0 || x.MaxLines > 0

	// Each command line gets a slice of its own: a tool or a goroutine may
	// still hold the previous one.
	fresh := func(size int) []string {
		argv := make([]string, len(command), max(size, len(command)+8))
		copy(argv, command)
		return argv
	}
	argv := fresh(0)
	chars, lines := base, 0
	flush := func() bool {
		ok := r.launch(argv)
		argv, chars, lines = fresh(len(argv)), base, 0
		return ok
	}

	for !r.stopped() {
		item, endsLine, err := in.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// GNU runs what it had collected before giving up on the input.
			if len(argv) > len(command) {
				flush()
			}
			r.inputError(err)
			return r.result()
		}

		if size := len(item) + 1; chars+size > limit {
			switch {
			case len(argv) == len(command):
				r.fail("cannot fit single argument within argument list size limit")
				return r.result()
			case exit && bounded:
				r.fail("argument list too long")
				return r.result()
			}
			if !flush() {
				break
			}
			// GNU would now pass the item whatever its size; the limit is
			// kept strict here, since a system limit may stand behind it.
			if base+size > limit {
				r.fail("cannot fit single argument within argument list size limit")
				return r.result()
			}
		}
		argv = append(argv, item)
		chars += len(item) + 1
		if endsLine {
			lines++
		}
		items := len(argv) - len(command)
		if (x.MaxArgs > 0 && items >= x.MaxArgs) || (x.MaxLines > 0 && lines >= x.MaxLines) {
			if !flush() {
				break
			}
		}
	}

	if !r.stopped() && (len(argv) > len(command) || (!x.NoRunIfEmpty && r.launched == 0)) {
		r.launch(argv)
	}
	return r.result()
}

// runReplace runs the command once per input line, replacing every
// occurrence of Replace in the command and its initial arguments by the line.
func (x *Xargs) runReplace(r *runner, in *input, command []string, limit int) error {
	for !r.stopped() {
		line, _, err := in.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			r.inputError(err)
			break
		}

		argv := make([]string, len(command))
		total := 0
		for i, a := range command {
			if strings.Contains(a, x.Replace) {
				a = strings.ReplaceAll(a, x.Replace, line)
			}
			if len(a)+1 > limit {
				r.fail("command too long")
				return r.result()
			}
			argv[i] = a
			total += len(a) + 1
		}
		if total > limit {
			// -I implies -x: the line is never split over two commands.
			r.fail("argument list too long")
			break
		}
		if !r.launch(argv) {
			break
		}
	}
	return r.result()
}

// showLimits prints the limits in the layout of GNU xargs --show-limits.
func (x *Xargs) showLimits(w io.Writer, name string, envSize, usable, limit int) {
	fmt.Fprintf(w, "Your environment variables take up %d bytes\n"+
		"POSIX upper limit on argument length (this system): %d\n"+
		"POSIX smallest allowable upper limit on argument length (all systems): %d\n"+
		"Maximum length of command we could actually use: %d\n"+
		"Size of command buffer we are actually using: %d\n"+
		"Maximum parallelism (--max-procs must be no greater): %d\n",
		envSize, systemArgMax-envSize, posixArgMax, usable, limit, math.MaxInt32)

	if (x.ArgFile == "" || x.ArgFile == "-") && core.IsTerminal(x.streams.In) {
		fmt.Fprint(w, "\nExecution of xargs will continue now, and it will try to read its input and "+
			"run commands; if this is not what you wanted to happen, please type the end-of-file keystroke.\n")
		if !x.NoRunIfEmpty {
			fmt.Fprintf(w, "Warning: %s will be run at least once.  If you do not want that to happen, "+
				"then press the interrupt keystroke.\n", name)
		}
	}
}
