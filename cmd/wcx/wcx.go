// Package wcx wires the wcx command (GNU wc) to its engine.
package wcx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/wc"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the wcx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "wc",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for wcx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    wc.Name,
		SeeAlso: []string{"catx", "headx", "tailx", "lsx"},
		Use:     wc.Name + " [OPTION]... [FILE]...",
		Short:   "Print newline, word, and byte counts for each file.",
		Long: "wcx prints the number of lines, words and bytes in each FILE, followed by the " +
			"file name, and a 'total' line when more than one FILE is given. With no FILE, or " +
			"when FILE is -, it reads standard input. -l, -w, -m, -c and -L select which counts " +
			"to print: newlines, words, characters, bytes and the display width of the longest " +
			"line. Whatever the order of the options, the counts are always printed in the " +
			"order lines, words, characters, bytes, maximum line length. A word is a run of " +
			"non-whitespace characters; input is decoded as UTF-8, and bytes that are not valid " +
			"UTF-8 count toward words and bytes but not characters. --total=WHEN controls the " +
			"summary line and --files0-from=F reads the file names, NUL-terminated, from F. " +
			"Columns are right-aligned to a width derived from the files' sizes, as in GNU wc.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. With -c alone on a regular file only its metadata is " +
			"consulted, so search permission on the directory is enough. Nothing is ever " +
			"modified.",
		UseCases: []string{
			"Counting the lines of a file or of a pipeline's output with -l.",
			"Measuring the size of a set of source files, with a total.",
			"Checking a text against a word or character limit.",
			"Finding the widest line of a file with -L before formatting it.",
			"Getting a file's size in bytes cheaply with -c.",
			"Counting the entries in a directory by piping lsx into it.",
			"Getting a single grand total for a large, generated list of files with --files0-from and --total=only.",
			"Verifying in a script that a file is not empty, or that output has an expected length.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Count lines, words and bytes of a file:",
				Bash:        "wcx README.md",
				PowerShell:  "wcx README.md",
			},
			{
				Description: "Count only the lines:",
				Bash:        "wcx -l access.log",
				PowerShell:  "wcx -l access.log",
			},
			{
				Description: "Count the words of an essay:",
				Bash:        "wcx -w essay.txt",
				PowerShell:  "wcx -w essay.txt",
			},
			{
				Description: "Print the size of a file in bytes:",
				Bash:        "wcx -c image.png",
				PowerShell:  "wcx -c image.png",
			},
			{
				Description: "Count characters rather than bytes:",
				Bash:        "wcx -m unicode.txt",
				PowerShell:  "wcx -m unicode.txt",
			},
			{
				Description: "Find the display width of the longest line:",
				Bash:        "wcx -L README.md",
				PowerShell:  "wcx -L README.md",
			},
			{
				Description: "Print lines and words together (always in that order, whatever the flag order):",
				Bash:        "wcx -wl essay.txt",
				PowerShell:  "wcx -wl essay.txt",
			},
			{
				Description: "Count the lines of several files, with a total:",
				Bash:        "wcx -l *.go",
				PowerShell:  "wcx -l (Get-ChildItem *.go -Name)",
			},
			{
				Description: "Count several files without the total line, or force it for a single file:",
				Bash:        "wcx -l --total=never *.go && wcx -l --total=always main.go",
				PowerShell:  "wcx -l --total=never (Get-ChildItem *.go -Name); wcx -l --total=always main.go",
			},
			{
				Description: "Count the entries a command produces:",
				Bash:        "lsx | wcx -l",
				PowerShell:  "lsx | wcx -l",
			},
			{
				Description: "Count the words of text from standard input:",
				Bash:        "echox 'the quick brown fox' | wcx -w",
				PowerShell:  "'the quick brown fox' | wcx -w",
			},
			{
				Description: "Read the list of files to count from a NUL-separated file:",
				Bash:        "wcx -l --files0-from=sources.lst",
				PowerShell:  "wcx -l --files0-from=sources.lst",
			},
			{
				Description: "Print only the grand total for a list of files found elsewhere:",
				Bash:        "find . -name '*.go' -print0 | wcx -l --files0-from=- --total=only",
				PowerShell:  "wcx -l --total=only (Get-ChildItem -Recurse *.go).FullName",
			},
			{
				Description: "Fail a script when a file is missing (exit status 1):",
				Bash:        "wcx -l missing.txt || echox \"exit status $?\"",
				PowerShell:  "wcx -l missing.txt; if ($LASTEXITCODE -ne 0) { \"exit status $LASTEXITCODE\" }",
			},
			{
				Description: "Count matching files in the portable gnux shell:",
				Bash:        "gnux shell -c 'lsx -1 *.md | wcx -l'",
				PowerShell:  "gnux shell -c 'lsx -1 *.md | wcx -l'",
			},
		},
	}
}

// Register installs the wcx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("bytes", "c", false, "print the byte counts")
	fs.BoolP("chars", "m", false, "print the character counts")
	fs.BoolP("lines", "l", false, "print the newline counts")
	fs.String("files0-from", "",
		"read input from the files specified by NUL-terminated names in file `F`; if F is - then read names from standard input")
	fs.BoolP("max-line-length", "L", false, "print the maximum display width")
	fs.BoolP("words", "w", false, "print the word counts")
	fs.String("total", "auto",
		"when to print a line with total counts; `WHEN` can be: auto, always, only, never")
}

// totalModes maps the --total argument to its mode.
var totalModes = map[string]wc.TotalMode{
	"auto":   wc.TotalAuto,
	"always": wc.TotalAlways,
	"only":   wc.TotalOnly,
	"never":  wc.TotalNever,
}

// buildOptions folds the parsed flags into wc.Options.
func buildOptions(fs *pflag.FlagSet) (wc.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := wc.Options{
		Lines:         on("lines"),
		Words:         on("words"),
		Chars:         on("chars"),
		Bytes:         on("bytes"),
		MaxLineLength: on("max-line-length"),
	}
	o.Files0From, _ = fs.GetString("files0-from")

	when, _ := fs.GetString("total")
	mode, ok := totalModes[when]
	if !ok {
		return wc.Options{}, core.Usagef("invalid argument '%s' for '--total'", when)
	}
	o.Total = mode
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return wc.New(opts, s).Run(args)
}
