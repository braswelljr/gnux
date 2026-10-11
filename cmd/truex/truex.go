// Package truex implements the truex command (GNU true). There is no engine
// package behind it: the tool does nothing but report a fixed exit status.
package truex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Name is the command name.
const Name = "truex"

// Spec describes the truex tool. Arguments are ignored rather than parsed, so
// an unknown option can never change the exit status.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "true",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for truex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    Name,
		SeeAlso: []string{"falsex", "yesx", "gnux shell"},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "always, whatever the arguments"},
		},
		Use:   Name + " [ignored command line arguments]",
		Short: "Do nothing, successfully.",
		Long: "truex exits with status 0 and does nothing else: it reads no input, writes " +
			"no output and ignores every argument it is given. The only exceptions are --help " +
			"and --version, which are honored when they are the sole argument. It exists so " +
			"scripts have a portable command whose result is always success.",
		Permissions: "Needs no permissions; it touches nothing.",
		UseCases: []string{
			"Writing an infinite loop: while truex; do ...; done.",
			"Forcing a command list to succeed, as in `might-fail || truex`.",
			"Standing in for a command that should be a no-op, such as a disabled hook.",
			"Keeping a 'gnux shell -e' script going past a step whose failure is acceptable.",
			"Testing how a script or CI step reacts to a command that always succeeds.",
			"Giving the always-true branch of a conditional a portable command to run.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Exit successfully:",
				Bash:        "truex",
				PowerShell:  "truex",
			},
			{
				Description: "Show the exit status:",
				Bash:        "truex; echox $?",
				PowerShell:  "truex; echox $LASTEXITCODE",
			},
			{
				Description: "Ignore every argument and still succeed:",
				Bash:        "truex --any thing at all; echox $?",
				PowerShell:  "truex --any thing at all; echox $LASTEXITCODE",
			},
			{
				Description: "Ignore the failure of a command:",
				Bash:        "rmx missing.txt || truex",
				PowerShell:  "rmx missing.txt; if (-not $?) { truex }",
			},
			{
				Description: "Chain a command that always runs:",
				Bash:        "truex && echox 'always runs'",
				PowerShell:  "truex; if ($LASTEXITCODE -eq 0) { echox 'always runs' }",
			},
			{
				Description: "Keep a gnux shell script with -e running past an allowed failure:",
				Bash:        "gnux shell -e -c 'rmx old.log || truex; echox survived'",
				PowerShell:  "gnux shell -e -c 'rmx old.log || truex; echox survived'",
			},
			{
				Description: "Loop until interrupted:",
				Bash:        "while truex; do date; sleepx 1; done",
				PowerShell:  "while ($true) { truex; if ($LASTEXITCODE -ne 0) { break }; Get-Date; sleepx 1 }",
			},
		},
	}
}

func run(_ *pflag.FlagSet, _ []string, _ core.Streams) error {
	return nil
}
