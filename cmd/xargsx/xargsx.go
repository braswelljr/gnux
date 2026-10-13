// Package xargsx wires the xargsx command (GNU xargs) to its engine.
package xargsx

import (
	"errors"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/xargs"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the xargsx tool. It parses its own arguments (see parseArgs):
// GNU's -e, -i and -l take an optional argument that must be attached
// ("-l3", "-iX"), a form pflag does not read. Option parsing stops at the
// first operand so the options of the command being run are left for that
// command.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "xargs",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for xargsx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    xargs.Name,
		SeeAlso: []string{"findx", "envx", "echox", "rmx", "gnux shell"},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "every command ran and exited with status 0"},
			{Code: "1", Meaning: "xargsx itself failed: a bad option value, an unmatched quote, an argument line too long, an input file it could not open"},
			{Code: "123", Meaning: "a command exited with a status from 1 to 254 (the remaining commands still ran)"},
			{Code: "124", Meaning: "a command exited with status 255; no further command was started"},
			{Code: "125", Meaning: "a signal ended a command; no further command was started"},
			{Code: "126", Meaning: "the command was found but could not be run"},
			{Code: "127", Meaning: "the command was not found"},
		},
		Use:   xargs.Name + " [OPTION]... [COMMAND [INITIAL-ARGUMENT]...]",
		Short: "Build and run command lines from standard input.",
		Long: "xargsx reads items from standard input and runs COMMAND (echo when none is given) " +
			"with the INITIAL-ARGUMENTs followed by the items, as many per command line as fit, " +
			"until the input is used up. Items are separated by blanks and newlines; single " +
			"quotes, double quotes and a backslash protect a character from being a separator, " +
			"a quote must close on the line it opens on, and a blank line is ignored. Because " +
			"file names may hold blanks, quotes and newlines, pair -0 with a producer that " +
			"separates names with NUL (findx -print0), or use -d to name a single delimiter " +
			"character (C escapes such as \\n and \\t, octal \\NNN and hexadecimal \\xHH are " +
			"understood); with either, every other character is literal. -a reads the items " +
			"from a file and leaves standard input to the commands; otherwise each command " +
			"gets an empty standard input, so that it cannot consume the items. -E names a " +
			"logical end-of-input item (not with -0 or -d). " +
			"-n limits the items per command and -L the non-blank input lines per command (a " +
			"line that ends in a blank continues on the next one). -s sets the size limit of " +
			"a command line, counting the command, the initial arguments, the items and one " +
			"terminating NUL for each: the default is 131072 bytes, or the system's limit when " +
			"that is smaller, taken as 524288 bytes on Unix and 32767 on Windows, less 2048 " +
			"and less the size of the environment. -x fails with 'argument list too long' " +
			"when the items asked for with -n or -L do not fit, instead of running a shorter " +
			"command. -I runs the command once per input line and replaces every REPLACE-STR " +
			"in the command and its initial arguments with the line (leading blanks removed, " +
			"quotes still processed); it implies -x and -L 1, and -L implies -x. " +
			"Without -r the command runs once even when the input holds no items. -t prints " +
			"each command line on the error stream before running it, quoted so that it can " +
			"be pasted into a shell; -p prints it followed by '?...' and runs it only when " +
			"the line read from the terminal (/dev/tty, or the console on Windows) begins " +
			"with y or Y. -P runs up to MAX-PROCS commands at once; their output is not " +
			"grouped, so lines of different commands may interleave. " +
			"A command that exits with status 255, or that a signal ends, stops xargsx at " +
			"once; any other nonzero status lets the remaining commands run and makes the " +
			"final status 123.\n\n" +
			"Differences from GNU xargs: a COMMAND named after a gnux tool, by its own name " +
			"(rmx) or its GNU name (rm, echo), runs inside the xargsx process instead of as a " +
			"program found on PATH, so that 'xargsx rm' works where no GNU tools are " +
			"installed; any other name is a program found on PATH. Use 'envx COMMAND' as the " +
			"command to force the program on PATH. GNU's last-one-wins between conflicting " +
			"options is replaced by a fixed precedence toward the most cautious one, in any " +
			"order: -I/-i wins over -L/-l, which wins over -n, each with GNU's warning about " +
			"the ignored option; -0 wins over -d; and of two spellings of one option, -I wins " +
			"over -i, -L over -l and -E over -e. -P 0 runs at most 128 commands at once " +
			"rather than as many as the system allows. The system's command-line limit is " +
			"the fixed figure above, not one asked of the system. A NUL in the input outside " +
			"-0 mode is dropped with GNU's warning, where GNU cuts the item short at it. A " +
			"quote left open at the very end of the input is always an error. -o/--open-tty " +
			"and --process-slot-var are not supported and are refused with a usage error. " +
			"Changing -P while xargsx runs, with SIGUSR1 and SIGUSR2, is not available.",
		Permissions: "xargsx needs read permission on the -a file. Running a program needs " +
			"execute permission on it and search permission on the directories leading to " +
			"it; a gnux tool run in-process needs whatever that tool needs for the files it " +
			"is given (rm needs write permission on the containing directory, and so on). " +
			"The commands run with the privileges of xargsx itself.",
		UseCases: []string{
			"Removing, moving or checksumming the files a search produced, a few commands for thousands of names.",
			"Handling file names with spaces, quotes or newlines safely with -0 and a NUL-separated producer.",
			"Running one command per input line with the line placed anywhere in it using -I.",
			"Splitting a long list into commands of N arguments with -n.",
			"Running independent jobs in parallel with -P.",
			"Reviewing destructive commands before they run with -t, or approving each one with -p.",
			"Keeping a command line under a size limit with -s when a program rejects long ones.",
			"Skipping the command entirely when a search found nothing, with -r.",
			"Feeding items from a file with -a while the command keeps the terminal as its standard input.",
			"Writing scripts that remove or copy files the same way on machines with no GNU tools installed.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Join the items of several lines into one command line:",
				Bash:        "printfx 'a b\\nc\\n' | xargsx echox",
				PowerShell:  "printfx 'a b\\nc\\n' | xargsx echox",
			},
			{
				Description: "Remove the files a search found, whatever characters their names hold:",
				Bash:        "findx . -name '*.tmp' -print0 | xargsx -0 rmx -f",
				PowerShell:  "findx . -name '*.tmp' -print0 | xargsx -0 rmx -f",
			},
			{
				Description: "Do nothing at all when the search finds no file:",
				Bash:        "findx . -name '*.orig' -print0 | xargsx -0 -r rmx",
				PowerShell:  "findx . -name '*.orig' -print0 | xargsx -0 -r rmx",
			},
			{
				Description: "Pass two items to each command:",
				Bash:        "seqx 6 | xargsx -n 2 echox",
				PowerShell:  "seqx 6 | xargsx -n 2 echox",
			},
			{
				Description: "Run one command per line, with the line placed inside an argument:",
				Bash:        "printfx 'a.txt\\nb.txt\\n' | xargsx -I {} cpx {} {}.bak",
				PowerShell:  "printfx 'a.txt\\nb.txt\\n' | xargsx -I '{}' cpx '{}' '{}.bak'",
			},
			{
				Description: "Treat each line as one item, blanks and quotes included:",
				Bash:        "xargsx -d '\\n' -a names.txt touchx",
				PowerShell:  "xargsx -d '\\n' -a names.txt touchx",
			},
			{
				Description: "Print each command line before it runs:",
				Bash:        "seqx 4 | xargsx -t -n 2 echox",
				PowerShell:  "seqx 4 | xargsx -t -n 2 echox",
			},
			{
				Description: "Ask before each removal (the answer is read from the terminal):",
				Bash:        "lsx | xargsx -p -n 1 rmx",
				PowerShell:  "lsx | xargsx -p -n 1 rmx",
			},
			{
				Description: "Checksum many files four commands at a time:",
				Bash:        "findx . -type f -print0 | xargsx -0 -n 50 -P 4 sha256sumx",
				PowerShell:  "findx . -type f -print0 | xargsx -0 -n 50 -P 4 sha256sumx",
			},
			{
				Description: "Use one input line per command, as -L counts them:",
				Bash:        "printfx 'a b\\nc d\\n' | xargsx -L 1 echox line:",
				PowerShell:  "printfx 'a b\\nc d\\n' | xargsx -L 1 echox line:",
			},
			{
				Description: "Stop reading at a marker item:",
				Bash:        "printfx 'a b END c\\n' | xargsx -E END echox",
				PowerShell:  "printfx 'a b END c\\n' | xargsx -E END echox",
			},
			{
				Description: "Keep every command line within 20 bytes:",
				Bash:        "seqx 10 | xargsx -s 20 echox",
				PowerShell:  "seqx 10 | xargsx -s 20 echox",
			},
			{
				Description: "Run a program from PATH rather than the gnux tool of the same name:",
				Bash:        "printfx 'a b\\n' | xargsx envx echo",
				PowerShell:  "printfx 'a b\\n' | xargsx envx cmd /c echo",
			},
			{
				Description: "Tell a failing command (status 123) from a missing one (127):",
				Bash:        "echox x | xargsx no-such-command; echox \"status: $?\"",
				PowerShell:  "echox x | xargsx no-such-command; echox \"status: $LASTEXITCODE\"",
			},
			{
				Description: "Show the size limits in effect without running anything:",
				Bash:        "xargsx --show-limits -r < /dev/null",
				PowerShell:  "'' | xargsx --show-limits -r",
			},
		},
	}
}

// noValue is what an option with an optional argument holds when it was
// given without one.
const noValue = "\x00"

// Register installs the xargsx flags on fs. GNU's -E, -I and -L have no long
// form of their own (--eof, --replace and --max-lines belong to -e, -i and
// -l, whose argument is optional); pflag requires one, so they get
// descriptive names.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("null", "0", false, "items are separated by a NUL, not whitespace;\n"+
		"disables quote and backslash processing and logical EOF processing")
	fs.StringP("arg-file", "a", "", "read arguments from `FILE`, not standard input")
	fs.StringP("delimiter", "d", "", "items in the input are separated by `CHARACTER`, not by whitespace;\n"+
		"disables quote and backslash processing and logical EOF processing")
	fs.StringP("eof-string", "E", "", "set logical EOF string to `END`; if END occurs as an item of input,\n"+
		"the rest of the input is ignored (ignored if -0 or -d was specified)")
	fs.StringP("eof", "e", "", "equivalent to -E `END` if END is specified; otherwise, there is no end-of-file string")
	fs.Lookup("eof").NoOptDefVal = noValue
	fs.StringP("replace-string", "I", "", "same as --replace=`R`")
	fs.StringP("replace", "i", "", "replace `R` in INITIAL-ARGS with names read from standard input,\n"+
		"split at newlines; if R is unspecified, assume {}")
	fs.Lookup("replace").NoOptDefVal = "{}"
	fs.StringP("max-input-lines", "L", "", "use at most `MAX-LINES` non-blank input lines per command line")
	fs.StringP("max-lines", "l", "", "similar to -L but defaults to at most one non-blank input line\n"+
		"if `MAX-LINES` is not specified")
	fs.Lookup("max-lines").NoOptDefVal = "1"
	fs.StringP("max-args", "n", "", "use at most `MAX-ARGS` arguments per command line")
	fs.BoolP("open-tty", "o", false, "reopen stdin as /dev/tty in the child process before executing\n"+
		"the command (not supported)")
	fs.StringP("max-procs", "P", "", "run at most `MAX-PROCS` processes at a time")
	fs.BoolP("interactive", "p", false, "prompt before running commands")
	fs.String("process-slot-var", "", "set environment variable `VAR` in child processes (not supported)")
	fs.BoolP("no-run-if-empty", "r", false, "if there are no arguments, then do not run COMMAND;\n"+
		"if this option is not given, COMMAND will be run at least once")
	fs.StringP("max-chars", "s", "", "limit length of command line to `MAX-CHARS`")
	fs.Bool("show-limits", false, "show limits on command-line length")
	fs.BoolP("verbose", "t", false, "print commands before executing them")
	fs.BoolP("exit", "x", false, "exit if the size (see -s) is exceeded")
}

// buildOptions folds the parsed flags into xargs.Options. GNU lets the last of
// -n, -L and -I win; pflag does not tell which came last, so the most cautious
// one wins in any order (-I over -L over -n) and the warnings GNU prints for
// the ignored options are returned with the options. The error is a usage
// error for a number that is not one.
func buildOptions(fs *pflag.FlagSet) (xargs.Options, []string, error) {
	var o xargs.Options
	var warnings []string

	o.Null, _ = fs.GetBool("null")
	o.ArgFile, _ = fs.GetString("arg-file")
	// NUL cannot occur in a file name, which makes -0 the safer of the two.
	if !o.Null {
		o.Delimiter, _ = fs.GetString("delimiter")
	}
	o.EOF = text(fs, "eof-string", "eof")
	o.Replace = text(fs, "replace-string", "replace")
	o.NoRunIfEmpty, _ = fs.GetBool("no-run-if-empty")
	o.Verbose, _ = fs.GetBool("verbose")
	o.Interactive, _ = fs.GetBool("interactive")
	o.Exit, _ = fs.GetBool("exit")
	o.ShowLimits, _ = fs.GetBool("show-limits")
	o.OpenTTY, _ = fs.GetBool("open-tty")
	o.ProcessSlotVar, _ = fs.GetString("process-slot-var")

	var err error
	if o.MaxArgs, err = number(fs, "max-args", 'n', 1); err != nil {
		return o, nil, err
	}
	if o.MaxLines, err = number(fs, "max-input-lines", 'L', 1); err != nil {
		return o, nil, err
	}
	if !fs.Changed("max-input-lines") {
		if o.MaxLines, err = number(fs, "max-lines", 'l', 1); err != nil {
			return o, nil, err
		}
	}
	if o.MaxChars, err = number(fs, "max-chars", 's', 1); err != nil {
		return o, nil, err
	}
	if fs.Changed("max-procs") {
		if o.MaxProcs, err = number(fs, "max-procs", 'P', 0); err != nil {
			return o, nil, err
		}
		if o.MaxProcs == 0 {
			o.MaxProcs = xargs.Unlimited
		}
	}

	ignore := func(winner, loser string, value *int) {
		if *value != 0 {
			warnings = append(warnings, "warning: options "+loser+" and "+winner+
				" are mutually exclusive, ignoring previous "+loser+" value")
			*value = 0
		}
	}
	switch {
	case o.Replace != "":
		ignore("--replace/-I/-i", "--max-args", &o.MaxArgs)
		ignore("--replace/-I/-i", "--max-lines", &o.MaxLines)
	case o.MaxLines != 0:
		ignore("--max-lines/-l/-L", "--max-args", &o.MaxArgs)
	}
	return o, warnings, nil
}

// text returns the value of an option that has two spellings: short, whose
// argument is required, and optional, whose argument may be left out. short
// wins when both were given.
func text(fs *pflag.FlagSet, short, optional string) string {
	if fs.Changed(short) {
		v, _ := fs.GetString(short)
		return v
	}
	v, _ := fs.GetString(optional)
	if v == noValue {
		return ""
	}
	return v
}

// number parses the value of a numeric option the way GNU xargs does, with
// its diagnostics, which name the option by its letter. An option that was
// not given is 0.
func number(fs *pflag.FlagSet, name string, letter rune, least int) (int, error) {
	if !fs.Changed(name) {
		return 0, nil
	}
	text, _ := fs.GetString(name)
	n, err := strconv.ParseInt(text, 10, 64)
	switch {
	case errors.Is(err, strconv.ErrRange) && text != "" && text[0] != '-':
		n = math.MaxInt32
	case errors.Is(err, strconv.ErrRange):
		n = math.MinInt32
	case err != nil:
		return 0, core.Usagef("invalid number %q for -%c option", text, letter)
	}
	if n < int64(least) {
		return 0, core.Usagef("value %s for -%c option should be >= %d", text, letter, least)
	}
	return int(min(n, math.MaxInt32)), nil
}

// parseArgs parses the command line the way GNU xargs reads it and returns
// the flag set holding the options, whose Args are the command. Options end at
// the first operand. The error is a usage error.
func parseArgs(args []string) (*pflag.FlagSet, error) {
	fs := pflag.NewFlagSet(xargs.Name, pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.SetInterspersed(false)
	fs.Bool("help", false, "")
	fs.Bool("version", false, "")
	Register(fs)
	if err := fs.Parse(attachOptional(fs, args)); err != nil {
		return nil, core.Usagef("%v", err)
	}
	return fs, nil
}

// attachOptional rewrites a short option whose argument is optional and was
// given attached, "-l3" or "-iX", as "-l=3" and "-i=X", the only spelling
// pflag takes an attached optional argument in. Everything from the first
// operand on belongs to the command and is left alone.
func attachOptional(fs *pflag.FlagSet, args []string) []string {
	out := append([]string(nil), args...)
	for n := 0; n < len(out); n++ {
		arg := out[n]
		switch {
		case arg == "--" || len(arg) < 2 || arg[0] != '-':
			return out
		case strings.HasPrefix(arg, "--"):
			// "--max-args 2": the value is the next word, not an operand.
			if f := fs.Lookup(arg[2:]); f != nil && takesValue(f) {
				n++
			}
			continue
		}
	cluster:
		for i := 1; i < len(arg); i++ {
			f := fs.ShorthandLookup(arg[i : i+1])
			switch {
			case f == nil:
				// Not an option: pflag reports it.
				break cluster
			case f.NoOptDefVal != "" && f.Value.Type() != "bool":
				if rest := arg[i+1:]; rest != "" {
					out[n] = arg[:i+1] + "=" + rest
				}
				break cluster
			case takesValue(f):
				if i == len(arg)-1 {
					n++ // the value is the next word
				}
				break cluster
			}
		}
	}
	return out
}

// takesValue reports whether f needs an argument.
func takesValue(f *pflag.Flag) bool {
	return f.NoOptDefVal == "" && f.Value.Type() != "bool"
}

func run(help *pflag.FlagSet, args []string, s core.Streams) error {
	fs, err := parseArgs(args)
	if err != nil {
		return err
	}
	if v, _ := fs.GetBool("help"); v {
		Spec().WriteHelp(s.Out, help, tool.StyleFor(s.Out))
		return nil
	}
	if v, _ := fs.GetBool("version"); v {
		tool.PrintVersion(s.Out, xargs.Name)
		return nil
	}

	opts, warnings, err := buildOptions(fs)
	if err != nil {
		return err
	}
	diag := core.NewDiag(xargs.Name, s)
	for _, w := range warnings {
		diag.Warnf("%s", w)
	}

	x := xargs.New(opts, s)
	// A command named after a gnux tool runs in-process, through the
	// registry package cmd installed.
	x.Exec = tool.RunTool
	return x.Run(fs.Args())
}
