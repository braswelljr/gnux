package grep_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/grep"
)

// result is what one run produced.
type result struct {
	out, err string
	// status is the exit status the run maps to; -1 stands for a usage error.
	status int
}

// run drives the engine with stdin as standard input.
func run(t *testing.T, opts grep.Options, stdin string, args ...string) result {
	t.Helper()
	return runWith(t, opts, stdin, nil, args...)
}

// runWith is run, with a chance to set the injection points first.
func runWith(t *testing.T, opts grep.Options, stdin string, tune func(*grep.Grep), args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	g := grep.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	g.Getenv = func(string) string { return "" }
	if tune != nil {
		tune(g)
	}
	code := status(t, g.Run(args))
	return result{out.String(), errb.String(), code}
}

func status(t *testing.T, err error) int {
	t.Helper()
	var exit *core.ExitError
	var usage *core.UsageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.Code
	case errors.As(err, &usage):
		return -1
	}
	t.Fatalf("unexpected error type %T: %v", err, err)
	return 0
}

// check compares a result with what was expected.
func check(t *testing.T, got result, out, errText string, status int) {
	t.Helper()
	if got.out != out {
		t.Errorf("stdout = %q, want %q", got.out, out)
	}
	if got.err != errText {
		t.Errorf("stderr = %q, want %q", got.err, errText)
	}
	if got.status != status {
		t.Errorf("status = %d, want %d", got.status, status)
	}
}

// tree creates files under a new temporary directory, makes it the working
// directory and returns it. A name ending in a slash is a directory.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

// p turns a slash-separated path into the platform's form, which is how
// names found by descending a directory are printed.
func p(path string) string { return filepath.FromSlash(path) }

func TestBasicRegexp(t *testing.T) {
	tests := []struct {
		name, pattern, input, want string
	}{
		{"literal", "abc", "xabcx\nabd\n", "xabcx\n"},
		{"dot", "a.c", "abc\na-c\nac\n", "abc\na-c\n"},
		{"start anchor", "^foo", "foo\nxfoo\n", "foo\n"},
		{"end anchor", "foo$", "foo\nfoox\n", "foo\n"},
		{"caret is literal inside", "a^b", "a^b\nab\n", "a^b\n"},
		{"dollar is literal inside", "a$b", "a$b\nab\n", "a$b\n"},
		{"star", "ab*c", "ac\nabbc\nadc\n", "ac\nabbc\n"},
		{"leading star is literal", "*a", "*a\na\n", "*a\n"},
		{"star after anchor is literal", "^*a", "*a\nxa\n", "*a\n"},
		{"star after open group is literal", `\(*a\)`, "*a\na\n", "*a\n"},
		{"interval", `a\{2\}`, "a\naa\n", "aa\n"},
		{"open interval", `^a\{2,\}$`, "a\naa\naaa\n", "aa\naaa\n"},
		{"bounded interval", `^a\{1,2\}$`, "a\naa\naaa\n", "a\naa\n"},
		{"group", `\(ab\)*c`, "ababc\nd\n", "ababc\n"},
		{"alternation", `cat\|dog`, "cat\ndog\ncow\n", "cat\ndog\n"},
		{"anchor in each branch", `^a\|^b`, "ax\nbx\nxa\nxb\n", "ax\nbx\n"},
		{"anchor at end of branch", `a$\|b$`, "xa\nxb\nax\n", "xa\nxb\n"},
		{"anchor inside group", `\(^a\)`, "ab\nba\n", "ab\n"},
		{"plus is literal", "a+", "a+\naa\n", "a+\n"},
		{"escaped plus repeats", `^a\+$`, "aaa\nb\n", "aaa\n"},
		{"question is literal", "a?", "a?\na\n", "a?\n"},
		{"escaped question is optional", `^ab\?$`, "a\nab\nabb\n", "a\nab\n"},
		{"braces are literal", "a{1}", "a{1}\na\n", "a{1}\n"},
		{"parentheses are literal", "(a)", "(a)\na\n", "(a)\n"},
		{"bar is literal", "a|b", "a|b\na\n", "a|b\n"},
		{"escaped dot", `a\.c`, "a.c\nabc\n", "a.c\n"},
		{"class", "[[:digit:]]", "a1\nab\n", "a1\n"},
		{"unicode letter class", "^[[:alpha:]]*$", "café\nabc1\n", "café\n"},
		{"upper class", "[[:upper:]]", "aB\nab\n", "aB\n"},
		{"negated set", "^[^a]", "abc\nbcd\n", "bcd\n"},
		{"bracket first in set", "[]x]", "a]\nb\n", "a]\n"},
		{"backslash is literal in set", `[\]`, "a\\b\nab\n", "a\\b\n"},
		{"range", "^[a-c]*$", "abc\nabd\n", "abc\n"},
		{"hyphen last in set", "[a-]", "x-y\nxy\n", "x-y\n"},
		{"equivalence class", "[[=a=]]", "a\nb\n", "a\n"},
		{"collating symbol", "[[.-.]]", "a-b\nab\n", "a-b\n"},
		{"word start and end", `\<cat\>`, "a cat\ncats\nconcat\n", "a cat\n"},
		{"word boundary", `\bcat\b`, "a cat\ncats\n", "a cat\n"},
		{"not a boundary", `\Bcat`, "concat\ncat\n", "concat\n"},
		{"word characters", `^\w\w*$`, "ab_1\na b\n", "ab_1\n"},
		{"unicode word characters", `^\w*$`, "naïve\na-b\n", "naïve\n"},
		{"non-word character", `a\Wb`, "a b\nab\n", "a b\n"},
		{"space", `a\sb`, "a\tb\nab\n", "a\tb\n"},
		{"non-space", `^\S*$`, "ab\na b\n", "ab\n"},
		{"buffer anchors", "\\`a.*c\\'", "abc\nxabc\n", "abc\n"},
		{"repeated repetition", `^a**$`, "aaa\nb\n", "aaa\n"},
		{"empty pattern matches all", "", "a\n\nb\n", "a\n\nb\n"},
		{"newline separates patterns", "a\nb", "a\nb\nc\n", "a\nb\n"},
		{"longest match wins in output", `a\|ab`, "ab\n", "ab\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, grep.Options{}, tt.input, tt.pattern), tt.want, "", 0)
		})
	}
}

func TestExtendedRegexp(t *testing.T) {
	tests := []struct {
		name, pattern, input, want string
	}{
		{"plus", "^a+$", "aaa\nb\n", "aaa\n"},
		{"question", "^ab?$", "a\nab\nabb\n", "a\nab\n"},
		{"group", "^(ab)+$", "abab\naba\n", "abab\n"},
		{"alternation", "cat|dog", "cat\ndog\ncow\n", "cat\ndog\n"},
		{"interval", "^a{2}$", "a\naa\n", "aa\n"},
		{"interval with only a maximum", "^a{,2}$", "aa\naaa\n", "aa\n"},
		{"brace that starts no interval", "a{", "a{\na\n", "a{\n"},
		{"brace with text is literal", "a{x}", "a{x}\nax\n", "a{x}\n"},
		{"inverted interval is literal", "a{2,1}", "a{2,1}\naa\n", "a{2,1}\n"},
		{"closing brace is literal", "a}", "a}\na\n", "a}\n"},
		{"anchor anywhere", "(^a|b$)", "ax\nxb\nxa\n", "ax\nxb\n"},
		{"escaped operators are literal", `\(a\)\|\+\?\{`, "(a)|+?{\na\n", "(a)|+?{\n"},
		{"word escapes", `\<\w+\>`, "--ab--\n--\n", "--ab--\n"},
		{"stacked repetition", "^a+*$", "aaa\nb\n", "aaa\n"},
		{"a question after a star is not lazy", "^a*?$", "aaa\nb\n", "aaa\n"},
		{"empty group", "a()b", "ab\na\n", "ab\n"},
		{"empty alternative", "^(a|)b$", "ab\nb\ncb\n", "ab\nb\n"},
		{"class in set", "^[[:alnum:]_]+$", "ab_1\na-b\n", "ab_1\n"},
		{"dot matches any character", "^.$", "é\nab\n", "é\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, grep.Options{Syntax: grep.SyntaxExtended}, tt.input, tt.pattern), tt.want, "", 0)
		})
	}
}

func TestFixedStrings(t *testing.T) {
	fixed := grep.Options{Syntax: grep.SyntaxFixed}
	tests := []struct {
		name    string
		opts    grep.Options
		pattern string
		input   string
		want    string
	}{
		{"metacharacters are literal", fixed, "a.c*", "a.c*\nabc\n", "a.c*\n"},
		{"several strings", fixed, "foo\nbar", "foo\nbar\nbaz\n", "foo\nbar\n"},
		{"backslash", fixed, `a\b`, "a\\b\nab\n", "a\\b\n"},
		{"ignore case", grep.Options{Syntax: grep.SyntaxFixed, IgnoreCase: true}, "a.c", "A.C\nabc\n", "A.C\n"},
		{"word", grep.Options{Syntax: grep.SyntaxFixed, WordRegexp: true}, "cat", "a cat\ncats\ncat\n", "a cat\ncat\n"},
		{"word found later in the line", grep.Options{Syntax: grep.SyntaxFixed, WordRegexp: true}, "cat", "cats cat\n", "cats cat\n"},
		{"whole line", grep.Options{Syntax: grep.SyntaxFixed, LineRegexp: true}, "cat\ndog", "cat\ncats\ndog\n", "cat\ndog\n"},
		{"invalid UTF-8 is searched as bytes", fixed, "\xff\xfe", "a\xff\xfeb\nab\n", "a\xff\xfeb\n"},
		{"empty string matches all", fixed, "", "a\nb\n", "a\nb\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.BinaryFiles = grep.BinaryFilesText
			check(t, run(t, tt.opts, tt.input, tt.pattern), tt.want, "", 0)
		})
	}
}

func TestPatternErrors(t *testing.T) {
	tests := []struct {
		name    string
		syntax  grep.Syntax
		pattern string
		message string
	}{
		{"trailing backslash", grep.SyntaxBasic, `a\`, "Trailing backslash"},
		{"trailing backslash, extended", grep.SyntaxExtended, `a\`, "Trailing backslash"},
		{"open bracket", grep.SyntaxBasic, "[a", "Unmatched [, [^, [:, [., or [="},
		{"open class", grep.SyntaxBasic, "[[:alpha:", "Unmatched [, [^, [:, [., or [="},
		{"open group", grep.SyntaxBasic, `\(a`, `Unmatched ( or \(`},
		{"open group, extended", grep.SyntaxExtended, "(a", `Unmatched ( or \(`},
		{"stray close", grep.SyntaxBasic, `a\)`, `Unmatched ) or \)`},
		{"stray close, extended", grep.SyntaxExtended, "a)", `Unmatched ) or \)`},
		{"open interval", grep.SyntaxBasic, `a\{1`, `Unmatched \{`},
		{"bad interval", grep.SyntaxBasic, `a\{x\}`, `Invalid content of \{\}`},
		{"inverted interval", grep.SyntaxBasic, `a\{2,1\}`, `Invalid content of \{\}`},
		{"interval with nothing before it", grep.SyntaxBasic, `\{1\}`, "Invalid preceding regular expression"},
		{"huge interval", grep.SyntaxExtended, "a{99999}", "Regular expression too big"},
		{"interval beyond the engine", grep.SyntaxExtended, "a{2000}", "Regular expression too big"},
		{"inverted range", grep.SyntaxBasic, "[z-a]", "Invalid range end"},
		{"unknown class", grep.SyntaxBasic, "[[:nope:]]", "Invalid character class name"},
		{"long collating symbol", grep.SyntaxBasic, "[[.ab.]]", "Invalid collation character"},
		{"class without its set", grep.SyntaxBasic, "[:space:]", "character class syntax is [[:space:]], not [:space:]"},
		{"back reference", grep.SyntaxBasic, `\(a\)\1`, "back references are not supported"},
		{"back reference, extended", grep.SyntaxExtended, `(a)\1`, "back references are not supported"},
		{"invalid UTF-8", grep.SyntaxBasic, "a\xffb", "Invalid regular expression"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run(t, grep.Options{Syntax: tt.syntax}, "a\n", tt.pattern)
			check(t, got, "", "grepx: "+tt.message+"\n", 2)
		})
	}
}

func TestPatternWarnings(t *testing.T) {
	tests := []struct {
		name    string
		syntax  grep.Syntax
		pattern string
		input   string
		out     string
		warning string
	}{
		{"stray backslash", grep.SyntaxBasic, `a\d`, "ad\n", "ad\n", `stray \ before 'd'`},
		{"stray backslash before a space", grep.SyntaxBasic, `a\ b`, "a b\n", "a b\n", `stray \ before white space`},
		{"stray backslash before a control character", grep.SyntaxBasic, "a\\\x01", "a\x01\n", "a\x01\n", `stray \ before unprintable character`},
		{"star first, extended", grep.SyntaxExtended, "*a", "a\n", "a\n", "* at start of expression"},
		{"plus first, extended", grep.SyntaxExtended, "+a", "a\n", "a\n", "+ at start of expression"},
		{"question first, extended", grep.SyntaxExtended, "(?a)", "a\n", "a\n", "? at start of expression"},
		{"interval first, extended", grep.SyntaxExtended, "{1}a", "a\n", "a\n", "{...} at start of expression"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := run(t, grep.Options{Syntax: tt.syntax}, tt.input, tt.pattern)
			check(t, got, tt.out, "grepx: warning: "+tt.warning+"\n", 0)
		})
	}

	t.Run("escapes that need no warning", func(t *testing.T) {
		got := run(t, grep.Options{}, "a.*[]^$\\}\n", `a\.\*\[\]\^\$\\\}`)
		check(t, got, "a.*[]^$\\}\n", "", 0)
	})
	t.Run("a warning is given once", func(t *testing.T) {
		got := run(t, grep.Options{Patterns: []string{`\d`, `x\d`}}, "d\n")
		check(t, got, "d\n", "grepx: warning: stray \\ before 'd'\n", 0)
	})
}

func TestPatternSources(t *testing.T) {
	tree(t, map[string]string{
		"pats":      "foo\nbar\n",
		"unended":   "foo\nbar",
		"empty":     "",
		"blank":     "\n",
		"input.txt": "foo\nbar\nbaz\n",
	})
	tests := []struct {
		name   string
		opts   grep.Options
		args   []string
		out    string
		status int
	}{
		{"operand", grep.Options{}, []string{"ba", "input.txt"}, "bar\nbaz\n", 0},
		{"-e", grep.Options{Patterns: []string{"foo"}}, []string{"input.txt"}, "foo\n", 0},
		{"-e twice", grep.Options{Patterns: []string{"foo", "baz"}}, []string{"input.txt"}, "foo\nbaz\n", 0},
		{"-e with a newline", grep.Options{Patterns: []string{"foo\nbaz"}}, []string{"input.txt"}, "foo\nbaz\n", 0},
		{"-e that looks like an option", grep.Options{Patterns: []string{"-x"}}, []string{"input.txt"}, "", 1},
		{"-f", grep.Options{PatternFiles: []string{"pats"}}, []string{"input.txt"}, "foo\nbar\n", 0},
		{"-f without a final newline", grep.Options{PatternFiles: []string{"unended"}}, []string{"input.txt"}, "foo\nbar\n", 0},
		{"-f empty matches nothing", grep.Options{PatternFiles: []string{"empty"}}, []string{"input.txt"}, "", 1},
		{"-f empty, inverted, selects all", grep.Options{PatternFiles: []string{"empty"}, Invert: true}, []string{"input.txt"}, "foo\nbar\nbaz\n", 0},
		{"-f with an empty line matches all", grep.Options{PatternFiles: []string{"blank"}}, []string{"input.txt"}, "foo\nbar\nbaz\n", 0},
		{"-f and -e together", grep.Options{PatternFiles: []string{"empty"}, Patterns: []string{"baz"}}, []string{"input.txt"}, "baz\n", 0},
		{"-f twice", grep.Options{PatternFiles: []string{"empty", "pats"}}, []string{"input.txt"}, "foo\nbar\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "", tt.args...), tt.out, "", tt.status)
		})
	}

	t.Run("-f from standard input", func(t *testing.T) {
		got := run(t, grep.Options{PatternFiles: []string{"-"}}, "baz\n", "input.txt")
		check(t, got, "baz\n", "", 0)
	})
	t.Run("-f missing", func(t *testing.T) {
		got := run(t, grep.Options{PatternFiles: []string{"nope"}, NoMessages: true}, "", "input.txt")
		check(t, got, "", "grepx: nope: No such file or directory\n", 2)
	})
	t.Run("no pattern at all", func(t *testing.T) {
		var out, errb bytes.Buffer
		err := grep.New(grep.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(nil)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "missing PATTERNS operand" {
			t.Fatalf("err = %v, want the missing-operand usage error", err)
		}
		if out.Len() != 0 || errb.Len() != 0 {
			t.Fatalf("wrote %q and %q, want nothing", out.String(), errb.String())
		}
	})
}

func TestMatchingOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    grep.Options
		pattern string
		input   string
		out     string
		status  int
	}{
		{"ignore case", grep.Options{IgnoreCase: true}, "hello", "HeLLo\nbye\n", "HeLLo\n", 0},
		{"ignore case, non-ASCII", grep.Options{IgnoreCase: true}, "été", "ÉTÉ\nete\n", "ÉTÉ\n", 0},
		{"ignore case in a set", grep.Options{IgnoreCase: true}, "[a-c]x", "Bx\ndx\n", "Bx\n", 0},
		{"invert", grep.Options{Invert: true}, "a", "a\nb\nca\nd\n", "b\nd\n", 0},
		{"invert with nothing left", grep.Options{Invert: true}, "", "a\nb\n", "", 1},
		{"word", grep.Options{WordRegexp: true}, "cat", "cat\na cat.\ncats\nconcat\ncat_x\ncat-x\n", "cat\na cat.\ncat-x\n", 0},
		{"word, a later occurrence qualifies", grep.Options{WordRegexp: true}, "cat", "cats cat\n", "cats cat\n", 0},
		{"word, a shorter match qualifies", grep.Options{WordRegexp: true, Syntax: grep.SyntaxExtended}, "cat|cats", "cats!\n", "cats!\n", 0},
		{"word with a regexp", grep.Options{WordRegexp: true}, "c.t", "a cut b\nscuts\n", "a cut b\n", 0},
		{"word next to a non-ASCII letter", grep.Options{WordRegexp: true}, "caf", "café\ncaf!\n", "caf!\n", 0},
		{"word starting with punctuation", grep.Options{WordRegexp: true}, "@home", "x@home\n @home\n", " @home\n", 0},
		{"line", grep.Options{LineRegexp: true}, "ab*", "abb\nabbc\nxab\n", "abb\n", 0},
		{"line with alternation", grep.Options{LineRegexp: true, Syntax: grep.SyntaxExtended}, "a|bc", "a\nbc\nabc\n", "a\nbc\n", 0},
		{"line, inverted", grep.Options{LineRegexp: true, Invert: true}, "a", "a\nab\n", "ab\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, tt.input, tt.pattern), tt.out, "", tt.status)
		})
	}
}

func TestOutputControl(t *testing.T) {
	tree(t, map[string]string{
		"a.txt": "one\ntwo\nthree\n",
		"b.txt": "four\nfive\n",
		"c.txt": "",
	})
	with := func(o grep.Options) grep.Options { o.Filename = grep.FilenameWith; return o }
	tests := []struct {
		name   string
		opts   grep.Options
		args   []string
		out    string
		status int
	}{
		{"one file has no prefix", grep.Options{}, []string{"o", "a.txt"}, "one\ntwo\n", 0},
		{"several files are prefixed", grep.Options{}, []string{"o", "a.txt", "b.txt"}, "a.txt:one\na.txt:two\nb.txt:four\n", 0},
		{"-h", grep.Options{Filename: grep.FilenameWithout}, []string{"o", "a.txt", "b.txt"}, "one\ntwo\nfour\n", 0},
		{"-H", with(grep.Options{}), []string{"o", "b.txt"}, "b.txt:four\n", 0},
		{"count", grep.Options{Count: true}, []string{"o", "a.txt"}, "2\n", 0},
		{"count per file", grep.Options{Count: true}, []string{"e", "a.txt", "b.txt", "c.txt"}, "a.txt:2\nb.txt:1\nc.txt:0\n", 0},
		{"count of nothing", grep.Options{Count: true}, []string{"zzz", "a.txt"}, "0\n", 1},
		{"count, inverted", grep.Options{Count: true, Invert: true}, []string{"o", "a.txt"}, "1\n", 0},
		{"count stops at the maximum", grep.Options{Count: true, MaxCount: 1, MaxCountSet: true}, []string{"o", "a.txt"}, "1\n", 0},
		{"count with a NUL after the name", with(grep.Options{Count: true, Null: true}), []string{"o", "a.txt"}, "a.txt\x002\n", 0},
		{"files with matches", grep.Options{FilesWithMatches: true}, []string{"f", "a.txt", "b.txt", "c.txt"}, "b.txt\n", 0},
		{"files with matches, one operand", grep.Options{FilesWithMatches: true}, []string{"one", "a.txt"}, "a.txt\n", 0},
		{"files with matches, none", grep.Options{FilesWithMatches: true}, []string{"zzz", "a.txt"}, "", 1},
		{"files with matches, NUL", grep.Options{FilesWithMatches: true, Null: true}, []string{"e", "a.txt", "b.txt"}, "a.txt\x00b.txt\x00", 0},
		{"files without match", grep.Options{FilesWithoutMatch: true}, []string{"f", "a.txt", "b.txt", "c.txt"}, "a.txt\nc.txt\n", 0},
		{"files without match, none selected", grep.Options{FilesWithoutMatch: true}, []string{"zzz", "a.txt"}, "a.txt\n", 1},
		{"-l wins over -L and -c", grep.Options{FilesWithMatches: true, FilesWithoutMatch: true, Count: true}, []string{"f", "a.txt", "b.txt"}, "b.txt\n", 0},
		{"-L wins over -c", grep.Options{FilesWithoutMatch: true, Count: true}, []string{"f", "a.txt", "b.txt"}, "a.txt\n", 0},
		{"max count", grep.Options{MaxCount: 2, MaxCountSet: true}, []string{"e", "a.txt"}, "one\nthree\n", 0},
		{"max count applies per file", grep.Options{MaxCount: 1, MaxCountSet: true}, []string{"e", "a.txt", "b.txt"}, "a.txt:one\nb.txt:five\n", 0},
		{"max count of zero", grep.Options{MaxCount: 0, MaxCountSet: true}, []string{"e", "a.txt"}, "", 1},
		{"max count of zero still lists", grep.Options{MaxCount: 0, MaxCountSet: true, FilesWithoutMatch: true}, []string{"e", "a.txt"}, "a.txt\n", 1},
		{"max count ignored when not set", grep.Options{MaxCount: 1}, []string{"e", "a.txt"}, "one\nthree\n", 0},
		{"only matching", grep.Options{OnlyMatching: true}, []string{"e", "a.txt"}, "e\ne\ne\n", 0},
		{"only matching, with prefixes", with(grep.Options{OnlyMatching: true, LineNumber: true, ByteOffset: true}), []string{"e", "a.txt"}, "a.txt:1:2:e\na.txt:3:11:e\na.txt:3:12:e\n", 0},
		{"only matching, inverted, prints nothing", grep.Options{OnlyMatching: true, Invert: true}, []string{"one", "a.txt"}, "", 0},
		{"quiet", grep.Options{Quiet: true}, []string{"one", "a.txt"}, "", 0},
		{"quiet without a match", grep.Options{Quiet: true}, []string{"zzz", "a.txt"}, "", 1},
		{"quiet wins over -c and -l", grep.Options{Quiet: true, Count: true, FilesWithMatches: true}, []string{"one", "a.txt"}, "", 0},
		{"line number", grep.Options{LineNumber: true}, []string{"t", "a.txt"}, "2:two\n3:three\n", 0},
		{"byte offset", grep.Options{ByteOffset: true}, []string{"t", "a.txt"}, "4:two\n8:three\n", 0},
		{"every prefix", with(grep.Options{LineNumber: true, ByteOffset: true}), []string{"two", "a.txt"}, "a.txt:2:4:two\n", 0},
		{"initial tab", with(grep.Options{LineNumber: true, InitialTab: true}), []string{"two", "a.txt"}, "a.txt:2:\ttwo\n", 0},
		{"initial tab needs a prefix", grep.Options{InitialTab: true}, []string{"two", "a.txt"}, "two\n", 0},
		{"NUL after the name", with(grep.Options{Null: true, LineNumber: true}), []string{"two", "a.txt"}, "a.txt\x002:two\n", 0},
		{"line buffered", grep.Options{LineBuffered: true}, []string{"o", "a.txt"}, "one\ntwo\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "", tt.args...), tt.out, "", tt.status)
		})
	}

	t.Run("initial tab skips an empty line", func(t *testing.T) {
		got := run(t, grep.Options{LineNumber: true, InitialTab: true}, "\nx\n", "")
		check(t, got, "1:\n2:\tx\n", "", 0)
	})
	t.Run("only matching skips empty matches", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true}, "aab\nb\n", "a*")
		check(t, got, "aa\n", "", 0)
	})
	t.Run("only matching takes each match in turn", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true, Syntax: grep.SyntaxExtended}, "ab12cd345\n", "[0-9]+|[a-z]+")
		check(t, got, "ab\n12\ncd\n345\n", "", 0)
	})
	t.Run("only matching keeps an anchor at the start of the line", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true}, "aaa\n", "^a")
		check(t, got, "a\n", "", 0)
	})
	t.Run("only matching whole words", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true, WordRegexp: true}, "cat cats cat-cat écat\n", "c.t")
		check(t, got, "cat\ncat\ncat\n", "", 0)
	})
	t.Run("only matching whole words, fixed", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true, WordRegexp: true, Syntax: grep.SyntaxFixed}, "cat cats cat\n", "cat")
		check(t, got, "cat\ncat\n", "", 0)
	})
	t.Run("only matching whole lines", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true, LineRegexp: true}, "ab\nabc\n", "ab")
		check(t, got, "ab\n", "", 0)
	})
	t.Run("only matching fixed strings prefers the longest", func(t *testing.T) {
		got := run(t, grep.Options{OnlyMatching: true, Syntax: grep.SyntaxFixed}, "abcab\n", "ab\nabc")
		check(t, got, "abc\nab\n", "", 0)
	})
}

func TestContext(t *testing.T) {
	input := "1\n2\nX\n4\n5\n6\n7\nX\n9\n"
	tests := []struct {
		name string
		opts grep.Options
		out  string
	}{
		{"after", grep.Options{After: 1}, "X\n4\n--\nX\n9\n"},
		{"before", grep.Options{Before: 1}, "2\nX\n--\n7\nX\n"},
		{"both", grep.Options{After: 1, Before: 1}, "2\nX\n4\n--\n7\nX\n9\n"},
		{"groups that touch are merged", grep.Options{After: 2, Before: 2}, "1\n2\nX\n4\n5\n6\n7\nX\n9\n"},
		{"groups that are adjacent get no separator", grep.Options{After: 2, Before: 1}, "2\nX\n4\n5\n--\n7\nX\n9\n"},
		{"more context than lines", grep.Options{After: 100, Before: 100}, input},
		{"zero context still separates", grep.Options{Context: true}, "X\n--\nX\n"},
		{"no context option, no separator", grep.Options{}, "X\nX\n"},
		{"separator", grep.Options{After: 1, GroupSeparator: "==", GroupSeparatorSet: true}, "X\n4\n==\nX\n9\n"},
		{"empty separator", grep.Options{After: 1, GroupSeparatorSet: true}, "X\n4\n\nX\n9\n"},
		{"separator ignored when not set", grep.Options{After: 1, GroupSeparator: "=="}, "X\n4\n--\nX\n9\n"},
		{"no separator", grep.Options{After: 1, NoGroupSeparator: true}, "X\n4\nX\n9\n"},
		{"prefixes", grep.Options{After: 1, Before: 1, LineNumber: true, ByteOffset: true},
			"2-2-2\n3:4:X\n4-6-4\n--\n7-12-7\n8:14:X\n9-16-9\n"},
		{"trailing context after the maximum", grep.Options{After: 1, MaxCount: 1, MaxCountSet: true}, "X\n4\n"},
		{"only matching prints no context but separates", grep.Options{After: 1, OnlyMatching: true}, "X\n--\nX\n"},
		{"negative counts are none", grep.Options{After: -3, Before: -3}, "X\nX\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, input, "X"), tt.out, "", 0)
		})
	}

	t.Run("trailing context stops at the next selected line once the maximum is reached", func(t *testing.T) {
		got := run(t, grep.Options{After: 3, MaxCount: 1, MaxCountSet: true}, "X\na\nX\nb\n", "X")
		check(t, got, "X\na\n", "", 0)
	})
	t.Run("inverted context marks the matching lines", func(t *testing.T) {
		got := run(t, grep.Options{Invert: true, After: 1, LineNumber: true}, "a\nX\nX\nb\n", "X")
		check(t, got, "1:a\n2-X\n--\n4:b\n", "", 0)
	})
	t.Run("more leading context than the ring first held", func(t *testing.T) {
		got := run(t, grep.Options{Before: 3}, "a\nX\nb\nc\nd\ne\nf\nX\n", "X")
		check(t, got, "a\nX\n--\nd\ne\nf\nX\n", "", 0)
	})
	t.Run("groups are separated across files", func(t *testing.T) {
		tree(t, map[string]string{"a": "1\nX\n3\n", "b": "X\n2\n"})
		got := run(t, grep.Options{After: 1}, "", "X", "a", "b")
		check(t, got, "a:X\na-3\n--\nb:X\nb-2\n", "", 0)
	})
	t.Run("NUL after the name in a context line", func(t *testing.T) {
		got := run(t, grep.Options{After: 1, Null: true, Filename: grep.FilenameWith, Label: "in"}, "X\ny\n", "X")
		check(t, got, "in\x00X\nin\x00y\n", "", 0)
	})
}

func TestColor(t *testing.T) {
	const (
		ms  = "\x1b[01;31m\x1b[K"
		end = "\x1b[m\x1b[K"
		fn  = "\x1b[35m\x1b[K"
		ln  = "\x1b[32m\x1b[K"
		se  = "\x1b[36m\x1b[K"
	)
	colon := se + ":" + end
	dash := se + "-" + end
	always := func(o grep.Options) grep.Options { o.Color = grep.ColorAlways; return o }
	tests := []struct {
		name    string
		opts    grep.Options
		pattern string
		input   string
		out     string
	}{
		{"match", always(grep.Options{}), "b", "abc\n", "a" + ms + "b" + end + "c\n"},
		{"every match", always(grep.Options{}), "a", "aXa\n", ms + "a" + end + "X" + ms + "a" + end + "\n"},
		{"empty matches are not colored", always(grep.Options{}), "x*", "abc\n", "abc\n"},
		{"prefixes", always(grep.Options{Filename: grep.FilenameWith, Label: "f", LineNumber: true, ByteOffset: true}), "b", "b\n",
			fn + "f" + end + colon + ln + "1" + end + colon + ln + "0" + end + colon + ms + "b" + end + "\n"},
		{"NUL after the name is not colored", always(grep.Options{Filename: grep.FilenameWith, Label: "f", Null: true}), "b", "b\n",
			fn + "f" + end + "\x00" + ms + "b" + end + "\n"},
		{"only matching", always(grep.Options{OnlyMatching: true, LineNumber: true}), "b", "abcb\n",
			ln + "1" + end + colon + ms + "b" + end + "\n" + ln + "1" + end + colon + ms + "b" + end + "\n"},
		{"context and separator", always(grep.Options{After: 1, LineNumber: true}), "X", "X\na\nb\nX\n",
			ln + "1" + end + colon + ms + "X" + end + "\n" + ln + "2" + end + dash + "a\n" +
				se + "--" + end + "\n" + ln + "4" + end + colon + ms + "X" + end + "\n"},
		{"inverted lines are not highlighted", always(grep.Options{Invert: true}), "X", "abc\n", "abc\n"},
		{"inverted context is", always(grep.Options{Invert: true, After: 1}), "X", "a\nXy\n", "a\n" + ms + "X" + end + "y\n"},
		{"count", always(grep.Options{Count: true, Filename: grep.FilenameWith, Label: "f"}), "b", "b\n", fn + "f" + end + colon + "1\n"},
		{"list", always(grep.Options{FilesWithMatches: true, Label: "f"}), "b", "b\n", fn + "f" + end + "\n"},
		{"never", grep.Options{Color: grep.ColorNever}, "b", "abc\n", "abc\n"},
		{"auto off a terminal", grep.Options{Color: grep.ColorAuto}, "b", "abc\n", "abc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, tt.input, tt.pattern), tt.out, "", 0)
		})
	}

	env := func(vars map[string]string, terminal bool) func(*grep.Grep) {
		return func(g *grep.Grep) {
			g.Getenv = func(name string) string { return vars[name] }
			g.IsTerminal = terminal
		}
	}
	t.Run("auto on a terminal", func(t *testing.T) {
		got := runWith(t, grep.Options{Color: grep.ColorAuto}, "abc\n", env(map[string]string{"TERM": "xterm"}, true), "b")
		check(t, got, "a"+ms+"b"+end+"c\n", "", 0)
	})
	t.Run("auto on a dumb terminal", func(t *testing.T) {
		got := runWith(t, grep.Options{Color: grep.ColorAuto}, "abc\n", env(map[string]string{"TERM": "dumb"}, true), "b")
		check(t, got, "abc\n", "", 0)
	})
	t.Run("GREP_COLORS", func(t *testing.T) {
		vars := map[string]string{"GREP_COLORS": "ms=4:ln=1;33:se=:ne"}
		got := runWith(t, always(grep.Options{LineNumber: true}), "abc\n", env(vars, false), "b")
		check(t, got, "\x1b[1;33m1\x1b[m:a\x1b[4mb\x1b[mc\n", "", 0)
	})
	t.Run("GREP_COLORS mt sets both match colors", func(t *testing.T) {
		vars := map[string]string{"GREP_COLORS": "mt=7"}
		got := runWith(t, always(grep.Options{Invert: true, After: 1}), "a\nX\n", env(vars, false), "X")
		check(t, got, "a\n\x1b[7m\x1b[KX"+end+"\n", "", 0)
	})
	t.Run("GREP_COLORS line colors", func(t *testing.T) {
		vars := map[string]string{"GREP_COLORS": "sl=1:cx=2"}
		got := runWith(t, always(grep.Options{After: 1}), "aXb\r\nc\n", env(vars, false), "X")
		check(t, got, "\x1b[1m\x1b[Ka"+ms+"X"+end+"\x1b[1m\x1b[Kb"+end+"\r\n\x1b[2m\x1b[Kc"+end+"\n", "", 0)
	})
	t.Run("GREP_COLORS rv swaps the line colors under -v", func(t *testing.T) {
		vars := map[string]string{"GREP_COLORS": "sl=1:cx=2:rv:ms=:mc="}
		got := runWith(t, always(grep.Options{Invert: true, After: 1}), "a\nX\n", env(vars, false), "X")
		check(t, got, "\x1b[2m\x1b[Ka"+end+"\n\x1b[1m\x1b[KX"+end+"\n", "", 0)
	})
	t.Run("a malformed GREP_COLORS keeps what came before", func(t *testing.T) {
		vars := map[string]string{"GREP_COLORS": "ms=4:ln=red:fn=1"}
		got := runWith(t, always(grep.Options{Filename: grep.FilenameWith, Label: "f"}), "b\n", env(vars, false), "b")
		check(t, got, fn+"f"+end+colon+"\x1b[4m\x1b[Kb"+end+"\n", "", 0)
	})
}

func TestStandardInput(t *testing.T) {
	tree(t, map[string]string{"a.txt": "apple\n"})
	tests := []struct {
		name string
		opts grep.Options
		args []string
		out  string
	}{
		{"no file", grep.Options{}, []string{"a"}, "banana\n"},
		{"dash", grep.Options{}, []string{"a", "-"}, "banana\n"},
		{"named when prefixed", grep.Options{Filename: grep.FilenameWith}, []string{"a"}, "(standard input):banana\n"},
		{"label", grep.Options{Filename: grep.FilenameWith, Label: "fruit"}, []string{"a"}, "fruit:banana\n"},
		{"among files", grep.Options{}, []string{"a", "a.txt", "-"}, "a.txt:apple\n(standard input):banana\n"},
		{"listed", grep.Options{FilesWithMatches: true}, []string{"a", "-"}, "(standard input)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "banana\ncherry\n", tt.args...), tt.out, "", 0)
		})
	}
}

func TestLines(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		check(t, run(t, grep.Options{}, "", "a"), "", "", 1)
	})
	t.Run("empty input, empty pattern", func(t *testing.T) {
		check(t, run(t, grep.Options{}, "", ""), "", "", 1)
	})
	t.Run("a last line without a newline gets one", func(t *testing.T) {
		check(t, run(t, grep.Options{}, "a\nab", "b"), "ab\n", "", 0)
	})
	t.Run("an empty line is a line", func(t *testing.T) {
		check(t, run(t, grep.Options{LineNumber: true}, "a\n\nb\n", "^$"), "2:\n", "", 0)
	})
	t.Run("carriage returns are data", func(t *testing.T) {
		check(t, run(t, grep.Options{}, "a\r\nb\r\n", "a.$"), "a\r\n", "", 0)
	})
	t.Run("a line longer than the buffer", func(t *testing.T) {
		long := strings.Repeat("x", 3*core.BufferSize) + "needle" + strings.Repeat("y", core.BufferSize)
		input := "short\n" + long + "\nneedle too\n"
		got := run(t, grep.Options{LineNumber: true, ByteOffset: true, OnlyMatching: true}, input, "needle")
		check(t, got, "2:196614:needle\n3:262157:needle\n", "", 0)
		got = run(t, grep.Options{Count: true, Invert: true}, input, "needle")
		check(t, got, "1\n", "", 0)
		got = run(t, grep.Options{}, long, "needle")
		check(t, got, long+"\n", "", 0)
	})
	t.Run("many lines", func(t *testing.T) {
		var in, want strings.Builder
		for i := range 50000 {
			line := "line " + strings.Repeat("z", i%97)
			if i%1000 == 0 {
				line += " hit"
				want.WriteString(line + "\n")
			}
			in.WriteString(line + "\n")
		}
		check(t, run(t, grep.Options{}, in.String(), "hit$"), want.String(), "", 0)
	})
}

func TestNullData(t *testing.T) {
	t.Run("lines end with NUL", func(t *testing.T) {
		got := run(t, grep.Options{NullData: true}, "one\x00two\x00three", "t")
		check(t, got, "two\x00three\x00", "", 0)
	})
	t.Run("a newline is data", func(t *testing.T) {
		got := run(t, grep.Options{NullData: true, LineNumber: true}, "a\nb\x00c\x00", "a.b")
		check(t, got, "1:a\nb\x00", "", 0)
	})
	t.Run("only matching ends each match with NUL", func(t *testing.T) {
		got := run(t, grep.Options{NullData: true, OnlyMatching: true}, "aXbX\x00", "X")
		check(t, got, "X\x00X\x00", "", 0)
	})
	t.Run("count is still a line of text", func(t *testing.T) {
		got := run(t, grep.Options{NullData: true, Count: true}, "a\x00b\x00a\x00", "a")
		check(t, got, "2\n", "", 0)
	})
}

func TestBinaryFiles(t *testing.T) {
	tree(t, map[string]string{
		"bin":   "text\x00more text\nsecond text\n",
		"late":  strings.Repeat("text line\n", 10000) + "nul \x00 text\nlast text\n",
		"plain": "text\n",
	})
	tests := []struct {
		name string
		opts grep.Options
		args []string
		out  string
		err  string
		code int
	}{
		{"reported, not printed", grep.Options{}, []string{"text", "bin"}, "", "grepx: bin: binary file matches\n", 0},
		{"no match, no report", grep.Options{}, []string{"zzz", "bin"}, "", "", 1},
		{"-s does not silence the report", grep.Options{NoMessages: true}, []string{"text", "bin"}, "", "grepx: bin: binary file matches\n", 0},
		{"among files", grep.Options{}, []string{"text", "plain", "bin"}, "plain:text\n", "grepx: bin: binary file matches\n", 0},
		{"as text", grep.Options{BinaryFiles: grep.BinaryFilesText}, []string{"second", "bin"}, "second text\n", "", 0},
		{"as text, the NUL is printed", grep.Options{BinaryFiles: grep.BinaryFilesText}, []string{"more", "bin"}, "text\x00more text\n", "", 0},
		{"without match", grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch}, []string{"text", "bin"}, "", "", 1},
		{"without match, listed as such", grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch, FilesWithoutMatch: true}, []string{"text", "bin", "plain"}, "bin\n", "", 0},
		{"without match, counted as none", grep.Options{BinaryFiles: grep.BinaryFilesWithoutMatch, Count: true}, []string{"text", "bin"}, "0\n", "", 1},
		{"count needs no report", grep.Options{Count: true}, []string{"text", "bin"}, "2\n", "", 0},
		{"list needs no report", grep.Options{FilesWithMatches: true}, []string{"text", "bin"}, "bin\n", "", 0},
		{"quiet needs no report", grep.Options{Quiet: true}, []string{"text", "bin"}, "", "", 0},
		{"inverted", grep.Options{Invert: true}, []string{"zzz", "bin"}, "", "grepx: bin: binary file matches\n", 0},
		{"only matching", grep.Options{OnlyMatching: true}, []string{"text", "bin"}, "", "grepx: bin: binary file matches\n", 0},
		{"the maximum is reached before a late NUL", grep.Options{MaxCount: 2, MaxCountSet: true, Invert: true}, []string{"zzz", "late"}, "text line\ntext line\n", "", 0},
		{"with -z nothing is binary", grep.Options{NullData: true}, []string{"second", "bin"}, "more text\nsecond text\n\x00", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "", tt.args...), tt.out, tt.err, tt.code)
		})
	}

	t.Run("lines before a late NUL are printed, then the report", func(t *testing.T) {
		got := run(t, grep.Options{}, "", "last\\|nul", "late")
		check(t, got, "", "grepx: late: binary file matches\n", 0)
		got = run(t, grep.Options{Count: true}, "", "text", "late")
		check(t, got, "10002\n", "", 0)
		got = run(t, grep.Options{}, "", "^text line$", "late")
		if n := strings.Count(got.out, "text line\n"); n != 10000 || got.err != "" || got.status != 0 {
			t.Fatalf("printed %d lines, stderr %q, status %d; want 10000 lines and no report", n, got.err, got.status)
		}
		got = run(t, grep.Options{}, "", "text", "late")
		if n := strings.Count(got.out, "\n"); n != 10000 || got.err != "grepx: late: binary file matches\n" || got.status != 0 {
			t.Fatalf("printed %d lines, stderr %q, status %d; want 10000 lines, then the report", n, got.err, got.status)
		}
	})
	t.Run("binary standard input", func(t *testing.T) {
		got := run(t, grep.Options{}, "a\x00b\n", "a")
		check(t, got, "", "grepx: (standard input): binary file matches\n", 0)
	})
}

func TestFileErrors(t *testing.T) {
	tree(t, map[string]string{"a.txt": "apple\n", "b.txt": "banana\n", "dir/": "", "dir/in.txt": "apple\n"})
	tests := []struct {
		name string
		opts grep.Options
		args []string
		out  string
		err  string
		code int
	}{
		{"missing file", grep.Options{}, []string{"a", "nope"}, "", "grepx: nope: No such file or directory\n", 2},
		{"missing file among others", grep.Options{}, []string{"a", "a.txt", "nope", "b.txt"},
			"a.txt:apple\nb.txt:banana\n", "grepx: nope: No such file or directory\n", 2},
		{"-s silences the message, not the status", grep.Options{NoMessages: true}, []string{"a", "a.txt", "nope"}, "a.txt:apple\n", "", 2},
		{"quiet with a match ignores the error", grep.Options{Quiet: true}, []string{"a", "nope", "a.txt"}, "", "grepx: nope: No such file or directory\n", 0},
		{"quiet without a match reports it", grep.Options{Quiet: true}, []string{"zzz", "nope", "a.txt"}, "", "grepx: nope: No such file or directory\n", 2},
		{"quiet stops at the first match", grep.Options{Quiet: true}, []string{"a", "a.txt", "nope"}, "", "", 0},
		{"directory", grep.Options{}, []string{"a", "dir"}, "", "grepx: dir: Is a directory\n", 2},
		{"directory among files", grep.Options{}, []string{"a", "dir", "a.txt"}, "a.txt:apple\n", "grepx: dir: Is a directory\n", 2},
		{"directory with -s", grep.Options{NoMessages: true}, []string{"a", "dir"}, "", "", 2},
		{"directory skipped", grep.Options{Directories: grep.DirectoriesSkip}, []string{"a", "dir", "a.txt"}, "a.txt:apple\n", "", 0},
		{"directory excluded", grep.Options{ExcludeDir: []string{"dir"}}, []string{"a", "dir", "a.txt"}, "a.txt:apple\n", "", 0},
		{"empty name", grep.Options{}, []string{"a", ""}, "", "grepx: : No such file or directory\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "", tt.args...), tt.out, tt.err, tt.code)
		})
	}
}

func TestRecursive(t *testing.T) {
	tree(t, map[string]string{
		"top.txt":           "match top\n",
		"b/two.go":          "match two\n",
		"b/deep/three.txt":  "match three\n",
		"a/one.txt":         "match one\nother\n",
		"a/skip.log":        "match log\n",
		"node_modules/x.js": "match dep\n",
		"empty/":            "",
	})
	rec := grep.Options{Directories: grep.DirectoriesRecurse}
	with := func(edit func(*grep.Options)) grep.Options {
		o := rec
		edit(&o)
		return o
	}
	all := p("a/one.txt") + ":match one\n" + p("a/skip.log") + ":match log\n" +
		p("b/deep/three.txt") + ":match three\n" + p("b/two.go") + ":match two\n" +
		p("node_modules/x.js") + ":match dep\ntop.txt:match top\n"
	tests := []struct {
		name string
		opts grep.Options
		args []string
		out  string
	}{
		{"working directory, in name order, without ./", rec, []string{"match"}, all},
		{"dot keeps its prefix", rec, []string{"match one", "."}, p("./a/one.txt") + ":match one\n"},
		{"named directory", rec, []string{"match", "b"}, p("b/deep/three.txt") + ":match three\n" + p("b/two.go") + ":match two\n"},
		{"trailing separator is not doubled", rec, []string{"match", "b/"}, "b/" + p("deep/three.txt") + ":match three\nb/two.go:match two\n"},
		{"a single file operand has no prefix", rec, []string{"match", "top.txt"}, "match top\n"},
		{"file and directory", rec, []string{"match", "top.txt", "a"},
			"top.txt:match top\n" + p("a/one.txt") + ":match one\n" + p("a/skip.log") + ":match log\n"},
		{"-h", with(func(o *grep.Options) { o.Filename = grep.FilenameWithout }), []string{"match", "a"}, "match one\nmatch log\n"},
		{"count", with(func(o *grep.Options) { o.Count = true }), []string{"other", "a"}, p("a/one.txt") + ":1\n" + p("a/skip.log") + ":0\n"},
		{"list", with(func(o *grep.Options) { o.FilesWithMatches = true }), []string{"t[wh]", "b"}, p("b/deep/three.txt") + "\n" + p("b/two.go") + "\n"},
		{"include", with(func(o *grep.Options) { o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobInclude, Pattern: "*.txt"}} }), []string{"match"},
			p("a/one.txt") + ":match one\n" + p("b/deep/three.txt") + ":match three\ntop.txt:match top\n"},
		{"two includes", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobInclude, Pattern: "*.go"}, {Kind: grep.GlobInclude, Pattern: "*.log"}}
		}), []string{"match"}, p("a/skip.log") + ":match log\n" + p("b/two.go") + ":match two\n"},
		{"exclude", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExclude, Pattern: "*.txt"}, {Kind: grep.GlobExclude, Pattern: "*.j?"}}
		}), []string{"match"}, p("a/skip.log") + ":match log\n" + p("b/two.go") + ":match two\n"},
		{"the last matching glob decides: include after exclude", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExclude, Pattern: "*"}, {Kind: grep.GlobInclude, Pattern: "t*.txt"}}
		}), []string{"match"}, p("b/deep/three.txt") + ":match three\ntop.txt:match top\n"},
		{"the last matching glob decides: exclude after include", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobInclude, Pattern: "*.txt"}, {Kind: grep.GlobExclude, Pattern: "t*"}}
		}), []string{"match"}, p("a/one.txt") + ":match one\n"},
		{"a glob matches the base name, not the path", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExclude, Pattern: "a/*"}, {Kind: grep.GlobExclude, Pattern: "[!t]*"}}
		}), []string{"match"}, p("b/deep/three.txt") + ":match three\n" + p("b/two.go") + ":match two\ntop.txt:match top\n"},
		{"exclude directories", with(func(o *grep.Options) { o.ExcludeDir = []string{"node_modules", "dee*/", "a"} }), []string{"match"},
			p("b/two.go") + ":match two\ntop.txt:match top\n"},
		// The operand is printed as it was typed; only what is joined to it
		// uses the platform's separator.
		{"an excluded directory operand is skipped", with(func(o *grep.Options) { o.ExcludeDir = []string{"a"} }), []string{"match", "a", "b/deep"},
			"b/deep" + p("/three.txt") + ":match three\n"},
		{"include applies to a file operand", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobInclude, Pattern: "*.go"}}
		}), []string{"match", "top.txt", p("b/two.go")}, p("b/two.go") + ":match two\n"},
		{"a glob matches a trailing part of an operand", with(func(o *grep.Options) {
			o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExclude, Pattern: "deep/*"}}
		}), []string{"match", "b/deep/three.txt", "b/two.go"}, "b/two.go:match two\n"},
		{"context separates files", with(func(o *grep.Options) { o.After = 1 }), []string{"match", "a"},
			p("a/one.txt") + ":match one\n" + p("a/one.txt") + "-other\n--\n" + p("a/skip.log") + ":match log\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check(t, run(t, tt.opts, "", tt.args...), tt.out, "", 0)
		})
	}

	t.Run("exclude from a file", func(t *testing.T) {
		if err := os.WriteFile("globs", []byte("*.txt\n\n*.js\nglobs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove("globs")
		o := rec
		o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExcludeFrom, Pattern: "globs"}}
		check(t, run(t, o, "", "match"), p("a/skip.log")+":match log\n"+p("b/two.go")+":match two\n", "", 0)

		o.FileGlobs = []grep.FileGlob{{Kind: grep.GlobExcludeFrom, Pattern: "nope"}}
		check(t, run(t, o, "", "match"), "", "grepx: nope: No such file or directory\n", 2)
	})
	t.Run("-d recurse is -r", func(t *testing.T) {
		check(t, run(t, rec, "", "three", "b"), p("b/deep/three.txt")+":match three\n", "", 0)
	})
	t.Run("nothing found", func(t *testing.T) {
		check(t, run(t, rec, "", "zzz"), "", "", 1)
	})
	t.Run("quiet stops the descent", func(t *testing.T) {
		o := rec
		o.Quiet = true
		check(t, run(t, o, "", "match"), "", "", 0)
	})
}

func TestSymlinks(t *testing.T) {
	dir := tree(t, map[string]string{
		"real/file.txt":  "match real\n",
		"outside/in.txt": "match outside\n",
		"tree/plain.txt": "match plain\n",
	})
	link := func(target, name string) {
		t.Helper()
		if err := os.Symlink(target, filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Skipf("symbolic links are not available: %v", err)
		}
	}
	link(filepath.Join(dir, "real", "file.txt"), "tree/filelink")
	link(filepath.Join(dir, "outside"), "tree/dirlink")
	link(filepath.Join(dir, "tree"), "tree/loop")
	link(filepath.Join(dir, "nowhere"), "tree/broken")
	link(filepath.Join(dir, "real"), "toplink")

	rec := grep.Options{Directories: grep.DirectoriesRecurse}
	t.Run("-r skips links met on the way", func(t *testing.T) {
		check(t, run(t, rec, "", "match", "tree"), p("tree/plain.txt")+":match plain\n", "", 0)
	})
	t.Run("-r follows a link named as an operand", func(t *testing.T) {
		check(t, run(t, rec, "", "match", "toplink"), p("toplink/file.txt")+":match real\n", "", 0)
		check(t, run(t, rec, "", "match", p("tree/filelink")), "match real\n", "", 0)
	})
	t.Run("-R follows them all and reports loops and broken links", func(t *testing.T) {
		deref := rec
		deref.Dereference = true
		got := run(t, deref, "", "match", "tree")
		check(t, got,
			p("tree/dirlink/in.txt")+":match outside\n"+p("tree/filelink")+":match real\n"+p("tree/plain.txt")+":match plain\n",
			"grepx: "+p("tree/broken")+": No such file or directory\n"+
				"grepx: "+p("tree/loop")+": warning: recursive directory loop\n", 2)
	})
}

func TestUnreadable(t *testing.T) {
	tree(t, map[string]string{"locked.txt": "match\n", "open.txt": "match\n"})
	if err := os.Chmod("locked.txt", 0); err != nil {
		t.Fatal(err)
	}
	if f, err := os.Open("locked.txt"); err == nil {
		f.Close()
		t.Skip("permissions are not enforced here")
	}
	got := run(t, grep.Options{}, "", "match", "locked.txt", "open.txt")
	check(t, got, "open.txt:match\n", "grepx: locked.txt: Permission denied\n", 2)
}

// failingWriter fails every write, as a full disk would.
type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteError(t *testing.T) {
	var errb bytes.Buffer
	g := grep.New(grep.Options{}, core.Streams{
		In:  strings.NewReader("a\n"),
		Out: failingWriter{errors.New("no space left on device")},
		Err: &errb,
	})
	if got := status(t, g.Run([]string{"a"})); got != 2 {
		t.Errorf("status = %d, want 2", got)
	}
	if want := "grepx: write error: No space left on device\n"; errb.String() != want {
		t.Errorf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestClosedPipe(t *testing.T) {
	var errb bytes.Buffer
	g := grep.New(grep.Options{LineBuffered: true}, core.Streams{
		In:  strings.NewReader(strings.Repeat("a\n", 1000)),
		Out: failingWriter{io.ErrClosedPipe},
		Err: &errb,
	})
	if got := status(t, g.Run([]string{"a"})); got != 0 {
		t.Errorf("status = %d, want 0", got)
	}
	if errb.Len() != 0 {
		t.Errorf("stderr = %q, want a quiet stop", errb.String())
	}
}
