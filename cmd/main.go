package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/tool"
)

// Main runs gnux for the given command line (argv[0] included) and returns the
// process exit status.
//
// gnux is a multi-call binary: when argv[0] names a tool - because the binary
// was linked or copied to "rmx", say - that tool runs directly with the
// remaining arguments. Otherwise the root command runs and the first argument
// selects the tool. A first argument that names a tool runs it directly too:
// so does gnux shell, and the root command, which carries every tool, is only
// built for what is left (gnux --list, gnux help, an unknown name).
func Main(argv []string, s core.Streams) int {
	var args []string
	if len(argv) > 0 {
		args = argv[1:]
		if status, ok := RunTool(progName(argv[0]), args, s); ok {
			return status
		}
	}
	if len(args) > 0 {
		if status, ok := RunTool(args[0], args[1:], s); ok {
			return status
		}
		// The shell is not a tool (no program name selects it), but a
		// script should not wait for every tool's command to be built.
		// Match the name before building a spec: building one writes out
		// its documentation, which a script run should not pay for.
		for name, spec := range map[string]func() tool.Spec{"shell": ShellSpec, "mcp": MCPSpec} {
			if args[0] == name {
				return runCommand(spec().Command(s), args[1:], s)
			}
		}
	}
	return execute(NewRoot(s), args, s)
}

// RunTool runs the tool called name - its gnux name ("rmx") or the GNU name it
// mirrors ("rm") - with args, and returns its exit status. ok is false when
// there is no such tool. It is how an application runs a tool by name without
// spawning a process, and what the gnux shell executes commands through.
func RunTool(name string, args []string, s core.Streams) (status int, ok bool) {
	spec, ok := Lookup(name)
	if !ok {
		return 0, false
	}
	return runCommand(spec.Command(s), args, s), true
}

// ExitInternal is the exit status of a tool that panicked: a bug in gnux, not
// a mistake in the command line or a failed operation.
const ExitInternal = 2

// execute runs the root command against args, through cobra, and maps the
// outcome to an exit status.
func execute(root *cobra.Command, args []string, s core.Streams) int {
	return contained(root, s, func() (*cobra.Command, error) {
		// cobra falls back to os.Args when handed a nil slice.
		root.SetArgs(append([]string{}, args...))
		return root.ExecuteC()
	})
}

// runCommand runs a command built by Spec.Command against args and maps the outcome
// to an exit status. It does what cobra's Execute does for a command with no
// subcommands (parse the flags, answer --help, call RunE) and nothing else:
// Execute is made for a command tree, and scans the whole argument list
// several times in search of a subcommand, which costs more than many tools
// do when xargsx or findx hands one tens of thousands of names.
func runCommand(command *cobra.Command, args []string, s core.Streams) int {
	return contained(command, s, func() (*cobra.Command, error) {
		err := tool.ParseFlags(command, args)
		// pflag asks for help itself when it meets -h on a tool that
		// defines no such option.
		if help, _ := command.Flags().GetBool("help"); help || errors.Is(err, pflag.ErrHelp) {
			command.HelpFunc()(command, args)
			return command, nil
		}
		if err != nil {
			return command, command.FlagErrorFunc()(command, err)
		}
		operands := command.Flags().Args()
		if command.DisableFlagParsing {
			operands = args
		}
		return command, command.RunE(command, operands)
	})
}

// contained calls fn, which runs command, and maps what it returns to an exit
// status.
//
// A panic in the tool is reported and becomes ExitInternal instead of taking
// the process down: the tool may be one stage of a gnux shell pipeline, a
// command in the playground, or a call from a program embedding gnux, and
// none of those should die of one tool's bug. An engine that starts
// goroutines brings their panics back to this one with a core.Trap.
func contained(command *cobra.Command, s core.Streams, fn func() (*cobra.Command, error)) (status int) {
	defer func() {
		if r := recover(); r != nil {
			name := tool.DiagName(command)
			stack := debug.Stack()
			// A panic carried over from another goroutine brings the stack
			// of the place it happened.
			if p, ok := r.(*core.Panic); ok {
				r, stack = p.Value, p.Stack
			}
			fmt.Fprintf(s.Err, "%s: internal error: %v\n%s", name, r, stack)
			fmt.Fprintf(s.Err, "%s: this is a bug in %s; please report it\n", name, common.AppName)
			status = ExitInternal
		}
	}()
	ran, err := fn()
	return exitStatus(ran, err, s)
}

// exitStatus reports err the way GNU tools do and maps it to an exit status.
func exitStatus(ran *cobra.Command, err error, s core.Streams) int {
	if err == nil {
		return 0
	}

	var exit *core.ExitError
	switch {
	case errors.Is(err, core.ErrFailed):
		// Diagnostics were already written by the engine.
		return 1
	case errors.As(err, &exit):
		return exit.Code
	}

	name := common.AppName
	if ran != nil {
		name = tool.DiagName(ran)
	}
	fmt.Fprintf(s.Err, "%s: %v\n", name, err)
	fmt.Fprintf(s.Err, "Try '%s --help' for more information.\n", name)
	if usage := (*core.UsageError)(nil); errors.As(err, &usage) && usage.Code != 0 {
		return usage.Code
	}
	return tool.UsageExit(ran)
}

// progName reduces argv[0] to the bare program name: no directory, no ".exe",
// lower-cased so RMX.EXE on a case-insensitive filesystem still selects rmx.
func progName(arg0 string) string {
	// Accept either separator so a Windows-style path is handled everywhere.
	if i := strings.LastIndexAny(arg0, `/\`); i >= 0 {
		arg0 = arg0[i+1:]
	}
	arg0 = filepath.Base(arg0)
	if ext := filepath.Ext(arg0); strings.EqualFold(ext, ".exe") {
		arg0 = strings.TrimSuffix(arg0, ext)
	}
	return strings.ToLower(arg0)
}
