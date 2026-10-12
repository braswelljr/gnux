// Package hostnamex wires the hostnamex command (GNU coreutils hostname) to its
// engine.
package hostnamex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hostname"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the hostnamex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "hostname",
		Run:  run,
	}
}

// Meta returns the documentation metadata for hostnamex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    hostname.Name,
		SeeAlso: []string{"unamex", "whoamix"},
		Use:     hostname.Name + " [NAME]",
		Short:   "Print the name of the current host system.",
		Long: "hostnamex prints the name of the current host, as the operating system reports " +
			"it. GNU coreutils hostname also sets the name when given NAME (with " +
			"sethostname(2), as root); gnux never changes system settings, so hostnamex " +
			"refuses NAME with 'cannot set name to NAME: Operation not supported' and exits 1. " +
			"Use the system's own tool (hostnamectl, scutil, Rename-Computer) to set it. Note " +
			"that the hostname program shipped by many Linux distributions is not the " +
			"coreutils one and has options such as -f and -I; hostnamex has none of them.",
		Permissions: "Needs no permissions to print the name.",
		UseCases: []string{
			"Recording which machine produced a log line or a build artifact.",
			"Naming per-host files or directories.",
			"Branching a script on the machine it runs on.",
			"Checking a container's or a CI runner's host name.",
			"Getting the host name the same way on Linux, macOS and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the host name:",
				Bash:        "hostnamex",
				PowerShell:  "hostnamex",
			},
			{
				Description: "Tag a log line with the host:",
				Bash:        "echox \"$(hostnamex): backup done\" >> backup.log",
				PowerShell:  "\"$(hostnamex): backup done\" >> backup.log",
			},
			{
				Description: "Name an archive after the machine:",
				Bash:        "cpx backup.tar \"backup-$(hostnamex).tar\"",
				PowerShell:  "cpx backup.tar \"backup-$(hostnamex).tar\"",
			},
			{
				Description: "Run a step only on one machine:",
				Bash:        "[ \"$(hostnamex)\" = build-01 ] && make release",
				PowerShell:  "if ((hostnamex) -eq 'build-01') { make release }",
			},
			{
				Description: "Print the user and the host together:",
				Bash:        "echox \"$(whoamix)@$(hostnamex)\"",
				PowerShell:  "\"$(whoamix)@$(hostnamex)\"",
			},
			{
				Description: "Compare with the network node name unamex reports:",
				Bash:        "hostnamex; unamex -n",
				PowerShell:  "hostnamex; unamex -n",
			},
			{
				Description: "Setting the name is refused (exit status 1):",
				Bash:        "hostnamex new-name || echox 'use hostnamectl instead'",
				PowerShell:  "hostnamex new-name; if ($LASTEXITCODE -ne 0) { 'use Rename-Computer instead' }",
			},
			{
				Description: "Print the host name in upper case:",
				Bash:        "hostnamex | trx a-z A-Z",
				PowerShell:  "hostnamex | trx a-z A-Z",
			},
			{
				Description: "Create a per-host directory:",
				Bash:        "mkdirx -p \"logs/$(hostnamex)\"",
				PowerShell:  "mkdirx -p \"logs/$(hostnamex)\"",
			},
			{
				Description: "Print the host name in the portable gnux shell:",
				Bash:        "gnux shell -c 'hostnamex'",
				PowerShell:  "gnux shell -c 'hostnamex'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return hostname.New(hostname.Options{}, s).Run(args)
}
