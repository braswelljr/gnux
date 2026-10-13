package grepx

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/grep"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (grep.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(grep.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	context := func(before, after int) grep.Options {
		return grep.Options{Before: before, After: after, Context: true}
	}
	recurse := grep.Options{Directories: grep.DirectoriesRecurse}
	deref := grep.Options{Directories: grep.DirectoriesRecurse, Dereference: true}
	tests := []struct {
		name string
		args []string
		want grep.Options
	}{
		{"defaults", nil, grep.Options{}},

		// Pattern syntax: -F beats -E, which beats -G.
		{"basic", []string{"-G"}, grep.Options{}},
		{"basic long", []string{"--basic-regexp"}, grep.Options{}},
		{"extended", []string{"-E"}, grep.Options{Syntax: grep.SyntaxExtended}},
		{"extended long", []string{"--extended-regexp"}, grep.Options{Syntax: grep.SyntaxExtended}},
		{"fixed", []string{"-F"}, grep.Options{Syntax: grep.SyntaxFixed}},
		{"fixed long", []string{"--fixed-strings"}, grep.Options{Syntax: grep.SyntaxFixed}},
		{"fixed then extended", []string{"-F", "-E"}, grep.Options{Syntax: grep.SyntaxFixed}},
		{"extended then fixed", []string{"-E", "-F"}, grep.Options{Syntax: grep.SyntaxFixed}},
		{"extended then basic", []string{"-E", "-G"}, grep.Options{Syntax: grep.SyntaxExtended}},
		{"basic then extended", []string{"-G", "-E"}, grep.Options{Syntax: grep.SyntaxExtended}},
		{"fixed then basic", []string{"-F", "-G"}, grep.Options{Syntax: grep.SyntaxFixed}},
		{"basic then fixed", []string{"-G", "-F"}, grep.Options{Syntax: grep.SyntaxFixed}},

		// Patterns.
		{"pattern", []string{"-e", "a"}, grep.Options{Patterns: []string{"a"}}},
		{"pattern attached", []string{"-ea"}, grep.Options{Patterns: []string{"a"}}},
		{"pattern long", []string{"--regexp=a"}, grep.Options{Patterns: []string{"a"}}},
		{"patterns", []string{"-e", "a", "-e", "b,c"}, grep.Options{Patterns: []string{"a", "b,c"}}},
		{"pattern that looks like an option", []string{"-e", "-v"}, grep.Options{Patterns: []string{"-v"}}},
		{"empty pattern", []string{"-e", ""}, grep.Options{Patterns: []string{""}}},
		{"pattern file", []string{"-f", "p"}, grep.Options{PatternFiles: []string{"p"}}},
		{"pattern files long", []string{"--file=p", "--file=q"}, grep.Options{PatternFiles: []string{"p", "q"}}},

		// Matching.
		{"ignore case", []string{"-i"}, grep.Options{IgnoreCase: true}},
		{"ignore case long", []string{"--ignore-case"}, grep.Options{IgnoreCase: true}},
		{"ignore case, obsolete spelling", []string{"-y"}, grep.Options{IgnoreCase: true}},
		{"no ignore case", []string{"--no-ignore-case"}, grep.Options{}},
		{"ignore case then not", []string{"-i", "--no-ignore-case"}, grep.Options{}},
		{"not, then ignore case", []string{"--no-ignore-case", "-i"}, grep.Options{}},
		{"-y then not", []string{"-y", "--no-ignore-case"}, grep.Options{}},
		{"not, then -y", []string{"--no-ignore-case", "-y"}, grep.Options{}},
		{"invert", []string{"-v"}, grep.Options{Invert: true}},
		{"invert long", []string{"--invert-match"}, grep.Options{Invert: true}},
		{"word", []string{"-w"}, grep.Options{WordRegexp: true}},
		{"word long", []string{"--word-regexp"}, grep.Options{WordRegexp: true}},
		{"line", []string{"-x"}, grep.Options{LineRegexp: true}},
		{"line long", []string{"--line-regexp"}, grep.Options{LineRegexp: true}},
		{"null data", []string{"-z"}, grep.Options{NullData: true}},
		{"null data long", []string{"--null-data"}, grep.Options{NullData: true}},
		{"no messages", []string{"-s"}, grep.Options{NoMessages: true}},
		{"no messages long", []string{"--no-messages"}, grep.Options{NoMessages: true}},
		{"binary is accepted and ignored", []string{"-U"}, grep.Options{}},
		{"binary long", []string{"--binary"}, grep.Options{}},

		// Output control. -l beats -L.
		{"count", []string{"-c"}, grep.Options{Count: true}},
		{"count long", []string{"--count"}, grep.Options{Count: true}},
		{"files with matches", []string{"-l"}, grep.Options{FilesWithMatches: true}},
		{"files with matches long", []string{"--files-with-matches"}, grep.Options{FilesWithMatches: true}},
		{"files without match", []string{"-L"}, grep.Options{FilesWithoutMatch: true}},
		{"files without match long", []string{"--files-without-match"}, grep.Options{FilesWithoutMatch: true}},
		{"-l then -L", []string{"-l", "-L"}, grep.Options{FilesWithMatches: true}},
		{"-L then -l", []string{"-L", "-l"}, grep.Options{FilesWithMatches: true}},
		{"max count", []string{"-m", "3"}, grep.Options{MaxCount: 3, MaxCountSet: true}},
		{"max count attached", []string{"-m3"}, grep.Options{MaxCount: 3, MaxCountSet: true}},
		{"max count long", []string{"--max-count=7"}, grep.Options{MaxCount: 7, MaxCountSet: true}},
		{"max count of zero", []string{"-m", "0"}, grep.Options{MaxCountSet: true}},
		{"negative max count is no limit", []string{"-m", "-1"}, grep.Options{}},
		{"huge max count is clamped", []string{"-m", "99999999999999999999"}, grep.Options{MaxCount: math.MaxInt64, MaxCountSet: true}},
		{"hugely negative max count is no limit", []string{"--max-count=-99999999999999999999"}, grep.Options{}},
		{"the last max count wins", []string{"-m", "3", "-m", "5"}, grep.Options{MaxCount: 5, MaxCountSet: true}},
		{"only matching", []string{"-o"}, grep.Options{OnlyMatching: true}},
		{"only matching long", []string{"--only-matching"}, grep.Options{OnlyMatching: true}},
		{"quiet", []string{"-q"}, grep.Options{Quiet: true}},
		{"quiet long", []string{"--quiet"}, grep.Options{Quiet: true}},
		{"silent", []string{"--silent"}, grep.Options{Quiet: true}},
		{"line buffered", []string{"--line-buffered"}, grep.Options{LineBuffered: true}},

		// Line prefix. -H beats -h.
		{"byte offset", []string{"-b"}, grep.Options{ByteOffset: true}},
		{"byte offset long", []string{"--byte-offset"}, grep.Options{ByteOffset: true}},
		{"line number", []string{"-n"}, grep.Options{LineNumber: true}},
		{"line number long", []string{"--line-number"}, grep.Options{LineNumber: true}},
		{"with filename", []string{"-H"}, grep.Options{Filename: grep.FilenameWith}},
		{"with filename long", []string{"--with-filename"}, grep.Options{Filename: grep.FilenameWith}},
		{"no filename", []string{"-h"}, grep.Options{Filename: grep.FilenameWithout}},
		{"no filename long", []string{"--no-filename"}, grep.Options{Filename: grep.FilenameWithout}},
		{"-H then -h", []string{"-H", "-h"}, grep.Options{Filename: grep.FilenameWith}},
		{"-h then -H", []string{"-h", "-H"}, grep.Options{Filename: grep.FilenameWith}},
		{"label", []string{"--label=in"}, grep.Options{Label: "in"}},
		{"initial tab", []string{"-T"}, grep.Options{InitialTab: true}},
		{"initial tab long", []string{"--initial-tab"}, grep.Options{InitialTab: true}},
		{"null", []string{"-Z"}, grep.Options{Null: true}},
		{"null long", []string{"--null"}, grep.Options{Null: true}},
		{"clustered", []string{"-rnwi"}, grep.Options{Directories: grep.DirectoriesRecurse, LineNumber: true, WordRegexp: true, IgnoreCase: true}},

		// Context. -A and -B beat -C, which beats -NUM.
		{"after", []string{"-A", "2"}, context(0, 2)},
		{"after long", []string{"--after-context=2"}, context(0, 2)},
		{"before", []string{"-B", "3"}, context(3, 0)},
		{"before long", []string{"--before-context=3"}, context(3, 0)},
		{"context", []string{"-C", "4"}, context(4, 4)},
		{"context attached", []string{"-C4"}, context(4, 4)},
		{"context long", []string{"--context=4"}, context(4, 4)},
		{"context of zero is still context", []string{"-A", "0"}, context(0, 0)},
		{"digits", []string{"-5"}, context(5, 5)},
		{"several digits", []string{"-12"}, context(12, 12)},
		{"digits among letters", []string{"-n2i"}, grep.Options{LineNumber: true, IgnoreCase: true, Before: 2, After: 2, Context: true}},
		{"-A then -C", []string{"-A", "1", "-C", "4"}, context(4, 1)},
		{"-C then -A", []string{"-C", "4", "-A", "1"}, context(4, 1)},
		{"-B then -C", []string{"-B", "1", "-C", "4"}, context(1, 4)},
		{"-C then -B", []string{"-C", "4", "-B", "1"}, context(1, 4)},
		{"-C then digits", []string{"-C", "4", "-2"}, context(4, 4)},
		{"digits then -C", []string{"-2", "-C", "4"}, context(4, 4)},
		{"-A then digits", []string{"-A", "1", "-7"}, context(7, 1)},
		{"digits then -A", []string{"-7", "-A", "1"}, context(7, 1)},
		{"-B then digits", []string{"-B", "1", "-7"}, context(1, 7)},
		{"digits then -B", []string{"-7", "-B", "1"}, context(1, 7)},
		{"-A, -B and -C", []string{"-C", "9", "-A", "1", "-B", "2"}, context(2, 1)},
		{"the last -A wins", []string{"-A", "1", "-A", "6"}, context(0, 6)},
		{"huge context is clamped", []string{"-C", "99999999999999999999"}, context(math.MaxInt32, math.MaxInt32)},
		{"group separator", []string{"--group-separator==="}, grep.Options{GroupSeparator: "==", GroupSeparatorSet: true}},
		{"empty group separator", []string{"--group-separator="}, grep.Options{GroupSeparatorSet: true}},
		{"no group separator", []string{"--no-group-separator"}, grep.Options{NoGroupSeparator: true}},
		{"separator then none", []string{"--group-separator=x", "--no-group-separator"}, grep.Options{GroupSeparator: "x", GroupSeparatorSet: true, NoGroupSeparator: true}},
		{"none then separator", []string{"--no-group-separator", "--group-separator=x"}, grep.Options{GroupSeparator: "x", GroupSeparatorSet: true, NoGroupSeparator: true}},

		// Color. --color beats its second spelling.
		{"color alone is auto", []string{"--color"}, grep.Options{Color: grep.ColorAuto}},
		{"second spelling alone is auto", []string{"--" + flagColour}, grep.Options{Color: grep.ColorAuto}},
		{"color always", []string{"--color=always"}, grep.Options{Color: grep.ColorAlways}},
		{"color yes", []string{"--color=yes"}, grep.Options{Color: grep.ColorAlways}},
		{"color force", []string{"--color=force"}, grep.Options{Color: grep.ColorAlways}},
		{"color never", []string{"--color=never"}, grep.Options{}},
		{"color no", []string{"--color=no"}, grep.Options{}},
		{"color none", []string{"--color=none"}, grep.Options{}},
		{"color auto", []string{"--color=auto"}, grep.Options{Color: grep.ColorAuto}},
		{"color tty", []string{"--color=tty"}, grep.Options{Color: grep.ColorAuto}},
		{"color if-tty", []string{"--color=if-tty"}, grep.Options{Color: grep.ColorAuto}},
		{"second spelling, always", []string{"--" + flagColour + "=always"}, grep.Options{Color: grep.ColorAlways}},
		{"the last color wins", []string{"--color=always", "--color=never"}, grep.Options{}},
		{"color then its second spelling", []string{"--color=always", "--" + flagColour + "=never"}, grep.Options{Color: grep.ColorAlways}},
		{"second spelling then color", []string{"--" + flagColour + "=never", "--color=always"}, grep.Options{Color: grep.ColorAlways}},

		// Binary files. -I beats -a, and either beats --binary-files.
		{"binary files binary", []string{"--binary-files=binary"}, grep.Options{}},
		{"binary files text", []string{"--binary-files=text"}, grep.Options{BinaryFiles: grep.BinaryFilesText}},
		{"binary files without-match", []string{"--binary-files=without-match"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},
		{"text", []string{"-a"}, grep.Options{BinaryFiles: grep.BinaryFilesText}},
		{"text long", []string{"--text"}, grep.Options{BinaryFiles: grep.BinaryFilesText}},
		{"-I", []string{"-I"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},
		{"-I then -a", []string{"-I", "-a"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},
		{"-a then -I", []string{"-a", "-I"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},
		{"-a then --binary-files", []string{"-a", "--binary-files=binary"}, grep.Options{BinaryFiles: grep.BinaryFilesText}},
		{"--binary-files then -a", []string{"--binary-files=without-match", "-a"}, grep.Options{BinaryFiles: grep.BinaryFilesText}},
		{"-I then --binary-files", []string{"-I", "--binary-files=text"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},
		{"--binary-files then -I", []string{"--binary-files=text", "-I"}, grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}},

		// Directories and devices. -R beats -r, and either beats -d.
		{"recursive", []string{"-r"}, recurse},
		{"recursive long", []string{"--recursive"}, recurse},
		{"dereference recursive", []string{"-R"}, deref},
		{"dereference recursive long", []string{"--dereference-recursive"}, deref},
		{"-r then -R", []string{"-r", "-R"}, deref},
		{"-R then -r", []string{"-R", "-r"}, deref},
		{"directories read", []string{"-d", "read"}, grep.Options{}},
		{"directories skip", []string{"-d", "skip"}, grep.Options{Directories: grep.DirectoriesSkip}},
		{"directories recurse", []string{"--directories=recurse"}, recurse},
		{"directories abbreviated", []string{"-d", "s"}, grep.Options{Directories: grep.DirectoriesSkip}},
		{"-r then -d", []string{"-r", "-d", "skip"}, recurse},
		{"-d then -r", []string{"-d", "skip", "-r"}, recurse},
		{"-R then -d", []string{"-R", "-d", "read"}, deref},
		{"-d then -R", []string{"-d", "read", "-R"}, deref},
		{"devices read", []string{"-D", "read"}, grep.Options{Devices: grep.DevicesRead}},
		{"devices skip", []string{"--devices=skip"}, grep.Options{Devices: grep.DevicesSkip}},

		// File selection keeps the order the options came in.
		{"include", []string{"--include=*.go"}, grep.Options{FileGlobs: []grep.FileGlob{{Kind: grep.GlobInclude, Pattern: "*.go"}}}},
		{"exclude", []string{"--exclude=*.o"}, grep.Options{FileGlobs: []grep.FileGlob{{Kind: grep.GlobExclude, Pattern: "*.o"}}}},
		{"exclude from", []string{"--exclude-from=list"}, grep.Options{FileGlobs: []grep.FileGlob{{Kind: grep.GlobExcludeFrom, Pattern: "list"}}}},
		{"include then exclude", []string{"--include=*.go", "--exclude=x*"}, grep.Options{FileGlobs: []grep.FileGlob{
			{Kind: grep.GlobInclude, Pattern: "*.go"}, {Kind: grep.GlobExclude, Pattern: "x*"},
		}}},
		{"exclude then include", []string{"--exclude=x*", "--include=*.go"}, grep.Options{FileGlobs: []grep.FileGlob{
			{Kind: grep.GlobExclude, Pattern: "x*"}, {Kind: grep.GlobInclude, Pattern: "*.go"},
		}}},
		{"all three, interleaved", []string{"--exclude=a", "--exclude-from=f", "--include=b", "--exclude=c"}, grep.Options{FileGlobs: []grep.FileGlob{
			{Kind: grep.GlobExclude, Pattern: "a"}, {Kind: grep.GlobExcludeFrom, Pattern: "f"},
			{Kind: grep.GlobInclude, Pattern: "b"}, {Kind: grep.GlobExclude, Pattern: "c"},
		}}},
		{"exclude dir", []string{"--exclude-dir=.git", "--exclude-dir=a,b"}, grep.Options{ExcludeDir: []string{".git", "a,b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) =\n%+v, want\n%+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		// trouble is a plain diagnostic with status 2; otherwise a usage error.
		trouble bool
		msg     string
	}{
		{"perl", []string{"-P"}, true, "Perl-compatible regular expressions (-P) are not supported"},
		{"perl long", []string{"--perl-regexp"}, true, "Perl-compatible regular expressions (-P) are not supported"},
		{"perl with another syntax", []string{"-E", "-P", "-F"}, true, "Perl-compatible regular expressions (-P) are not supported"},
		{"max count", []string{"-m", "x"}, true, "invalid max count"},
		{"empty max count", []string{"--max-count="}, true, "invalid max count"},
		{"after", []string{"-A", "x"}, true, "x: invalid context length argument"},
		{"before", []string{"-B", "-1"}, true, "-1: invalid context length argument"},
		{"context", []string{"-C", "1.5"}, true, "1.5: invalid context length argument"},
		{"binary files", []string{"--binary-files=maybe"}, true, "unknown binary-files type"},
		{"devices", []string{"-D", "recurse"}, true, "unknown devices method"},
		{"directories", []string{"-d", "walk"}, false, "invalid argument 'walk' for '--directories'"},
		{"ambiguous directories", []string{"-d", "re"}, false, "ambiguous argument 're' for '--directories'"},
		{"color", []string{"--color=sometimes"}, false, "invalid argument 'sometimes' for '--color'"},
		{"color, second spelling", []string{"--" + flagColour + "=sometimes"}, false, "invalid argument 'sometimes' for '--" + flagColour + "'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := options(t, tt.args...)
			var (
				tr    *trouble
				usage *core.UsageError
			)
			switch {
			case tt.trouble && errors.As(err, &tr):
			case !tt.trouble && errors.As(err, &usage):
			default:
				t.Fatalf("buildOptions(%v) error = %T %v", tt.args, err, err)
			}
			if err.Error() != tt.msg {
				t.Fatalf("buildOptions(%v) error = %q, want %q", tt.args, err, tt.msg)
			}
		})
	}
}

// command runs the tool as the binary would, through its cobra command.
func command(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	c.SetArgs(args)
	err = c.Execute()
	return out.String(), errb.String(), err
}

func exitCode(err error) int {
	var exit *core.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.Code
	}
	return -1
}

func TestCommand(t *testing.T) {
	const input = "one\ntwo\nthree\nfour\n"
	tests := []struct {
		name string
		args []string
		out  string
		errs string
		code int
	}{
		{"pattern operand", []string{"t"}, "two\nthree\n", "", 0},
		{"no match", []string{"zzz"}, "", "", 1},
		{"-e", []string{"-e", "one", "-e", "four"}, "one\nfour\n", "", 0},
		{"pattern after --", []string{"--", "-x"}, "", "", 1},
		{"-NUM", []string{"-1", "three"}, "two\nthree\nfour\n", "", 0},
		{"-h is not help", []string{"-h", "one"}, "one\n", "", 0},
		{"extended", []string{"-E", "^(one|four)$"}, "one\nfour\n", "", 0},
		{"color with no argument does not take the pattern", []string{"--color", "one"}, "one\n", "", 0},
		{"perl is refused", []string{"-P", "one"}, "", "grepx: Perl-compatible regular expressions (-P) are not supported\n", 2},
		{"bad context", []string{"-A", "x", "one"}, "", "grepx: x: invalid context length argument\n", 2},
		{"bad pattern", []string{"["}, "", "grepx: Unmatched [, [^, [:, [., or [=\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errs, err := command(t, input, tt.args...)
			if out != tt.out || errs != tt.errs || exitCode(err) != tt.code {
				t.Fatalf("grepx %v: stdout %q, stderr %q, status %d (%v); want %q, %q, %d",
					tt.args, out, errs, exitCode(err), err, tt.out, tt.errs, tt.code)
			}
		})
	}

	t.Run("a missing pattern is a usage error", func(t *testing.T) {
		_, _, err := command(t, input)
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("err = %T %v, want a *core.UsageError", err, err)
		}
	})
	t.Run("usage errors exit with 2", func(t *testing.T) {
		if got := Spec().UsageExit; got != 2 {
			t.Fatalf("UsageExit = %d, want 2", got)
		}
	})
	t.Run("the help lists -I with --binary-files and hides the plumbing", func(t *testing.T) {
		out, _, err := command(t, "", "--help")
		if err != nil {
			t.Fatal(err)
		}
		for _, hidden := range []string{flagI, flagY, digitFlag(0)} {
			if strings.Contains(out, hidden) {
				t.Errorf("--help shows the hidden flag %q", hidden)
			}
		}
		// The option list is wrapped, so compare without the line breaks.
		if !strings.Contains(strings.Join(strings.Fields(out), " "), "-I is --binary-files=without-match") {
			t.Errorf("--help does not mention -I:\n%s", out)
		}
	})
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != grep.Name || !strings.HasPrefix(m.Use, grep.Name+" ") {
		t.Fatalf("Name %q, Use %q", m.Name, m.Use)
	}
	if !strings.Contains(m.Long, "Differences from GNU grep:") {
		t.Error("Long has no \"Differences from GNU grep:\" passage")
	}
	if len(m.Examples) < 8 {
		t.Errorf("%d examples, want at least 8", len(m.Examples))
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Errorf("incomplete example: %+v", e)
		}
	}
	if len(m.ExitStatus) != 3 {
		t.Errorf("%d exit statuses documented, want 3", len(m.ExitStatus))
	}
}
