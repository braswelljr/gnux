// Package idx wires the idx command (GNU id) to its engine.
package idx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/id"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the idx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "id",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for idx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    id.Name,
		SeeAlso: []string{"whoamix", "chownx", "chgrpx", "whox"},
		Use:     id.Name + " [OPTION]... [USER]...",
		Short:   "Print real and effective user and group IDs.",
		Long: "idx prints the user and group IDs of the current process, or of each USER (a " +
			"name or a numeric ID), in GNU's default format: 'uid=1000(alice) " +
			"gid=1000(alice) groups=1000(alice),27(sudo)', with euid= and egid= added when " +
			"they differ from the real IDs. -u, -g and -G print only the effective user ID, " +
			"the effective group ID or every group ID; -n prints names instead of numbers and " +
			"-r the real IDs instead of the effective ones, with one of those three; -z ends " +
			"each entry with NUL. -a is accepted and ignored, as in GNU. -Z (SELinux " +
			"context) is not supported and fails, as GNU id does on a kernel without " +
			"SELinux. An unknown USER is reported ('no such user') and the remaining USERs " +
			"are still printed. On Windows the IDs are security identifiers (SIDs) and the " +
			"names come from the account database.",
		Permissions: "Needs no permissions; names are read from the system's user and group " +
			"databases.",
		UseCases: []string{
			"Checking which user and groups a script or service runs as.",
			"Getting the numeric UID and GID to pass to chownx or a container.",
			"Checking whether a user belongs to a group, such as docker or sudo.",
			"Refusing to run a script as root by testing idx -u.",
			"Listing a user's groups by name.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print your user and group IDs:",
				Bash:        "idx",
				PowerShell:  "idx",
			},
			{
				Description: "Print another user's IDs:",
				Bash:        "idx alice",
				PowerShell:  "idx alice",
			},
			{
				Description: "Print only the effective user ID:",
				Bash:        "idx -u",
				PowerShell:  "idx -u",
			},
			{
				Description: "Print your user name:",
				Bash:        "idx -un",
				PowerShell:  "idx -un",
			},
			{
				Description: "Print the effective group ID:",
				Bash:        "idx -g",
				PowerShell:  "idx -g",
			},
			{
				Description: "List your groups by name:",
				Bash:        "idx -Gn",
				PowerShell:  "idx -Gn",
			},
			{
				Description: "Print the real user ID rather than the effective one:",
				Bash:        "idx -ru",
				PowerShell:  "idx -ru",
			},
			{
				Description: "Refuse to run as root:",
				Bash:        "[ \"$(idx -u)\" -eq 0 ] && { echox 'do not run as root' >&2; exit 1; }",
				PowerShell:  "if ((idx -u) -eq '0') { Write-Error 'do not run as root'; exit 1 }",
			},
			{
				Description: "Check membership of a group:",
				Bash:        "idx -Gn | trx ' ' '\\n' | grep -qx docker && echox member",
				PowerShell:  "if ((idx -Gn) -split ' ' -contains 'docker') { 'member' }",
			},
			{
				Description: "Pass your IDs to a container:",
				Bash:        "docker run --user \"$(idx -u):$(idx -g)\" image",
				PowerShell:  "docker run --user \"$(idx -u):$(idx -g)\" image",
			},
			{
				Description: "Print IDs in the portable gnux shell:",
				Bash:        "gnux shell -c 'idx -un && idx -Gn'",
				PowerShell:  "gnux shell -c 'idx -un && idx -Gn'",
			},
		},
	}
}

// Register installs the idx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("all", "a", false, "ignore, for compatibility with other versions")
	fs.BoolP("context", "Z", false, "print only the security context of the process (not supported)")
	fs.BoolP("group", "g", false, "print only the effective group ID")
	fs.BoolP("groups", "G", false, "print all group IDs")
	fs.BoolP("name", "n", false, "print a name instead of a number, for -u,-g,-G")
	fs.BoolP("real", "r", false, "print the real ID instead of the effective ID, with -u,-g,-G")
	fs.BoolP("user", "u", false, "print only the effective user ID")
	fs.BoolP("zero", "z", false, "delimit entries with NUL characters, not whitespace; not permitted in default format")
}

// buildOptions folds the parsed flags into id.Options.
func buildOptions(fs *pflag.FlagSet) id.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	return id.Options{
		User:   on("user"),
		Group:  on("group"),
		Groups: on("groups"),
		Name:   on("name"),
		Real:   on("real"),
		Zero:   on("zero"),
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	if ctx, _ := fs.GetBool("context"); ctx {
		d := core.NewDiag(id.Name, s)
		d.Errorf("--context (-Z) works only on an SELinux-enabled kernel")
		return d.Result()
	}
	return id.New(buildOptions(fs), s).Run(args)
}
