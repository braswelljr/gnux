// Package catx wires the catx command (GNU cat) to its engine.
package catx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cat"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the catx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "cat",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for catx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    cat.Name,
		SeeAlso: []string{"headx", "tailx", "wcx", "echox"},
		Use:     cat.Name + " [OPTION]... [FILE]...",
		Short:   "Concatenate files and print them on standard output.",
		Long: "catx writes each FILE to standard output, one after another. With no FILE, or " +
			"when FILE is -, it reads standard input. Without options the bytes are copied " +
			"untouched, so it is safe for binary data. -n numbers every output line and -b only " +
			"the non-empty ones, with the numbering continuing across files; -s squeezes runs " +
			"of empty lines into one. -E marks line ends with $, -T shows tabs as ^I and -v " +
			"shows other control and non-ASCII bytes in ^X and M- notation; -A turns all three " +
			"on. A file that cannot be read is reported and the remaining files are still " +
			"printed.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified; a directory operand is " +
			"reported as 'Is a directory'.",
		UseCases: []string{
			"Printing a file, or joining several into one stream or file.",
			"Feeding a file or standard input into a pipeline.",
			"Numbering the lines of a listing with -n.",
			"Finding stray tabs, trailing spaces, CRLF line endings or control characters with -A.",
			"Appending one file to another with >>, such as adding a section to a changelog.",
			"Collapsing runs of blank lines in generated text with -s.",
			"Reassembling a file that was split into numbered parts.",
			"Writing a here-document or piped text into a file from a script.",
			"Checking that a set of files is readable, relying on the exit status.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print a file:",
				Bash:        "catx notes.txt",
				PowerShell:  "catx notes.txt",
			},
			{
				Description: "Join several files into one:",
				Bash:        "catx part1.txt part2.txt > whole.txt",
				PowerShell:  "catx part1.txt part2.txt | Out-File -Encoding utf8 whole.txt",
			},
			{
				Description: "Reassemble a file split into numbered parts (binary safe; on Windows let the gnux shell do the redirection):",
				Bash:        "catx archive.tar.part-* > archive.tar",
				PowerShell:  "gnux shell -c 'catx archive.tar.part-* > archive.tar'",
			},
			{
				Description: "Append a file to the end of another:",
				Bash:        "catx release-notes.txt >> CHANGELOG.txt",
				PowerShell:  "catx release-notes.txt >> CHANGELOG.txt",
			},
			{
				Description: "Number the lines of a file:",
				Bash:        "catx -n main.go",
				PowerShell:  "catx -n main.go",
			},
			{
				Description: "Number only the non-empty lines:",
				Bash:        "catx -b poem.txt",
				PowerShell:  "catx -b poem.txt",
			},
			{
				Description: "Squeeze runs of blank lines into one, then number the result:",
				Bash:        "catx -sn draft.txt",
				PowerShell:  "catx -sn draft.txt",
			},
			{
				Description: "Reveal tabs, line ends and control characters:",
				Bash:        "catx -A Makefile",
				PowerShell:  "catx -A Makefile",
			},
			{
				Description: "Find trailing spaces and Windows line endings (shown as ^M$):",
				Bash:        "catx -e config.ini",
				PowerShell:  "catx -e config.ini",
			},
			{
				Description: "Show only line ends, or only tabs:",
				Bash:        "catx -E list.txt && catx -T Makefile",
				PowerShell:  "catx -E list.txt; catx -T Makefile",
			},
			{
				Description: "Show tabs and nonprinting bytes, but not line ends:",
				Bash:        "catx -t data.tsv",
				PowerShell:  "catx -t data.tsv",
			},
			{
				Description: "Make control and non-ASCII bytes visible in ^X and M- notation:",
				Bash:        "catx -v binary.dat | headx -n 5",
				PowerShell:  "catx -v binary.dat | headx -n 5",
			},
			{
				Description: "Put standard input between two files (-u is accepted and ignored):",
				Bash:        "echox '---' | catx -u header.txt - footer.txt",
				PowerShell:  "'---' | catx -u header.txt - footer.txt",
			},
			{
				Description: "Write piped text into a file:",
				Bash:        "echox 'hello, world' | catx > greeting.txt",
				PowerShell:  "'hello, world' | catx > greeting.txt",
			},
			{
				Description: "Count the lines of several files combined:",
				Bash:        "catx *.log | wcx -l",
				PowerShell:  "catx (Get-ChildItem *.log -Name) | wcx -l",
			},
			{
				Description: "Report unreadable files but keep going, then act on the exit status:",
				Bash:        "catx a.txt missing.txt b.txt > all.txt || echox 'some files could not be read'",
				PowerShell:  "catx a.txt missing.txt b.txt > all.txt; if ($LASTEXITCODE -ne 0) { 'some files could not be read' }",
			},
			{
				Description: "Concatenate files in the portable gnux shell:",
				Bash:        "gnux shell -c 'catx header.txt *.md > book.md && wcx -l book.md'",
				PowerShell:  "gnux shell -c 'catx header.txt *.md > book.md && wcx -l book.md'",
			},
		},
	}
}

// Register installs the catx flags on fs. GNU's -e, -t and -u have no long
// form; pflag requires one, so they get descriptive names.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("show-all", "A", false, "equivalent to -vET")
	fs.BoolP("number-nonblank", "b", false, "number nonempty output lines, overrides -n")
	fs.BoolP("show-nonprinting-ends", "e", false, "equivalent to -vE")
	fs.BoolP("show-ends", "E", false, "display $ at end of each line")
	fs.BoolP("number", "n", false, "number all output lines")
	fs.BoolP("squeeze-blank", "s", false, "suppress repeated empty output lines")
	fs.BoolP("show-nonprinting-tabs", "t", false, "equivalent to -vT")
	fs.BoolP("show-tabs", "T", false, "display TAB characters as ^I")
	fs.BoolP("unbuffered", "u", false, "(ignored)")
	fs.BoolP("show-nonprinting", "v", false, "use ^ and M- notation, except for LFD and TAB")
}

// buildOptions folds the parsed flags into cat.Options, expanding the
// compound flags into the three display switches they stand for.
func buildOptions(fs *pflag.FlagSet) cat.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	all, e, t := on("show-all"), on("show-nonprinting-ends"), on("show-nonprinting-tabs")
	return cat.Options{
		Number:          on("number"),
		NumberNonblank:  on("number-nonblank"),
		SqueezeBlank:    on("squeeze-blank"),
		ShowEnds:        on("show-ends") || all || e,
		ShowTabs:        on("show-tabs") || all || t,
		ShowNonprinting: on("show-nonprinting") || all || e || t,
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return cat.New(buildOptions(fs), s).Run(args)
}
