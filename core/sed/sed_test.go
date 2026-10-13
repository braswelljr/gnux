package sed_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sed"
)

// run drives the engine over input as standard input and returns what it
// wrote and returned.
func run(t *testing.T, opts sed.Options, input string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	e := sed.New(opts, core.Streams{In: strings.NewReader(input), Out: &out, Err: &errOut})
	e.Getenv = func(string) string { return "" }
	err := e.Run(args)
	return out.String(), errOut.String(), err
}

// status is the exit status an error from Run stands for.
func status(err error) int {
	var exit *core.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.Code
	}
	return 1
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const five = "1\n2\n3\n4\n5\n"

func TestScripts(t *testing.T) {
	tests := []struct {
		name   string
		opts   sed.Options
		script string
		input  string
		want   string
	}{
		// Addresses.
		{"empty script", sed.Options{}, "", "a\nb\n", "a\nb\n"},
		{"line", sed.Options{}, "2d", five, "1\n3\n4\n5\n"},
		{"last", sed.Options{}, "$d", five, "1\n2\n3\n4\n"},
		{"last on one line", sed.Options{Quiet: true}, "$p", "only\n", "only\n"},
		{"regex", sed.Options{Quiet: true}, "/[24]/p", five, "2\n4\n"},
		{"custom delimiter", sed.Options{Quiet: true}, `\,3,p`, five, "3\n"},
		{"custom delimiter escaped", sed.Options{Quiet: true}, `\,a\,b,p`, "a,b\nab\n", "a,b\n"},
		{"regex I", sed.Options{Quiet: true}, "/abc/Ip", "ABC\nabd\n", "ABC\n"},
		{"regex M", sed.Options{Quiet: true}, "N;/^b$/Mp", "a\nb\n", "a\nb\n"},
		{"regex without M", sed.Options{Quiet: true}, "N;/^b$/p", "a\nb\n", ""},
		{"step", sed.Options{Quiet: true}, "1~2p", five, "1\n3\n5\n"},
		{"step from zero", sed.Options{Quiet: true}, "0~2p", five, "2\n4\n"},
		{"step of zero", sed.Options{Quiet: true}, "2~0p", five, "2\n"},
		{"range", sed.Options{Quiet: true}, "2,4p", five, "2\n3\n4\n"},
		{"range to last", sed.Options{}, "3,$d", five, "1\n2\n"},
		{"range backwards is one line", sed.Options{Quiet: true}, "4,2p", five, "4\n"},
		{"range of regexes", sed.Options{Quiet: true}, "/2/,/4/p", five, "2\n3\n4\n"},
		{"range restarts", sed.Options{Quiet: true}, "/a/,/b/p", "a\nb\nc\na\nd\n", "a\nb\na\nd\n"},
		{"range end checked from next line", sed.Options{Quiet: true}, "/a/,/a/p", "a\nb\na\nc\n", "a\nb\na\n"},
		{"range plus", sed.Options{Quiet: true}, "2,+2p", five, "2\n3\n4\n"},
		{"range plus zero", sed.Options{Quiet: true}, "2,+0p", five, "2\n"},
		{"range multiple", sed.Options{Quiet: true}, "2,~4p", five, "2\n3\n4\n"},
		{"range multiple already there", sed.Options{Quiet: true}, "4,~4p", five, "4\n"},
		{"zero address", sed.Options{Quiet: true}, "0,/1/p", five, "1\n"},
		{"one address", sed.Options{Quiet: true}, "1,/1/p", "1\n2\n1\n3\n", "1\n2\n1\n"},
		{"negation", sed.Options{}, "2!d", five, "2\n"},
		{"negation with blanks", sed.Options{}, "2 ! d", five, "2\n"},
		{"negated range", sed.Options{}, "2,4!d", five, "2\n3\n4\n"},
		{"negated last", sed.Options{}, "$!d", five, "5\n"},
		{"empty regex reuses the last", sed.Options{}, "/b/s//X/", "abc\nbbb\n", "aXc\nXbb\n"},
		{"empty regex address", sed.Options{Quiet: true}, "s/b/X/;//p", "abb\n", "aXb\n"},
		{"address then blanks", sed.Options{Quiet: true}, "2 p", five, "2\n"},

		// Blocks and separators.
		{"block", sed.Options{Quiet: true}, "2{p;p}", five, "2\n2\n"},
		{"block with semicolon before brace", sed.Options{Quiet: true}, "2{p;p;}", five, "2\n2\n"},
		{"nested blocks", sed.Options{Quiet: true}, "2,4{/3/!{p}}", five, "2\n4\n"},
		{"block over lines", sed.Options{Quiet: true}, "2{\n  p\n  p\n}", five, "2\n2\n"},
		{"newline separator", sed.Options{Quiet: true}, "1p\n2p", five, "1\n2\n"},
		{"comment", sed.Options{Quiet: true}, "# nothing\n1p # first\n", five, "1\n"},
		{"hash n", sed.Options{}, "#n\n1p", five, "1\n"},
		{"hash n needs the whole line", sed.Options{}, "#no\n1d", "1\n2\n", "2\n"},

		// Substitution.
		{"s", sed.Options{}, "s/a/b/", "aaa\n", "baa\n"},
		{"s g", sed.Options{}, "s/a/b/g", "aaa\n", "bbb\n"},
		{"s number", sed.Options{}, "s/a/b/2", "aaa\n", "aba\n"},
		{"s number g", sed.Options{}, "s/a/b/2g", "aaa\n", "abb\n"},
		{"s number too high", sed.Options{}, "s/a/b/4", "aaa\n", "aaa\n"},
		{"s p", sed.Options{Quiet: true}, "s/a/b/p", "a\nc\n", "b\n"},
		{"s gp", sed.Options{Quiet: true}, "s/a/b/gp", "aa\n", "bb\n"},
		{"s i", sed.Options{}, "s/A/x/i", "a\n", "x\n"},
		{"s I", sed.Options{}, "s/a/x/Ig", "aA\n", "xx\n"},
		{"s M", sed.Options{}, "N;s/^/>/Mg", "a\nb\n", ">a\n>b\n"},
		{"s without M", sed.Options{}, "N;s/^/>/g", "a\nb\n", ">a\nb\n"},
		{"s delimiter", sed.Options{}, "s|/usr|/opt|", "/usr/bin\n", "/opt/bin\n"},
		{"s escaped delimiter", sed.Options{}, `s/\/usr/\/opt/`, "/usr/bin\n", "/opt/bin\n"},
		{"s ampersand", sed.Options{}, "s/b/[&]/", "abc\n", "a[b]c\n"},
		{"s escaped ampersand", sed.Options{}, `s/b/\&/`, "abc\n", "a&c\n"},
		{"s group", sed.Options{}, `s/\(a\)\(b\)/\2\1/`, "abc\n", "bac\n"},
		{"s group zero", sed.Options{}, `s/b/<\0>/`, "abc\n", "a<b>c\n"},
		{"s unmatched group", sed.Options{}, `s/\(x\)*b/[\1]/`, "abc\n", "a[]c\n"},
		{"s newline in replacement", sed.Options{}, `s/b/\n/`, "abc\n", "a\nc\n"},
		{"s literal newline", sed.Options{}, "s/b/\\\n/", "abc\n", "a\nc\n"},
		{"s tab", sed.Options{}, `s/b/\t/`, "abc\n", "a\tc\n"},
		{"s newline in regex", sed.Options{}, `N;s/\n/+/`, "a\nb\n", "a+b\n"},
		{"s empty matches", sed.Options{}, "s/x*/-/g", "abc\n", "-a-b-c-\n"},
		{"s empty match after a match", sed.Options{}, "s/a*/x/g", "baaac\n", "xbxcx\n"},
		{"s second empty match", sed.Options{}, "s/b*/x/2", "abc\n", "axc\n"},
		{"s anchored g", sed.Options{}, "s/^a/x/g", "aaa\n", "xaa\n"},
		{"s end anchor", sed.Options{}, "s/$/!/", "a\n", "a!\n"},
		{"s longest", sed.Options{}, `s/a\|ab/X/`, "abc\n", "Xc\n"},
		{"s upper", sed.Options{}, `s/.*/\U&/`, "abc def\n", "ABC DEF\n"},
		{"s lower", sed.Options{}, `s/.*/\L&/`, "ABC\n", "abc\n"},
		{"s upper until E", sed.Options{}, `s/\(a\)\(b\)/\U\1\E\2/`, "ab\n", "Ab\n"},
		{"s upper first", sed.Options{}, `s/\w\+/\u&/g`, "one two\n", "One Two\n"},
		{"s lower first", sed.Options{}, `s/.*/\l&/`, "ABC\n", "aBC\n"},
		{"s upper first then lower", sed.Options{}, `s/.*/\L\u&/`, "hELLO\n", "Hello\n"},
		{"s case on literal", sed.Options{}, `s/b/\Uxy\Ez/`, "abc\n", "aXYzc\n"},
		{"s case skips empty group", sed.Options{}, `s/\(x*\)\(b\)/\u\1\2/`, "abc\n", "aBc\n"},
		{"s star first is literal", sed.Options{}, "s/*/x/", "a*b\n", "axb\n"},
		{"s interval", sed.Options{}, `s/a\{2\}/X/`, "aaa\n", "Xa\n"},
		{"s plus", sed.Options{}, `s/a\+/X/`, "caab\n", "cXb\n"},
		{"s question", sed.Options{}, `s/ab\?/X/g`, "aab\n", "XX\n"},
		{"s bracket", sed.Options{}, "s/[[:digit:]]/#/g", "a1b22\n", "a#b##\n"},
		{"s bracket closing first", sed.Options{}, "s/[]x]/#/g", "a]x\n", "a##\n"},
		{"s bracket negated", sed.Options{}, "s/[^a]/#/g", "abca\n", "a##a\n"},
		{"s bracket backslash", sed.Options{}, `s/[\.]/#/g`, `a\.b` + "\n", "a##b\n"},
		{"s bracket newline", sed.Options{}, `N;s/[\n]/+/`, "a\nb\n", "a+b\n"},
		{"s word", sed.Options{}, `s/\<the\>/THE/g`, "the other the\n", "THE other THE\n"},
		{"s buffer anchors", sed.Options{}, "N;s/\\`a/X/;s/b\\'/Y/", "a\nb\n", "X\nY\n"},
		{"s dot matches newline", sed.Options{}, "N;s/a.b/X/", "a\nb\n", "X\n"},
		{"s literal braces in a basic regex", sed.Options{}, "s/{a}/X/", "{a}\n", "X\n"},
		{"s caret in the middle", sed.Options{}, "s/a^b/X/", "a^b\n", "X\n"},
		{"s dollar in the middle", sed.Options{}, "s/a$b/X/", "a$b\n", "X\n"},
		{"s caret after group", sed.Options{}, `s/\(^a\)/X/`, "aa\n", "Xa\n"},
		{"s repeated repetition", sed.Options{}, "s/a**/X/", "aab\n", "Xb\n"},
		{"s hex escape", sed.Options{}, `s/\x41/X/`, "A\n", "X\n"},
		{"s utf8", sed.Options{}, "s/é*/X/", "éét\n", "Xt\n"},
		{"s upper utf8", sed.Options{}, `s/.*/\U&/`, "été\n", "ÉTÉ\n"},
		{"s blanks before next command", sed.Options{Quiet: true}, "s/a/b/ ; p", "a\n", "b\n"},
		{"s in block", sed.Options{}, "1{s/a/b/}", "a\n", "b\n"},

		// Extended regular expressions.
		{"E group", sed.Options{Extended: true}, `s/(a)(b)/\2\1/`, "abc\n", "bac\n"},
		{"E alternation", sed.Options{Extended: true}, "s/cat|dog/pet/g", "cat dog\n", "pet pet\n"},
		{"E plus", sed.Options{Extended: true}, "s/a+/X/", "caab\n", "cXb\n"},
		{"E question", sed.Options{Extended: true}, "s/ab?/X/g", "aab\n", "XX\n"},
		{"E interval", sed.Options{Extended: true}, "s/a{2,3}/X/", "aaaa\n", "Xa\n"},
		{"E interval open", sed.Options{Extended: true}, "s/a{2,}/X/", "aaaa\n", "X\n"},
		{"E literal brace", sed.Options{Extended: true}, "s/a{/X/", "a{b\n", "Xb\n"},
		{"E escaped parens", sed.Options{Extended: true}, `s/\(a\)/X/`, "(a)\n", "X\n"},
		{"E word boundaries", sed.Options{Extended: true}, `s/\<(\w+)\>/[\1]/g`, "ab cd\n", "[ab] [cd]\n"},
		{"E classes", sed.Options{Extended: true}, `s/\s+/_/g;s/\w/x/`, "a  b\n", "x_b\n"},
		{"E escapes", sed.Options{Extended: true}, `s/\t/T/`, "a\tb\n", "aTb\n"},
		{"E control", sed.Options{Extended: true}, `s/\cI/T/`, "a\tb\n", "aTb\n"},
		{"E not lazy", sed.Options{Extended: true}, "s/a+?/X/", "aab\n", "Xb\n"},
		{"E longest alternation", sed.Options{Extended: true}, "s/a|ab/X/", "abc\n", "Xc\n"},
		{"E address", sed.Options{Extended: true, Quiet: true}, "/^(a|b)$/p", "a\nc\nb\n", "a\nb\n"},
		{"E anchor in group", sed.Options{Extended: true}, "s/(^|,)a/X/g", "a,a,ba\n", "XX,ba\n"},

		// Transliteration.
		{"y", sed.Options{}, "y/abc/xyz/", "aabbcc\n", "xxyyzz\n"},
		{"y delimiter", sed.Options{}, "y,ab,xy,", "ab\n", "xy\n"},
		{"y escapes", sed.Options{}, `N;y/\n\\/+|/`, "a\\\nb\n", "a|+b\n"},
		{"y escaped delimiter", sed.Options{}, `y/\//|/`, "a/b\n", "a|b\n"},
		{"y utf8", sed.Options{}, "y/é/e/", "été\n", "ete\n"},
		{"y first mapping wins", sed.Options{}, "y/aa/xy/", "a\n", "x\n"},

		// Printing and deleting.
		{"p twice", sed.Options{}, "p", "a\n", "a\na\n"},
		{"P", sed.Options{Quiet: true}, "N;P", "a\nb\n", "a\n"},
		{"P without newline", sed.Options{Quiet: true}, "P", "a\n", "a\n"},
		{"d", sed.Options{}, "d", five, ""},
		{"d ends the cycle", sed.Options{}, "d;p", five, ""},
		{"D", sed.Options{}, "$!N;P;D", "a\nb\nc\n", "a\nb\nc\n"},
		{"D restarts without reading", sed.Options{}, "N;N;D", "1\n2\n3\n4\n5\n6\n", "3\n4\n5\n6\n"},
		{"D without newline", sed.Options{}, "D", "a\nb\n", ""},
		{"line number", sed.Options{Quiet: true}, "$=", five, "5\n"},
		{"line number before line", sed.Options{}, "=", "a\nb\n", "1\na\n2\nb\n"},
		{"z", sed.Options{}, "2z", "a\nb\n", "a\n\n"},
		{"l", sed.Options{Quiet: true}, "l", "a\tb\\c\x01\xc3\xa9\n", `a\tb\\c\001\303\251$` + "\n"},
		{"l embedded newline", sed.Options{Quiet: true}, "N;l", "a\nb\n", `a\nb$` + "\n"},
		{"l wraps", sed.Options{Quiet: true}, "l 5", "abcdefghij\n", "abcd\\\nefgh\\\nij$\n"},
		{"l never splits an escape", sed.Options{Quiet: true}, "l 5", "abc\tdef\n", "abc\\\n\\tde\\\nf$\n"},
		{"l zero", sed.Options{Quiet: true}, "l 0", strings.Repeat("x", 100) + "\n", strings.Repeat("x", 100) + "$\n"},
		{"l default width", sed.Options{Quiet: true}, "l", strings.Repeat("x", 70) + "\n", strings.Repeat("x", 69) + "\\\nx$\n"},
		{"l option width", sed.Options{Quiet: true, LineLength: 4}, "l", "abcdef\n", "abc\\\ndef$\n"},
		{"l option no wrap", sed.Options{Quiet: true, LineLength: -1}, "l", strings.Repeat("x", 80) + "\n", strings.Repeat("x", 80) + "$\n"},

		// Next line.
		{"n", sed.Options{}, "n;d", five, "1\n3\n5\n"},
		{"n quiet", sed.Options{Quiet: true}, "n;p", five, "2\n4\n"},
		{"n at the end prints", sed.Options{}, "n;s/./X/", "a\n", "a\n"},
		{"n at the end posix", sed.Options{Posix: true}, "n;s/./X/", "a\n", ""},
		{"N", sed.Options{}, `N;s/\n/+/`, "1\n2\n3\n4\n", "1+2\n3+4\n"},
		{"N at the end prints", sed.Options{}, `N;s/\n/+/`, "1\n2\n3\n", "1+2\n3\n"},
		{"N at the end quiet", sed.Options{Quiet: true}, "N;p", "1\n", ""},
		{"N at the end posix", sed.Options{Posix: true}, `N;s/\n/+/`, "1\n2\n3\n", "1+2\n"},
		{"N guarded", sed.Options{}, `$!N;s/\n/+/`, "1\n2\n3\n", "1+2\n3\n"},
		{"join all lines", sed.Options{}, `:a;N;$!ba;s/\n/,/g`, five, "1,2,3,4,5\n"},
		{"join all lines in a block", sed.Options{}, `:a;$!{N;ba};s/\n/,/g`, five, "1,2,3,4,5\n"},

		// Hold space.
		{"g", sed.Options{}, "g", "a\n", "\n"},
		{"G", sed.Options{}, "G", "a\nb\n", "a\n\nb\n\n"},
		{"h and G", sed.Options{}, "1h;2G", "a\nb\n", "a\nb\na\n"},
		{"H and g", sed.Options{Quiet: true}, "H;${g;p}", "a\nb\n", "\na\nb\n"},
		{"x", sed.Options{}, "x", "a\nb\n", "\na\n"},
		{"reverse lines", sed.Options{Quiet: true}, "1!G;h;$p", "1\n2\n3\n", "3\n2\n1\n"},

		// Text.
		{"a one line", sed.Options{}, "1a hello", "a\nb\n", "a\nhello\nb\n"},
		{"a without blank", sed.Options{}, "1ahello", "a\nb\n", "a\nhello\nb\n"},
		{"a strips leading blanks", sed.Options{}, "1a   hello", "a\n", "a\nhello\n"},
		{"a posix form", sed.Options{}, "1a\\\nhello", "a\nb\n", "a\nhello\nb\n"},
		{"a posix form keeps leading blanks", sed.Options{}, "1a\\\n  hello", "a\n", "a\n  hello\n"},
		{"a backslash keeps leading blanks", sed.Options{}, `1a\  hello`, "a\n", "a\n  hello\n"},
		{"a backslash text", sed.Options{}, `1a\text`, "a\n", "a\ntext\n"},
		{"a continuation", sed.Options{}, "1a one\\\ntwo", "a\nb\n", "a\none\ntwo\nb\n"},
		{"a removes backslash", sed.Options{}, `1a a\qb`, "a\n", "a\naqb\n"},
		{"a escape", sed.Options{}, `1a a\tb`, "a\n", "a\na\tb\n"},
		{"a takes the rest of the line", sed.Options{}, "1a x;p}", "a\n", "a\nx;p}\n"},
		{"a then command", sed.Options{}, "1a x\ns/a/b/", "a\n", "b\nx\n"},
		{"a in a block", sed.Options{}, "1{\na x\n}", "a\n", "a\nx\n"},
		{"a on the last line", sed.Options{}, "$a end", "a\nb\n", "a\nb\nend\n"},
		{"a after autoprint with d", sed.Options{}, "a x\nd", "a\n", "x\n"},
		{"a flushed by n", sed.Options{}, "1{a x\nn}", "a\nb\n", "a\nx\nb\n"},
		{"a flushed by N", sed.Options{}, "1{a x\nN}", "a\nb\n", "x\na\nb\n"},
		{"i", sed.Options{}, "2i hello", "a\nb\n", "a\nhello\nb\n"},
		{"i posix form", sed.Options{}, "1i\\\none\\\ntwo", "a\n", "one\ntwo\na\n"},
		{"c", sed.Options{}, "2c changed", "a\nb\nc\n", "a\nchanged\nc\n"},
		{"c range prints once", sed.Options{}, "2,3c changed", "a\nb\nc\nd\n", "a\nchanged\nd\n"},
		{"c negated", sed.Options{}, "2!c x", "a\nb\nc\n", "x\nb\nx\n"},
		{"c regex range", sed.Options{}, "/b/,/c/c X", "a\nb\nc\nd\n", "a\nX\nd\n"},
		{"c ends the cycle", sed.Options{}, "c x\np", "a\n", "x\n"},
		{"c with last", sed.Options{}, "$!N;c x", "a\nb\nc\n", "x\nx\n"},

		// Branching.
		{"b to the end", sed.Options{}, "2b;s/./X/", "a\nb\nc\n", "X\nb\nX\n"},
		{"b label", sed.Options{}, "/a/b skip\ns/$/!/\n:skip", "a\nb\n", "a\nb!\n"},
		{"b label in block", sed.Options{}, "/a/{b end};s/$/!/;:end", "a\nb\n", "a\nb!\n"},
		{"t", sed.Options{}, "s/a/b/;t;s/$/!/", "a\nc\n", "b\nc!\n"},
		{"t loop", sed.Options{}, ":a;s/^\\(x*\\)a/\\1x/;ta", "aaab\n", "xxxb\n"},
		{"t resets", sed.Options{}, "s/a/b/;t one;:one;t two;s/$/!/;:two", "a\n", "b!\n"},
		{"t reset by a new line", sed.Options{}, "s/a/b/;n;t;s/$/!/", "a\nc\n", "b\nc!\n"},
		{"T", sed.Options{}, "s/a/b/;T;s/$/!/", "a\nc\n", "b!\nc\n"},
		{"label with spaces", sed.Options{}, ": a\ns/a/b/\nt a", "aa\n", "bb\n"},

		// Quitting.
		{"q", sed.Options{}, "2q", five, "1\n2\n"},
		{"q quiet", sed.Options{Quiet: true}, "2q", five, ""},
		{"q flushes appends", sed.Options{}, "1{a x\nq}", five, "1\nx\n"},
		{"Q", sed.Options{}, "2Q", five, "1\n"},
		{"Q drops appends", sed.Options{}, "1{a x\nQ}", five, ""},

		// Other commands.
		{"F", sed.Options{Quiet: true}, "1F", "a\n", "-\n"},
		{"v", sed.Options{}, "v 4.2\ns/a/b/", "a\n", "b\n"},
		{"v alone", sed.Options{}, "v;s/a/b/", "a\n", "b\n"},

		// Trailing newlines.
		{"no trailing newline", sed.Options{}, "s/a/b/", "a\na", "b\nb"},
		{"no trailing newline p", sed.Options{}, "p", "a", "a\na"},
		{"no trailing newline append", sed.Options{}, "$a x", "a", "a\nx\n"},
		{"empty append adds the newline", sed.Options{}, `$a\`, "a\nb", "a\nb\n"},
		{"empty append with newline present", sed.Options{}, `$a\`, "a\nb\n", "a\nb\n"},
		{"no trailing newline G", sed.Options{}, "G", "a", "a\n"},
		{"empty input", sed.Options{}, "p;$=", "", ""},
		{"empty lines", sed.Options{}, "s/^$/x/", "\n\n", "x\nx\n"},

		// NUL-separated lines.
		{"z lines", sed.Options{NullData: true}, "s/^a/X/", "ab\x00ac\x00", "Xb\x00Xc\x00"},
		{"z newline is data", sed.Options{NullData: true}, `s/\n/+/g`, "a\nb\x00c\n", "a+b\x00c+"},
		{"z N joins with NUL", sed.Options{NullData: true, Quiet: true}, "N;l", "a\x00b\x00", `a\000b$` + "\n"},
		{"z last", sed.Options{NullData: true, Quiet: true}, "$p", "a\x00b\x00", "b\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(t, tt.opts, tt.input, tt.script)
			if err != nil || errOut != "" {
				t.Fatalf("sed %q: error %v, stderr %q", tt.script, err, errOut)
			}
			if out != tt.want {
				t.Fatalf("sed %q on %q:\n got %q\nwant %q", tt.script, tt.input, out, tt.want)
			}
		})
	}
}

func TestScriptSources(t *testing.T) {
	dir := t.TempDir()
	file := write(t, dir, "script.sed", "s/a/b/\n# comment\n$a\\\nend\n")
	quiet := write(t, dir, "quiet.sed", "#n\n2p\n")

	tests := []struct {
		name    string
		scripts []sed.Script
		input   string
		want    string
	}{
		{"two expressions", []sed.Script{{Text: "s/a/b/"}, {Text: "s/b/c/"}}, "a\n", "c\n"},
		{"text across expressions", []sed.Script{{Text: `1a\`}, {Text: "hello"}}, "a\n", "a\nhello\n"},
		{"block across expressions", []sed.Script{{Text: "1{"}, {Text: "s/a/b/"}, {Text: "}"}}, "a\na\n", "b\na\n"},
		{"one-line text ends with its expression", []sed.Script{{Text: "1a hello"}, {Text: "s/a/b/"}}, "a\n", "b\nhello\n"},
		{"file", []sed.Script{{Text: file, File: true}}, "a\na\n", "b\nb\nend\n"},
		{"file then expression", []sed.Script{{Text: file, File: true}, {Text: "s/b/c/"}}, "a\n", "c\nend\n"},
		{"hash n in a file", []sed.Script{{Text: quiet, File: true}}, five, "2\n"},
		{"hash n only in the first", []sed.Script{{Text: "1d"}, {Text: "#n"}}, "1\n2\n", "2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(t, sed.Options{Scripts: tt.scripts}, tt.input)
			if err != nil || errOut != "" || out != tt.want {
				t.Fatalf("got %q, stderr %q, error %v; want %q", out, errOut, err, tt.want)
			}
		})
	}

	t.Run("script from standard input", func(t *testing.T) {
		in := write(t, dir, "in.txt", "a\n")
		out, errOut, err := run(t, sed.Options{Scripts: []sed.Script{{Text: "-", File: true}}}, "s/a/b/\n", in)
		if err != nil || errOut != "" || out != "b\n" {
			t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
		}
	})

	t.Run("missing script file", func(t *testing.T) {
		_, errOut, err := run(t, sed.Options{Scripts: []sed.Script{{Text: filepath.Join(dir, "none"), File: true}}}, "")
		want := "sedx: couldn't open file " + filepath.Join(dir, "none") + ": No such file or directory\n"
		if status(err) != sed.ExitIO || errOut != want {
			t.Fatalf("status %d, stderr %q", status(err), errOut)
		}
	})

	t.Run("no script", func(t *testing.T) {
		_, _, err := run(t, sed.Options{}, "")
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "no script specified" {
			t.Fatalf("error %v", err)
		}
	})
}

func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		name   string
		opts   sed.Options
		script string
		want   string
	}{
		{"unknown command", sed.Options{}, "k", "-e expression #1, char 1: unknown command: `k'"},
		{"unknown command later", sed.Options{}, "p;k", "-e expression #1, char 3: unknown command: `k'"},
		{"unterminated s", sed.Options{}, "s/a", "-e expression #1, char 3: unterminated `s' command"},
		{"unterminated s replacement", sed.Options{}, "s/a/b", "-e expression #1, char 5: unterminated `s' command"},
		{"unterminated s newline", sed.Options{}, "s/a/b\np", "-e expression #1, char 5: unterminated `s' command"},
		{"unterminated y", sed.Options{}, "y/a/", "-e expression #1, char 4: unterminated `y' command"},
		{"y lengths", sed.Options{}, "y/ab/c/", "-e expression #1, char 7: strings for `y' command are different lengths"},
		{"unterminated address", sed.Options{}, "/a", "-e expression #1, char 2: unterminated address regex"},
		{"unknown s option", sed.Options{}, "s/a/b/k", "-e expression #1, char 7: unknown option to `s'"},
		{"multiple p", sed.Options{}, "s/a/b/pp", "-e expression #1, char 8: multiple `p' options to `s' command"},
		{"multiple g", sed.Options{}, "s/a/b/gg", "-e expression #1, char 8: multiple `g' options to `s' command"},
		{"multiple numbers", sed.Options{}, "s/a/b/2p3", "-e expression #1, char 9: multiple number options to `s' command"},
		{"zero number", sed.Options{}, "s/a/b/0", "-e expression #1, char 7: number option to `s' command may not be zero"},
		{"extra characters", sed.Options{}, "pq", "-e expression #1, char 2: extra characters after command"},
		{"missing command", sed.Options{}, "1", "-e expression #1, char 1: missing command"},
		{"missing command after range", sed.Options{}, "1,2", "-e expression #1, char 3: missing command"},
		{"unexpected comma", sed.Options{}, "1,p", "-e expression #1, char 3: unexpected `,'"},
		{"first address plus", sed.Options{}, "+1p", "-e expression #1, char 2: invalid usage of +N or ~N as first address"},
		{"line zero", sed.Options{}, "0p", "-e expression #1, char 2: invalid usage of line address 0"},
		{"line zero range", sed.Options{}, "0,5p", "-e expression #1, char 4: invalid usage of line address 0"},
		{"multiple bangs", sed.Options{}, "1!!p", "-e expression #1, char 3: multiple `!'s"},
		{"unmatched open", sed.Options{}, "1{p", "-e expression #1, char 0: unmatched `{'"},
		{"unexpected close", sed.Options{}, "p;}", "-e expression #1, char 3: unexpected `}'"},
		{"close with address", sed.Options{}, "{p;1}", "-e expression #1, char 5: `}' doesn't want any addresses"},
		{"colon with address", sed.Options{}, "1:a", "-e expression #1, char 2: : doesn't want any addresses"},
		{"colon without label", sed.Options{}, ":", "-e expression #1, char 1: \":\" lacks a label"},
		{"comment with address", sed.Options{}, "1# x", "-e expression #1, char 2: comments don't accept any addresses"},
		{"q with two addresses", sed.Options{}, "1,2q", "-e expression #1, char 4: command only uses one address"},
		{"a without text", sed.Options{}, "a", "-e expression #1, char 1: expected \\ after `a', `c' or `i'"},
		{"r without file", sed.Options{}, "r", "-e expression #1, char 1: missing filename in r/R/w/W commands"},
		{"w flag without file", sed.Options{}, "s/a/b/w", "-e expression #1, char 7: missing filename in r/R/w/W commands"},
		{"missing label", sed.Options{}, "b nowhere", "can't find label for jump to `nowhere'"},
		{"e command", sed.Options{}, "e date", "-e expression #1, char 1: the `e' command is not supported: sedx does not run commands"},
		{"e flag", sed.Options{}, "s/a/b/e", "-e expression #1, char 7: the `e' option to `s' is not supported: sedx does not run commands"},
		{"newer version", sed.Options{}, "v 9.0", "-e expression #1, char 5: expected newer version of sed"},
		{"modifier on empty regex", sed.Options{}, "//Ip", "-e expression #1, char 3: cannot specify modifiers on empty regexp"},
		{"modifier on empty s regex", sed.Options{}, "s//x/I", "-e expression #1, char 6: cannot specify modifiers on empty regexp"},
		{"bad reference", sed.Options{}, `s/a/\1/`, "-e expression #1, char 7: invalid reference \\1 on `s' command's RHS"},
		{"back reference", sed.Options{}, `s/\(a\)\1/x/`, "-e expression #1, char 12: back references in a regular expression are not supported"},
		{"back reference extended", sed.Options{Extended: true}, `/(a)\1/p`, "-e expression #1, char 7: back references in a regular expression are not supported"},
		{"unmatched paren", sed.Options{}, `s/\(a/x/`, `-e expression #1, char 8: Unmatched ( or \(`},
		{"unmatched close paren", sed.Options{}, `s/a\)/x/`, `-e expression #1, char 8: Unmatched ) or \)`},
		{"unmatched paren extended", sed.Options{Extended: true}, "s/(a/x/", `-e expression #1, char 7: Unmatched ( or \(`},
		{"unmatched bracket", sed.Options{}, "s/[a/x/", "-e expression #1, char 7: Unmatched [, [^, [:, [., or [="},
		{"a with nothing in a later expression", sed.Options{}, "p;a", "-e expression #1, char 3: expected \\ after `a', `c' or `i'"},
		{"bad class", sed.Options{}, "s/[[:foo:]]/x/", "-e expression #1, char 14: Invalid character class name"},
		{"bad interval", sed.Options{}, `s/a\{2,1\}/x/`, `-e expression #1, char 13: Invalid content of \{\}`},
		{"unmatched brace", sed.Options{}, `s/a\{2/x/`, `-e expression #1, char 9: Unmatched \{`},
		{"interval too big", sed.Options{}, `s/a\{5000\}/x/`, "-e expression #1, char 14: Regular expression too big"},
		{"interval without atom", sed.Options{}, `s/\{2\}/x/`, "-e expression #1, char 10: Invalid preceding regular expression"},
		{"trailing backslash", sed.Options{}, `/a\`, "-e expression #1, char 3: unterminated address regex"},
		{"range end", sed.Options{}, "s/[z-a]/x/", "-e expression #1, char 10: Invalid range end"},
		{"sandbox r", sed.Options{Sandbox: true}, "r file", "-e expression #1, char 1: e/r/w commands disabled in sandbox mode"},
		{"sandbox w", sed.Options{Sandbox: true}, "w file", "-e expression #1, char 1: e/r/w commands disabled in sandbox mode"},
		{"sandbox w flag", sed.Options{Sandbox: true}, "s/a/b/w file", "-e expression #1, char 7: e/r/w commands disabled in sandbox mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(t, tt.opts, "a\n", tt.script)
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("sed %q: error %v, want ErrFailed", tt.script, err)
			}
			if want := "sedx: " + tt.want + "\n"; errOut != want || out != "" {
				t.Fatalf("sed %q:\n got %q\nwant %q\nstdout %q", tt.script, errOut, want, out)
			}
		})
	}
}

func TestSyntaxErrorLocations(t *testing.T) {
	dir := t.TempDir()
	file := write(t, dir, "bad.sed", "p\ns/a/b/\nk\n")
	open := write(t, dir, "open.sed", "p\n2{\np\n")

	tests := []struct {
		name    string
		scripts []sed.Script
		want    string
	}{
		{"second expression", []sed.Script{{Text: "p"}, {Text: "pk"}}, "-e expression #2, char 2: extra characters after command"},
		{"end of first expression", []sed.Script{{Text: "s/a"}, {Text: "p"}}, "-e expression #1, char 3: unterminated `s' command"},
		{"file line", []sed.Script{{Text: file, File: true}}, "file " + file + " line 3: unknown command: `k'"},
		{"expression after a file", []sed.Script{{Text: open, File: true}, {Text: "}"}, {Text: "k"}}, "-e expression #2, char 1: unknown command: `k'"},
		{"unmatched open in a file", []sed.Script{{Text: open, File: true}}, "file " + open + " line 2: unmatched `{'"},
		{"unmatched open in an expression", []sed.Script{{Text: "p"}, {Text: "1{"}}, "-e expression #2, char 0: unmatched `{'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errOut, err := run(t, sed.Options{Scripts: tt.scripts}, "a\n")
			if want := "sedx: " + tt.want + "\n"; !errors.Is(err, core.ErrFailed) || errOut != want {
				t.Fatalf("got %q, error %v\nwant %q", errOut, err, want)
			}
		})
	}
}

// A mistake anywhere in the script is found before a w file is emptied.
func TestSyntaxErrorTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	keep := write(t, dir, "keep.txt", "precious\n")
	_, errOut, err := run(t, sed.Options{}, "a\n", "w "+keep+"\nk")
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errOut, "unknown command") {
		t.Fatalf("error %v, stderr %q", err, errOut)
	}
	if got := read(t, keep); got != "precious\n" {
		t.Fatalf("the w file was emptied: %q", got)
	}
}

func TestNoPreviousRegex(t *testing.T) {
	out, errOut, err := run(t, sed.Options{}, "a\n", "s//x/")
	if !errors.Is(err, core.ErrFailed) || errOut != "sedx: no previous regular expression\n" || out != "" {
		t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
	}
}

func TestExitStatus(t *testing.T) {
	tests := []struct {
		script string
		want   int
		out    string
	}{
		{"2q", 0, "1\n2\n"},
		{"2q5", 5, "1\n2\n"},
		{"2q 7", 7, "1\n2\n"},
		{"2Q3", 3, "1\n"},
		{"$!d;q 9", 9, "5\n"},
	}
	for _, tt := range tests {
		out, errOut, err := run(t, sed.Options{}, five, tt.script)
		if status(err) != tt.want || out != tt.out || errOut != "" {
			t.Errorf("sed %q: status %d, stdout %q, stderr %q", tt.script, status(err), out, errOut)
		}
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a.txt", "a1\na2\n")
	b := write(t, dir, "b.txt", "b1\nb2\n")
	bare := write(t, dir, "bare.txt", "x1\nx2")
	empty := write(t, dir, "empty.txt", "")
	missing := filepath.Join(dir, "missing.txt")

	tests := []struct {
		name   string
		opts   sed.Options
		script string
		stdin  string
		files  []string
		want   string
	}{
		{"one stream", sed.Options{Quiet: true}, "$p;3p", "", []string{a, b}, "b1\nb2\n"},
		{"line numbers continue", sed.Options{Quiet: true}, "=", "", []string{a, b}, "1\n2\n3\n4\n"},
		{"last skips empty files", sed.Options{Quiet: true}, "$p", "", []string{a, empty, empty}, "a2\n"},
		{"separate last", sed.Options{Quiet: true, Separate: true}, "$p", "", []string{a, b}, "a2\nb2\n"},
		{"separate line numbers", sed.Options{Quiet: true, Separate: true}, "1p", "", []string{a, b}, "a1\nb1\n"},
		{"separate N stops per file", sed.Options{Separate: true}, `N;N;s/\n/+/g`, "", []string{a, b}, "a1\na2\nb1\nb2\n"},
		{"N joins across files", sed.Options{}, `N;N;s/\n/+/g`, "", []string{a, b}, "a1+a2+b1\nb2\n"},
		{"dash is standard input", sed.Options{}, "s/^/>/", "in\n", []string{a, "-"}, ">a1\n>a2\n>in\n"},
		{"missing newline between files", sed.Options{}, "p", "", []string{bare, a}, "x1\nx1\nx2\nx2\na1\na1\na2\na2\n"},
		{"missing newline at the end", sed.Options{}, "s/x/y/", "", []string{a, bare}, "a1\na2\ny1\ny2"},
		{"file name", sed.Options{Quiet: true}, "1F;$F", "", []string{a, b}, a + "\n" + b + "\n"},
		{"file name separate", sed.Options{Quiet: true, Separate: true}, "1F", "", []string{a, b}, a + "\n" + b + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(t, tt.opts, tt.stdin, append([]string{tt.script}, tt.files...)...)
			if err != nil || errOut != "" || out != tt.want {
				t.Fatalf("got %q, stderr %q, error %v\nwant %q", out, errOut, err, tt.want)
			}
		})
	}

	t.Run("a missing file does not stop the rest", func(t *testing.T) {
		out, errOut, err := run(t, sed.Options{}, "", "s/^/>/", a, missing, b)
		want := "sedx: can't read " + missing + ": No such file or directory\n"
		if status(err) != sed.ExitBadInput || errOut != want || out != ">a1\n>a2\n>b1\n>b2\n" {
			t.Fatalf("status %d, stdout %q, stderr %q", status(err), out, errOut)
		}
	})

	t.Run("only a missing file", func(t *testing.T) {
		out, errOut, err := run(t, sed.Options{}, "", "p", missing)
		if status(err) != sed.ExitBadInput || out != "" || !strings.HasPrefix(errOut, "sedx: can't read ") {
			t.Fatalf("status %d, stdout %q, stderr %q", status(err), out, errOut)
		}
	})

	t.Run("q status wins over a missing file", func(t *testing.T) {
		_, _, err := run(t, sed.Options{}, "", "q 3", missing, a)
		if status(err) != 3 {
			t.Fatalf("status %d", status(err))
		}
	})

	t.Run("a directory", func(t *testing.T) {
		_, errOut, err := run(t, sed.Options{}, "", "p", dir)
		if status(err) != sed.ExitIO || !strings.HasPrefix(errOut, "sedx: read error on "+dir+": ") {
			t.Fatalf("status %d, stderr %q", status(err), errOut)
		}
	})
}

func TestReadAndWriteCommands(t *testing.T) {
	dir := t.TempDir()
	extra := write(t, dir, "extra.txt", "e1\ne2\n")
	bare := write(t, dir, "bare.txt", "tail")
	missing := filepath.Join(dir, "missing.txt")

	t.Run("r", func(t *testing.T) {
		out, _, err := run(t, sed.Options{}, "a\nb\n", "1r "+extra)
		if err != nil || out != "a\ne1\ne2\nb\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("r of a missing file is silent", func(t *testing.T) {
		out, errOut, err := run(t, sed.Options{}, "a\n", "r "+missing)
		if err != nil || out != "a\n" || errOut != "" {
			t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
		}
	})
	t.Run("r copies the file as it is", func(t *testing.T) {
		out, _, err := run(t, sed.Options{}, "a\nb\n", "1r "+bare)
		if err != nil || out != "a\ntailb\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("r after a deleted line", func(t *testing.T) {
		out, _, err := run(t, sed.Options{}, "a\nb\n", "1{r "+extra+"\nd}")
		if err != nil || out != "e1\ne2\nb\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("r standard input", func(t *testing.T) {
		in := write(t, dir, "in.txt", "a\nb\n")
		out, _, err := run(t, sed.Options{}, "from stdin\n", "1r /dev/stdin", in)
		if err != nil || out != "a\nfrom stdin\nb\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("R", func(t *testing.T) {
		out, _, err := run(t, sed.Options{}, "a\nb\nc\n", "R "+extra)
		if err != nil || out != "a\ne1\nb\ne2\nc\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("R of a missing file is silent", func(t *testing.T) {
		out, errOut, err := run(t, sed.Options{}, "a\n", "R "+missing)
		if err != nil || out != "a\n" || errOut != "" {
			t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
		}
	})
	t.Run("w", func(t *testing.T) {
		target := filepath.Join(dir, "w.txt")
		out, _, err := run(t, sed.Options{Quiet: true}, five, "/[24]/w "+target)
		if err != nil || out != "" || read(t, target) != "2\n4\n" {
			t.Fatalf("got %q, file %q, error %v", out, read(t, target), err)
		}
	})
	t.Run("w creates the file even unused", func(t *testing.T) {
		target := write(t, dir, "unused.txt", "old\n")
		if _, _, err := run(t, sed.Options{}, "a\n", "/zzz/w "+target); err != nil || read(t, target) != "" {
			t.Fatalf("file %q, error %v", read(t, target), err)
		}
	})
	t.Run("two w commands share a file", func(t *testing.T) {
		target := filepath.Join(dir, "shared.txt")
		_, _, err := run(t, sed.Options{Quiet: true}, "a\nb\n", "w "+target+"\ns/$/!/w "+target)
		if err != nil || read(t, target) != "a\na!\nb\nb!\n" {
			t.Fatalf("file %q, error %v", read(t, target), err)
		}
	})
	t.Run("W", func(t *testing.T) {
		target := filepath.Join(dir, "first.txt")
		_, _, err := run(t, sed.Options{Quiet: true}, "a\nb\nc\n", "$!N;W "+target)
		if err != nil || read(t, target) != "a\nc\n" {
			t.Fatalf("file %q, error %v", read(t, target), err)
		}
	})
	t.Run("s w flag", func(t *testing.T) {
		target := filepath.Join(dir, "changed.txt")
		out, _, err := run(t, sed.Options{}, "a\nb\n", "s/a/X/w "+target)
		if err != nil || out != "X\nb\n" || read(t, target) != "X\n" {
			t.Fatalf("got %q, file %q, error %v", out, read(t, target), err)
		}
	})
	t.Run("w standard output", func(t *testing.T) {
		out, _, err := run(t, sed.Options{}, "a\nb\n", "1w /dev/stdout")
		if err != nil || out != "a\na\nb\n" {
			t.Fatalf("got %q, error %v", out, err)
		}
	})
	t.Run("w standard error", func(t *testing.T) {
		out, errOut, err := run(t, sed.Options{Quiet: true}, "a\nb\n", "2w /dev/stderr")
		if err != nil || out != "" || errOut != "b\n" {
			t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
		}
	})
	t.Run("w to a file that cannot be made", func(t *testing.T) {
		target := filepath.Join(dir, "no", "such", "dir.txt")
		out, errOut, err := run(t, sed.Options{}, "a\n", "w "+target)
		want := "sedx: couldn't open file " + target + ": No such file or directory\n"
		if status(err) != sed.ExitIO || errOut != want || out != "" {
			t.Fatalf("status %d, stdout %q, stderr %q", status(err), out, errOut)
		}
	})
}

func TestInPlace(t *testing.T) {
	t.Run("edits each file", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "one\ntwo\n")
		b := write(t, dir, "b.txt", "one\nlast")
		out, errOut, err := run(t, sed.Options{InPlace: true}, "", "s/one/1/;$s/$/!/", a, b)
		if err != nil || out != "" || errOut != "" {
			t.Fatalf("stdout %q, stderr %q, error %v", out, errOut, err)
		}
		if read(t, a) != "1\ntwo!\n" || read(t, b) != "1\nlast!" {
			t.Fatalf("a %q, b %q", read(t, a), read(t, b))
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 2 {
			t.Fatalf("leftover files: %v", entries)
		}
	})

	t.Run("line numbers restart", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\ny\n")
		b := write(t, dir, "b.txt", "x\ny\n")
		if _, _, err := run(t, sed.Options{InPlace: true}, "", "1d", a, b); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "y\n" || read(t, b) != "y\n" {
			t.Fatalf("a %q, b %q", read(t, a), read(t, b))
		}
	})

	t.Run("keeps the mode", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.sh", "x\n")
		if err := os.Chmod(a, 0o750); err != nil {
			t.Fatal(err)
		}
		before, _ := os.Stat(a)
		if _, _, err := run(t, sed.Options{InPlace: true}, "", "s/x/y/", a); err != nil {
			t.Fatal(err)
		}
		after, _ := os.Stat(a)
		if after.Mode() != before.Mode() || read(t, a) != "y\n" {
			t.Fatalf("mode %v, was %v; content %q", after.Mode(), before.Mode(), read(t, a))
		}
	})

	t.Run("suffix", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		if _, _, err := run(t, sed.Options{InPlace: true, Suffix: ".bak"}, "", "s/x/y/", a); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "y\n" || read(t, a+".bak") != "x\n" {
			t.Fatalf("file %q, backup %q", read(t, a), read(t, a+".bak"))
		}
	})

	t.Run("suffix with a star", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		if _, _, err := run(t, sed.Options{InPlace: true, Suffix: "old-*"}, "", "s/x/y/", a); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "y\n" || read(t, filepath.Join(dir, "old-a.txt")) != "x\n" {
			t.Fatalf("file %q", read(t, a))
		}
	})

	t.Run("suffix naming a directory", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		backups := filepath.Join(dir, "backups")
		if err := os.Mkdir(backups, 0o755); err != nil {
			t.Fatal(err)
		}
		suffix := backups + string(filepath.Separator) + "*"
		if _, _, err := run(t, sed.Options{InPlace: true, Suffix: suffix}, "", "s/x/y/", a); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "y\n" || read(t, filepath.Join(backups, "a.txt")) != "x\n" {
			t.Fatalf("file %q", read(t, a))
		}
	})

	t.Run("quiet and p", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "1\n2\n3\n")
		if _, _, err := run(t, sed.Options{InPlace: true, Quiet: true}, "", "2p", a); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "2\n" {
			t.Fatalf("file %q", read(t, a))
		}
	})

	t.Run("i a and = go to the file", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		out, _, err := run(t, sed.Options{InPlace: true}, "", "i before\na after\n=", a)
		if err != nil || out != "" || read(t, a) != "before\n1\nx\nafter\n" {
			t.Fatalf("stdout %q, file %q, error %v", out, read(t, a), err)
		}
	})

	t.Run("w standard output still reaches it", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\ny\n")
		out, _, err := run(t, sed.Options{InPlace: true}, "", "1w /dev/stdout\ns/x/z/", a)
		if err != nil || out != "x\n" || read(t, a) != "z\ny\n" {
			t.Fatalf("stdout %q, file %q, error %v", out, read(t, a), err)
		}
	})

	t.Run("q keeps what was written and leaves the other files", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "1\n2\n3\n")
		b := write(t, dir, "b.txt", "1\n2\n3\n")
		if _, _, err := run(t, sed.Options{InPlace: true}, "", "2q", a, b); err != nil {
			t.Fatal(err)
		}
		if read(t, a) != "1\n2\n" || read(t, b) != "1\n2\n3\n" {
			t.Fatalf("a %q, b %q", read(t, a), read(t, b))
		}
	})

	t.Run("no input files", func(t *testing.T) {
		_, errOut, err := run(t, sed.Options{InPlace: true}, "", "p")
		if !errors.Is(err, core.ErrFailed) || errOut != "sedx: no input files\n" {
			t.Fatalf("stderr %q, error %v", errOut, err)
		}
	})

	t.Run("a missing file does not stop the rest", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		missing := filepath.Join(dir, "missing.txt")
		_, errOut, err := run(t, sed.Options{InPlace: true}, "", "s/x/y/", missing, a)
		want := "sedx: can't read " + missing + ": No such file or directory\n"
		if status(err) != sed.ExitBadInput || errOut != want || read(t, a) != "y\n" {
			t.Fatalf("status %d, stderr %q, file %q", status(err), errOut, read(t, a))
		}
	})

	t.Run("not a regular file", func(t *testing.T) {
		dir := t.TempDir()
		_, errOut, err := run(t, sed.Options{InPlace: true}, "", "p", dir)
		want := "sedx: couldn't edit " + dir + ": not a regular file\n"
		if status(err) != sed.ExitIO || errOut != want {
			t.Fatalf("status %d, stderr %q", status(err), errOut)
		}
	})

	t.Run("a script error leaves the file alone", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		if _, _, err := run(t, sed.Options{InPlace: true}, "", "s/x/y", a); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("error %v", err)
		}
		entries, _ := os.ReadDir(dir)
		if read(t, a) != "x\n" || len(entries) != 1 {
			t.Fatalf("file %q, entries %v", read(t, a), entries)
		}
	})

	t.Run("a run-time error leaves the file alone", func(t *testing.T) {
		dir := t.TempDir()
		a := write(t, dir, "a.txt", "x\n")
		if _, _, err := run(t, sed.Options{InPlace: true}, "", "s//y/", a); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("error %v", err)
		}
		entries, _ := os.ReadDir(dir)
		if read(t, a) != "x\n" || len(entries) != 1 {
			t.Fatalf("file %q, entries %v", read(t, a), entries)
		}
	})
}

func TestInPlaceSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := write(t, dir, "target.txt", "x\n")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}

	if _, _, err := run(t, sed.Options{InPlace: true, FollowSymlinks: true}, "", "s/x/y/", link); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 || read(t, target) != "y\n" {
		t.Fatalf("followed: link %v, target %q", info.Mode(), read(t, target))
	}

	// Without --follow-symlinks the link itself becomes the edited file.
	if _, _, err := run(t, sed.Options{InPlace: true}, "", "s/y/z/", link); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || !info.Mode().IsRegular() || read(t, link) != "z\n" || read(t, target) != "y\n" {
		t.Fatalf("replaced: link %v %q, target %q", info.Mode(), read(t, link), read(t, target))
	}
}

func TestPosixlyCorrect(t *testing.T) {
	var out bytes.Buffer
	e := sed.New(sed.Options{}, core.Streams{In: strings.NewReader("1\n2\n3\n"), Out: &out, Err: &out})
	e.Getenv = func(name string) string {
		if name == "POSIXLY_CORRECT" {
			return "1"
		}
		return ""
	}
	if err := e.Run([]string{`N;s/\n/+/`}); err != nil || out.String() != "1+2\n" {
		t.Fatalf("got %q, error %v", out.String(), err)
	}
}

// flushCounter counts the writes that reach it.
type flushCounter struct {
	bytes.Buffer
	writes int
}

func (f *flushCounter) Write(p []byte) (int, error) {
	f.writes++
	return f.Buffer.Write(p)
}

func TestUnbuffered(t *testing.T) {
	for _, tt := range []struct {
		unbuffered bool
		writes     int
	}{{false, 1}, {true, 5}} {
		var out flushCounter
		e := sed.New(sed.Options{Unbuffered: tt.unbuffered}, core.Streams{In: strings.NewReader(five), Out: &out, Err: &out})
		if err := e.Run([]string{"s/^/>/"}); err != nil || out.String() != ">1\n>2\n>3\n>4\n>5\n" || out.writes != tt.writes {
			t.Errorf("unbuffered %v: %d writes, output %q, error %v", tt.unbuffered, out.writes, out.String(), err)
		}
	}
}

// failingWriter fails every write with err.
type failingWriter struct{ err error }

func (f failingWriter) Write([]byte) (int, error) { return 0, f.err }

func TestWriteErrors(t *testing.T) {
	var errOut bytes.Buffer
	streams := core.Streams{In: strings.NewReader(five), Out: failingWriter{os.ErrPermission}, Err: &errOut}
	err := sed.New(sed.Options{}, streams).Run([]string{"p"})
	if status(err) != sed.ExitIO || errOut.String() != "sedx: couldn't write 1 item to stdout: Permission denied\n" {
		t.Fatalf("status %d, stderr %q", status(err), errOut.String())
	}

	// A reader that went away is a quiet stop.
	errOut.Reset()
	streams.In, streams.Out = strings.NewReader(five), failingWriter{io.ErrClosedPipe}
	if err := sed.New(sed.Options{Unbuffered: true}, streams).Run([]string{":a;p;ba"}); err != nil || errOut.Len() != 0 {
		t.Fatalf("error %v, stderr %q", err, errOut.String())
	}
}

// A long line is longer than the reader's buffer.
func TestLongLine(t *testing.T) {
	long := strings.Repeat("ab", 100_000)
	out, _, err := run(t, sed.Options{}, long+"\nshort\n", "s/b/c/g;$!N")
	if want := strings.ReplaceAll(long, "b", "c") + "\nshort\n"; err != nil || out != want {
		t.Fatalf("got %d bytes, error %v", len(out), err)
	}
}

func BenchmarkSubstitute(b *testing.B) {
	input := strings.Repeat("the quick brown fox jumps over the lazy dog\nnothing to see on this line\n", 500)
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	for b.Loop() {
		var out bytes.Buffer
		e := sed.New(sed.Options{}, core.Streams{In: strings.NewReader(input), Out: &out, Err: &out})
		if err := e.Run([]string{"s/fox/cat/"}); err != nil {
			b.Fatal(err)
		}
	}
}
