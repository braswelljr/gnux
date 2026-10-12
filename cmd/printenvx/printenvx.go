// Package printenvx wires the printenvx command (GNU printenv) to its engine.
package printenvx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printenv"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the printenvx tool. Like GNU printenv, a command-line
// mistake exits 2, so that it cannot be mistaken for "not set" (1).
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "printenv",
		Flags:     Register,
		Run:       run,
		UsageExit: 2,
	}
}

// Meta returns the documentation metadata for printenvx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    printenv.Name,
		SeeAlso: []string{"envx", "echox"},
		Use:     printenv.Name + " [OPTION]... [VARIABLE]...",
		Short:   "Print all or part of the environment.",
		Long: "printenvx prints the value of each environment VARIABLE, one per line. With no " +
			"VARIABLE it prints the whole environment as NAME=value lines. A VARIABLE that is not " +
			"set prints nothing, not even an empty line, and makes the exit status 1, with no " +
			"message; one that is set to the empty string prints an empty line. Names are " +
			"matched exactly, including their case, on every platform. -0 ends each item with " +
			"NUL instead of a newline, so values containing newlines survive.",
		Permissions: "Needs no permissions. Reads only the process's own environment.",
		UseCases: []string{
			"Checking what value a variable has in the current environment.",
			"Testing in a script whether a variable is set, from the exit status.",
			"Dumping the environment into a CI log for debugging.",
			"Comparing the environment before and after a change.",
			"Reading a variable on Windows with the same command as on Linux.",
			"Feeding the environment to another tool safely with -0.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "every VARIABLE was set (or none was named)"},
			{Code: "1", Meaning: "at least one VARIABLE was not set"},
			{Code: "2", Meaning: "a command-line mistake"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the whole environment:",
				Bash:        "printenvx",
				PowerShell:  "printenvx",
			},
			{
				Description: "Print one variable:",
				Bash:        "printenvx HOME",
				PowerShell:  "printenvx USERPROFILE",
			},
			{
				Description: "Print several variables, one per line:",
				Bash:        "printenvx USER SHELL",
				PowerShell:  "printenvx USERNAME COMSPEC",
			},
			{
				Description: "Test whether a variable is set:",
				Bash:        "printenvx CI > /dev/null && echox 'running in CI'",
				PowerShell:  "printenvx CI > $null; if ($LASTEXITCODE -eq 0) { 'running in CI' }",
			},
			{
				Description: "Show the search path one directory per line:",
				Bash:        "printenvx PATH | trx ':' '\\n'",
				PowerShell:  "printenvx PATH | trx ';' '\\n'",
			},
			{
				Description: "Find variables by name:",
				Bash:        "printenvx | sortx | headx -n 20",
				PowerShell:  "printenvx | sortx | headx -n 20",
			},
			{
				Description: "Print a variable set for one command:",
				Bash:        "envx GREETING=hello printenvx GREETING",
				PowerShell:  "envx GREETING=hello printenvx GREETING",
			},
			{
				Description: "Print the environment NUL-separated:",
				Bash:        "printenvx -0 | trx '\\0' '\\n' | wcx -l",
				PowerShell:  "printenvx -0 | trx '\\0' '\\n' | wcx -l",
			},
			{
				Description: "Save the environment for a bug report:",
				Bash:        "printenvx | sortx > env.txt",
				PowerShell:  "printenvx | sortx > env.txt",
			},
			{
				Description: "Read a variable in the portable gnux shell:",
				Bash:        "gnux shell -c 'printenvx HOME'",
				PowerShell:  "gnux shell -c 'printenvx USERPROFILE'",
			},
		},
	}
}

// Register installs the printenvx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("null", "0", false, "end each output line with NUL, not newline")
}

// buildOptions folds the parsed flags into printenv.Options.
func buildOptions(fs *pflag.FlagSet) printenv.Options {
	null, _ := fs.GetBool("null")
	return printenv.Options{Null: null}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return printenv.New(buildOptions(fs), s).Run(args)
}
