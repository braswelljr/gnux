// Package rmdirx wires the rmdirx command (GNU rmdir) to its engine.
package rmdirx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rmdir"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the rmdirx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "rmdir",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for rmdirx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    rmdir.Name,
		SeeAlso: []string{"rmx", "mkdirx"},
		Use:     rmdir.Name + " [OPTION]... DIRECTORY...",
		Short:   "Remove the DIRECTORY(ies), if they are empty.",
		Long: "rmdirx removes each DIRECTORY, but only when it is empty: a directory that " +
			"still has entries is reported and left untouched, and so is anything that is not " +
			"a directory. That makes it the safe counterpart to a recursive delete - it can " +
			"never destroy file contents. -p also removes each parent named in the operand, " +
			"so 'rmdirx -p a/b/c' behaves like 'rmdirx a/b/c a/b a', stopping at the first " +
			"one that cannot be removed. --ignore-fail-on-non-empty silences the " +
			"not-empty failure (and only that one), and -v reports every directory processed.",
		Permissions: "Removing a directory needs write and search permission on its parent, " +
			"not on the directory itself. A symbolic link to a directory is not a directory: " +
			"it is refused rather than followed.",
		UseCases: []string{
			"Removing a directory while guaranteeing nothing inside it is lost.",
			"Pruning a chain of now-empty parents after moving files out with -p.",
			"Best-effort cleanup in scripts with --ignore-fail-on-non-empty.",
			"Checking that a directory really is empty: rmdirx fails, and says why, if it is not.",
			"Tidying up the empty leftovers of a build or an extraction with find.",
			"Undoing a 'mkdirx -p' of a tree that never received any files.",
			"Auditing a cleanup with -v, which names every directory removed.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove an empty directory:",
				Bash:        "rmdirx build",
				PowerShell:  "rmdirx build",
			},
			{
				Description: "Remove several empty directories at once:",
				Bash:        "rmdirx a b c",
				PowerShell:  "rmdirx a b c",
			},
			{
				Description: "Report each directory as it is removed:",
				Bash:        "rmdirx -v empty1 empty2",
				PowerShell:  "rmdirx -v empty1 empty2",
			},
			{
				Description: "Remove a directory and its now-empty parents, reporting each:",
				Bash:        "rmdirx -pv out/reports/2024",
				PowerShell:  "rmdirx -pv out\\reports\\2024",
			},
			{
				Description: "Remove a chain of parents and confirm the top one is gone:",
				Bash:        "rmdirx -p x/y/z && echox 'x is gone'",
				PowerShell:  "rmdirx -p x\\y\\z; if ($LASTEXITCODE -eq 0) { echox 'x is gone' }",
			},
			{
				Description: "Clean up whatever is empty without failing on the rest:",
				Bash:        "rmdirx --ignore-fail-on-non-empty cache logs tmp",
				PowerShell:  "rmdirx --ignore-fail-on-non-empty cache logs tmp",
			},
			{
				Description: "Prune parents as far as they are empty, stopping quietly at the first that is not:",
				Bash:        "rmdirx -p --ignore-fail-on-non-empty keep/sub",
				PowerShell:  "rmdirx -p --ignore-fail-on-non-empty keep\\sub",
			},
			{
				Description: "Refuse to remove a directory that still has files (exit status 1):",
				Bash:        "rmdirx logs; echox \"exit $?\"",
				PowerShell:  "rmdirx logs; echox \"exit $LASTEXITCODE\"",
			},
			{
				Description: "Use the failure as an emptiness test:",
				Bash:        "rmdirx logs 2>/dev/null || echox 'logs still has files'",
				PowerShell:  "rmdirx logs 2>$null; if ($LASTEXITCODE -ne 0) { echox 'logs still has files' }",
			},
			{
				Description: "Remove every empty directory below a tree, deepest first:",
				Bash:        "find tree -depth -type d -empty -exec rmdirx -v {} \\;",
				PowerShell:  "Get-ChildItem tree -Directory -Recurse | Sort-Object { $_.FullName.Length } -Descending | Where-Object { -not (Get-ChildItem $_.FullName -Force) } | ForEach-Object { rmdirx -v $_.FullName }",
			},
			{
				Description: "Create and then remove a temporary staging tree with the gnux shell:",
				Bash:        "gnux shell -c 'mkdirx -p staging/a && rmdirx -pv staging/a'",
				PowerShell:  "gnux shell -c 'mkdirx -p staging/a && rmdirx -pv staging/a'",
			},
		},
	}
}

// Register installs the rmdirx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("ignore-fail-on-non-empty", false,
		"ignore each failure that is solely because a directory is non-empty")
	fs.BoolP("parents", "p", false,
		"remove DIRECTORY and its ancestors; e.g., 'rmdirx -p a/b/c' is similar to 'rmdirx a/b/c a/b a'")
	fs.BoolP("verbose", "v", false, "output a diagnostic for every directory processed")
}

// buildOptions folds the parsed flags into rmdir.Options.
func buildOptions(fs *pflag.FlagSet) rmdir.Options {
	var o rmdir.Options
	o.IgnoreNonEmpty, _ = fs.GetBool("ignore-fail-on-non-empty")
	o.Parents, _ = fs.GetBool("parents")
	o.Verbose, _ = fs.GetBool("verbose")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return rmdir.New(buildOptions(fs), s).Run(args)
}
