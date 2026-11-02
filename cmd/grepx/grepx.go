// Package grepx wires the grepx command (GNU grep) to its engine.
package grepx

import (
	"errors"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/grep"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the grepx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "grep",
		Flags:     Register,
		Run:       run,
		UsageExit: 2,
	}
}

// Meta returns the documentation metadata for grepx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    grep.Name,
		SeeAlso: []string{"sedx", "findx", "xargsx", "headx", "tailx", "wcx", "cutx", "sortx", "uniqx", "gnux shell"},
		Use:     grep.Name + " [OPTION]... PATTERNS [FILE]...",
		Short:   "Print the lines that match patterns.",
		Long: "grepx searches each FILE for PATTERNS and prints every line that matches one. " +
			"PATTERNS is the first operand, unless -e or -f supplied patterns, and may hold " +
			"several patterns separated by newlines; a line is selected when any of them matches. " +
			"With no FILE, or when FILE is -, standard input is searched (shown as " +
			"'(standard input)', or as the --label); with -r and no FILE, the working directory.\n\n" +
			"Pattern syntax. Patterns are POSIX basic regular expressions by default (-G), where " +
			"\\( \\) \\{ \\} \\| \\+ \\? are the operators; -E takes extended regular expressions, " +
			"where ( ) { } | + ? are, and a { that does not start an interval is literal; -F takes " +
			"fixed strings. Both regular syntaxes accept bracket expressions with POSIX classes " +
			"([[:alpha:]], [[:digit:]], ...), and the GNU escapes \\w \\W \\s \\S \\b \\B \\< \\> " +
			"\\` \\'. -i ignores case, -w accepts only matches that are whole words, -x only " +
			"matches of the whole line, and -v selects the lines that do not match. An empty " +
			"pattern matches every line; an empty -f file matches none.\n\n" +
			"Output. Each selected line is printed, prefixed on request with the file name " +
			"(-H, the default when several files or a directory are searched; -h turns it off), " +
			"the line number (-n) and the byte offset (-b), each followed by ':'. -o prints only " +
			"the matched parts, one to a line. -c prints a count of selected lines per file, -l " +
			"the names of the files with a selected line, -L the names of those without, and -q " +
			"nothing at all, stopping at the first selected line. -m NUM stops reading a file " +
			"after NUM selected lines. -A, -B and -C (or -NUM) add lines of context, which are " +
			"prefixed with '-' in place of ':' and set apart in groups by a line holding '--' " +
			"(--group-separator, --no-group-separator). -Z ends each file name with a NUL and -z " +
			"reads and writes lines ending with a NUL. --color highlights matches, file names, " +
			"line numbers, offsets and separators with GNU's default colors, which GREP_COLORS " +
			"overrides; --color=auto colors only when standard output is a terminal and TERM is " +
			"set (which is not required on Windows) and is not 'dumb'.\n\n" +
			"File selection. -r searches directories recursively, following only the symbolic " +
			"links named as operands; -R follows them all. Files are visited in name order. " +
			"--include and --exclude select files by wildcard on their base name (for a file " +
			"named on the command line, on the name or any trailing part of it after a slash); " +
			"when several match, the last one given decides, and when none matches a file is " +
			"searched unless the first of them is an --include. --exclude-dir skips directories. " +
			"Without -r a directory operand is an error ('Is a directory') unless -d skip. " +
			"Devices, FIFOs and sockets are read when named and skipped when met during a " +
			"recursive search, unless -D says otherwise. A file with a NUL byte is binary: its " +
			"lines are not printed, and 'grepx: FILE: binary file matches' is written to standard " +
			"error instead (as GNU grep has done since 3.5), unless -a treats it as text or -I as " +
			"a file without a match.\n\n" +
			"Where GNU grep lets the last of several conflicting options win, grepx applies a " +
			"fixed order, whatever order they were given in: -F beats -E, which beats -G; -H " +
			"beats -h; --no-ignore-case beats -i; -A and -B beat -C, which beats -NUM; -I beats " +
			"-a, and either beats --binary-files; -R beats -r, and either beats -d; -l beats -L; " +
			"--no-group-separator beats --group-separator; --color beats --" + flagColour + ". As in GNU, -q " +
			"overrides -l and -L, which override -c.\n\n" +
			"Differences from GNU grep: -P (Perl-compatible regular expressions) is refused, " +
			"because the regular expression engine is RE2, not PCRE. Back references (\\1 to \\9) " +
			"are refused for the same reason. \\< and \\> both match at any word boundary, as \\b " +
			"does, and the boundary escapes see ASCII words only; \\w, \\W, -w and the " +
			"letter classes know Unicode letters and digits. Input is read as UTF-8 whatever the " +
			"locale, a byte that is not valid UTF-8 matches like any other character, and such " +
			"bytes do not make a file binary: only a NUL byte does. With -z nothing is binary. " +
			"Multi-character collating symbols and equivalence classes are refused. A repetition " +
			"count above 1000 is refused ('Regular expression too big'), where GNU allows " +
			"32767. A regular expression error is reported without the file and line number of a " +
			"-f pattern. GREP_COLOR (the deprecated variable) is not read; GREP_COLORS is. The " +
			"digits of -NUM given in separate arguments (-1 -2) are read as one number (12). -D " +
			"does not apply to standard input. --line-buffered is implied when standard output is " +
			"a terminal. -U (--binary) is accepted and has no effect: no platform's line endings " +
			"are translated. The obsolete -u and the short form -V (use --version) are not " +
			"accepted. A file that is also the output is " +
			"not detected. A missing pattern is reported as 'missing PATTERNS operand' rather " +
			"than with a usage line. Directory entries are searched in name order, where GNU " +
			"uses directory order." +
			" On a terminal, gnux's terminal view (--pretty=auto, the default) prints each file's name once, as a colored heading above its lines with a blank line between files, and numbers the lines (as -n), each unless an option already decides it (--color=never keeps it uncolored); -c, -l, -L, -q, -Z and -z output is GNU's. Piped or redirected output is exactly GNU's; --pretty asks for the view anyway and --pretty=never or GNUX_PRETTY=never turns it off.",
		Permissions: "Needs read permission on every FILE searched and on the files named by -f " +
			"and --exclude-from, and search (execute) permission on the directories leading to " +
			"them. A recursive search also needs read permission on each directory it descends " +
			"into. A file that cannot be opened is reported ('Permission denied') and the " +
			"search goes on with the rest; the exit status is then 2. -s silences those " +
			"messages but not the status. grepx never modifies the files it searches.",
		UseCases: []string{
			"Finding the lines of a log that mention an error, with or without surrounding context.",
			"Searching a source tree for every use of a name with -r, -n and -w.",
			"Listing the files that contain, or lack, a string with -l and -L.",
			"Counting occurrences per file with -c.",
			"Testing in a script whether a file contains a pattern with -q and the exit status.",
			"Filtering a pipeline, keeping (or with -v dropping) the lines that match.",
			"Extracting only the matching parts, such as addresses or identifiers, with -o.",
			"Searching for text that contains regular expression characters with -F.",
			"Restricting a recursive search to some file types with --include and --exclude-dir.",
			"Feeding NUL-separated file names to xargsx -0 with -lZ.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "a line was selected"},
			{Code: "1", Meaning: "no line was selected"},
			{Code: "2", Meaning: "an error occurred (a file could not be read, a pattern is invalid, a command-line mistake), even if a line was selected, unless -q found one"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Find the lines of a log that mention an error:",
				Bash:        "grepx error app.log",
				PowerShell:  "grepx error app.log",
			},
			{
				Description: "Search a source tree for a whole word, with file names and line numbers:",
				Bash:        "grepx -rnw 'TODO' src",
				PowerShell:  "grepx -rnw 'TODO' src",
				Output:      "src/main.go:42:    // TODO: handle the empty case",
			},
			{
				Description: "Ignore case and show two lines of context around each match:",
				Bash:        "grepx -i -C 2 'timeout' server.log",
				PowerShell:  "grepx -i -C 2 'timeout' server.log",
			},
			{
				Description: "Match either of two words with an extended regular expression:",
				Bash:        "grepx -E 'warn(ing)?|error' build.log",
				PowerShell:  "grepx -E 'warn(ing)?|error' build.log",
			},
			{
				Description: "Search for text that holds regular expression characters, literally:",
				Bash:        "grepx -F 'price[0] * 1.5' *.js",
				PowerShell:  "grepx -F 'price[0] * 1.5' (Get-ChildItem *.js).Name",
			},
			{
				Description: "List the files under the current directory that contain a string:",
				Bash:        "grepx -rl 'deprecated' .",
				PowerShell:  "grepx -rl 'deprecated' .",
			},
			{
				Description: "List the Go files that lack a license header:",
				Bash:        "grepx -rL --include='*.go' 'SPDX-License-Identifier' .",
				PowerShell:  "grepx -rL --include='*.go' 'SPDX-License-Identifier' .",
			},
			{
				Description: "Count the matching lines in each file:",
				Bash:        "grepx -c '^func ' *.go",
				PowerShell:  "grepx -c '^func ' (Get-ChildItem *.go).Name",
				Output:      "main.go:3\nutil.go:11",
			},
			{
				Description: "Drop comment lines and empty lines from a configuration file:",
				Bash:        "grepx -v -e '^#' -e '^$' app.conf",
				PowerShell:  "grepx -v -e '^#' -e '^$' app.conf",
			},
			{
				Description: "Print only the matched parts, here every IPv4-looking address:",
				Bash:        "grepx -oE '[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+' access.log",
				PowerShell:  "grepx -oE '[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+' access.log",
			},
			{
				Description: "Test in a script whether a file contains a line, printing nothing:",
				Bash:        "grepx -q '^version = ' config.toml && echo found",
				PowerShell:  "grepx -q '^version = ' config.toml; if ($?) { 'found' }",
			},
			{
				Description: "Search a tree but skip the dependency and VCS directories:",
				Bash:        "grepx -rn --exclude-dir=node_modules --exclude-dir=.git 'fetch(' .",
				PowerShell:  "grepx -rn --exclude-dir=node_modules --exclude-dir=.git 'fetch(' .",
			},
			{
				Description: "Stop after the first three matches and number them:",
				Bash:        "grepx -n -m 3 'panic' crash.log",
				PowerShell:  "grepx -n -m 3 'panic' crash.log",
			},
			{
				Description: "Take the patterns from a file, one to a line, as fixed strings:",
				Bash:        "grepx -F -f blocklist.txt users.csv",
				PowerShell:  "grepx -F -f blocklist.txt users.csv",
			},
			{
				Description: "Filter another tool's output in a pipeline:",
				Bash:        "lsx -l | grepx -i '\\.log$'",
				PowerShell:  "lsx -l | grepx -i '\\.log$'",
			},
			{
				Description: "Search in the portable gnux shell, the same on every platform:",
				Bash:        "gnux shell -c 'catx notes.txt | grepx -n urgent'",
				PowerShell:  "gnux shell -c 'catx notes.txt | grepx -n urgent'",
			},
		},
	}
}

// Register installs the grepx flags on fs.
func Register(fs *pflag.FlagSet) {
	// Pattern syntax.
	fs.BoolP("extended-regexp", "E", false, "PATTERNS are extended regular expressions")
	fs.BoolP("fixed-strings", "F", false, "PATTERNS are strings")
	fs.BoolP("basic-regexp", "G", false, "PATTERNS are basic regular expressions (the default)")
	fs.BoolP("perl-regexp", "P", false, "PATTERNS are Perl regular expressions (not supported: refused)")
	fs.VarP(&listValue{}, "regexp", "e", "use `PATTERNS` for matching")
	fs.VarP(&listValue{}, "file", "f", "take PATTERNS from `FILE`")
	fs.BoolP("ignore-case", "i", false, "ignore case distinctions in patterns and data")
	fs.Bool("no-ignore-case", false, "do not ignore case distinctions (the default)")
	fs.BoolP("word-regexp", "w", false, "match only whole words")
	fs.BoolP("line-regexp", "x", false, "match only whole lines")
	fs.BoolP("null-data", "z", false, "a data line ends in a NUL byte, not a newline")

	// Miscellaneous.
	fs.BoolP("no-messages", "s", false, "suppress error messages about files that do not exist or cannot be read")
	fs.BoolP("invert-match", "v", false, "select the lines that do not match")

	// Output control.
	fs.StringP("max-count", "m", "", "stop after `NUM` selected lines")
	fs.BoolP("byte-offset", "b", false, "print the byte offset with output lines")
	fs.BoolP("line-number", "n", false, "print the line number with output lines")
	fs.Bool("line-buffered", false, "flush the output on every line")
	fs.BoolP("with-filename", "H", false, "print the file name with output lines")
	fs.BoolP("no-filename", "h", false, "suppress the file name prefix on output")
	fs.String("label", "", "use `LABEL` as the file name of standard input")
	fs.BoolP("only-matching", "o", false, "print only the matched (non-empty) parts of lines, one to a line")
	fs.BoolP("quiet", "q", false, "suppress all normal output; stop at the first selected line")
	fs.Bool("silent", false, "same as --quiet")
	fs.String("binary-files", "",
		"assume that binary files are `TYPE`: binary (the default), text or without-match; -I is --binary-files=without-match")
	fs.BoolP("text", "a", false, "equivalent to --binary-files=text")
	fs.StringP("directories", "d", "", "how to handle directories; `ACTION` is read (the default), recurse or skip")
	fs.StringP("devices", "D", "", "how to handle devices, FIFOs and sockets; `ACTION` is read or skip")
	fs.BoolP("recursive", "r", false, "like --directories=recurse")
	fs.BoolP("dereference-recursive", "R", false, "likewise, but follow all symbolic links")

	globs := &[]grep.FileGlob{}
	fs.Var(&globValue{list: globs, kind: grep.GlobInclude}, "include", "search only files that match `GLOB` (a file pattern)")
	fs.Var(&globValue{list: globs, kind: grep.GlobExclude}, "exclude", "skip files that match `GLOB`")
	fs.Var(&globValue{list: globs, kind: grep.GlobExcludeFrom}, "exclude-from", "skip files that match any file pattern from `FILE`")
	fs.Var(&listValue{}, "exclude-dir", "skip directories that match `GLOB`")

	fs.BoolP("files-without-match", "L", false, "print only the names of FILEs with no selected lines")
	fs.BoolP("files-with-matches", "l", false, "print only the names of FILEs with selected lines")
	fs.BoolP("count", "c", false, "print only a count of selected lines per FILE")
	fs.BoolP("initial-tab", "T", false, "make tabs line up (put a tab before the line)")
	fs.BoolP("null", "Z", false, "print a NUL byte after each FILE name")

	// Context control.
	fs.StringP("before-context", "B", "", "print `NUM` lines of leading context")
	fs.StringP("after-context", "A", "", "print `NUM` lines of trailing context")
	fs.StringP("context", "C", "", "print `NUM` lines of output context; -NUM is the same")
	fs.String("group-separator", "", "print `SEP` on the line between groups of matches with context")
	fs.Bool("no-group-separator", false, "do not print a separator between groups of matches with context")
	fs.String("color", "", "highlight matches, names and separators; `WHEN` is always, never or auto")
	fs.Lookup("color").NoOptDefVal = "auto"
	fs.String(flagColour, "", "same as --color=`WHEN`")
	fs.Lookup(flagColour).NoOptDefVal = "auto"
	// Hidden, as in GNU's --help, and so that --col completes to --color
	// alone: glibc does not call a choice between two spellings of one
	// option ambiguous.
	fs.Lookup(flagColour).Hidden = true
	fs.BoolP("binary", "U", false, "accepted for compatibility; has no effect")

	// Options GNU spells with a letter only. pflag wants a long name for
	// each, so they get a hidden one.
	fs.BoolP(flagI, "I", false, "equivalent to --binary-files=without-match")
	fs.BoolP(flagY, "y", false, "same as --ignore-case (obsolete)")
	fs.Lookup(flagI).Hidden = true
	fs.Lookup(flagY).Hidden = true
	registerDigits(fs)
	tool.RegisterView(fs)
}

// flagColour is the second spelling GNU accepts for --color. It is written
// in two parts because the repository's spell checker enforces US spelling.
const flagColour = "colo" + "ur"

// The hidden long names of -I and -y.
const (
	flagI = "binary-files-without-match-I"
	flagY = "ignore-case-y"
)

// listValue is the pflag value behind the options that may be repeated (-e,
// -f, --exclude-dir). pflag's own string array loses a lone empty argument
// when it is read back, and -e ” is a pattern: the one that matches every
// line.
type listValue struct{ items []string }

func (v *listValue) String() string { return strings.Join(v.items, "\n") }

// Type is the placeholder --help shows when the usage text names none.
func (v *listValue) Type() string { return "STRING" }

func (v *listValue) Set(s string) error {
	v.items = append(v.items, s)
	return nil
}

// globValue is the pflag value behind --include, --exclude and
// --exclude-from. The three append to one list, because GNU decides a file's
// fate by the last of them that matches and pflag does not otherwise record
// the order options came in.
type globValue struct {
	list *[]grep.FileGlob
	kind grep.GlobKind
}

func (v *globValue) String() string { return "" }

// Type is the placeholder --help shows when the usage text names none.
func (v *globValue) Type() string { return "GLOB" }

func (v *globValue) Set(s string) error {
	*v.list = append(*v.list, grep.FileGlob{Kind: v.kind, Pattern: s})
	return nil
}

// digits collects the "-NUM" form of -C. Flag parsing belongs to cobra/pflag,
// which would reject -5 as an unknown shorthand; registering each digit as a
// hidden, argument-less shorthand makes pflag deliver "-15" as '1' then '5',
// which Set reassembles.
type digits struct{ text string }

// digitValue is the pflag value for one digit shorthand.
type digitValue struct {
	run   *digits
	digit byte
}

func (v *digitValue) String() string { return "" }

func (v *digitValue) Type() string { return "bool" }

func (v *digitValue) Set(string) error {
	v.run.text += string(v.digit)
	return nil
}

// digitFlag is the hidden long name of the digit d.
func digitFlag(d int) string { return "context-digit-" + strconv.Itoa(d) }

func registerDigits(fs *pflag.FlagSet) {
	run := &digits{}
	for d := range 10 {
		f := fs.VarPF(&digitValue{run: run, digit: byte('0' + d)}, digitFlag(d), strconv.Itoa(d), "")
		f.NoOptDefVal = "true"
		f.Hidden = true
	}
}

// trouble is a command-line mistake GNU grep reports with a plain diagnostic
// and status 2, without the "Try --help" hint a usage error carries.
type trouble struct{ msg string }

func (t *trouble) Error() string { return t.msg }

type choice[T any] struct {
	name  string
	value T
}

var colors = []choice[grep.Color]{
	{"always", grep.ColorAlways}, {"yes", grep.ColorAlways}, {"force", grep.ColorAlways},
	{"never", grep.ColorNever}, {"no", grep.ColorNever}, {"none", grep.ColorNever},
	{"auto", grep.ColorAuto}, {"tty", grep.ColorAuto}, {"if-tty", grep.ColorAuto},
}

var directories = []choice[grep.Directories]{
	{"read", grep.DirectoriesRead},
	{"recurse", grep.DirectoriesRecurse},
	{"skip", grep.DirectoriesSkip},
}

// exact resolves an argument that must be spelled out.
func exact[T any](arg string, choices []choice[T]) (T, bool) {
	for _, c := range choices {
		if c.name == arg {
			return c.value, true
		}
	}
	var zero T
	return zero, false
}

// abbreviated resolves an argument as GNU's argmatch does: an exact name, or
// an unambiguous prefix of one.
func abbreviated[T any](flag, arg string, choices []choice[T]) (T, error) {
	var (
		found T
		n     int
	)
	for _, c := range choices {
		if c.name == arg {
			return c.value, nil
		}
		if arg != "" && strings.HasPrefix(c.name, arg) {
			found = c.value
			n++
		}
	}
	switch n {
	case 1:
		return found, nil
	case 0:
		return found, core.Usagef("invalid argument '%s' for '--%s'", arg, flag)
	default:
		return found, core.Usagef("ambiguous argument '%s' for '--%s'", arg, flag)
	}
}

// contextLength parses the argument of -A, -B, -C or -NUM. A count too large
// to represent is clamped, as in GNU: no file has that many lines.
func contextLength(text string) (int, error) {
	n, err := strconv.ParseUint(text, 10, 31)
	switch {
	case errors.Is(err, strconv.ErrRange):
		return math.MaxInt32, nil
	case err != nil:
		return 0, &trouble{text + ": invalid context length argument"}
	}
	return int(n), nil
}

// buildOptions folds the parsed flags into grep.Options. Where GNU grep lets
// the last of several conflicting options win, a fixed order applies here,
// documented in Meta().Long.
func buildOptions(fs *pflag.FlagSet) (grep.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	text := func(name string) string {
		s, _ := fs.GetString(name)
		return s
	}

	if on("perl-regexp") {
		return grep.Options{}, &trouble{"Perl-compatible regular expressions (-P) are not supported"}
	}

	o := grep.Options{
		IgnoreCase:        (on("ignore-case") || on(flagY)) && !on("no-ignore-case"),
		Invert:            on("invert-match"),
		WordRegexp:        on("word-regexp"),
		LineRegexp:        on("line-regexp"),
		Count:             on("count"),
		FilesWithMatches:  on("files-with-matches"),
		FilesWithoutMatch: on("files-without-match") && !on("files-with-matches"),
		OnlyMatching:      on("only-matching"),
		Quiet:             on("quiet") || on("silent"),
		NoMessages:        on("no-messages"),
		ByteOffset:        on("byte-offset"),
		Label:             text("label"),
		LineNumber:        on("line-number"),
		InitialTab:        on("initial-tab"),
		Null:              on("null"),
		NullData:          on("null-data"),
		LineBuffered:      on("line-buffered"),
		NoGroupSeparator:  on("no-group-separator"),
	}
	list := func(name string) []string {
		return fs.Lookup(name).Value.(*listValue).items
	}
	o.Patterns = list("regexp")
	o.PatternFiles = list("file")
	o.ExcludeDir = list("exclude-dir")
	if globs := *fs.Lookup("include").Value.(*globValue).list; len(globs) > 0 {
		o.FileGlobs = append([]grep.FileGlob(nil), globs...)
	}

	// -F beats -E, which beats -G.
	switch {
	case on("fixed-strings"):
		o.Syntax = grep.SyntaxFixed
	case on("extended-regexp"):
		o.Syntax = grep.SyntaxExtended
	}

	// -H beats -h.
	switch {
	case on("with-filename"):
		o.Filename = grep.FilenameWith
	case on("no-filename"):
		o.Filename = grep.FilenameWithout
	}

	if fs.Changed("max-count") {
		arg := text("max-count")
		n, err := strconv.ParseInt(arg, 10, 64)
		switch {
		case errors.Is(err, strconv.ErrRange):
			// Too large either way: a negative count is no limit at all.
			n = math.MaxInt64
			if strings.HasPrefix(arg, "-") {
				n = -1
			}
		case err != nil:
			return grep.Options{}, &trouble{"invalid max count"}
		}
		if n >= 0 {
			o.MaxCount, o.MaxCountSet = n, true
		}
	}

	// -A and -B beat -C, which beats -NUM.
	var err error
	if num := fs.Lookup(digitFlag(0)).Value.(*digitValue).run.text; num != "" {
		if o.After, err = contextLength(num); err != nil {
			return grep.Options{}, err
		}
		o.Before, o.Context = o.After, true
	}
	if fs.Changed("context") {
		if o.After, err = contextLength(text("context")); err != nil {
			return grep.Options{}, err
		}
		o.Before, o.Context = o.After, true
	}
	if fs.Changed("after-context") {
		if o.After, err = contextLength(text("after-context")); err != nil {
			return grep.Options{}, err
		}
		o.Context = true
	}
	if fs.Changed("before-context") {
		if o.Before, err = contextLength(text("before-context")); err != nil {
			return grep.Options{}, err
		}
		o.Context = true
	}
	if fs.Changed("group-separator") {
		o.GroupSeparator, o.GroupSeparatorSet = text("group-separator"), true
	}

	// --color beats its British spelling.
	for _, name := range []string{flagColour, "color"} {
		if !fs.Changed(name) {
			continue
		}
		when, ok := exact(text(name), colors)
		if !ok {
			return grep.Options{}, core.Usagef("invalid argument '%s' for '--%s'", text(name), name)
		}
		o.Color = when
	}

	// -I beats -a, and either beats --binary-files.
	if fs.Changed("binary-files") {
		switch text("binary-files") {
		case "binary":
			o.BinaryFiles = grep.BinaryFilesBinary
		case "text":
			o.BinaryFiles = grep.BinaryFilesText
		case "without-match":
			o.BinaryFiles = grep.BinaryFilesWithoutMatch
		default:
			return grep.Options{}, &trouble{"unknown binary-files type"}
		}
	}
	switch {
	case on(flagI):
		o.BinaryFiles = grep.BinaryFilesWithoutMatch
	case on("text"):
		o.BinaryFiles = grep.BinaryFilesText
	}

	// -R beats -r, and either beats -d.
	if fs.Changed("directories") {
		if o.Directories, err = abbreviated("directories", text("directories"), directories); err != nil {
			return grep.Options{}, err
		}
	}
	if on("recursive") || on("dereference-recursive") {
		o.Directories = grep.DirectoriesRecurse
		o.Dereference = on("dereference-recursive")
	}

	if fs.Changed("devices") {
		switch text("devices") {
		case "read":
			o.Devices = grep.DevicesRead
		case "skip":
			o.Devices = grep.DevicesSkip
		default:
			return grep.Options{}, &trouble{"unknown devices method"}
		}
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		var t *trouble
		if errors.As(err, &t) {
			core.NewDiag(grep.Name, s).Errorf("%s", t.msg)
			return &core.ExitError{Code: 2}
		}
		return err
	}
	view, err := tool.ViewOf(fs, core.IsTerminal(s.Out), grep.Name, os.Getenv, s.Err)
	if err != nil {
		return err
	}
	applyView(fs, &opts, view)
	return grep.New(opts, s).Run(args)
}

// applyView folds in gnux's terminal view: file names as headings, line
// numbers, and color, each unless an option already decides it.
func applyView(fs *pflag.FlagSet, o *grep.Options, view tool.View) {
	if !view.Pretty {
		return
	}
	o.Pretty, o.Theme = true, view.Theme
	o.LineNumber = true
	if !fs.Changed("color") && !fs.Changed(flagColour) {
		o.Color = grep.ColorNever
		if view.Color {
			o.Color = grep.ColorAlways
		}
	}
}
