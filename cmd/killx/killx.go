// Package killx wires the killx command (GNU kill) to its engine.
package killx

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the killx tool. A signal is usually written as an option
// (-9, -KILL), which a flag parser cannot know in advance, so the arguments
// are parsed here.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "kill",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for killx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    kill.Name,
		SeeAlso: []string{"timeoutx", "nohupx", "nicex"},
		Use:     kill.Name + " [-s SIGNAL | -SIGNAL] PID...",
		Short:   "Send a signal to processes, or list signals.",
		Long: "killx sends a signal, TERM by default, to each process PID. The signal is " +
			"given as -s SIGNAL, -n NUMBER, or -SIGNAL directly (-9, -KILL, -SIGKILL), by name " +
			"in any case with or without the SIG prefix, or by number; signal 0 sends nothing " +
			"and only checks that the process exists. A PID that cannot be signaled is " +
			"reported and the remaining PIDs are still tried. -l prints the signal names, or " +
			"converts each operand between a name and a number (an exit status above 128 is " +
			"taken as 128 plus a signal, so 'killx -l 137' prints KILL); -L (or -t) prints a " +
			"table of numbers, names and descriptions. Unlike GNU kill, killx reads options " +
			"wherever they appear before '--', so 'killx 1234 -9' sends KILL to 1234 rather " +
			"than signaling process group 9; a negative PID (a process group, or -1 for every " +
			"process) must follow '--'. Giving two signals is an error. On Windows, which has " +
			"no signals, only KILL and TERM can be sent, both by terminating the process, and " +
			"0 checks that it exists; any other signal fails with 'Operation not supported'.",
		Permissions: "A process may be signaled by its owner, or by root (or with CAP_KILL). " +
			"On Windows, terminating a process needs the right to terminate it, which an " +
			"administrator has for every process.",
		UseCases: []string{
			"Stopping a runaway or hung process.",
			"Asking a daemon to reload its configuration with HUP.",
			"Checking whether a process is still alive with signal 0.",
			"Translating a shell exit status such as 137 into a signal name.",
			"Stopping a process the same way on Linux, macOS and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Ask a process to terminate:",
				Bash:        "killx 1234",
				PowerShell:  "killx 1234",
			},
			{
				Description: "Kill a process that ignores TERM:",
				Bash:        "killx -9 1234",
				PowerShell:  "killx -9 1234",
			},
			{
				Description: "Name the signal:",
				Bash:        "killx -s KILL 1234",
				PowerShell:  "killx -s KILL 1234",
			},
			{
				Description: "Ask a daemon to reload its configuration:",
				Bash:        "killx -HUP \"$(catx /run/nginx.pid)\"",
				PowerShell:  "killx -HUP (catx nginx.pid)",
			},
			{
				Description: "Check whether a process is alive:",
				Bash:        "killx -0 1234 && echox alive",
				PowerShell:  "killx -0 1234; if ($LASTEXITCODE -eq 0) { 'alive' }",
			},
			{
				Description: "Signal several processes:",
				Bash:        "killx -TERM 1234 5678",
				PowerShell:  "killx -TERM 1234 5678",
			},
			{
				Description: "Signal a process group:",
				Bash:        "killx -TERM -- -1234",
				PowerShell:  "killx -TERM -- -1234",
			},
			{
				Description: "List the signal names:",
				Bash:        "killx -l",
				PowerShell:  "killx -l",
			},
			{
				Description: "Show the table of signals:",
				Bash:        "killx -L",
				PowerShell:  "killx -L",
			},
			{
				Description: "Find which signal ended a command that exited with 137:",
				Bash:        "killx -l 137",
				PowerShell:  "killx -l 137",
			},
			{
				Description: "Convert a signal name to its number:",
				Bash:        "killx -l TERM",
				PowerShell:  "killx -l TERM",
			},
			{
				Description: "Send a signal in the portable gnux shell:",
				Bash:        "gnux shell -c 'killx -0 1234 || echox gone'",
				PowerShell:  "gnux shell -c 'killx -0 1234 || echox gone'",
			},
		},
	}
}

// Register installs the killx flags on fs; it is used for the help text, the
// arguments themselves being parsed by parseArgs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("signal", "s", "", "specify the name or number of the signal to be sent")
	fs.StringP("number", "n", "", "same as -s, with a `NUMBER`")
	fs.BoolP("list", "l", false, "list signal names, or convert signal names to/from numbers")
	fs.BoolP("table", "L", false, "print a table of signal information (also -t)")
}

// parsed is a command line split into its parts.
type parsed struct {
	opts     kill.Options
	operands []string
	help     bool
	version  bool
}

// parseArgs reads GNU kill's command line. Options may appear anywhere before
// "--": a PID never starts with '-' unless it follows "--".
func parseArgs(args []string) (parsed, error) {
	var p parsed
	signals := 0
	setSignal := func(s string) error {
		signals++
		if signals > 1 {
			return core.Usagef("multiple signals specified")
		}
		p.opts.Signal = s
		return nil
	}
	setMode := func(m kill.Mode) error {
		if p.opts.Mode != kill.ModeSend && p.opts.Mode != m {
			return core.Usagef("multiple -l or -t options specified")
		}
		p.opts.Mode = m
		return nil
	}
	value := func(i *int, name string) (string, error) {
		if *i+1 >= len(args) {
			return "", core.Usagef("option requires an argument -- '%s'", name)
		}
		*i++
		return args[*i], nil
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		var err error
		switch {
		case a == "--":
			p.operands = append(p.operands, args[i+1:]...)
			i = len(args)
		case len(a) < 2 || a[0] != '-':
			p.operands = append(p.operands, a)
		case strings.HasPrefix(a, "--"):
			name, val, hasVal := strings.Cut(a[2:], "=")
			switch name {
			case "help":
				p.help = true
			case "version":
				p.version = true
			case "list":
				err = setMode(kill.ModeList)
			case "table":
				err = setMode(kill.ModeTable)
			case "signal":
				if !hasVal {
					val, err = value(&i, "signal")
				}
				if err == nil {
					err = setSignal(val)
				}
			default:
				err = core.Usagef("unrecognized option '%s'", a)
			}
		case a == "-l":
			err = setMode(kill.ModeList)
		case a == "-L" || a == "-t":
			err = setMode(kill.ModeTable)
		case a[1] == 's' || a[1] == 'n':
			val := a[2:]
			if val == "" {
				val, err = value(&i, a[1:2])
			}
			if err == nil {
				err = setSignal(val)
			}
		default:
			// -SIGNAL: a name or number.
			err = setSignal(a[1:])
		}
		if err != nil {
			return p, err
		}
	}
	if p.opts.Mode != kill.ModeSend && signals > 0 {
		return p, core.Usagef("cannot combine signal with -l or -t")
	}
	return p, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	p, err := parseArgs(args)
	if err != nil {
		return err
	}
	switch {
	case p.help:
		Spec().WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	case p.version:
		tool.PrintVersion(s.Out, kill.Name)
		return nil
	}
	return kill.New(p.opts, s).Run(p.operands)
}
