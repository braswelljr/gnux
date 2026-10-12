// Package tacx wires the tacx command (GNU tac) to its engine.
package tacx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tac"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the tacx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "tac",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for tacx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tac.Name,
		SeeAlso: []string{"catx", "tailx", "sortx"},
		Use:     tac.Name + " [OPTION]... [FILE]...",
		Short:   "Concatenate and print files in reverse.",
		Long: "tacx writes each FILE to standard output last line first. With no FILE, or when " +
			"FILE is -, it reads standard input. Each FILE is reversed on its own, and the " +
			"files are written in the order given. A record ends with its separator, a " +
			"newline by default, so a last line without one is joined to the line before it " +
			"in the output, as in GNU. -s STRING uses STRING as the separator instead, and -b " +
			"attaches the separator to the start of the record that follows it rather than " +
			"to the end of the one before. -r treats the separator as a regular expression; " +
			"gnux uses Go's RE2 syntax (close to POSIX extended regular expressions) rather " +
			"than the GNU regex dialect, and reports an invalid expression in Go's words. " +
			"Each input is read whole into memory before it is written.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Reading a log newest entry first.",
			"Reversing the order of a list or a history file.",
			"Processing records last to first, such as undoing steps in reverse.",
			"Reversing records separated by something other than a newline with -s.",
			"Finding the last match in a file by reversing it before searching.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print a file last line first:",
				Bash:        "tacx app.log",
				PowerShell:  "tacx app.log",
			},
			{
				Description: "Show the newest ten log lines, newest first:",
				Bash:        "tacx app.log | headx",
				PowerShell:  "tacx app.log | headx",
			},
			{
				Description: "Reverse standard input:",
				Bash:        "seqx 5 | tacx",
				PowerShell:  "seqx 5 | tacx",
			},
			{
				Description: "Reverse several files, each on its own:",
				Bash:        "tacx a.txt b.txt",
				PowerShell:  "tacx a.txt b.txt",
			},
			{
				Description: "Reverse comma-separated records:",
				Bash:        "printfx 'a,b,c,' | tacx -s ,",
				PowerShell:  "printfx 'a,b,c,' | tacx -s ,",
			},
			{
				Description: "Attach each separator to the following record:",
				Bash:        "tacx -b -s '---' notes.txt",
				PowerShell:  "tacx -b -s '---' notes.txt",
			},
			{
				Description: "Split records on a regular expression:",
				Bash:        "tacx -r -s '[0-9]+' data.txt",
				PowerShell:  "tacx -r -s '[0-9]+' data.txt",
			},
			{
				Description: "Save a reversed copy:",
				Bash:        "tacx history.txt > reversed.txt",
				PowerShell:  "tacx history.txt > reversed.txt",
			},
			{
				Description: "Undo a list of steps in reverse order:",
				Bash:        "tacx steps.txt | catx -n",
				PowerShell:  "tacx steps.txt | catx -n",
			},
			{
				Description: "Reverse a sorted list instead of sorting it again:",
				Bash:        "sortx names.txt | tacx",
				PowerShell:  "sortx names.txt | tacx",
			},
			{
				Description: "Reverse in the portable gnux shell:",
				Bash:        "gnux shell -c 'tacx app.log | headx -n 5'",
				PowerShell:  "gnux shell -c 'tacx app.log | headx -n 5'",
			},
		},
	}
}

// Register installs the tacx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("before", "b", false, "attach the separator before instead of after")
	fs.BoolP("regex", "r", false, "interpret the separator as a regular expression")
	fs.StringP("separator", "s", "", "use `STRING` as the separator instead of newline")
}

// buildOptions folds the parsed flags into tac.Options.
func buildOptions(fs *pflag.FlagSet) tac.Options {
	before, _ := fs.GetBool("before")
	regex, _ := fs.GetBool("regex")
	sep, _ := fs.GetString("separator")
	return tac.Options{Before: before, Regex: regex, Separator: sep, SeparatorSet: fs.Changed("separator")}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return tac.New(buildOptions(fs), s).Run(args)
}
