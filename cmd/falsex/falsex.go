// Package falsex implements the falsex command (GNU false). There is no engine
// package behind it: the tool does nothing but report a fixed exit status.
package falsex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Name is the command name.
const Name = "falsex"

// Spec describes the falsex tool. Arguments are ignored rather than parsed, so
// an unknown option can never change the exit status.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "false",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for falsex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    Name,
		SeeAlso: []string{"truex", "gnux shell"},
		ExitStatus: []docmeta.Status{
			{Code: "1", Meaning: "always, whatever the arguments"},
		},
		Use:   Name + " [ignored command line arguments]",
		Short: "Do nothing, unsuccessfully.",
		Long: "falsex exits with status 1 and does nothing else: it reads no input, writes " +
			"no output and ignores every argument it is given. The only exceptions are --help " +
			"and --version, which are honored when they are the sole argument. It exists so " +
			"scripts have a portable command whose result is always failure.",
		Permissions: "Needs no permissions; it touches nothing.",
		UseCases: []string{
			"Testing how a script or CI step reacts to a failing command.",
			"Disabling a hook or account shell by pointing it at a command that always fails.",
			"Terminating an until loop condition: until falsex; do ...; done never stops on its own.",
			"Checking that 'gnux shell -e' aborts a script at the first failure.",
			"Exercising the else branch of a conditional or the right side of ||.",
			"Standing in for a command that is expected to fail in examples and tests.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Exit with status 1:",
				Bash:        "falsex",
				PowerShell:  "falsex",
			},
			{
				Description: "Show the exit status:",
				Bash:        "falsex; echox $?",
				PowerShell:  "falsex; echox $LASTEXITCODE",
			},
			{
				Description: "Run a fallback when a command fails:",
				Bash:        "falsex || echox 'fallback ran'",
				PowerShell:  "falsex; if ($LASTEXITCODE -ne 0) { echox 'fallback ran' }",
			},
			{
				Description: "Skip the command after &&:",
				Bash:        "falsex && echox never; echox \"status: $?\"",
				PowerShell:  "falsex; if ($LASTEXITCODE -eq 0) { echox never }; echox \"status: $LASTEXITCODE\"",
			},
			{
				Description: "Take the else branch of a conditional:",
				Bash:        "if falsex; then echox yes; else echox no; fi",
				PowerShell:  "falsex; if ($LASTEXITCODE -eq 0) { echox yes } else { echox no }",
			},
			{
				Description: "See gnux shell -e stop at the first failure:",
				Bash:        "gnux shell -e -c 'echox before; falsex; echox after'; echox \"shell exited $?\"",
				PowerShell:  "gnux shell -e -c 'echox before; falsex; echox after'; echox \"shell exited $LASTEXITCODE\"",
			},
			{
				Description: "Pass the status through envx, which exits with the command's status:",
				Bash:        "envx falsex; echox $?",
				PowerShell:  "envx falsex; echox $LASTEXITCODE",
			},
		},
	}
}

func run(_ *pflag.FlagSet, _ []string, _ core.Streams) error {
	return &core.ExitError{Code: 1}
}
