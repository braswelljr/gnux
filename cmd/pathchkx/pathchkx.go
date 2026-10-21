// Package pathchkx wires the pathchkx command (GNU pathchk) to its engine.
package pathchkx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pathchk"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the pathchkx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "pathchk",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for pathchkx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    pathchk.Name,
		SeeAlso: []string{"realpathx", "mkdirx", "touchx"},
		Use:     pathchk.Name + " [OPTION]... NAME...",
		Short:   "Check whether file names are valid or portable.",
		Long: "pathchkx checks each NAME and reports the ones that could not be created. By " +
			"default it checks against this system: a name whose directory cannot be searched " +
			"or is not a directory fails, as does a name or component longer than the file " +
			"system holding it allows. NAME itself need not exist. With -p it checks against " +
			"the POSIX minimum instead: at most 255 bytes in all, 14 in a component, and only " +
			"the portable characters A-Z, a-z, 0-9, '.', '_', '-' and '/'. With -P it also " +
			"rejects an empty name and a component that starts with '-'; --portability is " +
			"-p and -P together. Every NAME is checked even after one fails; nothing is " +
			"printed for a valid name. On Windows, which has no pathconf, the limits are " +
			"NTFS's: 255 characters per component and 259 in all.",
		Permissions: "Needs search permission on the existing directories in each NAME (not with " +
			"-p, which never looks at the disk). Nothing is ever created or modified.",
		UseCases: []string{
			"Rejecting file names that will not survive a copy to another system before creating them.",
			"Checking user-supplied names in a script before writing to them.",
			"Enforcing portable names in a repository in CI.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Check that a name can be created here:",
				Bash:        "pathchkx reports/2026/summary.txt && echox ok",
				PowerShell:  "pathchkx reports/2026/summary.txt; if ($LASTEXITCODE -eq 0) { 'ok' }",
			},
			{
				Description: "Catch a name that will not travel to other systems:",
				Bash:        "pathchkx -p 'Quarterly report.pdf'",
				PowerShell:  "pathchkx -p 'Quarterly report.pdf'",
				Output:      "pathchkx: non-portable character ' ' in file name 'Quarterly report.pdf'",
			},
			{
				Description: "Check every file in a repository for portable names in CI:",
				Bash:        "findx . -path ./.git -prune -o -print | xargsx pathchkx --portability",
				PowerShell:  "findx . -path ./.git -prune -o -print | xargsx pathchkx --portability",
			},
			{
				Description: "Reject a name that a command would read as an option:",
				Bash:        "pathchkx -P \"$name\" || exit 1",
				PowerShell:  "pathchkx -P $name; if ($LASTEXITCODE) { exit 1 }",
			},
		},
	}
}

// Register installs the pathchkx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("basic", "p", false, "check for most POSIX systems")
	fs.BoolP("extra", "P", false, "check for empty names and leading \"-\"")
	fs.Bool("portability", false, "check for all POSIX systems (equivalent to -p -P)")
	tool.OwnNames(fs, "basic", "extra")
}

// buildOptions folds the parsed flags into pathchk.Options.
func buildOptions(fs *pflag.FlagSet) pathchk.Options {
	p, _ := fs.GetBool("basic")
	extra, _ := fs.GetBool("extra")
	both, _ := fs.GetBool("portability")
	return pathchk.Options{Portable: p || both, Extra: extra || both}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return pathchk.New(buildOptions(fs), s).Run(args)
}
