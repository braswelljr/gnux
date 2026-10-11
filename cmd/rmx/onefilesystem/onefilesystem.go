// Package onefilesystem implements rm's --one-file-system flag: when removing a
// hierarchy recursively, skip any directory on a different file system than the
// one the removal started on.
package onefilesystem

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Meta returns the documentation metadata for the one-file-system flag.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "onefilesystem",
		Use:   "one-file-system [DIR]...",
		Short: "Stay on one file system during recursive removal (--one-file-system).",
		Long: "When removing a hierarchy recursively, --one-file-system skips any subdirectory " +
			"that lives on a different file system than the argument the removal started from. " +
			"It guards against accidentally descending into a mounted volume. On Windows, where " +
			"device identity is unavailable, the flag is a no-op.",
		Permissions: "The flag compares device IDs (via stat) and needs no extra permissions " +
			"beyond those recursive removal already requires. Skipped mount points are reported " +
			"on stderr and are not treated as errors.",
		UseCases: []string{
			"Clearing a directory that contains mounted volumes without deleting across the mount.",
			"Safely removing a tree on the root file system while leaving mounted disks intact.",
			"Cleaning container or chroot scratch trees that have bind mounts inside.",
			"Adding a guard to scripted recursive deletes whose targets may contain mounts.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove a tree but skip file systems mounted beneath it:",
				Bash:        "rmx -r --one-file-system /data",
			},
			{
				Description: "Report each removal while staying on one file system:",
				Bash:        "rmx -rv --one-file-system scratch/",
				PowerShell:  "rmx -rv --one-file-system scratch",
			},
			{
				Description: "Unattended cleanup of a chroot that may contain bind mounts:",
				Bash:        "rmx -rf --one-file-system chroot/",
			},
			{
				Description: "Check the result in a script:",
				Bash:        "rmx -r --one-file-system build/ && echox \"build removed\"",
				PowerShell:  "rmx -r --one-file-system build; if ($LASTEXITCODE -eq 0) { echox \"build removed\" }",
			},
		},
	}
}

// Register installs the --one-file-system flag on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("one-file-system", false, "when removing recursively, skip directories on a different file system")
}

// Apply sets o.OneFileSystem from the flag.
func Apply(fs *pflag.FlagSet, o *rm.Options) {
	o.OneFileSystem, _ = fs.GetBool("one-file-system")
}
