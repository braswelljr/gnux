// Package dirnamex wires the dirnamex command (GNU dirname) to its engine.
package dirnamex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dirname"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the dirnamex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "dirname",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for dirnamex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    dirname.Name,
		SeeAlso: []string{"basenamex", "realpathx", "mkdirx"},
		Use:     dirname.Name + " [OPTION] NAME...",
		Short:   "Strip the last component from file names.",
		Long: "dirnamex prints each NAME with its last non-slash component and trailing " +
			"slashes removed. If NAME contains no slash the result is '.', the current " +
			"directory, and anything directly under the root yields the root itself. The rest " +
			"of the path is left exactly as written: no symbolic links are resolved and no " +
			"'.' or '..' components are collapsed. -z separates the output with NUL instead of " +
			"newlines, for names that may contain newlines.",
		Permissions: "dirnamex works on the text of its operands only: it never touches the " +
			"filesystem, so the names need not exist and no permissions are required.",
		UseCases: []string{
			"Finding the directory a script lives in from $0.",
			"Computing the parent directory to create before writing a file.",
			"Splitting a list of paths into their directories in a pipeline.",
			"Walking up a path several levels by applying dirnamex repeatedly.",
			"Getting the name of a file's containing folder together with basenamex.",
			"Finding the real directory of a file behind symbolic links together with realpathx.",
			"Producing NUL-separated output with -z for names that may contain newlines.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the directory part of a path:",
				Bash:        "dirnamex /usr/bin/sort",
				PowerShell:  "dirnamex C:\\Windows\\System32\\sort.exe",
			},
			{
				Description: "Handle several names at once:",
				Bash:        "dirnamex dir1/str dir2/str stdio.h",
				PowerShell:  "dirnamex dir1\\str dir2\\str stdio.h",
			},
			{
				Description: "Ignore trailing slashes:",
				Bash:        "dirnamex /usr/bin/",
				PowerShell:  "dirnamex C:\\Windows\\System32\\",
			},
			{
				Description: "Get '.' for a bare name and the root for a top-level entry:",
				Bash:        "dirnamex stdio.h /etc",
				PowerShell:  "dirnamex stdio.h \\Windows",
			},
			{
				Description: "Leave '.' and '..' exactly as written (no normalization):",
				Bash:        "dirnamex a/../b/./c",
				PowerShell:  "dirnamex a\\..\\b\\.\\c",
			},
			{
				Description: "Separate the output with NUL bytes instead of newlines:",
				Bash:        "dirnamex -z a/b c/d | od -c",
				PowerShell:  "dirnamex -z a\\b c\\d | Format-Hex",
			},
			{
				Description: "Create the directory a file will be written to:",
				Bash:        "mkdirx -p \"$(dirnamex out/reports/q1.csv)\"",
				PowerShell:  "mkdirx -p (dirnamex out\\reports\\q1.csv)",
			},
			{
				Description: "Create a log file's directory, then the file:",
				Bash:        "d=$(dirnamex logs/2024/app.log) && mkdirx -p \"$d\" && touchx logs/2024/app.log",
				PowerShell:  "$d = dirnamex logs\\2024\\app.log; mkdirx -p $d; touchx logs\\2024\\app.log",
			},
			{
				Description: "Change to the directory a script lives in:",
				Bash:        "cd \"$(dirnamex \"$0\")\" && pwdx",
				PowerShell:  "Set-Location (dirnamex $PSCommandPath); pwdx",
			},
			{
				Description: "Go up two levels:",
				Bash:        "dirnamex \"$(dirnamex /srv/app/releases/v1/bin)\"",
				PowerShell:  "dirnamex (dirnamex C:\\srv\\app\\releases\\v1\\bin)",
			},
			{
				Description: "Print the name of the folder that contains a file:",
				Bash:        "basenamex \"$(dirnamex /var/lib/app/data.db)\"",
				PowerShell:  "basenamex (dirnamex C:\\ProgramData\\app\\data.db)",
			},
			{
				Description: "Find the real directory of a file behind symbolic links:",
				Bash:        "dirnamex \"$(realpathx scripts/run.sh)\"",
				PowerShell:  "dirnamex (realpathx scripts\\run.sh)",
			},
			{
				Description: "Split several paths into their directories with the gnux shell:",
				Bash:        "gnux shell -c 'dirnamex src/main.go docs/guide.md README.md'",
				PowerShell:  "gnux shell -c 'dirnamex src/main.go docs/guide.md README.md'",
			},
		},
	}
}

// Register installs the dirnamex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline")
}

// buildOptions folds the parsed flags into dirname.Options.
func buildOptions(fs *pflag.FlagSet) dirname.Options {
	zero, _ := fs.GetBool("zero")
	return dirname.Options{Zero: zero}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return dirname.New(buildOptions(fs), s).Run(args)
}
