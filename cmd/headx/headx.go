// Package headx wires the headx command (GNU head) to its engine.
package headx

import (
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the headx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "head",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for headx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    head.Name,
		SeeAlso: []string{"tailx", "catx", "wcx"},
		Use:     head.Name + " [OPTION]... [FILE]...",
		Short:   "Print the first part of each file.",
		Long: "headx prints the first 10 lines of each FILE to standard output. With no FILE, " +
			"or when FILE is -, it reads standard input. -n NUM changes the number of lines and " +
			"-c NUM switches to bytes; a leading '-' on NUM inverts the selection and prints " +
			"everything except the last NUM lines or bytes. NUM accepts a multiplier suffix: b " +
			"is 512, K, M, G, T, P and E are powers of 1024 (also written KiB, MiB, ...), and " +
			"KB, MB, ... are powers of 1000. With more than one FILE each is preceded by a " +
			"'==> name <==' header; -q suppresses the headers and -v forces them. -z makes NUL " +
			"the line delimiter. The historical 'headx -NUM' spelling of '-n NUM' is accepted. " +
			"Reading stops as soon as enough has been seen, so headx is cheap on huge files.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Peeking at the start of a large file or log without reading all of it.",
			"Taking the first few results of a pipeline.",
			"Dropping a fixed-size trailer with -n -NUM or -c -NUM.",
			"Sampling the first bytes of a binary file with -c, for example to check a magic number.",
			"Reading the header row of a CSV file before processing it.",
			"Comparing the beginnings of several files side by side, with or without '==> name <==' headers.",
			"Cutting an endless stream such as yesx down to a fixed number of lines.",
			"Extracting a range of lines when combined with tailx.",
			"Handling NUL-separated file lists safely with -z.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the first 10 lines of a file:",
				Bash:        "headx access.log",
				PowerShell:  "headx access.log",
			},
			{
				Description: "Print the first 3 lines of each file, with headers:",
				Bash:        "headx -n 3 a.txt b.txt",
				PowerShell:  "headx -n 3 a.txt b.txt",
			},
			{
				Description: "Print only the header row of a CSV file:",
				Bash:        "headx -n 1 data.csv",
				PowerShell:  "headx -n 1 data.csv",
			},
			{
				Description: "Print the first line of every CSV file without the file name headers:",
				Bash:        "headx -q -n 1 *.csv",
				PowerShell:  "headx -q -n 1 (Get-ChildItem *.csv -Name)",
			},
			{
				Description: "Show the file name header even for a single file (--silent is the long spelling of -q):",
				Bash:        "headx -v -n 2 config.ini",
				PowerShell:  "headx -v -n 2 config.ini",
			},
			{
				Description: "Print the first kibibyte of a file:",
				Bash:        "headx -c 1K image.png > start.bin",
				PowerShell:  "gnux shell -c 'headx -c 1K image.png > start.bin'",
			},
			{
				Description: "Check a file's magic number from its first bytes:",
				Bash:        "headx -c 4 image.png | catx -v",
				PowerShell:  "headx -c 4 image.png | catx -v",
			},
			{
				Description: "Copy the first megabyte (10^6 bytes) of a large file:",
				Bash:        "headx -c 1MB disk.img > sample.img",
				PowerShell:  "gnux shell -c 'headx -c 1MB disk.img > sample.img'",
			},
			{
				Description: "Print everything except the last 2 lines:",
				Bash:        "headx -n -2 report.txt",
				PowerShell:  "headx -n -2 report.txt",
			},
			{
				Description: "Strip the final byte, such as a trailing newline:",
				Bash:        "headx -c -1 version.txt",
				PowerShell:  "headx -c -1 version.txt",
			},
			{
				Description: "Keep the first 5 lines of a pipeline, using the historical -NUM form:",
				Bash:        "lsx -t | headx -5",
				PowerShell:  "lsx -t | headx -5",
			},
			{
				Description: "Print lines 11 to 20 of a file:",
				Bash:        "headx -n 20 app.log | tailx -n 10",
				PowerShell:  "headx -n 20 app.log | tailx -n 10",
			},
			{
				Description: "Cut an endless stream down to three lines:",
				Bash:        "yesx ok | headx -n 3",
				PowerShell:  "yesx ok | headx -n 3",
			},
			{
				Description: "Take the first two entries of a NUL-separated list:",
				Bash:        "find . -name '*.txt' -print0 | headx -z -n 2 | xargs -0 wcx -l",
				PowerShell:  "headx -z -n 2 names.lst",
			},
			{
				Description: "Fail a script when a file cannot be read:",
				Bash:        "headx -n 1 missing.txt || echox \"exit status $?\"",
				PowerShell:  "headx -n 1 missing.txt; if ($LASTEXITCODE -ne 0) { \"exit status $LASTEXITCODE\" }",
			},
			{
				Description: "Preview the start of every log in the portable gnux shell:",
				Bash:        "gnux shell -c 'headx -n 3 *.log'",
				PowerShell:  "gnux shell -c 'headx -n 3 *.log'",
			},
		},
	}
}

// Register installs the headx flags on fs.
func Register(fs *pflag.FlagSet) {
	ord := &order{}
	fs.VarP(&countValue{ord: ord}, "bytes", "c",
		"print the first NUM bytes of each file; with a leading '-', print all but the last NUM bytes of each file")
	fs.VarP(&countValue{ord: ord}, "lines", "n",
		"print the first NUM lines instead of the first 10; with a leading '-', print all but the last NUM lines of each file")
	fs.BoolP("quiet", "q", false, "never print headers giving file names")
	fs.Bool("silent", false, "same as --quiet")
	fs.BoolP("verbose", "v", false, "always print headers giving file names")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
	registerObsolete(fs, ord)
}

// order stamps each count-setting flag as it is parsed. GNU head lets the last
// of -n, -c and -NUM win, and pflag does not otherwise record flag order.
type order struct{ n int }

func (o *order) next() int {
	o.n++
	return o.n
}

// countValue is the pflag value behind -n and -c: the raw text, validated
// later by buildOptions, plus when it was given.
type countValue struct {
	ord  *order
	text string
	at   int
}

func (v *countValue) String() string { return v.text }

// Type is what --help shows as the value placeholder.
func (v *countValue) Type() string { return "[-]NUM" }

func (v *countValue) Set(s string) error {
	v.text, v.at = s, v.ord.next()
	return nil
}

// digits collects the obsolete "-NUM" form. Flag parsing belongs to
// cobra/pflag, which would reject -5 as an unknown shorthand; registering each
// digit as a hidden, argument-less shorthand makes pflag deliver "-15" as '1'
// then '5', which Set reassembles.
type digits struct {
	ord  *order
	text string
	at   int
}

// digitValue is the pflag value for one digit shorthand.
type digitValue struct {
	run   *digits
	digit byte
}

func (v *digitValue) String() string { return "" }

func (v *digitValue) Type() string { return "bool" }

func (v *digitValue) Set(string) error {
	at := v.run.ord.next()
	// Digits given after another count flag start a new number.
	if v.run.at != at-1 {
		v.run.text = ""
	}
	v.run.text += string(v.digit)
	v.run.at = at
	return nil
}

// obsoleteFlag is the hidden long name of the digit d.
func obsoleteFlag(d int) string { return "obsolete-" + strconv.Itoa(d) }

func registerObsolete(fs *pflag.FlagSet, ord *order) {
	run := &digits{ord: ord}
	for d := range 10 {
		f := fs.VarPF(&digitValue{run: run, digit: byte('0' + d)}, obsoleteFlag(d), strconv.Itoa(d), "")
		f.NoOptDefVal = "true"
		f.Hidden = true
	}
}

// buildOptions folds the parsed flags into head.Options. Whichever of -n, -c
// and -NUM was given last decides the count.
func buildOptions(fs *pflag.FlagSet) (head.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := head.Options{
		Count:          head.DefaultLines,
		Quiet:          on("quiet") || on("silent"),
		Verbose:        on("verbose"),
		ZeroTerminated: on("zero-terminated"),
	}

	lines := fs.Lookup("lines").Value.(*countValue)
	bytes := fs.Lookup("bytes").Value.(*countValue)
	obsolete := fs.Lookup(obsoleteFlag(0)).Value.(*digitValue).run

	var err error
	switch max(lines.at, bytes.at, obsolete.at) {
	case 0: // none given: keep the default
	case bytes.at:
		o.Bytes = true
		o.Count, o.FromEnd, err = parseCount(bytes.text, "bytes")
	case lines.at:
		o.Count, o.FromEnd, err = parseCount(lines.text, "lines")
	default:
		o.Count, _, err = parseCount(obsolete.text, "lines")
	}
	if err != nil {
		return head.Options{}, err
	}
	return o, nil
}

// parseCount reads [-]NUM[SUFFIX]. A leading '-' selects "all but the last
// NUM"; a leading '+' is accepted and means nothing.
func parseCount(text, unit string) (n int64, fromEnd bool, err error) {
	num := text
	if rest, ok := strings.CutPrefix(num, "-"); ok {
		num, fromEnd = rest, true
	} else {
		num = strings.TrimPrefix(num, "+")
	}
	n, err = head.ParseSize(num)
	if err != nil {
		return 0, false, core.Usagef("invalid number of %s: '%s'", unit, text)
	}
	return n, fromEnd, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return head.New(opts, s).Run(args)
}
