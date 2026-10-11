// Package recursive implements rm's -r/-R/--recursive flag: remove directories
// and their contents recursively.
package recursive

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Meta returns the documentation metadata for the recursive flag.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "recursive",
		Use:   "recursive [DIR]...",
		Short: "Remove directories and their contents recursively (-r, -R).",
		Long: "By default rmx refuses to remove directories. -r (or its alias -R) removes each " +
			"directory and everything beneath it, walking the tree post-order so children are " +
			"removed before their parents. It composes with the other flags: -ri prompts per " +
			"entry, -rv reports each removal, -rf deletes without prompting.",
		Permissions: "Recursive removal needs read+execute permission to list and descend into " +
			"each directory, and write permission on a directory to unlink the entries inside " +
			"it. A directory you cannot write to fails with 'Permission denied' and its contents " +
			"are left in place. The --preserve-root failsafe additionally refuses to recurse on " +
			"'/'.",
		UseCases: []string{
			"Deleting a directory and all of its contents in one command.",
			"Clearing build/output trees (build/, dist/, node_modules/).",
			"Combining with -i/-I for confirmation or -v to audit a large delete.",
			"Resetting a generated directory before regenerating it.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove a directory and everything under it:",
				Bash:        "rmx -r logs/",
				PowerShell:  "rmx -r logs",
			},
			{
				Description: "Recursively remove a directory, printing each deletion:",
				Bash:        "rmx -rv cache/",
				PowerShell:  "rmx -rv cache",
			},
			{
				Description: "Use the -R spelling, as with GNU rm:",
				Bash:        "rmx -R dist/",
				PowerShell:  "rmx -R dist",
			},
			{
				Description: "Remove several trees without prompting:",
				Bash:        "rmx -rf node_modules/ coverage/",
				PowerShell:  "rmx -rf node_modules coverage",
			},
			{
				Description: "Reset a directory and recreate it empty:",
				Bash:        "rmx -rf out/ && mkdirx out",
				PowerShell:  "rmx -rf out; if ($LASTEXITCODE -eq 0) { mkdirx out }",
			},
			{
				Description: "Recursively clean up with the gnux shell:",
				Bash:        "gnux shell -c 'rmx -r tmp-*'",
				PowerShell:  "gnux shell -c 'rmx -r tmp-*'",
			},
		},
	}
}

// Register installs -r and its -R alias on fs. -R is hidden so --help stays
// close to rm's own output.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("recursive", "r", false, "remove directories and their contents recursively")
	fs.BoolP("recursive-upper", "R", false, "equivalent to -r")
	_ = fs.MarkHidden("recursive-upper")
}

// Apply sets o.Recursive when either -r or -R was given.
func Apply(fs *pflag.FlagSet, o *rm.Options) {
	r, _ := fs.GetBool("recursive")
	ru, _ := fs.GetBool("recursive-upper")
	o.Recursive = r || ru
}
