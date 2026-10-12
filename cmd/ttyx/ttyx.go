// Package ttyx wires the ttyx command (GNU tty) to its engine.
package ttyx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tty"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the ttyx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "tty",
		Flags:     Register,
		Run:       run,
		UsageExit: tty.ExitUsage,
	}
}

// Meta returns the documentation metadata for ttyx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tty.Name,
		SeeAlso: []string{"whox", "nohupx", "idx"},
		Use:     tty.Name + " [OPTION]...",
		Short:   "Print the file name of the terminal connected to standard input.",
		Long: "ttyx prints the file name of the terminal on standard input, such as " +
			"/dev/pts/3, or 'not a tty' when standard input is a pipe, a file or nothing, " +
			"and then exits with status 1. -s prints nothing, so only the exit status " +
			"tells; GNU deprecates it in favor of 'test -t 0', which gnux has as testx -t " +
			"0. The name is found without the C library: through /proc on Linux, and on " +
			"macOS and the BSDs by finding the device in /dev with the same device number, " +
			"as ttyname(3) does. Windows has no terminal file names, so a console prints as " +
			"CON (its device name) instead of failing. The exit status is 0 when standard " +
			"input is a terminal, 1 when it is not, 2 for a command-line mistake and 3 when " +
			"the name could not be written.",
		Permissions: "Needs no permissions.",
		UseCases: []string{
			"Making a script behave differently when run by hand than from cron or a pipe.",
			"Finding which terminal device a session uses, to write to it or check its permissions.",
			"Refusing to prompt for a password when there is no terminal to type it on.",
			"Logging which terminal a command was run from.",
			"Checking that a CI job really has no terminal attached.",
		},
		Examples: []docmeta.Example{
			{
				Description: "See which terminal device your shell is using:",
				Bash:        "ttyx",
				PowerShell:  "ttyx",
				Output:      "/dev/pts/3",
			},
			{
				Description: "See what a command reading from a pipe gets:",
				Bash:        "echox | ttyx",
				PowerShell:  "echox | ttyx",
				Output:      "not a tty",
			},
			{
				Description: "Only prompt for confirmation when someone is at the keyboard:",
				Bash:        "if ttyx -s; then read -p 'Deploy to production? ' ok; else ok=yes; fi",
				PowerShell:  "if (ttyx -s; $LASTEXITCODE -eq 0) { $ok = Read-Host 'Deploy to production?' } else { $ok = 'yes' }",
			},
			{
				Description: "Refuse to ask for a password in a cron job or CI run:",
				Bash:        "ttyx -s || { echox 'no terminal: set DB_PASSWORD instead' >&2; exit 1; }",
				PowerShell:  "ttyx -s; if ($LASTEXITCODE -ne 0) { Write-Error 'no terminal: set DB_PASSWORD instead'; exit 1 }",
			},
			{
				Description: "Record the terminal in an audit log next to the user and time:",
				Bash:        "echox \"$(datex -Iseconds) $(whoamix) $(ttyx)\" >> audit.log",
				PowerShell:  "\"$(datex -Iseconds) $(whoamix) $(ttyx)\" >> audit.log",
			},
			{
				Description: "Check who may write to your terminal (as mesg would):",
				Bash:        "lsx -l \"$(ttyx)\"",
				PowerShell:  "ttyx",
			},
			{
				Description: "Show progress only when a person is watching:",
				Bash:        "ttyx -s && echox 'Copying 4.2 GB, this takes a few minutes...'",
				PowerShell:  "ttyx -s; if ($LASTEXITCODE -eq 0) { 'Copying 4.2 GB, this takes a few minutes...' }",
			},
			{
				Description: "Confirm a CI job has no terminal attached:",
				Bash:        "ttyx || echox 'running unattended'",
				PowerShell:  "ttyx; if ($LASTEXITCODE -ne 0) { 'running unattended' }",
				Output:      "not a tty\nrunning unattended",
			},
			{
				Description: "Get the exit status alone, for a test in a script:",
				Bash:        "ttyx -s; echox $?",
				PowerShell:  "ttyx -s; $LASTEXITCODE",
				Output:      "0",
			},
			{
				Description: "Compare with the portable equivalent GNU recommends:",
				Bash:        "testx -t 0 && echox interactive",
				PowerShell:  "testx -t 0; if ($LASTEXITCODE -eq 0) { 'interactive' }",
			},
			{
				Description: "Check for a terminal in the portable gnux shell:",
				Bash:        "gnux shell -c 'ttyx -s && echox interactive || echox batch'",
				PowerShell:  "gnux shell -c 'ttyx -s && echox interactive || echox batch'",
			},
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "standard input is a terminal"},
			{Code: "1", Meaning: "standard input is not a terminal"},
			{Code: "2", Meaning: "a command-line mistake"},
			{Code: "3", Meaning: "a write error"},
		},
	}
}

// Register installs the ttyx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("silent", "s", false, "print nothing, only return an exit status")
	fs.Bool("quiet", false, "same as --silent")
}

// buildOptions folds the parsed flags into tty.Options.
func buildOptions(fs *pflag.FlagSet) tty.Options {
	silent, _ := fs.GetBool("silent")
	quiet, _ := fs.GetBool("quiet")
	return tty.Options{Silent: silent || quiet}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return tty.New(buildOptions(fs), s).Run(args)
}
