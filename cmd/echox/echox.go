// Package echox wires the echox command (GNU echo) to its engine.
package echox

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/echo"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the echox tool. Its arguments are not GNU-style options -
// "-x" and "--" are text to print - so flag parsing is left to the engine.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "echo",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for echox.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    echo.Name,
		SeeAlso: []string{"catx", "yesx", "gnux shell"},
		Use:     echo.Name + " [SHORT-OPTION]... [STRING]...",
		Short:   "Display a line of text.",
		Long: "echox writes each STRING to standard output, separated by single spaces and " +
			"followed by a newline. -n omits the trailing newline, -e enables interpretation " +
			"of backslash escapes (\\\\, \\a, \\b, \\c, \\e, \\f, \\n, \\r, \\t, \\v, \\0NNN, " +
			"\\xHH) and -E (the default) disables it; \\c stops all further output, the " +
			"newline included. As in GNU echo, options are recognized only as leading " +
			"arguments made up solely of those letters (-n, -ne, -En, ...): the first argument " +
			"that is anything else, such as -x or --, ends option parsing and is printed " +
			"literally. --help and --version are honored only when they are the sole argument.",
		Permissions: "Needs no filesystem permissions; it only writes to standard output.",
		UseCases: []string{
			"Printing messages and progress from scripts.",
			"Emitting text without a trailing newline with -n, for prompts or piping.",
			"Producing tabs, newlines or arbitrary bytes with -e, identically on every platform.",
			"Creating or appending to small files with redirection, such as config stubs or notes.",
			"Feeding a known string into another tool (wcx, headx, catx) to test or demonstrate it.",
			"Reporting the result of a previous command, such as its exit status.",
			"Printing text that starts with a dash or contains backslashes verbatim, which is awkward with the shell's own echo.",
			"Writing the same output in bash, PowerShell and the gnux shell, where built-in echo commands differ.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print a line of text:",
				Bash:        "echox Hello, world",
				PowerShell:  "echox Hello, world",
			},
			{
				Description: "Print without a trailing newline, as for a prompt:",
				Bash:        "echox -n 'Continue? '",
				PowerShell:  "echox -n 'Continue? '",
			},
			{
				Description: "Interpret backslash escapes to print a small table:",
				Bash:        `echox -e 'name\tsize\nrmx\t2M'`,
				PowerShell:  `echox -e 'name\tsize\nrmx\t2M'`,
			},
			{
				Description: "Combine options: interpret escapes and omit the final newline:",
				Bash:        `echox -ne 'no newline here\n'`,
				PowerShell:  `echox -ne 'no newline here\n'`,
			},
			{
				Description: "Print backslashes literally (the default, made explicit with -E):",
				Bash:        `echox -E 'C:\new\table'`,
				PowerShell:  `echox -E 'C:\new\table'`,
			},
			{
				Description: "Stop all further output, newline included, at \\c:",
				Bash:        `echox -e 'first\csecond'`,
				PowerShell:  `echox -e 'first\csecond'`,
			},
			{
				Description: "Write bytes by octal and hexadecimal code (prints ABC):",
				Bash:        `echox -e '\x41\0102C'`,
				PowerShell:  `echox -e '\x41\0102C'`,
			},
			{
				Description: "Print text that looks like an option; anything other than option letters is literal:",
				Bash:        "echox -- -n",
				PowerShell:  "echox -- -n",
			},
			{
				Description: "Create a file, then append a line to it:",
				Bash:        `echox -e 'line one\nline two' > notes.txt && echox 'line three' >> notes.txt`,
				PowerShell:  `echox -e 'line one\nline two' | Out-File notes.txt; echox 'line three' | Add-Content notes.txt`,
			},
			{
				Description: "Count the characters in a string, newline included:",
				Bash:        "echox \"home is $HOME\" | wcx -c",
				PowerShell:  "echox \"home is $HOME\" | wcx -c",
			},
			{
				Description: "Feed several lines to another tool:",
				Bash:        `echox -e 'b\na\nc' | headx -n 2`,
				PowerShell:  `echox -e 'b\na\nc' | headx -n 2`,
			},
			{
				Description: "Report the exit status of the previous command:",
				Bash:        "falsex; echox \"status: $?\"",
				PowerShell:  "falsex; echox \"status: $LASTEXITCODE\"",
			},
			{
				Description: "Show an escape sequence's bytes made visible by catx -v:",
				Bash:        `echox -e '\e[1mbold\e[0m' | catx -v`,
				PowerShell:  `echox -e '\e[1mbold\e[0m' | catx -v`,
			},
			{
				Description: "Use a variable in the gnux shell, identically on every platform:",
				Bash:        "gnux shell -c 'NAME=gnux; echox \"hello from $NAME\" > greet.txt; catx greet.txt'",
				PowerShell:  "gnux shell -c 'NAME=gnux; echox \"hello from $NAME\" > greet.txt; catx greet.txt'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	opts, operands := echo.Parse(args)
	return echo.New(opts, s).Run(operands)
}
