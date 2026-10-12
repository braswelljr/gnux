// Package cutx wires the cutx command (GNU cut) to its engine.
package cutx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cut"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the cutx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "cut",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for cutx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    cut.Name,
		SeeAlso: []string{"uniqx", "headx", "tailx", "wcx"},
		Use:     cut.Name + " OPTION... [FILE]...",
		Short:   "Remove sections from each line of files.",
		Long: "cutx prints selected parts of each line of each FILE to standard output. With no " +
			"FILE, or when FILE is -, it reads standard input. Exactly one of -b (bytes), -c " +
			"(characters) or -f (fields) must be given, with a LIST of positions: N, N- (from N " +
			"to the end of the line), N-M, or -M (from the first to M), separated by commas or " +
			"blanks. Positions are numbered from 1, and the selection is printed in input order " +
			"whatever the order of the LIST, each position at most once. With -f the line is " +
			"split on TAB, or on the single character given with -d (-d '' means NUL); a line " +
			"with no delimiter is printed whole, unless -s suppresses it. --complement selects " +
			"everything the LIST does not. --output-delimiter=STRING is written between the " +
			"selected fields (the input delimiter by default), and with -b or -c between " +
			"non-overlapping ranges. -z makes NUL the line delimiter. Every output line ends " +
			"with the line delimiter. As in GNU cut, -c counts bytes, not multibyte " +
			"characters, so -c and -b are the same, and -n is accepted and ignored. One " +
			"difference from GNU: giving the same kind of list twice (-f1 -f2) keeps the last " +
			"one instead of refusing.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Extracting columns from CSV, TSV or colon-separated files such as /etc/passwd.",
			"Taking a fixed-width slice of every line, such as a date prefix of a log.",
			"Dropping a column with --complement.",
			"Turning a delimited file into another delimiter with --output-delimiter.",
			"Skipping header or comment lines that have no delimiter with -s.",
			"Pulling one value out of each line of a command's output in a script.",
			"Splitting NUL-separated records safely with -z.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the first field of each line of a TAB-separated file:",
				Bash:        "cutx -f 1 data.tsv",
				PowerShell:  "cutx -f 1 data.tsv",
			},
			{
				Description: "Print the user names and shells from a colon-separated file:",
				Bash:        "cutx -d : -f 1,7 /etc/passwd",
				PowerShell:  "cutx -d : -f 1,7 passwd.txt",
			},
			{
				Description: "Print the second and third columns of a CSV file:",
				Bash:        "cutx -d , -f 2-3 people.csv",
				PowerShell:  "cutx -d , -f 2-3 people.csv",
			},
			{
				Description: "Print every column from the third on:",
				Bash:        "cutx -d , -f 3- people.csv",
				PowerShell:  "cutx -d , -f 3- people.csv",
			},
			{
				Description: "Drop the second column, keeping the rest:",
				Bash:        "cutx -d , -f 2 --complement people.csv",
				PowerShell:  "cutx -d , -f 2 --complement people.csv",
			},
			{
				Description: "Print the first 10 bytes of each line:",
				Bash:        "cutx -b 1-10 app.log",
				PowerShell:  "cutx -b 1-10 app.log",
			},
			{
				Description: "Print characters 5 to 8 of each line:",
				Bash:        "cutx -c 5-8 codes.txt",
				PowerShell:  "cutx -c 5-8 codes.txt",
			},
			{
				Description: "Convert a CSV file to TAB-separated output:",
				Bash:        "cutx -d , -f 1- --output-delimiter=$'\\t' people.csv",
				PowerShell:  "cutx -d , -f 1- --output-delimiter=\"`t\" people.csv",
			},
			{
				Description: "Join the selected columns with a custom separator:",
				Bash:        "cutx -d : -f 1,3 --output-delimiter=' -> ' passwd.txt",
				PowerShell:  "cutx -d : -f 1,3 --output-delimiter=' -> ' passwd.txt",
			},
			{
				Description: "Skip lines that contain no delimiter, such as comments:",
				Bash:        "cutx -s -d = -f 2 settings.ini",
				PowerShell:  "cutx -s -d = -f 2 settings.ini",
			},
			{
				Description: "Pull one field out of a command's output:",
				Bash:        "echox 'key=value' | cutx -d = -f 2",
				PowerShell:  "'key=value' | cutx -d = -f 2",
			},
			{
				Description: "Process NUL-separated records:",
				Bash:        "cutx -z -d / -f 1 paths.lst",
				PowerShell:  "cutx -z -d / -f 1 paths.lst",
			},
			{
				Description: "Count the distinct values of a column in the portable gnux shell:",
				Bash:        "gnux shell -c 'cutx -d , -f 1 people.csv | uniqx -c'",
				PowerShell:  "gnux shell -c 'cutx -d , -f 1 people.csv | uniqx -c'",
			},
		},
	}
}

// Register installs the cutx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("bytes", "b", "", "select only these bytes")
	fs.StringP("characters", "c", "", "select only these characters")
	fs.StringP("delimiter", "d", "", "use `DELIM` instead of TAB for field delimiter")
	fs.StringP("fields", "f", "",
		"select only these fields; also print any line that contains no delimiter character, unless the -s option is specified")
	// -n is GNU's "(ignored)" option; it has no long name, so it is hidden
	// under one and described in Long.
	fs.BoolP("ignored-n", "n", false, "(ignored)")
	_ = fs.MarkHidden("ignored-n")
	fs.Bool("complement", false, "complement the set of selected bytes, characters or fields")
	fs.BoolP("only-delimited", "s", false, "do not print lines not containing delimiters")
	fs.String("output-delimiter", "", "use `STRING` as the output delimiter; the default is to use the input delimiter")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
}

// lists maps each list flag to the mode it selects.
var lists = []struct {
	flag string
	mode cut.Mode
}{
	{"bytes", cut.ModeBytes},
	{"characters", cut.ModeChars},
	{"fields", cut.ModeFields},
}

// buildOptions folds the parsed flags into cut.Options. A missing list is
// left to the engine, which refuses it in GNU's words.
func buildOptions(fs *pflag.FlagSet) (cut.Options, error) {
	var o cut.Options
	for _, l := range lists {
		if !fs.Changed(l.flag) {
			continue
		}
		if o.Mode != cut.ModeNone {
			return cut.Options{}, core.Usagef("only one type of list may be specified")
		}
		o.Mode = l.mode
		o.List, _ = fs.GetString(l.flag)
	}

	if fs.Changed("delimiter") {
		o.Delimiter, _ = fs.GetString("delimiter")
		if o.Delimiter == "" {
			o.Delimiter = "\x00"
		}
	}
	if fs.Changed("output-delimiter") {
		o.OutputDelimiter, _ = fs.GetString("output-delimiter")
		o.OutputDelimiterSet = true
	}
	o.Complement, _ = fs.GetBool("complement")
	o.OnlyDelimited, _ = fs.GetBool("only-delimited")
	o.ZeroTerminated, _ = fs.GetBool("zero-terminated")
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return cut.New(opts, s).Run(args)
}
