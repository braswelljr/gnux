// Package commx wires the commx command (GNU comm) to its engine.
package commx

import (
	"errors"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/comm"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the commx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "comm",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for commx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    comm.Name,
		SeeAlso: []string{"sortx", "uniqx", "joinx"},
		Use:     comm.Name + " [OPTION]... FILE1 FILE2",
		Short:   "Compare two sorted files line by line.",
		Long: "commx compares two sorted files, FILE1 and FILE2, line by line. With no options " +
			"it prints three columns: lines only in FILE1, lines only in FILE2, and lines in " +
			"both, each column indented by one more TAB than the one before. -1, -2 and -3 " +
			"suppress the matching column, so -12 prints the common lines and -23 the lines only " +
			"in FILE1. Either FILE may be - for standard input. Both files must be sorted: by " +
			"default commx warns once per file when it finds a line out of order after a line " +
			"that appears in only one file, and then exits with status 1; --check-order fails at " +
			"the first line out of order, and --nocheck-order never checks. " +
			"--output-delimiter=STR separates the columns with STR (an empty STR writes a NUL " +
			"byte), --total adds a final line with the three counts and the word 'total', and " +
			"-z makes NUL the line delimiter. Lines are compared byte by byte, as in the C " +
			"locale, whatever LANG says, so sort the inputs with sortx (or LC_ALL=C sort). The " +
			"framework requires a long name for every option, so -1, -2 and -3 also answer to " +
			"gnux's own --suppress-file1, --suppress-file2 and --suppress-common. When both " +
			"--check-order and --nocheck-order are given, --check-order wins.",
		Permissions: "Needs read permission on FILE1 and FILE2 and search permission on the " +
			"directories leading to them. Nothing is ever modified.",
		UseCases: []string{
			"Finding the lines two sorted lists have in common with -12.",
			"Finding what was added to or removed from a list between two versions.",
			"Comparing the files in two directories by comparing their sorted listings.",
			"Checking which required packages or users are missing from an inventory.",
			"Counting how many lines two lists share with --total.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Show the three columns for two sorted files:",
				Bash:        "commx old.txt new.txt",
				PowerShell:  "commx old.txt new.txt",
			},
			{
				Description: "Print only the lines the files have in common:",
				Bash:        "commx -12 old.txt new.txt",
				PowerShell:  "commx -12 old.txt new.txt",
			},
			{
				Description: "Print the lines only in the first file (removed lines):",
				Bash:        "commx -23 old.txt new.txt",
				PowerShell:  "commx -23 old.txt new.txt",
			},
			{
				Description: "Print the lines only in the second file (added lines):",
				Bash:        "commx -13 old.txt new.txt",
				PowerShell:  "commx -13 old.txt new.txt",
			},
			{
				Description: "Compare two unsorted lists by sorting them first:",
				Bash:        "sortx a.txt > a.sorted && sortx b.txt > b.sorted && commx a.sorted b.sorted",
				PowerShell:  "sortx a.txt > a.sorted; sortx b.txt > b.sorted; commx a.sorted b.sorted",
			},
			{
				Description: "Compare standard input with a file:",
				Bash:        "lsx | commx - expected.txt",
				PowerShell:  "lsx | commx - expected.txt",
			},
			{
				Description: "Separate the columns with a comma:",
				Bash:        "commx --output-delimiter=, old.txt new.txt",
				PowerShell:  "commx --output-delimiter=, old.txt new.txt",
			},
			{
				Description: "Count the lines unique to each file and common to both:",
				Bash:        "commx --total -123 old.txt new.txt",
				PowerShell:  "commx --total -123 old.txt new.txt",
			},
			{
				Description: "Fail at the first line out of order:",
				Bash:        "commx --check-order old.txt new.txt || echox 'not sorted'",
				PowerShell:  "commx --check-order old.txt new.txt; if ($LASTEXITCODE -ne 0) { 'not sorted' }",
			},
			{
				Description: "Skip the order check:",
				Bash:        "commx --nocheck-order old.txt new.txt",
				PowerShell:  "commx --nocheck-order old.txt new.txt",
			},
			{
				Description: "Compare NUL-terminated lists:",
				Bash:        "commx -z -12 a.lst b.lst",
				PowerShell:  "commx -z -12 a.lst b.lst",
			},
			{
				Description: "Find common lines in the portable gnux shell:",
				Bash:        "gnux shell -c 'commx -12 old.txt new.txt | wcx -l'",
				PowerShell:  "gnux shell -c 'commx -12 old.txt new.txt | wcx -l'",
			},
		},
	}
}

// Register installs the commx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("suppress-file1", "1", false, "suppress column 1 (lines unique to FILE1)")
	fs.BoolP("suppress-file2", "2", false, "suppress column 2 (lines unique to FILE2)")
	fs.BoolP("suppress-common", "3", false, "suppress column 3 (lines that appear in both files)")
	fs.Bool("check-order", false, "check that the input is correctly sorted, even if all input lines are pairable")
	fs.Bool("nocheck-order", false, "do not check that the input is correctly sorted")
	fs.StringArray("output-delimiter", nil, "separate columns with `STR`")
	fs.Bool("total", false, "output a summary")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "suppress-common", "suppress-file1", "suppress-file2")
}

// fatalError is a mistake GNU comm reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// buildOptions folds the parsed flags into comm.Options.
func buildOptions(fs *pflag.FlagSet) (comm.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := comm.Options{
		Suppress1:      on("suppress-file1"),
		Suppress2:      on("suppress-file2"),
		Suppress3:      on("suppress-common"),
		Total:          on("total"),
		ZeroTerminated: on("zero-terminated"),
	}
	// The cautious choice wins whatever the order: checking.
	switch {
	case on("check-order"):
		o.Check = comm.CheckOrder
	case on("nocheck-order"):
		o.Check = comm.NoCheckOrder
	}
	// Read the values directly: GetStringArray loses a lone empty value, and
	// an empty delimiter is meaningful (a NUL byte).
	delims := fs.Lookup("output-delimiter").Value.(pflag.SliceValue).GetSlice()
	for _, d := range delims {
		if o.OutputDelimiterSet && d != o.OutputDelimiter {
			return comm.Options{}, &fatalError{"multiple output delimiters specified"}
		}
		o.OutputDelimiter, o.OutputDelimiterSet = d, true
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(comm.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return comm.New(opts, s).Run(args)
}
