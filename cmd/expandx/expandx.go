// Package expandx wires the expandx command (GNU expand) to its engine.
package expandx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expand"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/obsolete"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the expandx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "expand",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for expandx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    expand.Name,
		SeeAlso: []string{"unexpandx", "foldx", "trx"},
		Use:     expand.Name + " [OPTION]... [FILE]...",
		Short:   "Convert tabs to spaces.",
		Long: "expandx writes each FILE to standard output with every tab replaced by the " +
			"spaces that reach the next tab stop. With no FILE, or when FILE is -, it reads " +
			"standard input. Tab stops are every 8 columns unless -t says otherwise: -t N puts " +
			"them every N columns, and -t LIST (numbers separated by commas or blanks, in " +
			"ascending order) puts them at those columns; the last number may be written /N " +
			"for a stop at every multiple of N after it, or +N for a stop every N columns after " +
			"it. Past the last listed stop a tab becomes one space. Several -t options add to " +
			"one list, and the obsolete -N form (expandx -4) is a list of its own, in the order " +
			"given; GNU's comma form -4,8 is not accepted, and the digits of two -N options " +
			"in a row run together (-4 -8 reads as -48), so write -t 4,8 instead. -i converts only " +
			"the tabs before the first non-blank character of each line. A backspace moves " +
			"back one column. Like GNU expand in the C locale, expandx counts bytes rather " +
			"than characters.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Converting tab-indented source to spaces for a style guide.",
			"Making tab-separated output line up the same way in every viewer.",
			"Preparing text for tools that count characters, like cutx -c or foldx.",
			"Converting only the indentation with -i, leaving tabs inside lines alone.",
			"Laying out columns at specific positions with -t LIST.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Make a tab-indented C file look the same in every editor:",
				Bash:        "expandx main.c",
				PowerShell:  "expandx main.c",
			},
			{
				Description: "Convert a Go file to 4-space indentation for a code review tool:",
				Bash:        "expandx -t 4 main.go",
				PowerShell:  "expandx -t 4 main.go",
			},
			{
				Description: "Run an old script that uses the obsolete form:",
				Bash:        "expandx -4 main.go",
				PowerShell:  "expandx -4 main.go",
			},
			{
				Description: "Line a tab-separated report up in fixed columns for printing:",
				Bash:        "expandx -t 10,20,40 table.tsv",
				PowerShell:  "expandx -t 10,20,40 table.tsv",
			},
			{
				Description: "Indent the first level by 4 and the rest by 8:",
				Bash:        "expandx -t 4,/8 data.txt",
				PowerShell:  "expandx -t 4,/8 data.txt",
			},
			{
				Description: "Fix YAML indentation (tabs are invalid there) without touching tabs inside values:",
				Bash:        "expandx -i -t 2 config.yml",
				PowerShell:  "expandx -i -t 2 config.yml",
			},
			{
				Description: "See exactly where a tab lands:",
				Bash:        "printfx 'a\\tb\\n' | expandx",
				PowerShell:  "printfx 'a\\tb\\n' | expandx",
			},
			{
				Description: "Convert a Python file that mixes tabs and spaces:",
				Bash:        "expandx -t 4 tabs.py > spaces.py",
				PowerShell:  "expandx -t 4 tabs.py > spaces.py",
			},
			{
				Description: "Preview a TSV file with readable columns:",
				Bash:        "expandx -t 16 report.tsv | headx",
				PowerShell:  "expandx -t 16 report.tsv | headx",
			},
			{
				Description: "Combine several files into one, with tabs converted:",
				Bash:        "expandx a.txt b.txt > combined.txt",
				PowerShell:  "expandx a.txt b.txt > combined.txt",
			},
			{
				Description: "Find the longest line after expansion in the portable gnux shell:",
				Bash:        "gnux shell -c 'expandx -t 4 main.go | wcx -L'",
				PowerShell:  "gnux shell -c 'expandx -t 4 main.go | wcx -L'",
			},
		},
	}
}

// Register installs the expandx flags on fs.
func Register(fs *pflag.FlagSet) {
	tabs := obsolete.RegisterList(fs, "N, LIST")
	fs.BoolP("initial", "i", false, "do not convert tabs after non blanks")
	fs.VarP(tabs, "tabs", "t", "have tabs N characters apart, not 8, or use comma separated list of explicit tab positions")
}

// buildOptions folds the parsed flags into expand.Options.
func buildOptions(fs *pflag.FlagSet) expand.Options {
	initial, _ := fs.GetBool("initial")
	return expand.Options{Initial: initial, Tabs: obsolete.LookupList(fs).Items}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return expand.New(buildOptions(fs), s).Run(args)
}
