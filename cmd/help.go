package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/tool"
)

// newHelpCommand builds `gnux help [TOOL [OPTION]]`:
//
//	gnux help               every command, with how to read more
//	gnux help rmx           the full manual for rmx (rm works too)
//	gnux help rmx force     one option in depth (-f, --force, force)
//	gnux help shell         the manual for gnux shell
//
// It takes the place of cobra's default help command, which would only repeat --help.
func newHelpCommand(s core.Streams) *cobra.Command {
	c := &cobra.Command{
		Use:   "help [TOOL [OPTION]]",
		Short: "Show the full manual for a tool, or one of its options in depth.",
		Long: "With no argument, gnux help lists every command. With a TOOL - its gnux name " +
			"(rmx) or the GNU name (rm) - it prints that tool's full manual: synopsis, " +
			"description, every option, permissions, use cases, every example for both bash and " +
			"PowerShell, exit status and related tools. With an OPTION as well (force, -f or " +
			"--force) it explains that one option, with the examples that use it.",
		// The OPTION operand is itself an option (gnux help lsx -l), so
		// nothing here is parsed as a flag of help.
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(_ *cobra.Command, args []string) error {
			st := tool.StyleFor(s.Out)
			if len(args) > 2 {
				return core.Usagef("too many arguments: expected at most TOOL and OPTION")
			}
			if len(args) == 1 && args[0] == "--help" {
				args = []string{"help"}
			}
			if len(args) > 0 && args[0] == "help" {
				writeHelpHelp(s.Out, st)
				return nil
			}
			if len(args) == 0 {
				writeOverview(s.Out, st)
				return nil
			}
			spec, ok := lookupCommand(args[0])
			if !ok {
				return core.Usagef("no help for '%s': not a gnux command (see 'gnux --list')", args[0])
			}
			fs := spec.Command(core.Discard()).Flags()
			if len(args) == 1 {
				spec.WriteManual(s.Out, fs, st)
				return nil
			}
			if !spec.WriteFlagHelp(s.Out, fs, args[1], st) {
				return core.Usagef("%s has no option '%s' (see 'gnux help %s')", spec.Name(), args[1], spec.Name())
			}
			return nil
		},
	}
	tool.SetDiagName(c, common.AppName+" help")
	return c
}

// writeHelpHelp is the help for gnux help itself.
func writeHelpHelp(w io.Writer, st tool.Style) {
	fmt.Fprintf(w, "Usage: %s help [TOOL [OPTION]]\n", common.AppName)
	fmt.Fprintln(w, "Show the full manual for a tool, or one of its options in depth.")
	fmt.Fprintf(w, "\n%s\n", heading(st, "Forms:"))
	fmt.Fprintln(w, "  gnux help                 every command, with a one-line summary")
	fmt.Fprintln(w, "  gnux help TOOL            the full manual: synopsis, description, every option,")
	fmt.Fprintln(w, "                            permissions, use cases, every example (bash and")
	fmt.Fprintln(w, "                            PowerShell), exit status and related tools")
	fmt.Fprintln(w, "  gnux help TOOL OPTION     one option in depth, with the examples that use it")
	fmt.Fprintln(w, "\n  TOOL is a gnux name (rmx), a GNU name (rm) or shell. OPTION may be written as")
	fmt.Fprintln(w, "  a long name (--force or force) or a shorthand (-f or f).")
	fmt.Fprintf(w, "\n%s\n", heading(st, "Examples:"))
	fmt.Fprintf(w, "  $ %s\n  $ %s\n  $ %s\n", command(st, "gnux help cpx"), command(st, "gnux help rm force"), command(st, "gnux help lsx -l"))
}

// lookupCommand finds a tool by either name, or a gnux command such as shell.
func lookupCommand(name string) (tool.Spec, bool) {
	if spec, ok := Lookup(name); ok {
		return spec, true
	}
	for _, spec := range Commands() {
		if spec.Name() == name {
			return spec, true
		}
	}
	return tool.Spec{}, false
}

// writeOverview is gnux's own help: what it is, how to run a tool, every
// command with its summary, and where the manuals are.
func writeOverview(w io.Writer, st tool.Style) {
	fmt.Fprintf(w, "%s %s [OPTION]... [TOOL [ARGUMENT]...]\n", "Usage:", common.AppName)
	fmt.Fprintln(w, "The GNU core utilities' interface on every platform, in one Go binary.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  gnux brings the GNU core utilities' options, messages and exit statuses to")
	fmt.Fprintln(w, "  Linux, macOS, Windows, containers and Go programs, with nothing else to")
	fmt.Fprintln(w, "  install. Each tool is named after its GNU counterpart with an x appended (rm")
	fmt.Fprintln(w, "  is rmx). Run a tool as a subcommand (gnux rmx -rf build), or link the binary")
	fmt.Fprintln(w, "  to the tool's name and run it directly (rmx -rf build). The GNU names work")
	fmt.Fprintln(w, "  too: gnux rm, or a link called rm.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  gnux is an independent project, not affiliated with or endorsed by the GNU")
	fmt.Fprintln(w, "  Project or the FSF. It complements GNU coreutils; it does not replace them.")

	tools := Tools()
	width, gnuWidth := len("completion"), 0
	for _, t := range Commands() {
		width = max(width, len(t.Name()))
		gnuWidth = max(gnuWidth, len(t.GNU))
	}

	fmt.Fprintf(w, "\n%s\n", heading(st, "Tools:"))
	for _, t := range tools {
		fmt.Fprintf(w, "  %s  %s  %s\n", command(st, pad(t.Name(), width)), dim(st, pad(t.GNU, gnuWidth)), t.Meta.Short)
	}

	fmt.Fprintf(w, "\n%s\n", heading(st, "Commands:"))
	for _, t := range Commands()[len(tools):] {
		fmt.Fprintf(w, "  %s  %s  %s\n", command(st, pad(t.Name(), width)), pad("", gnuWidth), t.Meta.Short)
	}
	fmt.Fprintf(w, "  %s  %s  %s\n", command(st, pad("help", width)), pad("", gnuWidth), "Show the full manual for a tool, or one of its options.")
	fmt.Fprintf(w, "  %s  %s  %s\n", command(st, pad("completion", width)), pad("", gnuWidth), "Print the completion script for bash, zsh, fish or powershell.")

	fmt.Fprintf(w, "\n%s\n", heading(st, "Options:"))
	fmt.Fprintln(w, "      --help      display this help and exit")
	fmt.Fprintln(w, "      --list      print the name of every tool, one per line")
	fmt.Fprintln(w, "      --version   output version information and exit")

	fmt.Fprintf(w, "\n%s\n", heading(st, "Examples:"))
	for _, e := range [][2]string{
		{"Run a tool through gnux", "gnux rmx -rf build/"},
		{"Make every tool a command of its own", `for t in $(gnux --list); do ln -s "$(command -v gnux)" "$HOME/bin/$t"; done`},
		{"The same script on every platform", "gnux shell -e -c 'mkdirx -p dist && cpx -r assets dist/'"},
		{"Read a tool's full manual", "gnux help rmx"},
		{"Read about one option", "gnux help lsx -l"},
		{"Enable tab completion in bash", "source <(gnux completion bash)"},
	} {
		fmt.Fprintf(w, "  %s\n  $ %s\n\n", dim(st, "# "+e[0]), command(st, e[1]))
	}

	fmt.Fprintf(w, "%s\n", heading(st, "More help:"))
	fmt.Fprintln(w, "  TOOL --help                 a tool's options and first examples")
	fmt.Fprintln(w, "  gnux help TOOL              the full manual for a tool")
	fmt.Fprintln(w, "  gnux help TOOL OPTION       one option in depth, with the examples that use it")
	fmt.Fprintf(w, "  Online: %s\n", tool.DocsURL)
}

// The overview's decorations, matching the per-tool help.
func heading(st tool.Style, s string) string { return paint(st, "1;36", s) }
func command(st tool.Style, s string) string { return paint(st, "32", s) }
func dim(st tool.Style, s string) string     { return paint(st, "2", s) }

func paint(st tool.Style, code, s string) string {
	if !st.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func pad(s string, n int) string {
	return s + strings.Repeat(" ", max(0, n-len(s)))
}
