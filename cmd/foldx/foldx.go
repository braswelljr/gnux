// Package foldx wires the foldx command (GNU fold) to its engine.
package foldx

import (
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fold"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/obsolete"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the foldx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "fold",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for foldx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    fold.Name,
		SeeAlso: []string{"fmtx", "cutx", "expandx"},
		Use:     fold.Name + " [OPTION]... [FILE]...",
		Short:   "Wrap each input line to fit in a specified width.",
		Long: "foldx writes each FILE to standard output, breaking any line longer than 80 " +
			"columns into several. With no FILE, or when FILE is -, it reads standard input. " +
			"-w WIDTH sets the width; the obsolete form -WIDTH (foldx -72) is accepted too, and " +
			"whichever is given last wins. Columns are counted as a terminal would: a tab " +
			"advances to the next multiple of 8, a backspace moves back one column and a " +
			"carriage return goes back to column 0. -b counts bytes instead, so every byte " +
			"takes one position. -s breaks a long line after the last blank that fits, rather " +
			"than exactly at the width, so words are not split unless one is longer than the " +
			"width. Like GNU fold in the C locale, foldx counts bytes rather than characters, " +
			"so a multibyte UTF-8 character takes more than one column.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Wrapping long log lines or generated text to fit a terminal or an email.",
			"Splitting a long base64 or hex string into fixed-width lines.",
			"Breaking prose at word boundaries with -s.",
			"Chunking a single-line file into records of a fixed number of bytes with -b.",
			"Preparing text for a fixed-width display or printer.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Make a log with very long lines readable in an 80-column terminal:",
				Bash:        "foldx notes.txt",
				PowerShell:  "foldx notes.txt",
			},
			{
				Description: "Fit text into a narrow 40-column display or receipt printer:",
				Bash:        "foldx -w 40 notes.txt",
				PowerShell:  "foldx -w 40 notes.txt",
			},
			{
				Description: "Wrap a commit message body at 72 columns without breaking words:",
				Bash:        "foldx -s -w 72 essay.txt",
				PowerShell:  "foldx -s -w 72 essay.txt",
			},
			{
				Description: "Run an old script that uses the obsolete width form:",
				Bash:        "foldx -60 notes.txt",
				PowerShell:  "foldx -60 notes.txt",
			},
			{
				Description: "Break a one-line key or base64 blob into 64-byte lines:",
				Bash:        "foldx -b -w 64 key.txt",
				PowerShell:  "foldx -b -w 64 key.txt",
			},
			{
				Description: "Wrap the output of another command:",
				Bash:        "catx long.log | foldx -w 100",
				PowerShell:  "catx long.log | foldx -w 100",
			},
			{
				Description: "Split a word into one character per line, to count or sort letters:",
				Bash:        "echox abc | foldx -w 1",
				PowerShell:  "echox abc | foldx -w 1",
			},
			{
				Description: "Save a wrapped copy of a draft for an email client that does not wrap:",
				Bash:        "foldx -s -w 72 draft.txt > wrapped.txt",
				PowerShell:  "foldx -s -w 72 draft.txt > wrapped.txt",
			},
			{
				Description: "Wrap several chapter files in one go:",
				Bash:        "foldx -w 72 a.txt b.txt",
				PowerShell:  "foldx -w 72 a.txt b.txt",
			},
			{
				Description: "Check how many screen lines a note will take once wrapped:",
				Bash:        "foldx -w 72 notes.txt | wcx -l",
				PowerShell:  "foldx -w 72 notes.txt | wcx -l",
			},
			{
				Description: "Preview a wrapped file in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'foldx -s -w 60 notes.txt | headx'",
				PowerShell:  "gnux shell -c 'foldx -s -w 60 notes.txt | headx'",
			},
		},
	}
}

// Register installs the foldx flags on fs.
func Register(fs *pflag.FlagSet) {
	ord := &obsolete.Order{}
	fs.BoolP("bytes", "b", false, "count bytes rather than columns")
	fs.BoolP("spaces", "s", false, "break at spaces")
	fs.VarP(obsolete.NewString(ord, "WIDTH"), "width", "w", "use WIDTH columns instead of 80")
	obsolete.Register(fs, ord)
}

// buildOptions folds the parsed flags into fold.Options.
func buildOptions(fs *pflag.FlagSet) (fold.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := fold.Options{Bytes: on("bytes"), Spaces: on("spaces")}
	width := fs.Lookup("width").Value.(*obsolete.String)
	text, ok := obsolete.Last(width, obsolete.Lookup(fs))
	if ok {
		n, err := strconv.ParseUint(text, 10, 31)
		if err != nil || n == 0 {
			return fold.Options{}, core.Usagef("invalid number of columns: '%s'", text)
		}
		o.Width = int(n)
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return fold.New(opts, s).Run(args)
}
