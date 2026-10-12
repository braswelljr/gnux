// Package unlinkx wires the unlinkx command (GNU unlink) to its engine.
package unlinkx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unlink"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the unlinkx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "unlink",
		Run:  run,
	}
}

// Meta returns the documentation metadata for unlinkx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    unlink.Name,
		SeeAlso: []string{"rmx", "rmdirx", "linkx"},
		Use:     unlink.Name + " FILE",
		Short:   "Call the unlink function to remove the specified file.",
		Long: "unlinkx removes the single name FILE with the unlink(2) system call. It takes " +
			"exactly one operand and no options: it never recurses, never prompts, never " +
			"ignores a missing file, and refuses a directory, even an empty one, with 'Is a " +
			"directory' (use rmdirx for that). A symbolic link is removed, not its target. " +
			"Removing the last name of a file frees its data once no process has it open.",
		Permissions: "Needs write and search permission on the directory containing FILE; " +
			"permission on FILE itself does not matter. In a directory with the sticky bit " +
			"set (such as /tmp), only the file's owner, the directory's owner or root may " +
			"remove it. On Windows, a read-only file or one open in another process cannot be " +
			"removed.",
		UseCases: []string{
			"Removing exactly one file in a script, failing loudly if anything is unexpected.",
			"Removing a symbolic link without any risk of touching what it points to.",
			"Releasing a lock file created with linkx.",
			"Removing a file whose name could be mistaken for an option, with --.",
			"Removing one of several hard links while keeping the others.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove one file:",
				Bash:        "unlinkx old.log",
				PowerShell:  "unlinkx old.log",
			},
			{
				Description: "Remove a symbolic link, leaving its target alone:",
				Bash:        "unlinkx current",
				PowerShell:  "unlinkx current",
			},
			{
				Description: "A directory is refused (use rmdirx):",
				Bash:        "unlinkx build || rmdirx build",
				PowerShell:  "unlinkx build; if ($LASTEXITCODE -ne 0) { rmdirx build }",
			},
			{
				Description: "Release a lock file:",
				Bash:        "unlinkx app.lock",
				PowerShell:  "unlinkx app.lock",
			},
			{
				Description: "Remove a file whose name starts with a dash:",
				Bash:        "unlinkx -- -oops",
				PowerShell:  "unlinkx -- -oops",
			},
			{
				Description: "Drop one hard link and keep the other name:",
				Bash:        "linkx data data.bak && unlinkx data",
				PowerShell:  "linkx data data.bak; unlinkx data",
			},
			{
				Description: "Fail when the file is already gone (rmx -f would not):",
				Bash:        "unlinkx tmp.txt || echox 'nothing to remove'",
				PowerShell:  "unlinkx tmp.txt; if ($LASTEXITCODE -ne 0) { 'nothing to remove' }",
			},
			{
				Description: "Remove a file with spaces in its name:",
				Bash:        "unlinkx 'my file.txt'",
				PowerShell:  "unlinkx 'my file.txt'",
			},
			{
				Description: "Remove a temporary file at the end of a script:",
				Bash:        "tmp=$(mktempx) && echox data > \"$tmp\" && unlinkx \"$tmp\"",
				PowerShell:  "$tmp = mktempx; 'data' > $tmp; unlinkx $tmp",
			},
			{
				Description: "Remove a file in the portable gnux shell:",
				Bash:        "gnux shell -c 'unlinkx out.txt'",
				PowerShell:  "gnux shell -c 'unlinkx out.txt'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return unlink.New(unlink.Options{}, s).Run(args)
}
