// Package whoamix wires the whoamix command (GNU whoami) to its engine.
package whoamix

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/whoami"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the whoamix tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "whoami",
		Run:  run,
	}
}

// Meta returns the documentation metadata for whoamix.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    whoami.Name,
		SeeAlso: []string{"hostnamex", "printenvx", "envx"},
		Use:     whoami.Name + " [OPTION]...",
		Short:   "Print the user name associated with the current effective user ID.",
		Long: "whoamix prints the name of the user the process runs as: the effective user, so " +
			"inside sudo it prints root. It takes no operands. On Windows, where the account " +
			"name is DOMAIN\\user, only the user part is printed, as GNU whoami does under " +
			"Cygwin and MSYS. If the user ID has no name, whoamix reports 'cannot find name for " +
			"user ID N' and exits 1.",
		Permissions: "Needs read access to the user database (/etc/passwd or the system's " +
			"directory service) to turn the user ID into a name.",
		UseCases: []string{
			"Refusing to run a script as root, or insisting on it.",
			"Naming per-user files, directories or caches in a portable way.",
			"Recording who ran a job in a log line.",
			"Checking which account a CI runner or container uses.",
			"Getting the user name on Windows without USERNAME and on Unix without USER.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print your user name:",
				Bash:        "whoamix",
				PowerShell:  "whoamix",
			},
			{
				Description: "Refuse to run as root:",
				Bash:        "[ \"$(whoamix)\" = root ] && { echox 'do not run as root' >&2; exit 1; }",
				PowerShell:  "if ((whoamix) -eq 'Administrator') { throw 'do not run as Administrator' }",
			},
			{
				Description: "Create a per-user scratch directory:",
				Bash:        "mkdirx -p \"/tmp/$(whoamix)/cache\"",
				PowerShell:  "mkdirx -p \"$env:TEMP/$(whoamix)/cache\"",
			},
			{
				Description: "Record who ran a job:",
				Bash:        "echox \"deployed by $(whoamix) on $(hostnamex)\" >> deploy.log",
				PowerShell:  "\"deployed by $(whoamix) on $(hostnamex)\" >> deploy.log",
			},
			{
				Description: "Compare with the user sudo runs as:",
				Bash:        "whoamix; sudo whoamix",
				PowerShell:  "whoamix",
			},
			{
				Description: "Print the user with a label:",
				Bash:        "printfx 'user: %s\\n' \"$(whoamix)\"",
				PowerShell:  "printfx 'user: %s\\n' (whoamix)",
			},
			{
				Description: "Check which user a container runs as:",
				Bash:        "docker run --rm gnux whoamix",
				PowerShell:  "docker run --rm gnux whoamix",
			},
			{
				Description: "Use the name in a file name:",
				Bash:        "cpx notes.md \"notes-$(whoamix).md\"",
				PowerShell:  "cpx notes.md \"notes-$(whoamix).md\"",
			},
			{
				Description: "Fail clearly when the user has no name (exit status 1):",
				Bash:        "whoamix || echox 'no user name for this ID'",
				PowerShell:  "whoamix; if ($LASTEXITCODE -ne 0) { 'no user name for this ID' }",
			},
			{
				Description: "Print the user in the portable gnux shell:",
				Bash:        "gnux shell -c 'whoamix'",
				PowerShell:  "gnux shell -c 'whoamix'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return whoami.New(whoami.Options{}, s).Run(args)
}
