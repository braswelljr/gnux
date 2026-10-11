// Package force implements rm's -f/--force flag: ignore nonexistent files and
// arguments, never prompt, and suppress the resulting errors.
package force

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Register installs the -f/--force flag on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("force", "f", false, "ignore nonexistent files and arguments, never prompt")
}

// Meta returns the documentation metadata for the force flag.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "force",
		Use:   "force [FILE]...",
		Short: "Ignore nonexistent files and arguments, and never prompt.",
		Long: "The -f/--force flag makes rmx ignore files that do not exist (exiting " +
			"successfully instead of reporting an error) and suppresses every confirmation " +
			"prompt, including the write-protected-file prompt. It is what turns rmx into an " +
			"unattended, script-safe delete. -f only affects prompting and missing-file errors; " +
			"it does not enable directory removal - combine it with -r for that.",
		Permissions: "-f skips the write-protected-file prompt, so read-only files are removed " +
			"without asking. Deletion is still governed by the parent directory's permissions, " +
			"not the file's own mode: if you lack write+execute on the containing directory the " +
			"removal fails with 'Permission denied' even under -f.",
		UseCases: []string{
			"Unattended cleanup in CI pipelines and shell scripts where prompts would hang.",
			"Deleting paths that may or may not exist without treating absence as an error.",
			"Removing read-only or write-protected files without per-file confirmation.",
			"Making cleanup steps idempotent so a script can be re-run safely.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove a file, succeeding silently if it is already gone:",
				Bash:        "rmx -f maybe-missing.log",
				PowerShell:  "rmx -f maybe-missing.log",
			},
			{
				Description: "Recursively delete a build directory with no prompts:",
				Bash:        "rmx -rf build/",
				PowerShell:  "rmx -rf build",
			},
			{
				Description: "Remove a write-protected file without being asked:",
				Bash:        "rmx -f readonly.cfg",
				PowerShell:  "rmx -f readonly.cfg",
			},
			{
				Description: "Run with no operands at all, which -f also tolerates:",
				Bash:        "rmx -f && echox \"nothing to do\"",
				PowerShell:  "rmx -f; if ($LASTEXITCODE -eq 0) { echox \"nothing to do\" }",
			},
			{
				Description: "Make a cleanup step idempotent in the gnux shell:",
				Bash:        "gnux shell -c 'rmx -fv out.log err.log; echox $?'",
				PowerShell:  "gnux shell -c 'rmx -fv out.log err.log; echox $?'",
			},
		},
	}
}

// Apply reads the force flag and folds it into o. Force also drops the prompt
// mode to PromptNever; a later interactive flag may raise it again.
func Apply(fs *pflag.FlagSet, o *rm.Options) {
	force, _ := fs.GetBool("force")
	o.Force = force
	if force {
		o.Interactive = rm.PromptNever
	}
}
