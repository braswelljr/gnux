// Package cmd assembles the gnux multi-call binary: a root command carrying one
// subcommand per tool, and a dispatcher that runs a tool directly when the
// binary is invoked under that tool's name.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/tool"
)

// NewRoot builds the gnux root command with every registered tool attached as a
// subcommand. main and the docs generator both build the command through here
// so they stay in sync.
func NewRoot(s core.Streams) *cobra.Command {
	root := &cobra.Command{
		Use:   common.AppName + " [OPTION]... [TOOL [ARGUMENT]...]",
		Short: "The GNU core utilities' interface on every platform, in one Go binary.",
		Long: "gnux brings the interface of the GNU core utilities to every platform. Each tool " +
			"is named after its GNU counterpart with an x appended (rm is rmx, cp is cpx) and " +
			"mirrors its flags, semantics and diagnostics.\n\n" +
			"Run a tool as a subcommand (gnux rmx -rf build), or link the binary to the tool's " +
			"name and invoke it directly (rmx -rf build): gnux looks at the name it was started " +
			"under and behaves as that tool. --list prints every tool name, one per line.\n\n" +
			"gnux shell is a small portable shell whose commands are these tools: it runs the " +
			"same script on every platform, with no dependency on the system's own shell.\n\n" +
			"Every tool answers --help with its options, first examples and exit status. gnux " +
			"help TOOL prints the tool's full manual - every option, permissions, use cases, " +
			"every example for bash and PowerShell, exit status and related tools - and gnux " +
			"help TOOL OPTION explains one option with the examples that use it.",
		Example: "  # run a tool through gnux\n" +
			"  gnux rmx -rf build/\n\n" +
			"  # make every tool a command of its own\n" +
			"  for t in $(gnux --list); do ln -s \"$(command -v gnux)\" \"$HOME/bin/$t\"; done\n" +
			"  rmx -rf build/\n\n" +
			"  # read a tool's full manual, or one option in depth\n" +
			"  gnux help rmx\n" +
			"  gnux help rmx --force",
		Args:                  cobra.ArbitraryArgs,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		RunE: func(command *cobra.Command, args []string) error {
			fs := command.Flags()
			if v, _ := fs.GetBool("version"); v {
				tool.PrintVersion(s.Out, common.AppName)
				return nil
			}
			if l, _ := fs.GetBool("list"); l {
				for _, t := range Tools() {
					fmt.Fprintln(s.Out, t.Name())
				}
				return nil
			}
			if len(args) > 0 {
				return core.Usagef("unknown tool '%s'", args[0])
			}
			return command.Help()
		},
	}
	root.SetIn(s.In)
	root.SetOut(s.Out)
	root.SetErr(s.Err)
	// `gnux completion SHELL` prints a completion script; the overview lists it.
	root.CompletionOptions.HiddenDefaultCmd = false
	root.SetHelpCommand(newHelpCommand(s))
	root.SetHelpFunc(func(*cobra.Command, []string) {
		writeOverview(s.Out, tool.StyleFor(s.Out))
	})

	// Defined without a shorthand, as on every tool: GNU tools take --help only.
	root.Flags().Bool("help", false, "display this help and exit")
	root.Flags().Bool("list", false, "print the name of every tool, one per line")
	root.Flags().Bool("version", false, "output version information and exit")

	for _, t := range Commands() {
		root.AddCommand(t.Command(s))
	}
	return root
}
