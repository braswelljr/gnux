package shell_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shell"
)

func TestControlFlow(t *testing.T) {
	tests := []struct {
		name, script string
		wantStatus   int
		wantOut      string
	}{
		{"if true", "if true; then echo yes; fi", 0, "yes\n"},
		{"if false has status 0", "if fail; then echo yes; fi", 0, ""},
		{"else", "if fail; then echo yes; else echo no; fi", 0, "no\n"},
		{"elif", "if fail; then echo a; elif true; then echo b; else echo c; fi", 0, "b\n"},
		{"status is the body's", "if true; then fail; fi", 3, ""},
		{"condition is a list", "if fail; true; then echo last-wins; fi", 0, "last-wins\n"},
		{"across lines", "if true\nthen\n  echo a\n  echo b\nfi\necho after", 0, "a\nb\nafter\n"},
		{"nested", "if true; then if fail; then echo x; else echo inner; fi; fi", 0, "inner\n"},
		{"empty body", "if true; then\nfi\necho $?", 0, "0\n"},
		{"keywords are ordinary arguments", "echo if then fi do done", 0, "if then fi do done\n"},
		{"bang", "! fail; echo $?; ! true; echo $?", 0, "0\n1\n"},
		{"bang in a condition", "if ! fail; then echo ok; fi", 0, "ok\n"},
		{"bracket is test", `if [ a = a ]; then echo same; fi; [ a = b ] || echo differ`, 0, "same\ndiffer\n"},

		{"for", "for x in a b c; do echo $x; done", 0, "a\nb\nc\n"},
		{"for across lines", "for x in a b\ndo\n  echo $x\ndone", 0, "a\nb\n"},
		{"for keeps the variable", "for x in a b; do true; done; echo $x", 0, "b\n"},
		{"for over nothing", "for x in; do echo $x; done; echo $?", 0, "0\n"},
		{"for items are not re-split", "V='a b'; for x in $V c; do args $x; done", 0, "[a b]\n[c]\n"},
		{"for over the arguments", "set -- p 'q r'; for x; do args $x; done", 0, "[p]\n[q r]\n"},
		{"break", "for x in a b c; do if eq $x b; then break; fi; echo $x; done", 0, "a\n"},
		{"continue", "for x in a b c; do if eq $x b; then continue; fi; echo $x; done", 0, "a\nc\n"},
		{"break 2", "for x in 1 2; do for y in a b; do echo $x$y; break 2; done; done", 0, "1a\n"},
		{"continue 2", "for x in 1 2; do for y in a b; do echo $x$y; continue 2; done; done", 0, "1a\n2a\n"},
		{"break outside a loop", "break", 1, ""},

		{"while", "X=a; while ! eq $X aaa; do X=${X}a; echo $X; done", 0, "aa\naaa\n"},
		{"while that never runs", "while fail; do echo x; done; echo $?", 0, "0\n"},
		{"while with break", "while true; do echo once; break; done", 0, "once\n"},

		{"case", "case b in a) echo A ;; b) echo B ;; esac", 0, "B\n"},
		{"case alternatives", "case b in a|b) echo AB ;; esac", 0, "AB\n"},
		{"case glob", "case src/main.go in *.go) echo go ;; *) echo other ;; esac", 0, "go\n"},
		{"case default", "case z in a) echo A ;; *) echo other ;; esac", 0, "other\n"},
		{"case class", "case x7 in [a-w]*) echo low ;; x[!0-5]) echo hit ;; esac", 0, "hit\n"},
		{"case no match", "case z in a) echo A ;; esac; echo $?", 0, "0\n"},
		{"case quoted pattern is literal", `case ab in "*") echo star ;; a?) echo two ;; esac`, 0, "two\n"},
		{"case pattern from a variable is literal", "P='*'; case ab in $P) echo star ;; *) echo any ;; esac", 0, "any\n"},
		{"case across lines", "case $X in\n  (a)\n    echo A\n    ;;\n  *)\n    echo other\nesac", 0, "other\n"},
		{"stray double semicolon", "echo a ;; echo b", 0, "a\nb\n"},

		{"loop into a pipe", "for x in b a; do echo $x; done | upper", 0, "B\nA\n"},
		{"a piped loop is a subshell", "for x in a; do Y=set; done | cat; args ${Y:-unset}", 0, "[unset]\n"},
		{"pipe into a condition", "if echo hi | upper; then echo ok; fi", 0, "HI\nok\n"},
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

func TestSubstitution(t *testing.T) {
	tests := []struct {
		name, script string
		wantStatus   int
		wantOut      string
	}{
		{"in an argument", "args $(echo hi)", 0, "[hi]\n"},
		{"trailing newlines go", "args \"$(lines a b)\"", 0, "[a\nb]\n"},
		{"unquoted splits into lines, not words", "args $(lines 'a b' c)", 0, "[a b]\n[c]\n"},
		{"empty output is no argument", "args a $(true) b", 0, "[a]\n[b]\n"},
		{"quoted empty output is one", `args a "$(true)" b`, 0, "[a]\n[]\n[b]\n"},
		{"joins its neighbors", "args pre$(lines a b)post", 0, "[prea]\n[bpost]\n"},
		{"output is not a glob", "args $(echo '*')", 0, "[*]\n"},
		{"assignment takes it whole", "X=$(lines a b); args \"$X\"", 0, "[a\nb]\n"},
		{"assignment status", "X=$(fail); echo $?", 0, "3\n"},
		{"nested", `args "$(echo a $(echo b))"`, 0, "[a b]\n"},
		{"pipeline inside", "args $(echo hi | upper)", 0, "[HI]\n"},
		{"across lines", "args $(\n  echo a\n  echo b\n)", 0, "[a]\n[b]\n"},
		{"is a subshell", "X=1; echo $(X=2; echo $X) $X", 0, "2 1\n"},
		{"exit inside ends only it", "echo $(echo a; exit 3; echo b); echo after", 0, "a\nafter\n"},
		{"for over output", "for f in $(lines 'my file' other); do args $f; done", 0, "[my file]\n[other]\n"},
		{"case inside", "args $(case a in a) echo A ;; esac)", 0, "[A]\n"},
		{"single quotes keep it literal", "args '$(echo hi)'", 0, "[$(echo hi)]\n"},
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

	// cd inside $(...) is real while it runs and undone afterwards.
	t.Run("cd does not leak", func(t *testing.T) {
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		before, _ := os.Getwd()
		_, out, errb := run(t, shell.Options{}, "echo $(cd $D && pwd)\npwd",
			func(sh *shell.Shell) { sh.Set("D", dir) })
		got := strings.Split(strings.TrimSpace(out), "\n")
		if len(got) != 2 {
			t.Fatalf("out=%q stderr=%q", out, errb)
		}
		if inside, _ := filepath.EvalSymlinks(got[0]); inside != dir {
			t.Errorf("inside $(...) pwd = %q, want %q", inside, dir)
		}
		if got[1] != before {
			t.Errorf("after $(...) pwd = %q, want %q", got[1], before)
		}
	})
}

func TestParameters(t *testing.T) {
	withArgs := func(sh *shell.Shell) { sh.SetArgs("build.gnux", []string{"one", "two words", "3"}) }
	tests := []struct {
		name, script, wantOut string
	}{
		{"positional", "args $1 $2 $3 $4", "[one]\n[two words]\n[3]\n[]\n"},
		{"count and name", "args $# $0", "[3]\n[build.gnux]\n"},
		{"at is one word each", `args "$@"`, "[one]\n[two words]\n[3]\n"},
		{"unquoted at is not re-split either", "args $@", "[one]\n[two words]\n[3]\n"},
		{"at joins its neighbors", `args "<$@>"`, "[<one]\n[two words]\n[3>]\n"},
		{"star is one word", `args "$*"`, "[one two words 3]\n"},
		{"braces", "args ${1}x ${10}", "[onex]\n[]\n"},
		{"shift", "shift; args $1 $#; shift 2; args $#", "[two words]\n[2]\n[0]\n"},
		{"set --", "set -- a b; args $# $2; set --; args $#", "[2]\n[b]\n[0]\n"},
		{"an assignment sees the one before it", "A=1 B=${A}x; args $B", "[1x]\n"},
		{"a command's words are expanded before its assignments", "X=old; X=new args $X; args $X", "[old]\n[new]\n"},
		{"an assignment is not globbed or split", `X=*; Y=$@; args "$X" "$Y"`, "[*]\n[one two words 3]\n"},
		{"tilde in an assignment", "X=~/bin; args $X", "[/home/u/bin]\n"},
		{"default for unset", "args ${NOPE:-fallback} ${1:-x}", "[fallback]\n[one]\n"},
		{"default expands", "D=dist; args ${OUT:-$D/bin}", "[dist/bin]\n"},
		{"default for empty", "E=; args ${E:-was-empty}", "[was-empty]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, out, errb := run(t, shell.Options{}, tt.script, withArgs); status != 0 || out != tt.wantOut {
				t.Fatalf("status=%d out=%q (stderr %q); want %q", status, out, errb, tt.wantOut)
			}
		})
	}

	t.Run("no arguments", func(t *testing.T) {
		if _, out, _ := run(t, shell.Options{}, `args a "$@" $@ b; args $# $0`); out != "[a]\n[b]\n[0]\n[gnux]\n" {
			t.Fatalf("out = %q", out)
		}
	})
	t.Run("shift too far", func(t *testing.T) {
		status, _, errb := run(t, shell.Options{}, "shift 4", withArgs)
		if status != 1 || errb != "gnux: shift: 4: shift count out of range\n" {
			t.Fatalf("status=%d stderr=%q", status, errb)
		}
	})
}

func TestNoUnset(t *testing.T) {
	u := shell.Options{NoUnset: true}

	status, out, errb := run(t, u, "echo before\necho $NOPE\necho after")
	if status != 1 || out != "before\n" || errb != "gnux: NOPE: unbound variable\n" {
		t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
	}

	for name, script := range map[string]string{
		"set variable":       "X=1; echo $X",
		"empty but set":      "X=; echo $X",
		"environment":        "echo $FROM_ENV",
		"default":            "echo ${NOPE:-ok}",
		"special parameters": `echo $? $# $0 "$@" $*`,
		"turned off again":   "set +u; echo $NOPE",
	} {
		if status, _, errb := run(t, u, script); status != 0 || errb != "" {
			t.Errorf("%s: status=%d stderr=%q", name, status, errb)
		}
	}

	// set -u turns it on from inside the script, and it reaches every place
	// a variable is read.
	for name, script := range map[string]string{
		"set -u":           "set -u\necho $NOPE\necho after",
		"positional":       "set -u\necho $1\necho after",
		"in an assignment": "set -u\nX=$NOPE\necho after",
		"in a for list":    "set -u\nfor x in $NOPE; do echo x; done\necho after",
		"in a case":        "set -u\ncase $NOPE in *) echo x ;; esac\necho after",
		"in a pipeline":    "set -u\necho $NOPE | cat\necho after",
		"in a redirection": "set -u\necho x > $NOPE\necho after",
		"inside a loop":    "set -u\nfor x in a b; do echo $NOPE; done\necho after",
	} {
		if status, out, errb := run(t, shell.Options{}, script); status != 1 || out != "" || !strings.Contains(errb, "unbound variable") {
			t.Errorf("%s: status=%d out=%q stderr=%q", name, status, out, errb)
		}
	}

	// An interactive session loses the command, not the session.
	status, out, _ = run(t, shell.Options{NoUnset: true, Interactive: true}, "echo $NOPE; echo same-line\necho next\n")
	if status != 0 || out != "next\n" {
		t.Fatalf("interactive: status=%d out=%q", status, out)
	}
}

func TestErrExitAndConditions(t *testing.T) {
	e := shell.Options{ErrExit: true}
	tests := []struct {
		name, script string
		wantStatus   int
		wantOut      string
	}{
		{"an if condition may fail", "if fail; then echo x; fi\necho after", 0, "after\n"},
		{"a while condition may fail", "while fail; do echo x; done\necho after", 0, "after\n"},
		{"a negated command may fail", "! true\necho after", 0, "after\n"},
		{"a body may not", "if true; then fail; echo unreachable; fi\necho after", 3, ""},
		{"a loop body may not", "for x in a b; do echo $x; fail; done\necho after", 3, "a\n"},
		{"a failed substitution in an assignment", "X=$(fail)\necho after", 3, ""},
		{"set -e", "set -e\nfail\necho after", 3, ""},
		{"set +e", "set +e\nfail\necho after", 0, "after\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := e
			if strings.HasPrefix(tt.name, "set -e") {
				opts = shell.Options{}
			}
			status, out, errb := run(t, opts, tt.script)
			if status != tt.wantStatus || out != tt.wantOut {
				t.Fatalf("status=%d out=%q (stderr %q); want status=%d out=%q",
					status, out, errb, tt.wantStatus, tt.wantOut)
			}
		})
	}
}

func TestControlSyntaxErrors(t *testing.T) {
	tests := []struct{ script, want string }{
		{"if true; then echo a", "expected 'fi'"},
		{"if; then echo a; fi", "expected a command after 'if'"},
		{"if true; echo a; fi", "expected 'then', found 'fi'"},
		{"fi", "unexpected 'fi'"},
		{"for 1x in a; do echo; done", "expected a variable name after 'for'"},
		{"for x in a; echo; done", "expected 'do', found 'echo'"},
		{"while true; do echo a", "expected 'done'"},
		{"case a in a) echo", "expected a pattern or 'esac'"},
		{"case a in a echo ;; esac", "expected ')' after a case pattern"},
		{"echo $(echo a", "expected ')' to close '$('"},
		{"echo a )", "unexpected ')'"},
		{"( echo a )", "unexpected '('"},
		{"for x in a; do echo; done | while true; do break; done", "a pipeline can hold only one"},
		{"if true; then echo a; fi extra", "unexpected 'extra'"},
		{"echo 'spans\nlines'", "unterminated single quote"},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			status, out, errb := run(t, shell.Options{}, "echo before\n"+tt.script)
			if status != 2 || !strings.Contains(errb, "gnux: syntax error: "+tt.want) {
				t.Errorf("status=%d stderr=%q, want status 2 and %q", status, errb, tt.want)
			}
			if out != "before\n" {
				t.Errorf("out=%q: the commands before the error run, nothing after", out)
			}
		})
	}
}

// A command that spans lines is prompted for with ContinuePrompt until it is
// whole, and Incomplete is how a host of its own asks the same question.
func TestContinuation(t *testing.T) {
	var out, errb bytes.Buffer
	sh := shell.New(shell.Options{Interactive: true}, fakeExec,
		core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	if err := sh.RunString("for x in a b\ndo\necho $x\ndone\necho end\n"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "a\nb\nend\n" {
		t.Fatalf("out = %q", out.String())
	}
	if want := "gnux> > > > gnux> gnux> \n"; errb.String() != want {
		t.Fatalf("prompts = %q, want %q", errb.String(), want)
	}

	for src, want := range map[string]bool{
		"if true; then":       true,
		"for x in a; do":      true,
		"case a in":           true,
		"echo $(echo":         true,
		"if true; then x; fi": false,
		"echo 'unterminated":  false,
		"echo a &&":           false,
		"fi":                  false,
		"":                    false,
	} {
		if got := shell.Incomplete(src); got != want {
			t.Errorf("Incomplete(%q) = %v, want %v", src, got, want)
		}
	}

	// Eval takes a whole command, however many lines it spans.
	out.Reset()
	if status := sh.Eval("for x in 1 2\ndo echo $x\ndone"); status != 0 || out.String() != "1\n2\n" {
		t.Fatalf("Eval: status=%d out=%q", status, out.String())
	}
}

func TestLoopLimit(t *testing.T) {
	status, out, errb := run(t, shell.Options{}, "while true; do echo x; done\necho after",
		func(sh *shell.Shell) { sh.LoopLimit = 3 })
	if status != 1 || out != "x\nx\nx\n" || errb != "gnux: loop stopped after 3 iterations\n" {
		t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
	}
}

// Redirecting a compound command redirects every command inside it.
func TestCompoundRedirection(t *testing.T) {
	dir := t.TempDir()
	setDir := func(sh *shell.Shell) { sh.Set("D", dir) }

	status, out, errb := run(t, shell.Options{},
		"for x in a b; do echo $x; nosuch; done > $D/out.txt 2> $D/err.txt\nupper < $D/out.txt", setDir)
	if status != 0 || out != "A\nB\n" || errb != "" {
		t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "err.txt")); strings.Count(string(b), "nosuch: command not found") != 2 {
		t.Fatalf("err.txt = %q", b)
	}

	status, out, _ = run(t, shell.Options{}, "if true; then cat; fi < $D/out.txt", setDir)
	if status != 0 || out != "a\nb\n" {
		t.Fatalf("status=%d out=%q", status, out)
	}
}
