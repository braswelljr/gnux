// Package sleepx wires the sleepx command (GNU sleep) to its engine.
package sleepx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sleep"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the sleepx tool. It has no flags beyond --help and --version.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "sleep",
		Run:  run,
	}
}

// Meta returns the documentation metadata for sleepx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    sleep.Name,
		SeeAlso: []string{"envx", "yesx", "gnux shell"},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "the full interval elapsed"},
			{Code: "1", Meaning: "an invalid interval or a command-line mistake"},
		},
		Use:   sleep.Name + " NUMBER[SUFFIX]...",
		Short: "Delay for a specified amount of time.",
		Long: "sleepx pauses for NUMBER seconds. SUFFIX may be 's' for seconds (the default), " +
			"'m' for minutes, 'h' for hours or 'd' for days. NUMBER need not be an integer: " +
			"fractions (0.5), exponents (1e3) and hexadecimal floating-point values are " +
			"accepted, as is 'infinity' (or 'inf') to pause until interrupted. Given two or " +
			"more arguments, sleepx pauses for the sum of their values. Every operand is " +
			"checked before the pause begins, so an invalid interval fails immediately.",
		Permissions: "Needs no permissions; it touches nothing.",
		UseCases: []string{
			"Pacing a retry or polling loop in a script.",
			"Giving a service time to start before the next step runs.",
			"Writing delays that behave the same in every shell, fractional seconds included.",
			"Rate-limiting a loop that calls an API or writes to a slow device.",
			"Waiting for a file or flag to appear before continuing.",
			"Keeping a container or background job alive with sleepx infinity.",
			"Spacing out progress messages or demo output so it can be read.",
			"Simulating a slow command when testing timeouts, spinners or pipelines.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Pause for half a second:",
				Bash:        "sleepx 0.5",
				PowerShell:  "sleepx 0.5",
			},
			{
				Description: "Pause for one and a half minutes:",
				Bash:        "sleepx 1m 30s",
				PowerShell:  "sleepx 1m 30s",
			},
			{
				Description: "Pause for two hours:",
				Bash:        "sleepx 2h",
				PowerShell:  "sleepx 2h",
			},
			{
				Description: "Use a fraction of a larger unit (6 minutes):",
				Bash:        "sleepx 0.1h",
				PowerShell:  "sleepx 0.1h",
			},
			{
				Description: "Write the interval in scientific or hexadecimal notation:",
				Bash:        "sleepx 1e-1 0x0.1p0",
				PowerShell:  "sleepx 1e-1 0x0.1p0",
			},
			{
				Description: "Run the next command only after the pause:",
				Bash:        "sleepx 2 && echox ready",
				PowerShell:  "sleepx 2; if ($LASTEXITCODE -eq 0) { echox ready }",
			},
			{
				Description: "Print a line every second, three times:",
				Bash:        "for i in 1 2 3; do echox \"tick $i\"; sleepx 1; done",
				PowerShell:  "foreach ($i in 1..3) { echox \"tick $i\"; sleepx 1 }",
			},
			{
				Description: "Retry a command every two seconds until it succeeds:",
				Bash:        "until curl -fs localhost:8080/health; do sleepx 2; done",
				PowerShell:  "while (-not (Test-NetConnection localhost -Port 8080 -InformationLevel Quiet)) { sleepx 2 }",
			},
			{
				Description: "Wait until a file appears:",
				Bash:        "until [ -e ready.flag ]; do sleepx 0.5; done; echox found",
				PowerShell:  "while (-not (Test-Path ready.flag)) { sleepx 0.5 }; echox found",
			},
			{
				Description: "Run a slow stand-in command in the background:",
				Bash:        "sleepx 30 & echox \"started $!\"",
				PowerShell:  "Start-Job { sleepx 30 }",
			},
			{
				Description: "See an invalid interval rejected before any pause:",
				Bash:        "sleepx 10x; echox \"status: $?\"",
				PowerShell:  "sleepx 10x; echox \"status: $LASTEXITCODE\"",
			},
			{
				Description: "Space out the steps of a gnux shell script:",
				Bash:        "gnux shell -c 'echox start; sleepx 1; echox done'",
				PowerShell:  "gnux shell -c 'echox start; sleepx 1; echox done'",
			},
			{
				Description: "Block until interrupted:",
				Bash:        "sleepx infinity",
				PowerShell:  "sleepx infinity",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return sleep.New(s).Run(args)
}
