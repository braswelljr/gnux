// Package tailx wires the tailx command (GNU tail) to its engine.
package tailx

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
	"github.com/braswelljr/gnux/core/tail"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the tailx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "tail",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for tailx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tail.Name,
		SeeAlso: []string{"headx", "catx", "wcx"},
		Use:     tail.Name + " [OPTION]... [FILE]...",
		Short:   "Print the last part of each file, optionally following it as it grows.",
		Long: "tailx prints the last 10 lines of each FILE to standard output. With no FILE, " +
			"or when FILE is -, it reads standard input. -n NUM changes the number of lines and " +
			"-c NUM switches to bytes; a leading '+' on NUM starts output at line or byte NUM " +
			"instead of counting from the end. NUM accepts a multiplier suffix: b is 512, K, M, " +
			"G, T, P and E are powers of 1024 (also written KiB, MiB, ...), and KB, MB, ... are " +
			"powers of 1000. With more than one FILE each is preceded by a '==> name <==' " +
			"header; -q suppresses the headers and -v forces them. -f keeps running and prints " +
			"data as it is appended, polling every -s seconds; --follow=name tracks the file " +
			"name rather than the open file, so it survives log rotation, and --retry keeps " +
			"waiting for a file that is missing. -F is shorthand for --follow=name --retry. " +
			"--pid ends following when the given process exits. On a regular file tailx seeks " +
			"to the end and reads backward, so its cost does not depend on the file's size.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified. With --follow=name the file " +
			"is reopened on every poll, so permissions are re-checked as it is rotated.",
		UseCases: []string{
			"Reading the most recent entries of a log.",
			"Watching a log live with -f, or across rotations with -F.",
			"Skipping a header with -n +2.",
			"Following a build log only for as long as the build runs, with --pid.",
			"Waiting for a log file that a service has not created yet, with --retry.",
			"Printing only new log lines from now on with -n 0 -f.",
			"Stripping a fixed-size prefix, such as a byte-order mark, with -c +NUM.",
			"Extracting a range of lines when combined with headx.",
			"Reading the last record of a NUL-separated list with -z.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the last 10 lines of a file:",
				Bash:        "tailx app.log",
				PowerShell:  "tailx app.log",
			},
			{
				Description: "Print the last 50 lines:",
				Bash:        "tailx -n 50 app.log",
				PowerShell:  "tailx -n 50 app.log",
			},
			{
				Description: "Print the last 3 lines, using the historical -NUM form:",
				Bash:        "tailx -3 app.log",
				PowerShell:  "tailx -3 app.log",
			},
			{
				Description: "Print everything from the second line on (skip a header):",
				Bash:        "tailx -n +2 data.csv",
				PowerShell:  "tailx -n +2 data.csv",
			},
			{
				Description: "Print the last 100 bytes of a file:",
				Bash:        "tailx -c 100 app.log",
				PowerShell:  "tailx -c 100 app.log",
			},
			{
				Description: "Drop a 3-byte UTF-8 byte-order mark by starting at byte 4:",
				Bash:        "tailx -c +4 with-bom.txt > clean.txt",
				PowerShell:  "gnux shell -c 'tailx -c +4 with-bom.txt > clean.txt'",
			},
			{
				Description: "Print the last line of each log, with and without file name headers:",
				Bash:        "tailx -n 1 a.log b.log && tailx -q -n 1 a.log b.log",
				PowerShell:  "tailx -n 1 a.log b.log; tailx -q -n 1 a.log b.log",
			},
			{
				Description: "Force the header for a single file (--silent is the long spelling of -q):",
				Bash:        "tailx -v -n 5 app.log",
				PowerShell:  "tailx -v -n 5 app.log",
			},
			{
				Description: "Print lines 11 to 20 of a file:",
				Bash:        "headx -n 20 app.log | tailx -n 10",
				PowerShell:  "headx -n 20 app.log | tailx -n 10",
			},
			{
				Description: "Print the oldest entry of a directory:",
				Bash:        "lsx -t | tailx -n 1",
				PowerShell:  "lsx -t | tailx -n 1",
			},
			{
				Description: "Print the last entry of a NUL-separated list:",
				Bash:        "tailx -z -n 1 names.lst | xargs -0 echox",
				PowerShell:  "tailx -z -n 1 names.lst",
			},
			{
				Description: "Follow a log, printing only lines appended from now on:",
				Bash:        "tailx -n 0 -f app.log",
				PowerShell:  "tailx -n 0 -f app.log",
			},
			{
				Description: "Follow a log across rotations, polling twice a second:",
				Bash:        "tailx -F -s 0.5 /var/log/app.log",
				PowerShell:  "tailx -F -s 0.5 C:\\logs\\app.log",
			},
			{
				Description: "Follow by name and wait for a file that does not exist yet:",
				Bash:        "tailx --follow=name --retry service.log",
				PowerShell:  "tailx --follow=name --retry service.log",
			},
			{
				Description: "Follow a log until a process exits:",
				Bash:        "make > build.log 2>&1 & tailx -f --pid=$! build.log",
				PowerShell:  "$p = Start-Process make -RedirectStandardOutput build.log -PassThru; tailx -f --pid=$($p.Id) build.log",
			},
			{
				Description: "Report a missing file through the exit status:",
				Bash:        "tailx missing.log || echox \"exit status $?\"",
				PowerShell:  "tailx missing.log; if ($LASTEXITCODE -ne 0) { \"exit status $LASTEXITCODE\" }",
			},
			{
				Description: "Show the end of every log in the portable gnux shell:",
				Bash:        "gnux shell -c 'catx *.log | tailx -n 5'",
				PowerShell:  "gnux shell -c 'catx *.log | tailx -n 5'",
			},
		},
	}
}

// minSleep bounds -s 0, which GNU tail turns into a busy loop.
const minSleep = time.Millisecond

// Register installs the tailx flags on fs. GNU's -F has no long form; pflag
// requires one, so it gets a descriptive name.
func Register(fs *pflag.FlagSet) {
	ord := &order{}
	fs.VarP(&countValue{ord: ord}, "bytes", "c",
		"output the last NUM bytes; or use -c +NUM to output starting with byte NUM of each file")
	fs.StringP("follow", "f", "",
		"output appended data as the file grows; `HOW` is 'name' or 'descriptor' (the default)")
	fs.Lookup("follow").NoOptDefVal = "descriptor"
	fs.BoolP("follow-retry", "F", false, "same as --follow=name --retry")
	fs.VarP(&countValue{ord: ord}, "lines", "n",
		"output the last NUM lines, instead of the last 10; or use -n +NUM to output starting with line NUM")
	fs.Int("pid", 0, "with -f, terminate after process ID, `PID`, dies")
	fs.BoolP("quiet", "q", false, "never output headers giving file names")
	fs.Bool("silent", false, "same as --quiet")
	fs.Bool("retry", false, "keep trying to open a file if it is inaccessible")
	fs.Float64P("sleep-interval", "s", 1.0,
		"with -f, sleep for approximately `N` seconds (default 1.0) between iterations")
	fs.BoolP("verbose", "v", false, "always output headers giving file names")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
	registerObsolete(fs, ord)

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "follow-retry")
}

// order stamps each count-setting flag as it is parsed. GNU tail lets the last
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
func (v *countValue) Type() string { return "[+]NUM" }

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

// buildOptions folds the parsed flags into tail.Options. Whichever of -n, -c
// and -NUM was given last decides the count.
func buildOptions(fs *pflag.FlagSet) (tail.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := tail.Options{
		Count:          tail.DefaultLines,
		Quiet:          on("quiet") || on("silent"),
		Verbose:        on("verbose"),
		ZeroTerminated: on("zero-terminated"),
		Retry:          on("retry"),
	}

	lines := fs.Lookup("lines").Value.(*countValue)
	bytes := fs.Lookup("bytes").Value.(*countValue)
	obsolete := fs.Lookup(obsoleteFlag(0)).Value.(*digitValue).run

	var err error
	switch max(lines.at, bytes.at, obsolete.at) {
	case 0: // none given: keep the default
	case bytes.at:
		o.Bytes = true
		o.Count, o.FromStart, err = parseCount(bytes.text, "bytes")
	case lines.at:
		o.Count, o.FromStart, err = parseCount(lines.text, "lines")
	default:
		o.Count, _, err = parseCount(obsolete.text, "lines")
	}
	if err != nil {
		return tail.Options{}, err
	}

	if fs.Changed("follow") {
		how, _ := fs.GetString("follow")
		switch {
		case abbreviates(how, "descriptor"):
			o.Follow = tail.FollowDescriptor
		case abbreviates(how, "name"):
			o.Follow = tail.FollowName
		default:
			return tail.Options{}, core.Usagef("invalid argument '%s' for '--follow'", how)
		}
	}
	if on("follow-retry") {
		o.Follow, o.Retry = tail.FollowName, true
	}

	secs, _ := fs.GetFloat64("sleep-interval")
	if secs < 0 || math.IsNaN(secs) {
		return tail.Options{}, core.Usagef("invalid number of seconds: '%s'", fs.Lookup("sleep-interval").Value.String())
	}
	o.SleepInterval = max(time.Duration(secs*float64(time.Second)), minSleep)

	o.PID, _ = fs.GetInt("pid")
	if o.PID < 0 {
		return tail.Options{}, core.Usagef("invalid PID: '%d'", o.PID)
	}
	return o, nil
}

// abbreviates reports whether s is a non-empty prefix of full; GNU accepts
// any unambiguous abbreviation of an option's argument.
func abbreviates(s, full string) bool {
	return s != "" && len(s) <= len(full) && full[:len(s)] == s
}

// parseCount reads [+]NUM[SUFFIX]. A leading '+' selects "starting at NUM"; a
// leading '-' is accepted and means the default, counting from the end.
func parseCount(text, unit string) (n int64, fromStart bool, err error) {
	num := text
	if rest, ok := strings.CutPrefix(num, "+"); ok {
		num, fromStart = rest, true
	} else {
		num = strings.TrimPrefix(num, "-")
	}
	n, err = head.ParseSize(num)
	if err != nil {
		return 0, false, core.Usagef("invalid number of %s: '%s'", unit, text)
	}
	return n, fromStart, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return tail.New(opts, s).Run(args)
}
