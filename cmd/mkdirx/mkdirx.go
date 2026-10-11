// Package mkdirx wires the mkdirx command (GNU mkdir) to its engine.
package mkdirx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mkdir"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the mkdirx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "mkdir",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for mkdirx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    mkdir.Name,
		SeeAlso: []string{"rmdirx", "touchx", "cpx"},
		Use:     mkdir.Name + " [OPTION]... DIRECTORY...",
		Short:   "Create the DIRECTORY(ies), if they do not already exist.",
		Long: "mkdirx creates each DIRECTORY in the order given. Without options the parent " +
			"must already exist and the directory must not, otherwise the operand is reported " +
			"and the rest are still processed. -p creates any missing parents along the way " +
			"and treats a directory that is already there as success, which makes the command " +
			"safe to repeat. -m sets the permissions of the new directory using either an " +
			"octal mode (755) or a chmod-style symbolic one (u=rwx,go=rx), and -v reports each " +
			"directory as it is created.",
		Permissions: "Creating a directory needs write and search permission on its parent. " +
			"New directories get a=rwx minus the umask. A mode given with -m is applied " +
			"exactly, without being reduced by the umask, but only to the operand: parents " +
			"made by -p keep the default mode, plus owner write and search so the next level " +
			"can be created inside them. On Windows only the read-only attribute is " +
			"representable, so most of a mode has no effect there.",
		UseCases: []string{
			"Creating a nested output tree in one step with -p.",
			"Idempotent setup in scripts and CI, where the directory may already exist.",
			"Making a private directory with -m 700 regardless of the umask.",
			"Logging what a provisioning script created with -v.",
			"Scaffolding a project layout (src, tests, docs) in a single command.",
			"Creating a shared drop directory with the sticky bit, such as mode 1777.",
			"Making sure the parent of an output file exists before writing it.",
			"Guarding a script step with && so later commands only run once the directory is there.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Create a directory:",
				Bash:        "mkdirx build",
				PowerShell:  "mkdirx build",
			},
			{
				Description: "Create several sibling directories at once:",
				Bash:        "mkdirx src docs tests",
				PowerShell:  "mkdirx src docs tests",
			},
			{
				Description: "Create a nested tree, reporting each level:",
				Bash:        "mkdirx -pv out/reports/2024",
				PowerShell:  "mkdirx -pv out\\reports\\2024",
			},
			{
				Description: "Scaffold a project layout, reporting every directory made:",
				Bash:        "mkdirx -pv project/src project/tests project/docs",
				PowerShell:  "mkdirx -pv project\\src project\\tests project\\docs",
			},
			{
				Description: "Create a directory only its owner can enter:",
				Bash:        "mkdirx -m 700 ~/.secrets",
				PowerShell:  "mkdirx -m 700 $HOME\\.secrets",
			},
			{
				Description: "Use a symbolic mode:",
				Bash:        "mkdirx -m u=rwx,go=rx shared",
				PowerShell:  "mkdirx -m 'u=rwx,go=rx' shared",
			},
			{
				Description: "Grant everything except write for others, using a symbolic mode with a removal:",
				Bash:        "mkdirx -m a=rwx,o-w drop",
				PowerShell:  "mkdirx -m 'a=rwx,o-w' drop",
			},
			{
				Description: "Create a world-writable scratch directory with the sticky bit set:",
				Bash:        "mkdirx -m 1777 scratch",
				PowerShell:  "mkdirx -m 1777 scratch",
			},
			{
				Description: "Create missing parents but give only the last directory an explicit mode:",
				Bash:        "mkdirx -p -m 755 www/public",
				PowerShell:  "mkdirx -p -m 755 www\\public",
			},
			{
				Description: "Run the next step only once the directory exists (safe to repeat):",
				Bash:        "mkdirx -p out/reports/2024 && echox ready",
				PowerShell:  "mkdirx -p out\\reports\\2024; if ($LASTEXITCODE -eq 0) { echox ready }",
			},
			{
				Description: "React to a failure, here a directory that already exists:",
				Bash:        "mkdirx build 2>/dev/null || echox 'build already exists'",
				PowerShell:  "mkdirx build 2>$null; if ($LASTEXITCODE -ne 0) { echox 'build already exists' }",
			},
			{
				Description: "Create the directory a file is about to be written to:",
				Bash:        "mkdirx -p \"$(dirnamex out/reports/q1.csv)\"",
				PowerShell:  "mkdirx -p (dirnamex out\\reports\\q1.csv)",
			},
			{
				Description: "Keep an otherwise empty directory under version control:",
				Bash:        "mkdirx -p cache && touchx cache/.keep && lsx -A cache",
				PowerShell:  "mkdirx -p cache; touchx cache\\.keep; lsx -A cache",
			},
			{
				Description: "Do the same setup portably with the gnux shell:",
				Bash:        "gnux shell -c 'mkdirx -pv site/assets && touchx site/index.html && lsx site'",
				PowerShell:  "gnux shell -c 'mkdirx -pv site/assets && touchx site/index.html && lsx site'",
			},
		},
	}
}

// Register installs the mkdirx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("mode", "m", "", "set file `MODE` (as in chmod), not a=rwx - umask")
	fs.BoolP("parents", "p", false, "no error if existing, make parent directories as needed")
	fs.BoolP("verbose", "v", false, "print a message for each created directory")
}

// buildOptions folds the parsed flags into mkdir.Options.
func buildOptions(fs *pflag.FlagSet) mkdir.Options {
	var o mkdir.Options
	o.Mode, _ = fs.GetString("mode")
	o.Parents, _ = fs.GetBool("parents")
	o.Verbose, _ = fs.GetBool("verbose")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return mkdir.New(buildOptions(fs), s).Run(args)
}
