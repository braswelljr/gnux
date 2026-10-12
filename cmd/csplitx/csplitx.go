// Package csplitx wires the csplitx command (GNU csplit) to its engine.
package csplitx

import (
	"errors"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/csplit"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the csplitx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "csplit",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for csplitx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    csplit.Name,
		SeeAlso: []string{"splitx", "headx", "tailx", "catx"},
		Use:     csplit.Name + " [OPTION]... FILE PATTERN...",
		Short:   "Split a file into sections determined by context lines.",
		Long: "csplitx copies FILE (- for standard input) into files xx00, xx01, ..., starting a " +
			"new file at each PATTERN, and prints the size in bytes of each file it makes. A " +
			"PATTERN is a line number N (the next file starts at line N), /REGEXP/ (the next " +
			"file starts at the next line matching the POSIX basic regular expression " +
			"REGEXP), or %REGEXP% (the lines up to that match are skipped, not written). A " +
			"regular expression may be followed by an offset, +N or -N, that moves the split " +
			"that many lines after or before the matching line. A pattern followed by {N} is " +
			"repeated N more times, and by {*} as many times as it matches; a line number " +
			"repeated advances by its own value each time. The lines after the last split go " +
			"to a final file. -f PREFIX replaces xx, -n N uses N digits, and -b FORMAT names " +
			"the files with a printf format holding one integer conversion (%02d.txt, %03x). " +
			"-s (or -q) does not print the sizes, -z removes empty files (and reuses their " +
			"number), and --suppress-matched leaves the matching lines out. If a pattern " +
			"cannot be satisfied (a line number past the end, a regular expression that never " +
			"matches), csplitx reports it and removes every file it made, so a failed run " +
			"leaves nothing half-done behind; -k keeps them. Lines are compared as bytes.",
		Permissions: "Needs read permission on FILE and write permission on the directory the " +
			"files are created in.",
		UseCases: []string{
			"Splitting a book or a long Markdown document into one file per chapter.",
			"Breaking a concatenated log into one file per day or per request.",
			"Separating the sections of a config file or a multi-document YAML stream.",
			"Cutting off a file's header or preamble with %REGEXP%.",
			"Pulling the first N lines into one file and the rest into another.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Split a Markdown book into one file per chapter (each starting with '# '):",
				Bash:        "csplitx -f chapter- -b '%02d.md' book.md '/^# /' '{*}'",
				PowerShell:  "csplitx -f chapter- -b '%02d.md' book.md '/^# /' '{*}'",
				Output:      "0\n18234\n22917\n15002",
			},
			{
				Description: "Do the same without the empty file before the first chapter:",
				Bash:        "csplitx -z -f chapter- -b '%02d.md' book.md '/^# /' '{*}'",
				PowerShell:  "csplitx -z -f chapter- -b '%02d.md' book.md '/^# /' '{*}'",
			},
			{
				Description: "Split a multi-document Kubernetes YAML into one manifest per file, dropping the '---' lines:",
				Bash:        "csplitx -z --suppress-matched -f manifest- -b '%02d.yaml' all.yaml '/^---$/' '{*}'",
				PowerShell:  "csplitx -z --suppress-matched -f manifest- -b '%02d.yaml' all.yaml '/^---$/' '{*}'",
			},
			{
				Description: "Separate the 3-line header of a report from its body:",
				Bash:        "csplitx -s -f report- report.txt 4",
				PowerShell:  "csplitx -s -f report- report.txt 4",
				Output:      "(creates report-00 with lines 1-3 and report-01 with the rest)",
			},
			{
				Description: "Cut a log into one file per day, where each day starts with a '=== 2026-' banner:",
				Bash:        "csplitx -z -f day- app.log '/^=== 2026-/' '{*}'",
				PowerShell:  "csplitx -z -f day- app.log '/^=== 2026-/' '{*}'",
			},
			{
				Description: "Drop everything before the line 'BEGIN DATA' and keep the rest:",
				Bash:        "csplitx -s export.txt '%^BEGIN DATA%1'",
				PowerShell:  "csplitx -s export.txt '%^BEGIN DATA%1'",
			},
			{
				Description: "Start each file two lines before the match, to keep a heading's preceding blank line and title:",
				Bash:        "csplitx notes.txt '/^-----$/-2' '{*}'",
				PowerShell:  "csplitx notes.txt '/^-----$/-2' '{*}'",
			},
			{
				Description: "Split a 400-line file into 100-line pieces:",
				Bash:        "csplitx data.txt 101 '{2}'",
				PowerShell:  "csplitx data.txt 101 '{2}'",
				Output:      "4290\n4400\n4400\n4380",
			},
			{
				Description: "Keep the pieces even if a later pattern fails:",
				Bash:        "csplitx -k data.txt 100 200 5000",
				PowerShell:  "csplitx -k data.txt 100 200 5000",
				Output:      "2890\n2900\ncsplitx: '5000': line number out of range",
			},
			{
				Description: "Split standard input from another command:",
				Bash:        "git log --format='commit %H%n%B' | csplitx -s -z -f commit- - '/^commit /' '{*}'",
				PowerShell:  "git log --format='commit %H%n%B' | csplitx -s -z -f commit- - '/^commit /' '{*}'",
			},
			{
				Description: "Split by chapter in the portable gnux shell and count the pieces:",
				Bash:        "gnux shell -c 'csplitx -s -z -f ch- book.md \"/^# /\" \"{*}\" && lsx ch-* | wcx -l'",
				PowerShell:  "gnux shell -c 'csplitx -s -z -f ch- book.md \"/^# /\" \"{*}\" && lsx ch-* | wcx -l'",
			},
		},
	}
}

// Register installs the csplitx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("suffix-format", "b", "", "use sprintf `FORMAT` instead of %02d")
	fs.StringP("prefix", "f", "", "use `PREFIX` instead of 'xx'")
	fs.BoolP("keep-files", "k", false, "do not remove output files on errors")
	fs.Bool("suppress-matched", false, "suppress the lines matching PATTERN")
	fs.StringP("digits", "n", "", "use specified number of `digits` instead of 2")
	fs.BoolP("quiet", "s", false, "do not print counts of output file sizes")
	fs.BoolP("silent", "q", false, "same as --quiet")
	fs.BoolP("elide-empty-files", "z", false, "suppress generation of empty output files")
}

// fatalError is a mistake GNU csplit reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// buildOptions folds the parsed flags into csplit.Options.
func buildOptions(fs *pflag.FlagSet) (csplit.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := csplit.Options{
		KeepFiles:       on("keep-files"),
		SuppressMatched: on("suppress-matched"),
		Quiet:           on("quiet") || on("silent"),
		ElideEmpty:      on("elide-empty-files"),
	}
	o.SuffixFormat, _ = fs.GetString("suffix-format")
	o.Prefix, _ = fs.GetString("prefix")
	o.PrefixSet = fs.Changed("prefix")
	if fs.Changed("digits") {
		text, _ := fs.GetString("digits")
		n, err := strconv.ParseUint(text, 10, 31)
		if err != nil {
			return o, &fatalError{"invalid number: '" + text + "'"}
		}
		// No padding at all reads the same as one digit.
		o.Digits = max(int(n), 1)
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(csplit.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return csplit.New(opts, s).Run(args)
}
