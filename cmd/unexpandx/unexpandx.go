// Package unexpandx wires the unexpandx command (GNU unexpand) to its engine.
package unexpandx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unexpand"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/obsolete"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the unexpandx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "unexpand",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for unexpandx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    unexpand.Name,
		SeeAlso: []string{"expandx", "foldx", "trx"},
		Use:     unexpand.Name + " [OPTION]... [FILE]...",
		Short:   "Convert spaces to tabs.",
		Long: "unexpandx writes each FILE to standard output with runs of blanks replaced by " +
			"tabs where they reach a tab stop. With no FILE, or when FILE is -, it reads " +
			"standard input. By default only the blanks at the start of each line are " +
			"converted; -a converts every run of two or more blanks that ends at a tab stop, " +
			"and --first-only goes back to the leading blanks only. Tab stops are every 8 " +
			"columns unless -t says otherwise, with the same syntax as expandx: -t N, or -t " +
			"LIST with an optional final /N or +N. -t implies -a, as in GNU. Several -t " +
			"options add to one list, and the obsolete -N form (unexpandx -4) is a list of its " +
			"own, in the order given, without implying -a; GNU's comma form -4,8 is not " +
			"accepted, and the digits of two -N options in a row run together (-4 -8 reads " +
			"as -48), so write -t 4,8 instead. When both -a and --first-only are given, " +
			"--first-only wins.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Converting space-indented files to tab indentation (Makefiles, Go code).",
			"Shrinking files that are padded with spaces.",
			"Reversing expandx after editing.",
			"Converting the indentation alone, leaving aligned columns inside lines untouched.",
			"Normalizing indentation width with -t before a commit.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Convert leading spaces to tabs every 8 columns:",
				Bash:        "unexpandx notes.txt",
				PowerShell:  "unexpandx notes.txt",
			},
			{
				Description: "Convert all runs of blanks, not just leading ones:",
				Bash:        "unexpandx -a table.txt",
				PowerShell:  "unexpandx -a table.txt",
			},
			{
				Description: "Use 4-column tab stops (implies -a):",
				Bash:        "unexpandx -t 4 main.go",
				PowerShell:  "unexpandx -t 4 main.go",
			},
			{
				Description: "Use 4-column stops but only for indentation:",
				Bash:        "unexpandx -t 4 --first-only main.go",
				PowerShell:  "unexpandx -t 4 --first-only main.go",
			},
			{
				Description: "Use the obsolete form, which converts only leading blanks:",
				Bash:        "unexpandx -4 main.go",
				PowerShell:  "unexpandx -4 main.go",
			},
			{
				Description: "Put tab stops at specific columns:",
				Bash:        "unexpandx -t 10,20,40 report.txt",
				PowerShell:  "unexpandx -t 10,20,40 report.txt",
			},
			{
				Description: "Round-trip a file through expandx:",
				Bash:        "expandx -t 4 a.go | unexpandx -t 4 --first-only > b.go",
				PowerShell:  "expandx -t 4 a.go | unexpandx -t 4 --first-only > b.go",
			},
			{
				Description: "Convert standard input:",
				Bash:        "printfx '        x\\n' | unexpandx",
				PowerShell:  "printfx '        x\\n' | unexpandx",
			},
			{
				Description: "Save a converted copy:",
				Bash:        "unexpandx -t 8 spaces.mk > Makefile",
				PowerShell:  "unexpandx -t 8 spaces.mk > Makefile",
			},
			{
				Description: "See how many bytes the conversion saves:",
				Bash:        "unexpandx -a big.txt | wcx -c",
				PowerShell:  "unexpandx -a big.txt | wcx -c",
			},
			{
				Description: "Convert in the portable gnux shell:",
				Bash:        "gnux shell -c 'unexpandx -t 4 main.go > tabs.go'",
				PowerShell:  "gnux shell -c 'unexpandx -t 4 main.go > tabs.go'",
			},
		},
	}
}

// Register installs the unexpandx flags on fs.
func Register(fs *pflag.FlagSet) {
	tabs := obsolete.RegisterList(fs, "N, LIST")
	fs.BoolP("all", "a", false, "convert all blanks, instead of just initial blanks")
	fs.Bool("first-only", false, "convert only leading sequences of blanks (overrides -a)")
	fs.VarP(tabs, "tabs", "t", "have tabs N characters apart instead of 8 (enables -a), or use comma separated list of explicit tab positions")
}

// buildOptions folds the parsed flags into unexpand.Options.
func buildOptions(fs *pflag.FlagSet) unexpand.Options {
	all, _ := fs.GetBool("all")
	firstOnly, _ := fs.GetBool("first-only")
	return unexpand.Options{
		All:       all || fs.Changed("tabs"),
		FirstOnly: firstOnly,
		Tabs:      obsolete.LookupList(fs).Items,
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return unexpand.New(buildOptions(fs), s).Run(args)
}
