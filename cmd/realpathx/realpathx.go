// Package realpathx wires the realpathx command (GNU realpath) to its engine.
package realpathx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/realpath"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the realpathx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "realpath",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for realpathx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    realpath.Name,
		SeeAlso: []string{"pwdx", "dirnamex", "basenamex", "lnx"},
		Use:     realpath.Name + " [OPTION]... FILE...",
		Short:   "Print the resolved absolute file name.",
		Long: "realpathx prints the canonical absolute name of each FILE: every symbolic " +
			"link is expanded and every '.', '..' and repeated separator is removed. By " +
			"default all components but the last must exist; -e requires the whole name to " +
			"exist and -m requires none of it to. Links are resolved as they are met (-P, the " +
			"default), so '..' after a link refers to the parent of the link's target; -L " +
			"collapses '..' first, and -s does not expand links at all. --relative-to prints " +
			"each name relative to a directory, and --relative-base does so only for names " +
			"under that directory, leaving the rest absolute. A name that cannot be resolved " +
			"is reported (unless -q) and the remaining names are still processed.",
		Permissions: "Resolving a name needs search permission on every directory leading " +
			"to it, including those reached through symbolic links. The files themselves are " +
			"never opened, so no read permission on them is required.",
		UseCases: []string{
			"Finding where a symlinked command or directory really lives.",
			"Normalizing user-supplied paths before comparing or storing them.",
			"Computing a path relative to a project root with --relative-to.",
			"Building the absolute name of a file that is about to be created.",
			"Checking that a path exists in a script with -e and the exit status.",
			"Cleaning up '.', '..' and doubled separators without touching symbolic links (-s).",
			"Choosing how '..' after a symbolic link is read: physically (-P) or logically (-L).",
			"Emitting NUL-terminated names with -z for names that may contain newlines.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Resolve a path to its canonical absolute name:",
				Bash:        "realpathx ../lib/./util.go",
				PowerShell:  "realpathx ..\\lib\\.\\util.go",
			},
			{
				Description: "Print the absolute name of the current directory:",
				Bash:        "realpathx .",
				PowerShell:  "realpathx .",
			},
			{
				Description: "Follow a symbolic link to the file it points to:",
				Bash:        "realpathx bin/tool",
				PowerShell:  "realpathx bin\\tool",
			},
			{
				Description: "Require the file to exist:",
				Bash:        "realpathx -e /usr/bin/python3",
				PowerShell:  "realpathx -e $env:WINDIR\\System32\\cmd.exe",
			},
			{
				Description: "Test for existence quietly and branch on the exit status:",
				Bash:        "realpathx -q -e config.yaml >/dev/null || echox 'config.yaml is missing'",
				PowerShell:  "realpathx -q -e config.yaml > $null; if ($LASTEXITCODE -ne 0) { echox 'config.yaml is missing' }",
			},
			{
				Description: "Normalize a path that does not exist yet:",
				Bash:        "realpathx -m build/out/../report.txt",
				PowerShell:  "realpathx -m build\\out\\..\\report.txt",
			},
			{
				Description: "Clean up a path without expanding symbolic links:",
				Bash:        "realpathx -s bin/tool",
				PowerShell:  "realpathx -s bin\\tool",
			},
			{
				Description: "Same thing, spelled out:",
				Bash:        "realpathx --no-symlinks bin/tool",
				PowerShell:  "realpathx --no-symlinks bin\\tool",
			},
			{
				Description: "Read '..' after a link logically (-L) rather than physically (-P, the default):",
				Bash:        "realpathx -L src/tools/.. && realpathx -P src/tools/..",
				PowerShell:  "realpathx -L src\\tools\\..; realpathx -P src\\tools\\..",
			},
			{
				Description: "Print a path relative to another directory:",
				Bash:        "realpathx --relative-to=/usr/lib /usr/bin/sort",
				PowerShell:  "realpathx --relative-to=C:\\Windows C:\\Windows\\System32\\sort.exe",
			},
			{
				Description: "Make names under the project relative, but leave outside ones absolute:",
				Bash:        "realpathx --relative-base=. lib/util.go /etc/hosts",
				PowerShell:  "realpathx --relative-base=. lib\\util.go $env:WINDIR\\System32\\drivers\\etc\\hosts",
			},
			{
				Description: "Separate the output with NUL bytes instead of newlines:",
				Bash:        "realpathx -z lib/util.go src/app/main.go | od -c",
				PowerShell:  "realpathx -z lib\\util.go src\\app\\main.go | Format-Hex",
			},
			{
				Description: "Find the directory that really contains a file:",
				Bash:        "dirnamex \"$(realpathx scripts/run.sh)\"",
				PowerShell:  "dirnamex (realpathx scripts\\run.sh)",
			},
			{
				Description: "Print several paths relative to a source directory with the gnux shell:",
				Bash:        "gnux shell -c 'realpathx --relative-to=src lib/util.go bin/tool'",
				PowerShell:  "gnux shell -c 'realpathx --relative-to=src lib/util.go bin/tool'",
			},
		},
	}
}

// Register installs the realpathx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("canonicalize-existing", "e", false, "all components of the path must exist")
	fs.BoolP("canonicalize-missing", "m", false, "no path components need exist or be a directory")
	fs.BoolP("logical", "L", false, "resolve '..' components before symlinks")
	fs.BoolP("physical", "P", false, "resolve symlinks as encountered (default)")
	fs.BoolP("quiet", "q", false, "suppress most error messages")
	fs.String("relative-to", "", "print the resolved path relative to `DIR`")
	fs.String("relative-base", "", "print absolute paths unless paths below `DIR`")
	fs.BoolP("strip", "s", false, "don't expand symlinks")
	fs.Bool("no-symlinks", false, "don't expand symlinks (same as --strip)")
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline")
}

// buildOptions folds the parsed flags into realpath.Options. GNU lets the
// last of two opposing flags win, but cobra does not expose flag order, so
// conflicts resolve one fixed way: -e over -m (the stricter check) and -L
// over -P (the non-default, as pwdx does).
func buildOptions(fs *pflag.FlagSet) realpath.Options {
	var o realpath.Options

	if missing, _ := fs.GetBool("canonicalize-missing"); missing {
		o.Mode = realpath.ModeMissing
	}
	if existing, _ := fs.GetBool("canonicalize-existing"); existing {
		o.Mode = realpath.ModeExisting
	}

	o.Logical, _ = fs.GetBool("logical")
	o.Quiet, _ = fs.GetBool("quiet")
	o.Zero, _ = fs.GetBool("zero")
	o.RelativeTo, _ = fs.GetString("relative-to")
	o.RelativeBase, _ = fs.GetString("relative-base")

	strip, _ := fs.GetBool("strip")
	noSymlinks, _ := fs.GetBool("no-symlinks")
	o.Strip = strip || noSymlinks

	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return realpath.New(buildOptions(fs), s).Run(args)
}
