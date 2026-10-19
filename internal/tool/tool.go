// Package tool defines what a gnux tool is - its metadata, flags and entry
// point - and turns that description into a cobra command. Every cmd/<tool>x
// package exposes a Spec; cmd/registry.go lists them and the root command, the
// argv[0] dispatcher and the doc generator all build from that one list.
package tool

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Cobra annotations recording, on a built command, what its Spec asked for.
const (
	usageExitKey = "gnux/usage-exit" // exit status for a usage error
	diagNameKey  = "gnux/diag-name"  // name diagnostics are prefixed with
	// noInterspersedKey marks a command whose options stop at the first
	// operand, which ParseFlags needs to know and pflag does not tell.
	noInterspersedKey = "gnux/no-interspersed"
)

// RunTool runs a registered tool by either of its names and reports whether
// there is one: it is cmd.RunTool, set by package cmd, which owns the registry
// and which the adapters cannot import. The adapters of tools that run other
// commands (xargsx, findx) hand it to their engine, so that a command named
// after a tool runs in-process. It is nil only in a program that never loaded
// package cmd.
var RunTool func(name string, args []string, s core.Streams) (status int, found bool)

// Spec describes one tool.
type Spec struct {
	// Meta is the tool's documentation. Meta.Name is the command name ("rmx")
	// and Meta.Use its full synopsis ("rmx [OPTION]... [FILE]...").
	Meta docmeta.Meta

	// GNU is the name of the GNU tool this one mirrors ("rm"). It is accepted as
	// an alias, both as a subcommand and as the invoked program name. It is
	// empty for a gnux command with no GNU counterpart (gnux shell).
	GNU string

	// Flags registers the tool's flags. --help and --version are added for it.
	Flags func(fs *pflag.FlagSet)

	// Run executes the tool against the parsed flags and remaining operands.
	// It returns core.ErrFailed (diagnostics already written), a
	// *core.ExitError, a *core.UsageError, or nil.
	Run func(fs *pflag.FlagSet, args []string, s core.Streams) error

	// RawArgs disables flag parsing: Run receives every argument untouched.
	// For tools whose arguments are not GNU-style options (echo, true, false).
	// A lone --help or --version is still honored.
	RawArgs bool

	// NoInterspersed stops option parsing at the first operand, for tools that
	// take another command line as their operands (env).
	NoInterspersed bool

	// UsageExit is the exit status for a command-line error. Zero means 1;
	// some tools reserve other values (ls: 2, env: 125).
	UsageExit int

	// Topics are optional per-flag documentation pages.
	Topics []Topic
}

// Topic documents one flag (or a small family of flags) of a tool on its own
// reference page.
type Topic struct {
	Meta     docmeta.Meta
	Register func(fs *pflag.FlagSet)
}

// Name returns the tool's command name.
func (s Spec) Name() string {
	return s.Meta.Name
}

// Command builds the cobra command for the tool, wired to the given streams.
func (s Spec) Command(streams core.Streams) *cobra.Command {
	c := &cobra.Command{
		Use:                s.Meta.Use,
		Short:              s.Meta.Short,
		Long:               s.Meta.Long,
		Example:            docmeta.ExampleBlock(s.Meta.Examples),
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableFlagParsing: s.RawArgs,
		// Use already spells out the synopsis, [OPTION]... included.
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			usageExitKey:      strconv.Itoa(s.usageExit()),
			diagNameKey:       s.DiagName(),
			noInterspersedKey: strconv.FormatBool(s.NoInterspersed),
		},
	}
	if s.GNU != "" {
		c.Aliases = []string{s.GNU}
	}
	c.SetIn(streams.In)
	c.SetOut(streams.Out)
	c.SetErr(streams.Err)

	fs := c.Flags()
	registerStandard(fs)
	if s.Flags != nil {
		s.Flags(fs)
	}
	fs.SetInterspersed(!s.NoInterspersed)

	// --help prints the structured screen; `gnux help <name>` the full manual.
	c.SetHelpFunc(func(command *cobra.Command, _ []string) {
		s.WriteHelp(command.OutOrStdout(), fs, StyleFor(streams.Out))
	})

	c.RunE = func(command *cobra.Command, args []string) error {
		if s.RawArgs && len(args) == 1 {
			switch args[0] {
			case "--help":
				return command.Help()
			case "--version":
				PrintVersion(streams.Out, s.DiagName())
				return nil
			}
		}
		if v, _ := fs.GetBool("version"); v && !s.RawArgs {
			PrintVersion(streams.Out, s.DiagName())
			return nil
		}
		return s.Run(fs, args, streams)
	}
	return c
}

// DiagName returns the name the command's diagnostics are prefixed with: the
// tool name, or "gnux <name>" for a gnux command that is not a tool.
func (s Spec) DiagName() string {
	if s.GNU == "" {
		return common.AppName + " " + s.Name()
	}
	return s.Name()
}

func (s Spec) usageExit() int {
	if s.UsageExit == 0 {
		return 1
	}
	return s.UsageExit
}

// UsageExit returns the usage-error exit status recorded on a command built by
// Spec.Command, or 1 for any other command.
func UsageExit(c *cobra.Command) int {
	if c != nil {
		if n, err := strconv.Atoi(c.Annotations[usageExitKey]); err == nil {
			return n
		}
	}
	return 1
}

// SetDiagName records the name a command's usage errors are prefixed with, for
// a command not built by Spec.Command (gnux help).
func SetDiagName(c *cobra.Command, name string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[diagNameKey] = name
}

// DiagName returns the diagnostic prefix recorded on a command built by
// Spec.Command, or the command's own name for any other command.
func DiagName(c *cobra.Command) string {
	if name := c.Annotations[diagNameKey]; name != "" {
		return name
	}
	return c.Name()
}
