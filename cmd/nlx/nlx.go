// Package nlx wires the nlx command (GNU nl) to its engine.
package nlx

import (
	"errors"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nl"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the nlx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "nl",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for nlx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    nl.Name,
		SeeAlso: []string{"catx", "fmtx", "headx", "tailx"},
		Use:     nl.Name + " [OPTION]... [FILE]...",
		Short:   "Number lines of files.",
		Long: "nlx writes each FILE to standard output with line numbers added. With no FILE, " +
			"or when FILE is -, it reads standard input; several FILEs are numbered as one " +
			"stream. By default non-empty lines are numbered from 1, right-aligned in a " +
			"6-column field followed by a tab, and empty lines get the same width of blanks " +
			"so the text stays aligned. -b STYLE chooses which lines are numbered: a (all), t " +
			"(non-empty, the default), n (none) or pBRE (those matching the POSIX basic " +
			"regular expression BRE). -n FORMAT is ln (left-aligned), rn (right-aligned) or " +
			"rz (zero-padded); -w sets the width, -s the separator, -v the first number and " +
			"-i the increment (both may be zero or negative). -l N with -b a numbers only " +
			"every Nth of a run of empty lines. Input may be split into logical pages of " +
			"header, body and footer sections by lines holding only '\\:\\:\\:', '\\:\\:' or " +
			"'\\:' (change the two characters with -d; an empty -d turns sections off); each " +
			"delimiter prints as an empty line, -h and -f choose the numbering of headers and " +
			"footers (none by default), and numbering restarts at every header unless -p. " +
			"Every output line ends with a newline, a last line without one included.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Quoting a snippet of code or a config file with line numbers in a review or a bug report.",
			"Numbering the entries of a list, such as a playlist or a set of steps.",
			"Finding the line numbers of matching lines with -b p.",
			"Producing zero-padded numbers for fixed-width records with -n rz.",
			"Numbering paginated documents section by section with the \\: delimiters.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Quote a config file with line numbers in a bug report:",
				Bash:        "nlx nginx.conf | headx -n 5",
				PowerShell:  "nlx nginx.conf | headx -n 5",
				Output:      "     1\tuser www-data;\n     2\tworker_processes auto;\n       \n     3\tevents {\n     4\t    worker_connections 768;",
			},
			{
				Description: "Number every line, empty ones included, to match an editor's line numbers:",
				Bash:        "nlx -b a main.go",
				PowerShell:  "nlx -b a main.go",
			},
			{
				Description: "Find which lines of a crontab are active jobs (not comments):",
				Bash:        "nlx -b 'p^[^#]' crontab.txt",
				PowerShell:  "nlx -b 'p^[^#]' crontab.txt",
			},
			{
				Description: "Turn a list of steps into a numbered checklist:",
				Bash:        "nlx -w 2 -s '. ' steps.txt",
				PowerShell:  "nlx -w 2 -s '. ' steps.txt",
				Output:      " 1. Back up the database\n 2. Stop the service\n 3. Run the migration",
			},
			{
				Description: "Give records fixed-width, zero-padded IDs:",
				Bash:        "nlx -n rz -w 5 -s , names.txt > names.csv",
				PowerShell:  "nlx -n rz -w 5 -s , names.txt > names.csv",
				Output:      "00001,Ada\n00002,Grace\n00003,Linus",
			},
			{
				Description: "Continue numbering where yesterday's log left off:",
				Bash:        "nlx -v 1043 today.log",
				PowerShell:  "nlx -v 1043 today.log",
			},
			{
				Description: "Number in steps of 10, leaving room to insert lines later:",
				Bash:        "nlx -i 10 -v 10 program.bas",
				PowerShell:  "nlx -i 10 -v 10 program.bas",
			},
			{
				Description: "Number a git diff's lines to discuss them in a review:",
				Bash:        "git diff | nlx -b a",
				PowerShell:  "git diff | nlx -b a",
			},
			{
				Description: "Number lines left-aligned with no padding before the text:",
				Bash:        "nlx -n ln -w 1 -s ' ' todo.txt",
				PowerShell:  "nlx -n ln -w 1 -s ' ' todo.txt",
			},
			{
				Description: "Number two chapters as one continuous document:",
				Bash:        "nlx chapter1.txt chapter2.txt > book.txt",
				PowerShell:  "nlx chapter1.txt chapter2.txt > book.txt",
			},
			{
				Description: "Collapse runs of empty lines so only every second one gets a number:",
				Bash:        "nlx -b a -l 2 poem.txt",
				PowerShell:  "nlx -b a -l 2 poem.txt",
			},
			{
				Description: "Number lines in the portable gnux shell:",
				Bash:        "gnux shell -c 'sortx names.txt | uniqx | nlx -w 3'",
				PowerShell:  "gnux shell -c 'sortx names.txt | uniqx | nlx -w 3'",
			},
		},
	}
}

// Register installs the nlx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("body-numbering", "b", "", "use `STYLE` for numbering body lines (a, t, n or pBRE; default t)")
	fs.StringP("section-delimiter", "d", "", "use `CC` for logical page delimiters (default \\:)")
	fs.StringP("footer-numbering", "f", "", "use `STYLE` for numbering footer lines (default n)")
	fs.StringP("header-numbering", "h", "", "use `STYLE` for numbering header lines (default n)")
	fs.StringP("line-increment", "i", "", "line number increment at each line (default 1)")
	fs.StringP("join-blank-lines", "l", "", "group of `NUMBER` empty lines counted as one (default 1)")
	fs.StringP("number-format", "n", "", "insert line numbers according to `FORMAT` (ln, rn or rz; default rn)")
	fs.BoolP("no-renumber", "p", false, "do not reset line numbers for each section")
	fs.StringP("number-separator", "s", "", "add `STRING` after (possible) line number (default TAB)")
	fs.StringP("starting-line-number", "v", "", "first line number for each section (default 1)")
	fs.StringP("number-width", "w", "", "use `NUMBER` columns for line numbers (default 6)")
}

// fatalError is a mistake GNU nl reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// buildOptions folds the parsed flags into nl.Options.
func buildOptions(fs *pflag.FlagSet) (nl.Options, error) {
	str := func(name string) string {
		s, _ := fs.GetString(name)
		return s
	}
	o := nl.Options{
		BodyStyle:    str("body-numbering"),
		HeaderStyle:  str("header-numbering"),
		FooterStyle:  str("footer-numbering"),
		Delimiter:    str("section-delimiter"),
		DelimiterSet: fs.Changed("section-delimiter"),
		Format:       str("number-format"),
		Separator:    str("number-separator"),
		SeparatorSet: fs.Changed("number-separator"),
	}
	o.NoRenumber, _ = fs.GetBool("no-renumber")

	integer := func(name, what string) (int64, bool, error) {
		if !fs.Changed(name) {
			return 0, false, nil
		}
		n, err := strconv.ParseInt(str(name), 10, 64)
		if err != nil {
			return 0, false, &fatalError{"invalid " + what + ": '" + str(name) + "'"}
		}
		return n, true, nil
	}
	positive := func(name, what string) (int, error) {
		n, set, err := integer(name, what)
		if err == nil && set && (n < 1 || n > 1<<31-1) {
			err = &fatalError{"invalid " + what + ": '" + str(name) + "'"}
		}
		return int(n), err
	}
	var err error
	if o.Increment, o.IncrementSet, err = integer("line-increment", "line number increment"); err != nil {
		return nl.Options{}, err
	}
	if o.StartingLine, o.StartingLineSet, err = integer("starting-line-number", "starting line number"); err != nil {
		return nl.Options{}, err
	}
	if o.JoinBlank, err = positive("join-blank-lines", "line number of blank lines"); err != nil {
		return nl.Options{}, err
	}
	if o.Width, err = positive("number-width", "line number field width"); err != nil {
		return nl.Options{}, err
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(nl.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return nl.New(opts, s).Run(args)
}
