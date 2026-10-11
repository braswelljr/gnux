// Package basenamex wires the basenamex command (GNU basename) to its engine.
package basenamex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basename"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the basenamex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "basename",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for basenamex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    basename.Name,
		SeeAlso: []string{"dirnamex", "realpathx"},
		Use:     basename.Name + " [OPTION]... NAME [SUFFIX]",
		Short:   "Strip the directory and suffix from file names.",
		Long: "basenamex prints NAME with any leading directory components removed. Trailing " +
			"slashes are ignored, so 'dir/' yields 'dir', and a name made only of slashes " +
			"yields the root. If SUFFIX is given and NAME ends with it, the suffix is removed " +
			"too - unless that would leave nothing. With -a every operand is a NAME; -s " +
			"supplies the suffix for all of them and implies -a. -z separates the output with " +
			"NUL instead of newlines, for names that may contain newlines.",
		Permissions: "basenamex works on the text of its operands only: it never touches the " +
			"filesystem, so the names need not exist and no permissions are required.",
		UseCases: []string{
			"Getting a script's own name from $0.",
			"Deriving an output file name by dropping the directory and extension of an input.",
			"Stripping one extension from a whole batch of paths with -s.",
			"Naming a directory after an archive, minus its .tar.gz.",
			"Renaming or copying files in a loop while keeping their base names.",
			"Printing several base names at once with -a, without a suffix.",
			"Producing NUL-separated output with -z for names that may contain newlines.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the last component of a path:",
				Bash:        "basenamex /usr/bin/sort",
				PowerShell:  "basenamex C:\\Windows\\System32\\sort.exe",
			},
			{
				Description: "Ignore trailing slashes:",
				Bash:        "basenamex /home/user/",
				PowerShell:  "basenamex C:\\Users\\user\\",
			},
			{
				Description: "Drop the directory and the extension:",
				Bash:        "basenamex include/stdio.h .h",
				PowerShell:  "basenamex include\\stdio.h .h",
			},
			{
				Description: "Remove a multi-part extension:",
				Bash:        "basenamex --suffix=.tar.gz release/gnux-1.0.tar.gz",
				PowerShell:  "basenamex --suffix=.tar.gz release\\gnux-1.0.tar.gz",
			},
			{
				Description: "Keep the name when removing the suffix would leave nothing:",
				Bash:        "basenamex .h .h",
				PowerShell:  "basenamex .h .h",
			},
			{
				Description: "Treat every operand as a NAME:",
				Bash:        "basenamex -a /var/log/syslog /etc/hosts",
				PowerShell:  "basenamex -a C:\\Windows\\win.ini C:\\Windows\\System32\\drivers\\etc\\hosts",
			},
			{
				Description: "Strip a suffix from several names at once:",
				Bash:        "basenamex -s .go cmd/main.go core/core.go",
				PowerShell:  "basenamex -s .go cmd\\main.go core\\core.go",
			},
			{
				Description: "Strip an extension from every match of a glob:",
				Bash:        "basenamex -s .jpg photos/*.jpg",
				PowerShell:  "basenamex -s .jpg (Get-ChildItem photos\\*.jpg)",
			},
			{
				Description: "Separate the output with NUL bytes instead of newlines:",
				Bash:        "basenamex -a -z a/b c/d | od -c",
				PowerShell:  "basenamex -a -z a\\b c\\d | Format-Hex",
			},
			{
				Description: "Print the name of the running script:",
				Bash:        "basenamex \"$0\"",
				PowerShell:  "basenamex $PSCommandPath",
			},
			{
				Description: "Back up every photo under its base name:",
				Bash:        "for f in photos/*.jpg; do cpx \"$f\" \"out/$(basenamex \"$f\" .jpg).bak\"; done",
				PowerShell:  "Get-ChildItem photos\\*.jpg | ForEach-Object { cpx $_.FullName \"out\\$(basenamex $_.Name .jpg).bak\" }",
			},
			{
				Description: "Create a directory named after an archive:",
				Bash:        "name=$(basenamex -s .tar.gz gnux-1.0.tar.gz) && mkdirx \"$name\"",
				PowerShell:  "$name = basenamex -s .tar.gz gnux-1.0.tar.gz; mkdirx $name",
			},
			{
				Description: "Strip a directory and extension from several logs with the gnux shell:",
				Bash:        "gnux shell -c 'basenamex -s .log /var/log/app.log /var/log/db.log'",
				PowerShell:  "gnux shell -c 'basenamex -s .log /var/log/app.log /var/log/db.log'",
			},
		},
	}
}

// Register installs the basenamex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("multiple", "a", false, "support multiple arguments and treat each as a NAME")
	fs.StringP("suffix", "s", "", "remove a trailing `SUFFIX`; implies -a")
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline")
}

// buildOptions folds the parsed flags into basename.Options.
func buildOptions(fs *pflag.FlagSet) basename.Options {
	var o basename.Options
	o.Multiple, _ = fs.GetBool("multiple")
	o.Suffix, _ = fs.GetString("suffix")
	o.Zero, _ = fs.GetBool("zero")
	// -s implies -a even when the suffix it carries is empty.
	if fs.Changed("suffix") {
		o.Multiple = true
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return basename.New(buildOptions(fs), s).Run(args)
}
