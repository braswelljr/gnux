// Package linkx wires the linkx command (GNU link) to its engine.
package linkx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/link"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the linkx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "link",
		Run:  run,
	}
}

// Meta returns the documentation metadata for linkx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    link.Name,
		SeeAlso: []string{"lnx", "unlinkx", "lsx"},
		Use:     link.Name + " FILE1 FILE2",
		Short:   "Call the link function to create a hard link to a file.",
		Long: "linkx creates FILE2 as a hard link to the existing FILE1: a second name for the " +
			"same file, sharing its contents and metadata. It is the bare link(2) system call, " +
			"with no options: it never overwrites FILE2, never makes a symbolic link, and never " +
			"places the link inside a directory. Exactly two operands are required. lnx does " +
			"the same with options; linkx is for scripts that want exactly one system call and " +
			"its error.",
		Permissions: "Needs write and search permission on the directory that will hold " +
			"FILE2. FILE1 and FILE2 must be on the same filesystem, and most systems refuse a " +
			"hard link to a directory. On Windows, hard links need an NTFS volume.",
		UseCases: []string{
			"Giving a file a second name without copying its data.",
			"Creating a hard link in a script that must fail rather than overwrite.",
			"Keeping a snapshot name for a file that is about to be renamed or replaced.",
			"Testing whether two paths are on the same filesystem (the link fails if not).",
			"Implementing a simple lock: link succeeds for exactly one of several racing processes.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Give a file a second name:",
				Bash:        "linkx report.txt report-final.txt",
				PowerShell:  "linkx report.txt report-final.txt",
			},
			{
				Description: "Refuse to overwrite an existing name (exit status 1):",
				Bash:        "linkx a.txt existing.txt || echox 'already there'",
				PowerShell:  "linkx a.txt existing.txt; if ($LASTEXITCODE -ne 0) { 'already there' }",
			},
			{
				Description: "Check that both names are the same file (same inode in the first column):",
				Bash:        "linkx data.bin data.bak && lsx -i data.bin data.bak",
				PowerShell:  "linkx data.bin data.bak; lsx -i data.bin data.bak",
			},
			{
				Description: "Keep the old contents reachable before replacing a file:",
				Bash:        "linkx config.yaml config.yaml.old && cpx new.yaml config.yaml",
				PowerShell:  "linkx config.yaml config.yaml.old; cpx new.yaml config.yaml",
			},
			{
				Description: "Take a lock that only one process can get:",
				Bash:        "touchx lock.$$ && linkx lock.$$ lock && echox 'got the lock'",
				PowerShell:  "touchx lock.$PID; linkx lock.$PID lock; if ($?) { 'got the lock' }",
			},
			{
				Description: "Release that lock:",
				Bash:        "unlinkx lock",
				PowerShell:  "unlinkx lock",
			},
			{
				Description: "Link a file into another directory (FILE2 names the link, not the directory):",
				Bash:        "linkx build/app dist/app",
				PowerShell:  "linkx build/app dist/app",
			},
			{
				Description: "Count a file's names after linking it:",
				Bash:        "linkx a b && lsx -l a",
				PowerShell:  "linkx a b; lsx -l a",
			},
			{
				Description: "Link a file whose name starts with a dash:",
				Bash:        "linkx -- -file copy",
				PowerShell:  "linkx -- -file copy",
			},
			{
				Description: "Make a hard link in the portable gnux shell:",
				Bash:        "gnux shell -c 'linkx notes.md notes-link.md'",
				PowerShell:  "gnux shell -c 'linkx notes.md notes-link.md'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return link.New(link.Options{}, s).Run(args)
}
