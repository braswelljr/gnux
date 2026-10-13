package xargs_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/xargs"
)

// fake stands in for the gnux tools: it records every command line it is
// asked to run and behaves by the command's name. "echo" prints its
// arguments, "cat" copies standard input to standard output, "status" exits
// with its first argument; any other name without a path separator succeeds
// silently, and "missing" (like any path) is not a tool at all.
type fake struct {
	mu    sync.Mutex
	calls [][]string
	hook  func(argv []string) // called for every command, when set
}

func (f *fake) exec(name string, args []string, s core.Streams) (int, bool) {
	if name == "missing" || strings.ContainsAny(name, `/\`) {
		return 0, false
	}
	argv := append([]string{name}, args...)
	f.mu.Lock()
	f.calls = append(f.calls, argv)
	f.mu.Unlock()
	if f.hook != nil {
		f.hook(argv)
	}

	switch name {
	case "echo":
		_, _ = io.WriteString(s.Out, strings.Join(args, " ")+"\n")
	case "cat":
		_, _ = io.Copy(s.Out, s.In)
	case "status":
		n, _ := strconv.Atoi(args[0])
		return n, true
	}
	return 0, true
}

// result is everything one run produced.
type result struct {
	calls  [][]string
	stdout string
	stderr string
	err    error
}

// drive runs the engine over stdin with the fake tools and an empty
// environment; setup may adjust the engine and the fake first.
func drive(t *testing.T, opts xargs.Options, stdin string, args []string, setup func(*xargs.Xargs, *fake)) result {
	t.Helper()
	var out, errb bytes.Buffer
	f := &fake{}
	x := xargs.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	x.Exec = f.exec
	x.Environ = func() []string { return nil }
	if setup != nil {
		setup(x, f)
	}
	err := x.Run(args)
	return result{calls: f.calls, stdout: out.String(), stderr: errb.String(), err: err}
}

// status is the exit status err stands for.
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

// testBinary is the path of the running test binary, which is the helper
// program when the engine starts it.
func testBinary(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return self
}

// c builds one expected command line.
func c(argv ...string) []string { return argv }

func TestInputAndBatching(t *testing.T) {
	file := filepath.Join(t.TempDir(), "items")
	if err := os.WriteFile(file, []byte("x y\nz\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		opts  xargs.Options
		in    string
		args  []string
		calls [][]string
	}{
		// The default parser.
		{"default command is echo", xargs.Options{}, "a b\nc\n", nil, [][]string{c("echo", "a", "b", "c")}},
		{"initial arguments come first", xargs.Options{}, "a\n", c("cmd", "-v", "--"), [][]string{c("cmd", "-v", "--", "a")}},
		{"no trailing newline", xargs.Options{}, "a b", nil, [][]string{c("echo", "a", "b")}},
		{"blank lines and tabs", xargs.Options{}, "\n\n a\t\tb \n\n\r\nc\n", nil, [][]string{c("echo", "a", "b", "c")}},
		{"single quotes", xargs.Options{}, "'a b' 'say \"hi\"'\n", nil, [][]string{c("echo", "a b", `say "hi"`)}},
		{"double quotes", xargs.Options{}, `"a b" "it's"` + "\n", nil, [][]string{c("echo", "a b", "it's")}},
		{"quotes join their neighbors", xargs.Options{}, `a"b c"d'e f'` + "\n", nil, [][]string{c("echo", "ab cde f")}},
		{"backslash in quotes is literal", xargs.Options{}, `'a\b' "c\d"` + "\n", nil, [][]string{c("echo", `a\b`, `c\d`)}},
		{"backslash", xargs.Options{}, `a\ b \'c\' d\\e \"` + "\n", nil, [][]string{c("echo", "a b", "'c'", `d\e`, `"`)}},
		{"backslash newline", xargs.Options{}, "a\\\nb\n", nil, [][]string{c("echo", "a\nb")}},
		{"trailing backslash", xargs.Options{}, `a\`, nil, [][]string{c("echo", "a")}},
		{"empty quoted items", xargs.Options{}, `"" a ''` + "\n", nil, [][]string{c("echo", "", "a", "")}},
		{"empty quoted item at the end", xargs.Options{}, `a ""`, nil, [][]string{c("echo", "a", "")}},
		{"carriage return stays in the item", xargs.Options{}, "a\r\n", nil, [][]string{c("echo", "a\r")}},

		// -0.
		{"-0", xargs.Options{Null: true}, "a b\x00c\n\x00'q'\x00\\", nil, [][]string{c("echo", "a b", "c\n", "'q'", `\`)}},
		{"-0 keeps empty items", xargs.Options{Null: true}, "a\x00\x00b\x00", nil, [][]string{c("echo", "a", "", "b")}},
		{"-0 wins over -d", xargs.Options{Null: true, Delimiter: ","}, "a,b\x00c", nil, [][]string{c("echo", "a,b", "c")}},

		// -d.
		{"-d character", xargs.Options{Delimiter: ","}, "a b,'c',\n", nil, [][]string{c("echo", "a b", "'c'", "\n")}},
		{"-d newline", xargs.Options{Delimiter: `\n`}, "a b\n\"c\n", nil, [][]string{c("echo", "a b", `"c`)}},
		{"-d tab", xargs.Options{Delimiter: `\t`}, "a b\tc", nil, [][]string{c("echo", "a b", "c")}},
		{"-d NUL", xargs.Options{Delimiter: `\0`}, "a\x00b", nil, [][]string{c("echo", "a", "b")}},
		{"-d octal", xargs.Options{Delimiter: `\054`}, "a,b", nil, [][]string{c("echo", "a", "b")}},
		{"-d hex", xargs.Options{Delimiter: `\x2C`}, "a,b", nil, [][]string{c("echo", "a", "b")}},
		{"-d backslash", xargs.Options{Delimiter: `\\`}, `a\b`, nil, [][]string{c("echo", "a", "b")}},
		{"-d literal backslash", xargs.Options{Delimiter: `\`}, `a\b`, nil, [][]string{c("echo", "a", "b")}},

		// -a.
		{"-a", xargs.Options{ArgFile: file}, "ignored", nil, [][]string{c("echo", "x", "y", "z")}},
		{"-a -", xargs.Options{ArgFile: "-"}, "a b", nil, [][]string{c("echo", "a", "b")}},

		// -E.
		{"-E", xargs.Options{EOF: "END"}, "a b END c\nd\n", nil, [][]string{c("echo", "a", "b")}},
		{"-E on its own line", xargs.Options{EOF: "_"}, "a\n_\nb\n", nil, [][]string{c("echo", "a")}},
		{"-E must match the whole item", xargs.Options{EOF: "END"}, "ENDED END2 xEND\n", nil, [][]string{c("echo", "ENDED", "END2", "xEND")}},
		{"-E first", xargs.Options{EOF: "END", NoRunIfEmpty: true}, "END a\n", nil, nil},
		{"-E is off with -0", xargs.Options{EOF: "END", Null: true}, "a\x00END\x00b", nil, [][]string{c("echo", "a", "END", "b")}},
		{"-E is off with -d", xargs.Options{EOF: "END", Delimiter: ","}, "a,END,b", nil, [][]string{c("echo", "a", "END", "b")}},

		// -n.
		{"-n 2", xargs.Options{MaxArgs: 2}, "1 2 3\n4 5\n", nil, [][]string{c("echo", "1", "2"), c("echo", "3", "4"), c("echo", "5")}},
		{"-n 1", xargs.Options{MaxArgs: 1}, "a b", c("cmd", "-x"), [][]string{c("cmd", "-x", "a"), c("cmd", "-x", "b")}},
		{"-n exact multiple", xargs.Options{MaxArgs: 2}, "1 2 3 4", nil, [][]string{c("echo", "1", "2"), c("echo", "3", "4")}},
		{"-n with -0", xargs.Options{MaxArgs: 1, Null: true}, "a b\x00c", nil, [][]string{c("echo", "a b"), c("echo", "c")}},

		// -L.
		{"-L 1", xargs.Options{MaxLines: 1}, "a b\n\nc\nd e f\n", nil, [][]string{c("echo", "a", "b"), c("echo", "c"), c("echo", "d", "e", "f")}},
		{"-L 2", xargs.Options{MaxLines: 2}, "a\nb\nc\n", nil, [][]string{c("echo", "a", "b"), c("echo", "c")}},
		{"-L trailing blank continues the line", xargs.Options{MaxLines: 1}, "a \nb\nc\t\n\nd\n", nil, [][]string{c("echo", "a", "b"), c("echo", "c", "d")}},
		{"-L last line unterminated", xargs.Options{MaxLines: 1}, "a\nb c", nil, [][]string{c("echo", "a"), c("echo", "b", "c")}},
		{"-L with -0 counts items", xargs.Options{MaxLines: 2, Null: true}, "a\x00b\x00c", nil, [][]string{c("echo", "a", "b"), c("echo", "c")}},
		{"-L and -n together", xargs.Options{MaxLines: 1, MaxArgs: 2}, "a b c\nd\n", nil, [][]string{c("echo", "a", "b"), c("echo", "c"), c("echo", "d")}},

		// -s: "echo" and its NUL take 5 bytes, each one-letter item 2.
		{"-s", xargs.Options{MaxChars: 9}, "a b c d e", nil, [][]string{c("echo", "a", "b"), c("echo", "c", "d"), c("echo", "e")}},
		{"-s exact fit", xargs.Options{MaxChars: 11}, "a b c", nil, [][]string{c("echo", "a", "b", "c")}},
		{"-s with -n", xargs.Options{MaxChars: 9, MaxArgs: 3}, "a b c d", nil, [][]string{c("echo", "a", "b"), c("echo", "c", "d")}},
		{"-x alone changes nothing", xargs.Options{MaxChars: 9, Exit: true}, "a b c", nil, [][]string{c("echo", "a", "b"), c("echo", "c")}},

		// -I.
		{"-I", xargs.Options{Replace: "{}"}, "a\nb c\n", c("mv", "{}", "dir/{}.bak"), [][]string{c("mv", "a", "dir/a.bak"), c("mv", "b c", "dir/b c.bak")}},
		{"-I strips leading blanks and blank lines", xargs.Options{Replace: "X"}, "  a b \n\n\tc\n", c("cmd", "X"), [][]string{c("cmd", "a b "), c("cmd", "c")}},
		{"-I several in one argument", xargs.Options{Replace: "%"}, "ab\n", c("cmd", "%-%", "x"), [][]string{c("cmd", "ab-ab", "x")}},
		{"-I replaces the command too", xargs.Options{Replace: "{}"}, "echo\n", c("{}", "hi"), [][]string{c("echo", "hi")}},
		{"-I still reads quotes", xargs.Options{Replace: "{}"}, "'a  b' \\\"c\n", c("cmd", "{}"), [][]string{c("cmd", `a  b "c`)}},
		{"-I without the string in the command", xargs.Options{Replace: "{}"}, "a\nb\n", c("cmd"), [][]string{c("cmd"), c("cmd")}},
		{"-I default command", xargs.Options{Replace: "{}"}, "a\n", nil, [][]string{c("echo")}},
		{"-I with -0", xargs.Options{Replace: "{}", Null: true}, " a\nb\x00c", c("cmd", "<{}>"), [][]string{c("cmd", "< a\nb>"), c("cmd", "<c>")}},
		{"-I ignores -n and -L", xargs.Options{Replace: "{}", MaxArgs: 5, MaxLines: 5}, "a\nb\n", c("cmd", "{}"), [][]string{c("cmd", "a"), c("cmd", "b")}},
		{"-I with no input runs nothing", xargs.Options{Replace: "{}"}, "", c("cmd", "{}"), nil},

		// Empty input.
		{"empty input runs once", xargs.Options{}, "", c("cmd", "x"), [][]string{c("cmd", "x")}},
		{"blank input runs once", xargs.Options{MaxArgs: 2}, " \n\n", nil, [][]string{c("echo")}},
		{"-r empty input", xargs.Options{NoRunIfEmpty: true}, "", c("cmd"), nil},
		{"-r blank input", xargs.Options{NoRunIfEmpty: true, MaxLines: 1}, "\n \n", c("cmd"), nil},
		{"-r with input", xargs.Options{NoRunIfEmpty: true}, "a", c("cmd"), [][]string{c("cmd", "a")}},
		{"-0 empty input runs once", xargs.Options{Null: true}, "", nil, [][]string{c("echo")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := drive(t, tt.opts, tt.in, tt.args, nil)
			if res.err != nil || res.stderr != "" {
				t.Fatalf("err = %v, stderr = %q", res.err, res.stderr)
			}
			if !reflect.DeepEqual(res.calls, tt.calls) {
				t.Errorf("commands = %q, want %q", res.calls, tt.calls)
			}
		})
	}
}

func TestFailures(t *testing.T) {
	long := strings.Repeat("x", 30)
	tests := []struct {
		name   string
		opts   xargs.Options
		in     string
		args   []string
		calls  [][]string
		stderr string
	}{
		{"unmatched single quote", xargs.Options{}, "a 'b\nc'\n", nil, [][]string{c("echo", "a")},
			"xargsx: unmatched single quote; by default quotes are special to xargs unless you use the -0 option\n"},
		{"unmatched double quote", xargs.Options{}, "a\nb \"c d\n", nil, [][]string{c("echo", "a", "b")},
			"xargsx: unmatched double quote; by default quotes are special to xargs unless you use the -0 option\n"},
		{"quote open at the end of input", xargs.Options{}, `"abc`, nil, nil,
			"xargsx: unmatched double quote; by default quotes are special to xargs unless you use the -0 option\n"},
		{"unmatched quote with -I", xargs.Options{Replace: "{}"}, "a\n'b\n", c("cmd", "{}"), [][]string{c("cmd", "a")},
			"xargsx: unmatched single quote; by default quotes are special to xargs unless you use the -0 option\n"},
		{"item longer than -s", xargs.Options{MaxChars: 20}, "a " + long + " b\n", nil, [][]string{c("echo", "a")},
			"xargsx: argument line too long\n"},
		{"-0 item longer than -s", xargs.Options{MaxChars: 20, Null: true}, "a\x00" + long + "\x00b", nil, [][]string{c("echo", "a")},
			"xargsx: argument line too long\n"},
		{"-0 unterminated item longer than -s", xargs.Options{MaxChars: 20, Null: true}, long, nil, nil,
			"xargsx: argument line too long\n"},
		{"item that fits only without the command", xargs.Options{MaxChars: 20}, "a " + long[:17] + "\n", nil, [][]string{c("echo", "a")},
			"xargsx: cannot fit single argument within argument list size limit\n"},
		{"command longer than -s", xargs.Options{MaxChars: 4}, "a\n", nil, nil,
			"xargsx: cannot fit single argument within argument list size limit\n"},
		{"-x with -n", xargs.Options{MaxChars: 9, MaxArgs: 3, Exit: true}, "a b c d", nil, nil,
			"xargsx: argument list too long\n"},
		{"-L implies -x", xargs.Options{MaxChars: 9, MaxLines: 1}, "a b\nc d e\n", nil, [][]string{c("echo", "a", "b")},
			"xargsx: argument list too long\n"},
		{"-I line too long for one argument", xargs.Options{MaxChars: 12, Replace: "{}"}, "aaaaaaa\n", c("c", "{}{}"), nil,
			"xargsx: command too long\n"},
		{"-I command line too long", xargs.Options{MaxChars: 12, Replace: "{}"}, "a\nbbbbb\n", c("c", "{}", "{}"), [][]string{c("c", "a", "a")},
			"xargsx: argument list too long\n"},
		{"-a missing file", xargs.Options{ArgFile: "no/such/file"}, "", nil, nil,
			"xargsx: Cannot open input file 'no/such/file': No such file or directory\n"},
		{"-d two characters", xargs.Options{Delimiter: "ab"}, "", nil, nil,
			"xargsx: Invalid input delimiter specification ab: the delimiter must be either a single character or an escape sequence starting with \\.\n"},
		{"-d unknown escape", xargs.Options{Delimiter: `\q`}, "", nil, nil,
			"xargsx: Invalid escape sequence \\q in input delimiter specification.\n"},
		{"-d octal too large", xargs.Options{Delimiter: `\400`}, "", nil, nil,
			"xargsx: Invalid escape sequence \\400 in input delimiter specification; character values must not exceed 377.\n"},
		{"-d hex too large", xargs.Options{Delimiter: `\x100`}, "", nil, nil,
			"xargsx: Invalid escape sequence \\x100 in input delimiter specification; character values must not exceed ff.\n"},
		{"-d trailing characters", xargs.Options{Delimiter: `\x41z`}, "", nil, nil,
			"xargsx: Invalid escape sequence \\x41z in input delimiter specification; trailing characters z not recognized.\n"},
		{"-d trailing characters after an escape", xargs.Options{Delimiter: `\nn`}, "", nil, nil,
			"xargsx: Invalid escape sequence \\nn in input delimiter specification; trailing characters n not recognized.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := drive(t, tt.opts, tt.in, tt.args, nil)
			if !errors.Is(res.err, core.ErrFailed) {
				t.Errorf("err = %v, want core.ErrFailed", res.err)
			}
			if res.stderr != tt.stderr {
				t.Errorf("stderr = %q, want %q", res.stderr, tt.stderr)
			}
			if !reflect.DeepEqual(res.calls, tt.calls) {
				t.Errorf("commands = %q, want %q", res.calls, tt.calls)
			}
		})
	}
}

func TestParseDelimiter(t *testing.T) {
	for spec, want := range map[string]byte{
		"x": 'x', `\a`: '\a', `\b`: '\b', `\f`: '\f', `\n`: '\n', `\r`: '\r', `\t`: '\t', `\v`: '\v',
		`\\`: '\\', `\0`: 0, `\7`: 7, `\101`: 'A', `\377`: 0xff, `\x0`: 0, `\x41`: 'A', `\xfF`: 0xff,
	} {
		if got, err := xargs.ParseDelimiter(spec); err != nil || got != want {
			t.Errorf("ParseDelimiter(%q) = %d, %v, want %d", spec, got, err, want)
		}
	}
	for _, spec := range []string{"", "ab", "é", `\x`, `\8`, `\z`, `\1x`} {
		if _, err := xargs.ParseDelimiter(spec); err == nil {
			t.Errorf("ParseDelimiter(%q) succeeded", spec)
		}
	}
}

func TestNULInInput(t *testing.T) {
	res := drive(t, xargs.Options{}, "a\x00b c\x00\n", nil, nil)
	want := "xargsx: WARNING: a NUL character occurred in the input.  It cannot be passed through " +
		"in the argument list.  Did you mean to use the --null option?\n"
	if res.err != nil || res.stderr != want {
		t.Errorf("err = %v, stderr = %q", res.err, res.stderr)
	}
	if want := [][]string{c("echo", "ab", "c")}; !reflect.DeepEqual(res.calls, want) {
		t.Errorf("commands = %q, want %q", res.calls, want)
	}
}

func TestCommandOutputAndInput(t *testing.T) {
	res := drive(t, xargs.Options{MaxArgs: 2}, "a b c\n", nil, nil)
	if res.err != nil || res.stdout != "a b\nc\n" || res.stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, err = %v", res.stdout, res.stderr, res.err)
	}

	// Without -a the command must not see the items on its standard input.
	res = drive(t, xargs.Options{MaxArgs: 1}, "a\nb\n", c("cat"), nil)
	if res.err != nil || res.stdout != "" || len(res.calls) != 2 {
		t.Errorf("cat got %q on its standard input (%d runs, err = %v)", res.stdout, len(res.calls), res.err)
	}

	// With -a the command reads what xargsx was given as standard input.
	file := filepath.Join(t.TempDir(), "items")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res = drive(t, xargs.Options{ArgFile: file}, "from stdin\n", c("cat"), nil)
	if res.err != nil || res.stdout != "from stdin\n" {
		t.Errorf("stdout = %q, err = %v, want the standard input", res.stdout, res.err)
	}
}

func TestDefaultBatchingFillsTheCommandLine(t *testing.T) {
	const items = 100000
	var in strings.Builder
	for i := 0; i < items; i++ {
		in.WriteString("item")
		in.WriteString(strconv.Itoa(i))
		in.WriteByte('\n')
	}
	res := drive(t, xargs.Options{}, in.String(), c("rm", "-f"), nil)
	if res.err != nil || res.stderr != "" {
		t.Fatalf("err = %v, stderr = %q", res.err, res.stderr)
	}

	next, total := 0, 0
	for n, argv := range res.calls {
		size := 0
		for _, a := range argv {
			size += len(a) + 1
		}
		total += size
		if size > xargs.DefaultMaxChars {
			t.Fatalf("command %d takes %d bytes, over the limit of %d", n, size, xargs.DefaultMaxChars)
		}
		if argv[0] != "rm" || argv[1] != "-f" {
			t.Fatalf("command %d starts with %q", n, argv[:2])
		}
		for _, a := range argv[2:] {
			if want := "item" + strconv.Itoa(next); a != want {
				t.Fatalf("command %d has %q where %q belongs", n, a, want)
			}
			next++
		}
	}
	if next != items {
		t.Errorf("%d items were passed, want %d", next, items)
	}
	// Every command but the last is full, so there are about as few as the
	// size allows (the limit is lower than the default on some platforms).
	if avg := total / len(res.calls); len(res.calls) > 1 && avg < 16000 {
		t.Errorf("%d commands averaging %d bytes: the command lines are not filled", len(res.calls), avg)
	}
}

func TestMaxCharsOverTheSystemLimit(t *testing.T) {
	res := drive(t, xargs.Options{MaxChars: 1 << 30}, "a\n", nil, nil)
	if res.err != nil || len(res.calls) != 1 {
		t.Fatalf("err = %v, commands = %q", res.err, res.calls)
	}
	const prefix = "xargsx: warning: value 1073741824 for -s option is too large, using "
	if !strings.HasPrefix(res.stderr, prefix) || !strings.HasSuffix(res.stderr, " instead\n") {
		t.Errorf("stderr = %q", res.stderr)
	}
}

func TestVerbose(t *testing.T) {
	tests := []struct {
		name string
		in   string
		args []string
		want string
	}{
		{"plain", "a b\n", nil, "echo a b\n"},
		{"batches", "a\x00b\x00", c("cmd", "-n1"), "cmd -n1 a b\n"},
		{"blank", "'a b'\n", nil, "echo 'a b'\n"},
		{"empty", `""` + "\n", nil, "echo ''\n"},
		{"single quote", `"it's"` + "\n", nil, `echo 'it'\''s'` + "\n"},
		{"shell characters", `a\$b c\*  d=e \#f g#h \~ {} \{` + "\n", nil, `echo 'a$b' 'c*' 'd=e' '#f' g#h '~' {} '{'` + "\n"},
		{"newline", "a\\\nb\n", nil, `echo 'a'$'\n''b'` + "\n"},
		{"control character", "\x01x\n", nil, `echo ''$'\001''x'` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := xargs.Options{Verbose: true, Null: strings.Contains(tt.in, "\x00")}
			res := drive(t, opts, tt.in, tt.args, nil)
			if res.err != nil || res.stderr != tt.want {
				t.Errorf("stderr = %q, want %q (err = %v)", res.stderr, tt.want, res.err)
			}
		})
	}
}

func TestVerboseGoesBeforeEachCommand(t *testing.T) {
	// One buffer for both streams shows the order of the trace and the output.
	var both bytes.Buffer
	f := &fake{}
	x := xargs.New(xargs.Options{Verbose: true, MaxArgs: 1}, core.Streams{In: strings.NewReader("a b"), Out: &both, Err: &both})
	x.Exec = f.exec
	if err := x.Run(nil); err != nil {
		t.Fatal(err)
	}
	if want := "echo a\na\necho b\nb\n"; both.String() != want {
		t.Errorf("output = %q, want %q", both.String(), want)
	}
}

func TestInteractive(t *testing.T) {
	var asked []string
	res := drive(t, xargs.Options{Interactive: true, MaxArgs: 1}, "a\n'b c'\nd\n", nil, func(x *xargs.Xargs, _ *fake) {
		x.PromptFn = func(command string) bool {
			asked = append(asked, command)
			return command != "echo 'b c'"
		}
	})
	if res.err != nil {
		t.Fatal(res.err)
	}
	if want := []string{"echo a", "echo 'b c'", "echo d"}; !reflect.DeepEqual(asked, want) {
		t.Errorf("asked = %q, want %q", asked, want)
	}
	if want := "echo a?...echo 'b c'?...echo d?..."; res.stderr != want {
		t.Errorf("stderr = %q, want %q", res.stderr, want)
	}
	if want := [][]string{c("echo", "a"), c("echo", "d")}; !reflect.DeepEqual(res.calls, want) {
		t.Errorf("commands = %q, want %q", res.calls, want)
	}
}

func TestInteractiveDeclinedOnEmptyInput(t *testing.T) {
	res := drive(t, xargs.Options{Interactive: true}, "", c("cmd"), func(x *xargs.Xargs, _ *fake) {
		x.PromptFn = func(string) bool { return false }
	})
	if res.err != nil || res.stderr != "cmd?..." || res.calls != nil {
		t.Errorf("stderr = %q, commands = %q, err = %v", res.stderr, res.calls, res.err)
	}
}

func TestExitStatuses(t *testing.T) {
	tests := []struct {
		name   string
		opts   xargs.Options
		in     string
		args   []string
		code   int
		runs   int
		stderr string
	}{
		{"success", xargs.Options{MaxArgs: 1}, "0 0", c("status"), 0, 2, ""},
		{"status 1 lets the rest run", xargs.Options{MaxArgs: 1}, "1 0 0", c("status"), 123, 3, ""},
		{"status 125", xargs.Options{MaxArgs: 1}, "0 125 0", c("status"), 123, 3, ""},
		{"status 126 from the command", xargs.Options{MaxArgs: 1}, "126 0", c("status"), 123, 2, ""},
		{"status 255 stops", xargs.Options{MaxArgs: 1}, "0 255 0", c("status"), 124, 2,
			"xargsx: status: exited with status 255; aborting\n"},
		{"status 255 after a failure", xargs.Options{MaxArgs: 1}, "1 255 0", c("status"), 124, 2,
			"xargsx: status: exited with status 255; aborting\n"},
		{"status 255 with -I", xargs.Options{Replace: "{}"}, "255\n0\n", c("status", "{}"), 124, 1,
			"xargsx: status: exited with status 255; aborting\n"},
		{"not found", xargs.Options{MaxArgs: 1}, "a b", c("missing"), 127, 0,
			"xargsx: missing: No such file or directory\n"},
		{"not found by path", xargs.Options{}, "a", c(filepath.Join("no", "such", "program")), 127, 0,
			"xargsx: " + filepath.Join("no", "such", "program") + ": No such file or directory\n"},
		{"failure then unmatched quote", xargs.Options{MaxArgs: 1}, "1 'x", c("status"), 1, 1,
			"xargsx: unmatched single quote; by default quotes are special to xargs unless you use the -0 option\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Keep the search for "missing" away from whatever PATH holds.
			t.Setenv("PATH", t.TempDir())
			res := drive(t, tt.opts, tt.in, tt.args, nil)
			if got := status(res.err); got != tt.code {
				t.Errorf("status = %d (%v), want %d", got, res.err, tt.code)
			}
			if len(res.calls) != tt.runs {
				t.Errorf("%d commands ran (%q), want %d", len(res.calls), res.calls, tt.runs)
			}
			if res.stderr != tt.stderr {
				t.Errorf("stderr = %q, want %q", res.stderr, tt.stderr)
			}
		})
	}
}

func TestUnsupportedOptions(t *testing.T) {
	for want, opts := range map[string]xargs.Options{
		"the --open-tty (-o) option is not supported":    {OpenTTY: true},
		"the --process-slot-var option is not supported": {ProcessSlotVar: "SLOT"},
	} {
		res := drive(t, opts, "a\n", nil, nil)
		var usage *core.UsageError
		if !errors.As(res.err, &usage) || usage.Msg != want {
			t.Errorf("err = %v, want the usage error %q", res.err, want)
		}
		if res.calls != nil || res.stderr != "" {
			t.Errorf("commands = %q, stderr = %q, want nothing run", res.calls, res.stderr)
		}
	}
}

func TestShowLimits(t *testing.T) {
	res := drive(t, xargs.Options{ShowLimits: true, MaxChars: 5000, NoRunIfEmpty: true}, "", nil, func(x *xargs.Xargs, _ *fake) {
		x.Environ = func() []string { return []string{"A=1", "BC=23"} }
	})
	if res.err != nil || res.calls != nil {
		t.Fatalf("err = %v, commands = %q", res.err, res.calls)
	}
	lines := strings.Split(res.stderr, "\n")
	if len(lines) != 7 {
		t.Fatalf("stderr = %q, want six lines", res.stderr)
	}
	for i, want := range map[int]string{
		0: "Your environment variables take up 10 bytes",
		2: "POSIX smallest allowable upper limit on argument length (all systems): 4096",
		4: "Size of command buffer we are actually using: 5000",
		5: "Maximum parallelism (--max-procs must be no greater): 2147483647",
	} {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
	for i, prefix := range map[int]string{
		1: "POSIX upper limit on argument length (this system): ",
		3: "Maximum length of command we could actually use: ",
	} {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d = %q, want it to start with %q", i, lines[i], prefix)
		}
	}
}

func TestParallel(t *testing.T) {
	for name, procs := range map[string]int{"-P 3": 3, "-P 0": xargs.Unlimited} {
		t.Run(name, func(t *testing.T) {
			// Every command waits until three are running at once, which
			// only happens if xargsx does not wait for one before the next.
			var running sync.WaitGroup
			running.Add(3)
			together := make(chan struct{})
			go func() { running.Wait(); close(together) }()

			res := drive(t, xargs.Options{MaxProcs: procs, MaxArgs: 1}, "a b c", nil, func(_ *xargs.Xargs, f *fake) {
				f.hook = func([]string) {
					running.Done()
					select {
					case <-together:
					case <-time.After(10 * time.Second):
						t.Error("the commands did not run at the same time")
					}
				}
			})
			if res.err != nil || len(res.calls) != 3 {
				t.Fatalf("err = %v, commands = %q", res.err, res.calls)
			}
			got := strings.Split(strings.TrimSuffix(res.stdout, "\n"), "\n")
			if len(got) != 3 || !strings.Contains(res.stdout, "a\n") || !strings.Contains(res.stdout, "b\n") || !strings.Contains(res.stdout, "c\n") {
				t.Errorf("stdout = %q, want the lines a, b and c in any order", res.stdout)
			}
		})
	}
}

func TestParallelIsBounded(t *testing.T) {
	var mu sync.Mutex
	now, most := 0, 0
	res := drive(t, xargs.Options{MaxProcs: 2, MaxArgs: 1, Verbose: true}, strings.Repeat("x ", 40), nil, func(_ *xargs.Xargs, f *fake) {
		f.hook = func([]string) {
			mu.Lock()
			now++
			most = max(most, now)
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			now--
			mu.Unlock()
		}
	})
	if res.err != nil || len(res.calls) != 40 {
		t.Fatalf("err = %v, %d commands", res.err, len(res.calls))
	}
	if most > 2 {
		t.Errorf("%d commands ran at once, want at most 2", most)
	}
	// Both streams are plain buffers written by every command at once: each
	// line must still arrive whole.
	if want := strings.Repeat("x\n", 40); res.stdout != want {
		t.Errorf("stdout = %q", res.stdout)
	}
	if want := strings.Repeat("echo x\n", 40); res.stderr != want {
		t.Errorf("stderr = %q", res.stderr)
	}
}

func TestParallelStopsAfterStatus255(t *testing.T) {
	res := drive(t, xargs.Options{MaxProcs: 2, MaxArgs: 1}, "255 "+strings.Repeat("0 ", 2000), c("status"), nil)
	if code := status(res.err); code != xargs.ExitAborted {
		t.Fatalf("status = %d (%v), want %d", code, res.err, xargs.ExitAborted)
	}
	if res.stderr != "xargsx: status: exited with status 255; aborting\n" {
		t.Errorf("stderr = %q", res.stderr)
	}
	if len(res.calls) > 1000 {
		t.Errorf("%d commands ran after the one that asked xargsx to stop", len(res.calls))
	}
}

func TestSequentialOrder(t *testing.T) {
	for _, procs := range []int{0, 1} {
		res := drive(t, xargs.Options{MaxProcs: procs, MaxArgs: 1}, "1 2 3 4 5 6 7 8", nil, nil)
		if res.err != nil || res.stdout != "1\n2\n3\n4\n5\n6\n7\n8\n" {
			t.Errorf("MaxProcs %d: stdout = %q, err = %v", procs, res.stdout, res.err)
		}
	}
}

// The tests below start a real program: the test binary itself, which acts as
// the helper of helper_test.go.

func TestProgram(t *testing.T) {
	self := testBinary(t)
	res := drive(t, xargs.Options{MaxArgs: 2}, "a b c\n", c(self, "echo"), nil)
	if res.err != nil || res.stderr != "" {
		t.Fatalf("err = %v, stderr = %q", res.err, res.stderr)
	}
	// The fake tools do not know a path, so nothing ran in-process.
	if res.calls != nil {
		t.Errorf("the program ran in-process: %q", res.calls)
	}
	if got := strings.ReplaceAll(res.stdout, "\r\n", "\n"); got != "a b\nc\n" {
		t.Errorf("stdout = %q, want %q", got, "a b\nc\n")
	}
}

func TestProgramWithoutExec(t *testing.T) {
	// A nil Exec, as in a program that never loaded the tool registry.
	self := testBinary(t)
	res := drive(t, xargs.Options{}, "a\n", c(self, "echo", "got"), func(x *xargs.Xargs, _ *fake) { x.Exec = nil })
	if got := strings.ReplaceAll(res.stdout, "\r\n", "\n"); res.err != nil || got != "got a\n" {
		t.Errorf("stdout = %q, err = %v", res.stdout, res.err)
	}
}

func TestProgramStandardInput(t *testing.T) {
	self := testBinary(t)
	res := drive(t, xargs.Options{}, "a\n", c(self, "cat"), nil)
	if res.err != nil || res.stdout != "" {
		t.Errorf("the program read %q from its standard input (err = %v)", res.stdout, res.err)
	}
}

func TestProgramExitStatuses(t *testing.T) {
	self := testBinary(t)
	res := drive(t, xargs.Options{MaxArgs: 1}, "3 0\n", c(self, "exit"), nil)
	if code := status(res.err); code != xargs.ExitCommandFailed || res.stderr != "" {
		t.Errorf("status = %d, stderr = %q, want %d", code, res.stderr, xargs.ExitCommandFailed)
	}

	res = drive(t, xargs.Options{MaxArgs: 1}, "255 0\n", c(self, "exit"), nil)
	if code := status(res.err); code != xargs.ExitAborted {
		t.Errorf("status = %d, want %d", code, xargs.ExitAborted)
	}
	if want := "xargsx: " + self + ": exited with status 255; aborting\n"; res.stderr != want {
		t.Errorf("stderr = %q, want %q", res.stderr, want)
	}
}

func TestProgramsInParallel(t *testing.T) {
	self := testBinary(t)
	res := drive(t, xargs.Options{MaxArgs: 1, MaxProcs: 4}, "1 2 3 4 5 6 7 8\n", c(self, "echo"), nil)
	if res.err != nil || res.stderr != "" {
		t.Fatalf("err = %v, stderr = %q", res.err, res.stderr)
	}
	lines := strings.Fields(res.stdout)
	if len(lines) != 8 {
		t.Errorf("stdout = %q, want eight lines", res.stdout)
	}
}

// A panic on a goroutine a command runs on (-P) is raised on the goroutine
// that called Run, where a host can recover it.
func TestParallelPanicReachesTheCaller(t *testing.T) {
	var got *core.Panic
	func() {
		defer func() { got, _ = recover().(*core.Panic) }()
		drive(t, xargs.Options{MaxArgs: 1, MaxProcs: 4}, "a b c d e f\n", []string{"cmd"}, func(x *xargs.Xargs, _ *fake) {
			x.Exec = func(_ string, args []string, _ core.Streams) (int, bool) {
				if len(args) > 0 && args[0] == "c" {
					panic("command failed badly")
				}
				return 0, true
			}
		})
	}()
	if got == nil || got.Value != "command failed badly" {
		t.Fatalf("recovered %+v, want the command's panic", got)
	}
}
