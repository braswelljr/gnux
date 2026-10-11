package env_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/env"
)

// helperVar selects what TestHelperProcess does when the test binary is
// re-executed as the command under test. Using the test binary itself keeps
// the command-running tests independent of any shell or system utility.
const helperVar = "GNUX_ENVX_HELPER"

// TestHelperProcess is not a test: it is the command envx runs in the tests
// below. It does nothing unless helperVar is set.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv(helperVar)
	if mode == "" {
		return
	}
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}

	switch mode {
	case "args":
		fmt.Println(strings.Join(args, "|"))
	case "env":
		for _, name := range args {
			if v, ok := os.LookupEnv(name); ok {
				fmt.Printf("%s=%s\n", name, v)
			} else {
				fmt.Printf("%s is unset\n", name)
			}
		}
	case "cwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(wd)
	case "exit":
		fmt.Fprintln(os.Stderr, "helper failing")
		code, _ := strconv.Atoi(args[0])
		os.Exit(code)
	case "stdin":
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			os.Exit(2)
		}
	case "argv0":
		fmt.Println(os.Args[0])
	}
	os.Exit(0)
}

// quickExit stops a race-enabled helper from lingering for a second when it
// exits, which is the race detector's default. It means nothing otherwise.
const quickExit = "GORACE=atexit_sleep_ms=0"

// helper returns envx operands that run the test binary in the given helper
// mode: the assignments selecting the mode, then the command line.
func helper(mode string, args ...string) []string {
	return append([]string{helperVar + "=" + mode, quickExit, os.Args[0], "-test.run=^TestHelperProcess$", "--"}, args...)
}

type result struct {
	stdout, stderr string
	err            error
}

// run executes the engine against buffers with a fixed inherited environment.
func run(opts env.Options, environ []string, stdin string, args ...string) result {
	var out, errb bytes.Buffer
	e := env.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	e.Environ = func() []string { return environ }
	err := e.Run(args)
	return result{out.String(), errb.String(), err}
}

// wantExit fails the test unless err is a silent exit with the given status.
func wantExit(t *testing.T, err error, code int) {
	t.Helper()
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != code {
		t.Fatalf("err = %v, want exit status %d", err, code)
	}
}

func TestPrint(t *testing.T) {
	inherited := []string{"HOME=/home/u", "PATH=/bin", "EMPTY=", "LANG=C"}
	tests := []struct {
		name string
		opts env.Options
		args []string
		want string
	}{
		{"inherited order is kept", env.Options{}, nil, "HOME=/home/u\nPATH=/bin\nEMPTY=\nLANG=C\n"},
		{"new variables are appended in order", env.Options{}, []string{"B=2", "A=1"},
			"HOME=/home/u\nPATH=/bin\nEMPTY=\nLANG=C\nB=2\nA=1\n"},
		{"an existing variable is replaced in place", env.Options{}, []string{"PATH=/opt/bin", "Z=1"},
			"HOME=/home/u\nPATH=/opt/bin\nEMPTY=\nLANG=C\nZ=1\n"},
		{"the last assignment of a name wins", env.Options{}, []string{"A=1", "B=2", "A=3"},
			"HOME=/home/u\nPATH=/bin\nEMPTY=\nLANG=C\nA=3\nB=2\n"},
		{"empty value", env.Options{}, []string{"LANG="}, "HOME=/home/u\nPATH=/bin\nEMPTY=\nLANG=\n"},
		{"value containing an equals sign", env.Options{IgnoreEnvironment: true}, []string{"A=b=c"}, "A=b=c\n"},
		{"ignore environment", env.Options{IgnoreEnvironment: true}, nil, ""},
		{"ignore environment with assignments", env.Options{IgnoreEnvironment: true}, []string{"A=1", "B=2"}, "A=1\nB=2\n"},
		{"lone dash means ignore environment", env.Options{}, []string{"-", "A=1"}, "A=1\n"},
		{"unset", env.Options{Unset: []string{"PATH", "LANG"}}, nil, "HOME=/home/u\nEMPTY=\n"},
		{"unset a missing variable", env.Options{Unset: []string{"NOPE"}}, nil, "HOME=/home/u\nPATH=/bin\nEMPTY=\nLANG=C\n"},
		{"unset then set again appends", env.Options{Unset: []string{"HOME"}}, []string{"HOME=/x"},
			"PATH=/bin\nEMPTY=\nLANG=C\nHOME=/x\n"},
		{"null terminated", env.Options{Null: true, IgnoreEnvironment: true}, []string{"A=1", "B=two words"}, "A=1\x00B=two words\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(tt.opts, inherited, "", tt.args...)
			if r.err != nil {
				t.Fatalf("Run: %v", r.err)
			}
			if r.stdout != tt.want {
				t.Errorf("stdout = %q, want %q", r.stdout, tt.want)
			}
			if r.stderr != "" {
				t.Errorf("stderr = %q, want empty", r.stderr)
			}
		})
	}
	if want := []string{"HOME=/home/u", "PATH=/bin", "EMPTY=", "LANG=C"}; strings.Join(inherited, "\n") != strings.Join(want, "\n") {
		t.Errorf("the inherited environment was modified: %q", inherited)
	}
}

func TestDuplicateInheritedEntriesCollapseOnAssignment(t *testing.T) {
	r := run(env.Options{}, []string{"A=1", "B=2", "A=3"}, "", "A=9")
	if want := "A=9\nB=2\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v; want %q", r.stdout, r.err, want)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts env.Options
		args []string
		want string
	}{
		{"null with a command", env.Options{Null: true}, []string{"cmd"}, "cannot specify --null (-0) with command"},
		{"chdir without a command", env.Options{Chdir: "/"}, []string{"A=1"}, "must specify command with --chdir (-C)"},
		{"argv0 without a command", env.Options{Argv0: "x"}, nil, "must specify command with --argv0 (-a)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(tt.opts, []string{"A=0"}, "", tt.args...)
			var usage *core.UsageError
			if !errors.As(r.err, &usage) || usage.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", r.err, tt.want)
			}
			if r.stdout != "" || r.stderr != "" {
				t.Errorf("stdout = %q, stderr = %q, want both empty", r.stdout, r.stderr)
			}
		})
	}
}

func TestInvalidUnset(t *testing.T) {
	for name, want := range map[string]string{
		"":    "envx: cannot unset '': Invalid argument\n",
		"A=B": "envx: cannot unset 'A=B': Invalid argument\n",
	} {
		r := run(env.Options{Unset: []string{name}}, []string{"A=0"}, "")
		wantExit(t, r.err, env.ExitCanceled)
		if r.stderr != want || r.stdout != "" {
			t.Errorf("unset %q: stdout = %q, stderr = %q, want stderr %q", name, r.stdout, r.stderr, want)
		}
	}
}

func TestCommandSeesTheModifiedEnvironment(t *testing.T) {
	inherited := []string{"KEEP=kept", "DROP=dropped", "CHANGE=old"}
	tests := []struct {
		name string
		opts env.Options
		pre  []string // operands ahead of the helper command
		want string
	}{
		{"inherited and assigned", env.Options{}, []string{"CHANGE=new", "ADD=added"},
			"KEEP=kept\nDROP=dropped\nCHANGE=new\nADD=added\n"},
		{"unset", env.Options{Unset: []string{"DROP"}}, nil,
			"KEEP=kept\nDROP is unset\nCHANGE=old\nADD is unset\n"},
		{"ignore environment", env.Options{IgnoreEnvironment: true}, []string{"ADD=only"},
			"KEEP is unset\nDROP is unset\nCHANGE is unset\nADD=only\n"},
		{"lone dash", env.Options{}, []string{"-", "ADD=only"},
			"KEEP is unset\nDROP is unset\nCHANGE is unset\nADD=only\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(tt.pre, helper("env", "KEEP", "DROP", "CHANGE", "ADD")...)
			r := run(tt.opts, inherited, "", args...)
			if r.err != nil {
				t.Fatalf("Run: %v (stderr %q)", r.err, r.stderr)
			}
			if r.stdout != tt.want {
				t.Errorf("stdout = %q, want %q", r.stdout, tt.want)
			}
			if r.stderr != "" {
				t.Errorf("stderr = %q, want empty", r.stderr)
			}
		})
	}
}

func TestCommandArguments(t *testing.T) {
	// Operands after the command belong to it, assignments and options included.
	r := run(env.Options{}, nil, "", helper("args", "a b", "X=1", "-i", "")...)
	if want := "a b|X=1|-i|\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v (stderr %q); want %q", r.stdout, r.err, r.stderr, want)
	}
}

func TestCommandInheritsStreams(t *testing.T) {
	r := run(env.Options{}, nil, "from stdin\n", helper("stdin")...)
	if want := "from stdin\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v (stderr %q); want %q", r.stdout, r.err, r.stderr, want)
	}
}

func TestCommandExitStatusIsPropagated(t *testing.T) {
	for _, code := range []int{1, 3, 42} {
		r := run(env.Options{}, nil, "", helper("exit", strconv.Itoa(code))...)
		wantExit(t, r.err, code)
		// The command's own stderr passes through; envx adds nothing to it.
		if want := "helper failing\n"; r.stderr != want || r.stdout != "" {
			t.Errorf("exit %d: stdout = %q, stderr = %q, want stderr %q", code, r.stdout, r.stderr, want)
		}
	}
}

func TestCommandIsFoundOnTheModifiedPath(t *testing.T) {
	dir, file := filepath.Split(os.Args[0])
	inherited := []string{"PATH=" + t.TempDir()}
	command := []string{helperVar + "=args", quickExit, file, "-test.run=^TestHelperProcess$", "--", "found"}

	// The inherited PATH does not hold the test binary...
	r := run(env.Options{}, inherited, "", command...)
	wantExit(t, r.err, env.ExitNotFound)
	if want := "envx: '" + file + "': No such file or directory\n"; r.stderr != want || r.stdout != "" {
		t.Fatalf("stdout = %q, stderr = %q, want stderr %q", r.stdout, r.stderr, want)
	}

	// ...but the PATH assigned on the command line does.
	r = run(env.Options{}, inherited, "", append([]string{"PATH=" + dir}, command...)...)
	if want := "found\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v (stderr %q); want %q", r.stdout, r.err, r.stderr, want)
	}
}

func TestCommandNotFound(t *testing.T) {
	empty := t.TempDir()
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"bare name", "gnux-no-such-command", "envx: 'gnux-no-such-command': No such file or directory\n"},
		{"path", filepath.Join(empty, "nope"), "envx: '" + filepath.Join(empty, "nope") + "': No such file or directory\n"},
		{"name with spaces hints at -S", "perl -w",
			"envx: 'perl -w': No such file or directory\nenvx: use -[v]S to pass options in shebang lines\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(env.Options{}, []string{"PATH=" + empty}, "", tt.command, "arg")
			wantExit(t, r.err, env.ExitNotFound)
			if r.stderr != tt.want || r.stdout != "" {
				t.Errorf("stdout = %q, stderr = %q, want stderr %q", r.stdout, r.stderr, tt.want)
			}
		})
	}
}

func TestChdir(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := run(env.Options{Chdir: dir}, nil, "", helper("cwd")...)
	if r.err != nil {
		t.Fatalf("Run: %v (stderr %q)", r.err, r.stderr)
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(r.stdout))
	if err != nil || got != dir {
		t.Fatalf("command ran in %q (%v), want %q", r.stdout, err, dir)
	}
}

func TestChdirFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	tests := []struct {
		name, chdir, want string
	}{
		{"missing directory", missing, "envx: cannot change directory to '" + missing + "': No such file or directory\n"},
		{"not a directory", file, "envx: cannot change directory to '" + file + "': Not a directory\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := run(env.Options{Chdir: tt.chdir}, nil, "", helper("cwd")...)
			wantExit(t, r.err, env.ExitCanceled)
			if r.stderr != tt.want || r.stdout != "" {
				t.Errorf("stdout = %q, stderr = %q, want stderr %q", r.stdout, r.stderr, tt.want)
			}
		})
	}
}

func TestArgv0(t *testing.T) {
	r := run(env.Options{Argv0: "renamed"}, nil, "", helper("argv0")...)
	if want := "renamed\n"; r.err != nil || r.stdout != want {
		t.Fatalf("stdout = %q, err = %v (stderr %q); want %q", r.stdout, r.err, r.stderr, want)
	}
}

func TestDebug(t *testing.T) {
	t.Run("printing", func(t *testing.T) {
		r := run(env.Options{Debug: true, Unset: []string{"A"}}, []string{"A=1"}, "", "B=2")
		want := "envx: unset:    A\nenvx: setenv:   B=2\n"
		if r.err != nil || r.stdout != "B=2\n" || r.stderr != want {
			t.Fatalf("stdout = %q, stderr = %q, err = %v; want stderr %q", r.stdout, r.stderr, r.err, want)
		}
	})
	t.Run("running a command", func(t *testing.T) {
		r := run(env.Options{Debug: true, IgnoreEnvironment: true}, nil, "", helper("args", "x")...)
		want := "envx: cleaning environ\n" +
			"envx: setenv:   " + helperVar + "=args\n" +
			"envx: setenv:   " + quickExit + "\n" +
			"envx: executing: " + os.Args[0] + "\n" +
			"envx:    arg[0]= '" + os.Args[0] + "'\n" +
			"envx:    arg[1]= '-test.run=^TestHelperProcess$'\n" +
			"envx:    arg[2]= '--'\n" +
			"envx:    arg[3]= 'x'\n"
		if r.err != nil || r.stdout != "x\n" || r.stderr != want {
			t.Fatalf("stdout = %q, stderr = %q, err = %v; want stderr %q", r.stdout, r.stderr, r.err, want)
		}
	})
}

func TestSplitString(t *testing.T) {
	vars := map[string]string{"HOME": "/home/u", "EMPTY": "", "SP": "a b"}
	lookup := func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
	tests := []struct {
		name    string
		in      string
		want    []string
		wantErr string
	}{
		{"empty", "", nil, ""},
		{"only whitespace", " \t\n ", nil, ""},
		{"words", "perl -w  -T", []string{"perl", "-w", "-T"}, ""},
		{"surrounding whitespace", "\t a  b\n", []string{"a", "b"}, ""},
		{"single quotes", `a 'b c' d`, []string{"a", "b c", "d"}, ""},
		{"double quotes", `a "b c" d`, []string{"a", "b c", "d"}, ""},
		{"quotes join with adjacent text", `a'b c'"d e"f`, []string{"ab cd ef"}, ""},
		{"empty quoted argument", `a '' "" b`, []string{"a", "", "", "b"}, ""},
		{"double quote inside single quotes", `'a"b'`, []string{`a"b`}, ""},
		{"single quote inside double quotes", `"a'b"`, []string{"a'b"}, ""},
		{"control escapes", `a\tb\nc\fd\re\vf`, []string{"a\tb\nc\fd\re\vf"}, ""},
		{"literal escapes", `\"\#\$\'\\`, []string{`"#$'\`}, ""},
		{"escapes inside double quotes", `"a\tb\"c\$d"`, []string{"a\tb\"c$d"}, ""},
		{"single quotes keep backslashes", `'a\tb\$c'`, []string{`a\tb\$c`}, ""},
		{"single quotes allow escaped quote and backslash", `'a\'b\\c'`, []string{`a'b\c`}, ""},
		{"underscore escape separates arguments", `a\_b`, []string{"a", "b"}, ""},
		{"underscore escape is a space in double quotes", `"a\_b"`, []string{"a b"}, ""},
		{"\\c ends the string", `a b\c d e`, []string{"a", "b"}, ""},
		{"comment at the start", `# nothing`, nil, ""},
		{"comment after arguments", `a b #c d`, []string{"a", "b"}, ""},
		{"hash inside an argument", `a#b c`, []string{"a#b", "c"}, ""},
		{"hash inside quotes", `'#a' "#b"`, []string{"#a", "#b"}, ""},
		{"variable", `${HOME}/bin x`, []string{"/home/u/bin", "x"}, ""},
		{"variable value is not split again", `${SP}`, []string{"a b"}, ""},
		{"variable in double quotes", `"${HOME} x"`, []string{"/home/u x"}, ""},
		{"variable in single quotes is literal", `'${HOME}'`, []string{"${HOME}"}, ""},
		{"empty variable is an empty argument", `a ${EMPTY} b`, []string{"a", "", "b"}, ""},
		{"unset variable vanishes", `a ${NOPE} b`, []string{"a", "b"}, ""},
		{"unset variable inside an argument", `a${NOPE}b`, []string{"ab"}, ""},

		{"unterminated single quote", `a 'b`, nil, "no terminating quote in -S string"},
		{"unterminated double quote", `a "b`, nil, "no terminating quote in -S string"},
		{"trailing backslash", `a\`, nil, "invalid backslash at end of string in -S"},
		{"unknown escape", `a\qb`, nil, `invalid sequence '\q' in -S`},
		{"\\c in double quotes", `"a\c"`, nil, `'\c' must not appear in double-quoted -S string`},
		{"bare dollar", `a $HOME`, nil, "only ${VARNAME} expansion is supported, error at: $HOME"},
		{"unclosed variable", `${HOME`, nil, "only ${VARNAME} expansion is supported, error at: ${HOME"},
		{"empty variable name", `${}`, nil, "only ${VARNAME} expansion is supported, error at: ${}"},
		{"variable name starting with a digit", `${1A}`, nil, "only ${VARNAME} expansion is supported, error at: ${1A}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.SplitString(tt.in, lookup)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("SplitString(%q) error = %v, want %q", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitString(%q): %v", tt.in, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SplitString(%q) = %q, want %q", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("SplitString(%q) = %q, want %q", tt.in, got, tt.want)
				}
			}
		})
	}
}
