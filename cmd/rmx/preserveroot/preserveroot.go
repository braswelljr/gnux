// Package preserveroot implements rm's --preserve-root / --no-preserve-root
// flags, the failsafe that refuses to recurse on the filesystem root '/'.
package preserveroot

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Meta returns the documentation metadata for the preserve-root flags.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "preserveroot",
		Use:   "preserve-root [FILE]...",
		Short: "Refuse to recurse on '/' (--preserve-root, default; --no-preserve-root overrides).",
		Long: "--preserve-root is the default failsafe: rmx refuses to operate recursively on the " +
			"filesystem root '/'. --no-preserve-root disables the failsafe for the rare, " +
			"deliberate case where you really do mean to recurse from a root. Use it with " +
			"extreme care.",
		Permissions: "This is a path check, not a permission check: rmx refuses '/' before it " +
			"touches the filesystem, regardless of whether you would otherwise have permission. " +
			"It never grants access you do not already have.",
		UseCases: []string{
			"Left at its default, protecting against a catastrophic 'rmx -r /' typo.",
			"Deliberately overridden (--no-preserve-root) only in throwaway or container roots.",
			"Guarding scripts where an empty variable could turn a path into '/'.",
			"Stating the default explicitly (--preserve-root) in scripts for reviewers.",
		},
		Examples: []docmeta.Example{
			{
				Description: "The default failsafe refuses to recurse on the root:",
				Bash:        "rmx -r /",
				PowerShell:  "rmx -r C:\\",
			},
			{
				Description: "Disable the failsafe (dangerous - only when you mean it):",
				Bash:        "rmx -r --no-preserve-root /mnt/scratch",
			},
			{
				Description: "State the default explicitly in a script:",
				Bash:        "rmx -rf --preserve-root \"$BUILD_DIR/\"",
				PowerShell:  "rmx -rf --preserve-root \"$env:BUILD_DIR\\\"",
			},
			{
				Description: "The failsafe makes the command fail, which a script can detect:",
				Bash:        "rmx -rf / || echox \"refused, status $?\"",
				PowerShell:  "rmx -rf C:\\; if ($LASTEXITCODE -ne 0) { echox \"refused, status $LASTEXITCODE\" }",
			},
		},
	}
}

// Register installs --preserve-root (default) and --no-preserve-root on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("preserve-root", true, "do not remove '/' (default)")
	fs.Bool("no-preserve-root", false, "do not treat '/' specially")
}

// Apply sets o.NoPreserveRoot; --no-preserve-root disables the failsafe.
func Apply(fs *pflag.FlagSet, o *rm.Options) {
	no, _ := fs.GetBool("no-preserve-root")
	o.NoPreserveRoot = no
}
