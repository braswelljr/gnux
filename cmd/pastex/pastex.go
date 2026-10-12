// Package pastex wires the pastex command (GNU paste) to its engine.
package pastex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/paste"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the pastex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "paste",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for pastex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    paste.Name,
		SeeAlso: []string{"cutx", "joinx", "commx", "catx"},
		Use:     paste.Name + " [OPTION]... [FILE]...",
		Short:   "Merge lines of files.",
		Long: "pastex writes lines made of the corresponding lines of each FILE, separated by " +
			"TABs, to standard output. With no FILE, or when FILE is -, it reads standard input; " +
			"every - reads the same standard input, so 'pastex - -' joins pairs of lines. A file " +
			"that runs out of lines contributes empty fields while the others continue. -s pastes " +
			"one file at a time instead, turning each file into a single line. -d LIST uses the " +
			"characters of LIST as delimiters instead of TAB, cycling through them and starting " +
			"again from the first on every output line (and, with -s, for every file). LIST " +
			"understands the escapes \\n, \\t, \\\\, \\b, \\f, \\r, \\v and \\0, the empty " +
			"delimiter. -z makes NUL the line delimiter. Delimiters are single bytes: a " +
			"multibyte character in LIST is used one byte at a time.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Putting two lists side by side, one column each.",
			"Joining every two or three lines of a file into one with 'pastex - - < file'.",
			"Turning a list of lines into one comma-separated line with -s -d ','.",
			"Building a CSV from columns kept in separate files.",
			"Summing a column of numbers by joining it with '+' for another tool.",
			"Merging a file of keys and a file of values line by line.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Put two files side by side, separated by a TAB:",
				Bash:        "pastex names.txt ages.txt",
				PowerShell:  "pastex names.txt ages.txt",
			},
			{
				Description: "Use a comma instead of a TAB:",
				Bash:        "pastex -d , names.txt ages.txt",
				PowerShell:  "pastex -d , names.txt ages.txt",
			},
			{
				Description: "Join all the lines of a file into one comma-separated line:",
				Bash:        "pastex -s -d , hosts.txt",
				PowerShell:  "pastex -s -d , hosts.txt",
			},
			{
				Description: "Join every two lines of standard input:",
				Bash:        "printf 'a\\nb\\nc\\nd\\n' | pastex - -",
				PowerShell:  "'a','b','c','d' | pastex - -",
			},
			{
				Description: "Lay a list out in three columns:",
				Bash:        "seqx 9 | pastex - - -",
				PowerShell:  "seqx 9 | pastex - - -",
			},
			{
				Description: "Cycle through several delimiters:",
				Bash:        "pastex -d ',;' a.txt b.txt c.txt",
				PowerShell:  "pastex -d ',;' a.txt b.txt c.txt",
			},
			{
				Description: "Join columns with no delimiter at all:",
				Bash:        "pastex -d '\\0' first.txt last.txt",
				PowerShell:  "pastex -d '\\0' first.txt last.txt",
			},
			{
				Description: "Show each file as one TAB-separated line:",
				Bash:        "pastex -s a.txt b.txt",
				PowerShell:  "pastex -s a.txt b.txt",
			},
			{
				Description: "Paste NUL-terminated records:",
				Bash:        "pastex -z keys.bin values.bin",
				PowerShell:  "pastex -z keys.bin values.bin",
			},
			{
				Description: "Turn a column of numbers into one sum expression (1+2+3+4+5):",
				Bash:        "seqx 5 | pastex -s -d +",
				PowerShell:  "seqx 5 | pastex -s -d +",
			},
			{
				Description: "Paste in the portable gnux shell:",
				Bash:        "gnux shell -c 'seqx 6 | pastex - - | sortx -rn'",
				PowerShell:  "gnux shell -c 'seqx 6 | pastex - - | sortx -rn'",
			},
		},
	}
}

// Register installs the pastex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("delimiters", "d", "", "reuse characters from `LIST` instead of TABs")
	fs.BoolP("serial", "s", false, "paste one file at a time instead of in parallel")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
}

// buildOptions folds the parsed flags into paste.Options.
func buildOptions(fs *pflag.FlagSet) paste.Options {
	o := paste.Options{}
	o.Serial, _ = fs.GetBool("serial")
	o.ZeroTerminated, _ = fs.GetBool("zero-terminated")
	if fs.Changed("delimiters") {
		o.Delimiters, _ = fs.GetString("delimiters")
		o.DelimitersSet = true
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return paste.New(buildOptions(fs), s).Run(args)
}
