package shell_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shell"
)

// fakeExec is a tiny command set that lets the interpreter be tested without
// the real tools: echo, cat, upper (stdin upper-cased), fail (status 3), true,
// eq (are the two arguments equal?), test (A = B only), args (one bracketed
// argument per line), lines (one per line, bare), exists (is the path there?)
// and spam (writes until the pipe closes).
func fakeExec(name string, args []string, s core.Streams) (int, bool) {
	switch name {
	case "echo":
		fmt.Fprintln(s.Out, strings.Join(args, " "))
	case "args":
		for _, a := range args {
			fmt.Fprintf(s.Out, "[%s]\n", a)
		}
	case "cat":
		_, _ = io.Copy(s.Out, s.In)
	case "upper":
		b, _ := io.ReadAll(s.In)
		fmt.Fprint(s.Out, strings.ToUpper(string(b)))
	case "first":
		line, _ := bufio.NewReader(s.In).ReadString('\n')
		fmt.Fprint(s.Out, line)
	case "spam":
		for {
			if _, err := fmt.Fprintln(s.Out, "y"); err != nil {
				if errors.Is(err, syscall.EPIPE) {
					return 0, true
				}
				return 1, true
			}
		}
	case "exists":
		if _, err := os.Stat(args[0]); err != nil {
			fmt.Fprintln(s.Out, "no")
		} else {
			fmt.Fprintln(s.Out, "yes")
		}
	case "warn":
		fmt.Fprintln(s.Err, strings.Join(args, " "))
		fmt.Fprintln(s.Out, "out")
	case "fail":
		return 3, true
	case "true":
	case "eq", "test":
		// eq A B, and test A = B: the one test the fake set knows.
		if name == "test" && len(args) == 3 && args[1] == "=" {
			args = []string{args[0], args[2]}
		}
		if len(args) != 2 || args[0] != args[1] {
			return 1, true
		}
	case "lines":
		for _, a := range args {
			fmt.Fprintln(s.Out, a)
		}
	case "pwd":
		wd, _ := os.Getwd()
		fmt.Fprintln(s.Out, wd)
	default:
		return 0, false
	}
	return 0, true
}

// run executes script and returns the exit status and both streams.
func run(t *testing.T, opts shell.Options, script string, setup ...func(*shell.Shell)) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	sh := shell.New(opts, fakeExec, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	sh.Getenv = func(k string) string {
		if k == "FROM_ENV" {
			return "env-value"
		}
		return ""
	}
	sh.HomeDir = func() (string, error) { return "/home/u", nil }
	for _, fn := range setup {
		fn(sh)
	}

	status := 0
	if err := sh.RunString(script); err != nil {
		var exit *core.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("Run returned %v, want nil or *core.ExitError", err)
		}
		status = exit.Code
	}
	return status, out.String(), errb.String()
}

func TestScripts(t *testing.T) {
	tests := []struct {
		name, script string
		wantStatus   int
		wantOut      string
	}{
		{"simple command", "echo hello world", 0, "hello world\n"},
		{"several lines", "echo a\necho b\n", 0, "a\nb\n"},
		{"crlf line endings", "echo a\r\necho b\r\n", 0, "a\nb\n"},
		{"semicolons", "echo a; echo b;", 0, "a\nb\n"},
		{"comments and blanks", "# intro\n\necho a # trailing\n", 0, "a\n"},
		{"hash inside a word", "echo a#b", 0, "a#b\n"},

		{"single quotes are literal", `args 'a b' '$X' '*'`, 0, "[a b]\n[$X]\n[*]\n"},
		{"double quotes keep spaces", `args "a  b"`, 0, "[a  b]\n"},
		{"empty strings are arguments", `args '' ""`, 0, "[]\n[]\n"},
		{"adjacent pieces join", `args a"b c"'d'`, 0, "[ab cd]\n"},
		{"backslash escapes specials", `args a\ b \$X \#`, 0, "[a b]\n[$X]\n[#]\n"},
		{"backslash kept otherwise", `args C:\tools\bin`, 0, "[C:\\tools\\bin]\n"},
		{"escapes in double quotes", `args "a\"b" "c\$d" "e\nf"`, 0, "[a\"b]\n[c$d]\n[e\\nf]\n"},

		{"variables", "X=1; args $X ${X}2 \"<$X>\"", 0, "[1]\n[12]\n[<1>]\n"},
		{"unset variable is empty", "args a$NOPE", 0, "[a]\n"},
		{"environment fallback", "args $FROM_ENV", 0, "[env-value]\n"},
		{"shell variable shadows env", "FROM_ENV=mine; args $FROM_ENV", 0, "[mine]\n"},
		{"value is not re-split", "X='a b'; args $X", 0, "[a b]\n"},
		{"lone dollar is literal", "args $ a$", 0, "[$]\n[a$]\n"},
		{"assignment before a command", "X=1 args a; args $X", 0, "[a]\n[1]\n"},
		{"digit before > inside a word is literal", "args a2 b2c", 0, "[a2]\n[b2c]\n"},
		{"tilde", "args ~ ~/x a~ '~'", 0, "[/home/u]\n[/home/u/x]\n[a~]\n[~]\n"},

		{"status of last command", "echo a\nfail", 3, "a\n"},
		{"dollar question", "fail; echo $?; echo $?", 0, "3\n0\n"},
		{"and runs on success", "echo a && echo b", 0, "a\nb\n"},
		{"and skips on failure", "fail && echo b", 3, ""},
		{"or runs on failure", "fail || echo b", 0, "b\n"},
		{"or skips on success", "echo a || echo b", 0, "a\n"},
		{"chain", "fail && echo x || echo y", 0, "y\n"},
		{"without errexit a failure continues", "fail\necho after", 0, "after\n"},

		{"pipe", "echo hello | upper", 0, "HELLO\n"},
		{"three stages", "echo hello | upper | cat", 0, "HELLO\n"},
		{"pipeline status is the last stage", "fail | echo ok", 0, "ok\n"},
		{"reader closing ends the writer", "spam | first", 0, "y\n"},

		{"exit stops the script", "echo a\nexit 4\necho b", 4, "a\n"},
		{"exit keeps the last status", "fail\nexit", 3, ""},
		{"exit in a pipeline is a subshell", "exit 9 | echo a\necho b", 0, "a\nb\n"},
		{"unknown command", "nosuch", 127, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, out, errb := run(t, shell.Options{}, tt.script)
			if status != tt.wantStatus || out != tt.wantOut {
				t.Fatalf("status=%d out=%q (stderr %q); want status=%d out=%q",
					status, out, errb, tt.wantStatus, tt.wantOut)
			}
		})
	}
}

func TestErrExit(t *testing.T) {
	status, out, _ := run(t, shell.Options{ErrExit: true}, "echo a\nfail\necho b")
	if status != 3 || out != "a\n" {
		t.Fatalf("status=%d out=%q, want 3 and only the first line", status, out)
	}

	// A status tested by && or || is a condition, not a failure.
	status, out, _ = run(t, shell.Options{ErrExit: true}, "fail || echo recovered\necho b")
	if status != 0 || out != "recovered\nb\n" {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestSyntaxErrors(t *testing.T) {
	for _, script := range []string{
		"echo 'unterminated", `echo "unterminated`, "| echo", "echo |", "echo a &&", "echo >", "> out", "echo a | | b",
		// No background jobs, and no file descriptors beyond 1 and 2: refused,
		// never read as an argument "&" or a file named "&3".
		"echo a & echo b", "echo a&", "echo a >&3", "echo a 2>&3",
	} {
		t.Run(script, func(t *testing.T) {
			status, out, errb := run(t, shell.Options{}, script+"\necho after")
			if status != 2 || !strings.HasPrefix(errb, "gnux: syntax error: ") {
				t.Errorf("status=%d stderr=%q, want status 2 and a syntax error", status, errb)
			}
			if out != "" {
				t.Errorf("a script must stop at a syntax error, got output %q", out)
			}
		})
	}

	t.Run("interactive carries on", func(t *testing.T) {
		status, out, errb := run(t, shell.Options{Interactive: true}, "echo 'oops\necho after\n")
		if status != 0 || out != "after\n" {
			t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
		}
		if strings.Count(errb, shell.Prompt) != 3 {
			t.Fatalf("want a prompt before each read, stderr=%q", errb)
		}
	})
}

func TestDiagnostics(t *testing.T) {
	_, _, errb := run(t, shell.Options{}, "nosuch arg")
	if want := "gnux: nosuch: command not found\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestSetAndHelp(t *testing.T) {
	_, out, _ := run(t, shell.Options{}, "args $TARGET", func(sh *shell.Shell) { sh.Set("TARGET", "dist") })
	if out != "[dist]\n" {
		t.Fatalf("out = %q", out)
	}
	_, out, _ = run(t, shell.Options{}, "help")
	if !strings.Contains(out, "Builtins: cd") {
		t.Fatalf("help output = %q", out)
	}
}

func TestRedirection(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.txt")
	setDir := func(sh *shell.Shell) { sh.Set("D", dir) }

	status, stdout, errb := run(t, shell.Options{}, "echo one > $D/out.txt\necho two >> $D/out.txt\nupper < $D/out.txt", setDir)
	if status != 0 || stdout != "ONE\nTWO\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout, errb)
	}
	if b, _ := os.ReadFile(out); string(b) != "one\ntwo\n" {
		t.Fatalf("file = %q", b)
	}

	// > truncates.
	if _, _, _ = run(t, shell.Options{}, "echo fresh > $D/out.txt", setDir); true {
		if b, _ := os.ReadFile(out); string(b) != "fresh\n" {
			t.Fatalf("file after truncate = %q", b)
		}
	}

	// A pipeline's redirections are opened before any stage runs, as in sh.
	status, stdout, _ = run(t, shell.Options{}, "exists $D/new.txt | cat > $D/new.txt\ncat < $D/new.txt", setDir)
	if status != 0 || stdout != "yes\n" {
		t.Fatalf("status=%d stdout=%q, want the earlier stage to see the file", status, stdout)
	}

	// 2> captures standard error, leaving standard output alone.
	status, stdout, errb = run(t, shell.Options{}, "warn oops 2> $D/err.txt\nwarn more 2>> $D/err.txt", setDir)
	if b, _ := os.ReadFile(filepath.Join(dir, "err.txt")); status != 0 || stdout != "out\nout\n" || errb != "" || string(b) != "oops\nmore\n" {
		t.Fatalf("status=%d stdout=%q stderr=%q file=%q", status, stdout, errb, b)
	}

	// /dev/null is understood by the shell itself, on every platform.
	status, stdout, errb = run(t, shell.Options{}, "warn gone 2> /dev/null > /dev/null\ncat < /dev/null")
	if status != 0 || stdout != "" || errb != "" {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout, errb)
	}

	// 1> and 1>> are > and >>: the 1 is not an argument.
	status, stdout, errb = run(t, shell.Options{}, "echo a 1> $D/one.txt\necho b 1>> $D/one.txt\ncat < $D/one.txt", setDir)
	if status != 0 || stdout != "a\nb\n" {
		t.Fatalf("1> and 1>>: status=%d stdout=%q stderr=%q", status, stdout, errb)
	}

	// 2>&1 and >&2 point a stream at wherever the other goes at that point,
	// so the order of the redirections matters, as in sh.
	for _, tt := range []struct{ script, out, err, file string }{
		{"warn x > $D/both.txt 2>&1", "", "", "x\nout\n"},
		{"warn x 2>&1 > $D/both.txt", "x\n", "", "out\n"},
		{"warn x 2>&1 | upper", "X\nOUT\n", "", ""},
		{"echo y >&2", "", "y\n", ""},
		{"echo y 1>&2 2> /dev/null", "", "y\n", ""},
	} {
		_ = os.Remove(filepath.Join(dir, "both.txt"))
		status, stdout, errb = run(t, shell.Options{}, tt.script, setDir)
		b, _ := os.ReadFile(filepath.Join(dir, "both.txt"))
		if status != 0 || stdout != tt.out || errb != tt.err || string(b) != tt.file {
			t.Errorf("%s: status=%d stdout=%q stderr=%q file=%q; want stdout=%q stderr=%q file=%q",
				tt.script, status, stdout, errb, b, tt.out, tt.err, tt.file)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "&") {
				t.Errorf("a duplication created the file %q", e.Name())
			}
		}
	}

	status, _, errb = run(t, shell.Options{}, "cat < $D/missing", setDir)
	if status != 1 || !strings.HasSuffix(errb, "missing: No such file or directory\n") {
		t.Fatalf("status=%d stderr=%q", status, errb)
	}
}

func TestGlob(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.go", "a.go", "c.txt", "we ird.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setDir := func(sh *shell.Shell) { sh.Set("D", dir) }
	j := func(names ...string) string {
		var b strings.Builder
		for _, n := range names {
			fmt.Fprintf(&b, "[%s]\n", filepath.Join(dir, n))
		}
		return b.String()
	}

	tests := []struct{ name, script, want string }{
		{"star, sorted", "args $D/*.go", j("a.go", "b.go", "we ird.go")},
		{"question mark", "args $D/?.go", j("a.go", "b.go")},
		{"class", "args $D/[ab].go", j("a.go", "b.go")},
		{"quoted pattern is literal", `args "$D/*.go"`, "[" + dir + "/*.go]\n"},
		{"no match passes through", "args $D/*.rs", "[" + dir + "/*.rs]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, out, _ := run(t, shell.Options{}, tt.script, setDir); out != tt.want {
				t.Fatalf("out = %q, want %q", out, tt.want)
			}
		})
	}

	// Glob characters that arrive through a variable are not a pattern.
	_, out, _ := run(t, shell.Options{}, "args $P", func(sh *shell.Shell) { sh.Set("P", filepath.Join(dir, "*.go")) })
	if want := "[" + filepath.Join(dir, "*.go") + "]\n"; out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
}

func TestCd(t *testing.T) {
	start, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// $PWD follows cd.
	_, pwd, _ := run(t, shell.Options{}, "cd $D\necho $PWD", func(sh *shell.Shell) { sh.Set("D", dir) })
	if got, _ := filepath.EvalSymlinks(strings.TrimSpace(pwd)); got != dir {
		t.Fatalf("$PWD after cd = %q, want %q", got, dir)
	}

	status, out, errb := run(t, shell.Options{}, "cd $D\npwd", func(sh *shell.Shell) { sh.Set("D", dir) })
	if status != 0 {
		t.Fatalf("status=%d stderr=%q", status, errb)
	}
	if got, _ := filepath.EvalSymlinks(strings.TrimSpace(out)); got != dir {
		t.Fatalf("pwd after cd = %q, want %q", got, dir)
	}

	// Run must hand the process back where it found it.
	if after, _ := os.Getwd(); after != start {
		t.Fatalf("working directory leaked: %q, want %q", after, start)
	}

	status, _, errb = run(t, shell.Options{}, "cd /no/such/dir")
	if status != 1 || !strings.Contains(errb, "cd: /no/such/dir: No such file or directory") {
		t.Fatalf("status=%d stderr=%q", status, errb)
	}
}

func TestEvalKeepsState(t *testing.T) {
	var out bytes.Buffer
	sh := shell.New(shell.Options{Interactive: true}, fakeExec, core.Streams{In: strings.NewReader(""), Out: &out, Err: io.Discard})

	if status := sh.Eval("X=kept"); status != 0 {
		t.Fatalf("assignment status = %d", status)
	}
	if status := sh.Eval("fail"); status != 3 {
		t.Fatalf("fail status = %d, want 3", status)
	}
	if status := sh.Eval("exit 7"); status != 7 {
		t.Fatalf("exit status = %d, want 7", status)
	}
	// The session survives an exit, and still has its variable and $?.
	sh.Eval("echo $X $?")
	if got := out.String(); got != "kept 7\n" {
		t.Fatalf("out = %q", got)
	}
}

// TestGlobInjection checks that globbing goes through Shell.Glob, so a host
// with its own filesystem (or a test) decides what a pattern matches.
func TestGlobInjection(t *testing.T) {
	var seen []string
	glob := func(pattern string) ([]string, error) {
		seen = append(seen, pattern)
		if pattern == "*.md" {
			return []string{"b.md", "a.md"}, nil
		}
		return nil, nil
	}
	_, out, _ := run(t, shell.Options{}, `echo *.md "*.md" none*`, func(sh *shell.Shell) { sh.Glob = glob })
	if want := "a.md b.md *.md none*\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if want := []string{"*.md", "none*"}; strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("Glob called with %q, want %q (a quoted pattern is not globbed)", seen, want)
	}
}

// A stage that panics closes its pipes, so its neighbors end too, and the
// panic is raised on the goroutine that called Run.
func TestPipelineStagePanicReachesTheCaller(t *testing.T) {
	exec := func(name string, args []string, s core.Streams) (int, bool) {
		if name == "boom" {
			panic("stage failed badly")
		}
		return fakeExec(name, args, s)
	}
	sh := shell.New(shell.Options{}, exec, core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard})

	var got *core.Panic
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { got, _ = recover().(*core.Panic) }()
		_ = sh.Run(strings.NewReader("echo a | boom | cat\n"))
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the pipeline did not end after a stage panicked")
	}
	if got == nil || got.Value != "stage failed badly" {
		t.Fatalf("recovered %+v, want the stage's panic", got)
	}
}

func TestRead(t *testing.T) {
	tests := []struct {
		name, script, stdin, want string
		status                    int
	}{
		{"one line", "read x; args $x", "hello world\n", "[hello world]\n", 0},
		{"fields", "read a b; args $a $b", "  one two  three  \n", "[one]\n[two  three]\n", 0},
		{"more names than fields", "read a b c; args $a $b $c", "one\n", "[one]\n[]\n[]\n", 0},
		{"REPLY", "read; args $REPLY", "  kept as is?  \n", "[kept as is?]\n", 0},
		{"each line in turn", "read a; read b; args $b $a", "1\n2\n", "[2]\n[1]\n", 0},
		{"while read", "while read l; do args $l; done", "a b\nc\n", "[a b]\n[c]\n", 0},
		{"last line without a newline", "while read l; do args $l; done; args $l", "a\nb", "[a]\n[b]\n", 0},
		{"partial line at the end", "read l; args $? $l", "tail", "[1]\n[tail]\n", 0},
		{"end of input", "read l; args $?", "", "[1]\n", 0},
		{"backslash escapes", `read a b; args $a $b`, "x\\ y z\n", "[x y]\n[z]\n", 0},
		{"backslash newline joins", "read a; args $a", "one \\\ntwo\n", "[one two]\n", 0},
		{"-r keeps backslashes", "read -r a; args $a", "C:\\tools\\bin\n", "[C:\\tools\\bin]\n", 0},
		{"IFS", "IFS=:; read a b c; args $a $b $c", "x::z\n", "[x]\n[]\n[z]\n", 0},
		{"IFS without blanks keeps them", "IFS=,; read a b; args $a $b", " a , b \n", "[ a ]\n[ b ]\n", 0},
		{"empty IFS", "IFS=; read a; args $a", "  as is  \n", "[  as is  ]\n", 0},
		{"UTF-8", "read a b; args $b", "é ü\n", "[ü]\n", 0},
		{"from a redirected loop", "while read l; do args $l; done < $F", "", "[one]\n[two]\n", 0},
		{"in a pipeline stage nothing is set", "x=old; echo new | read x; args $x", "", "[old]\n", 0},
		{"a piped loop", "echo a | while read l; do args got-$l; done", "", "[got-a]\n", 0},
		{"bad name", "read 1x", "", "", 2},
		{"bad option", "read -z x", "", "", 2},
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(file, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			sh := shell.New(shell.Options{}, fakeExec, core.Streams{In: strings.NewReader(tt.stdin), Out: &out, Err: &errb})
			sh.Getenv = func(string) string { return "" }
			sh.Set("F", file)
			status := 0
			if err := sh.RunString(tt.script); err != nil {
				var exit *core.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				status = exit.Code
			}
			if status != tt.status || out.String() != tt.want {
				t.Fatalf("status=%d out=%q stderr=%q; want %d %q", status, out.String(), errb.String(), tt.status, tt.want)
			}
		})
	}

	t.Run("reads no further than its line", func(t *testing.T) {
		var out bytes.Buffer
		sh := shell.New(shell.Options{}, fakeExec, core.Streams{In: strings.NewReader("first\nrest 1\nrest 2\n"), Out: &out, Err: io.Discard})
		if err := sh.RunString("read x; cat; args $x"); err != nil {
			t.Fatal(err)
		}
		if out.String() != "rest 1\nrest 2\n[first]\n" {
			t.Fatalf("out = %q", out.String())
		}
	})
}
