package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

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
// selects the tool.
func Main(argv []string, s core.Streams) int {
	var args []string
	if len(argv) > 0 {
		args = argv[1:]
		if status, ok := RunTool(progName(argv[0]), args, s); ok {
			return status
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
	return execute(spec.Command(s), args, s), true
}

// execute runs command against args and maps the outcome to an exit status.
func execute(command *cobra.Command, args []string, s core.Streams) int {
	// cobra falls back to os.Args when handed a nil slice.
	command.SetArgs(append([]string{}, args...))
	ran, err := command.ExecuteC()
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
