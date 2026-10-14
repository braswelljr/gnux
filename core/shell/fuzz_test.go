package shell

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

// seeds are lines in the shape real scripts take, plus the corners the parser
// has to refuse cleanly.
var seeds = []string{
	"",
	"echox hello",
	"lsx -la | wcx -l > count",
	`mkdirx -p "dist/$NAME" && cpx -r assets dist/ || echox 'failed: $?'`,
	"A=1 B=${A}x; echox $A $B ${UNSET} $?",
	`catx < in.txt >> out.txt 2> err.txt`,
	`echox C:\tools\bin \$HOME \"q\" # comment`,
	"rmx -rf build/*.o ~/tmp/[ab]?.log",
	"| x",
	"x &&",
	"> out",
	`echox "unterminated`,
	"echox 'unterminated",
	"echox ${",
	"a ;; b",
	`\`,
	"if testx -f a; then echox yes; elif falsex; then echox no; else echox other; fi",
	"for f in *.go $(lsx) \"$@\"; do echox $f; done | sortx",
	"while falsex; do break; done; while truex; do continue 2; done",
	"case $1 in (a|b) echox ab ;; *.go) echox go ;; *) echox other ;; esac",
	"X=$(echox \"a $(echox b)\") echox ${X:-none} $# $0 $1 ${10} $*",
	"! truex && set -eu; shift 2; [ -n \"$A\" ]",
	"if truex; then",
	"echox $(echox",
	"for do in; done; fi; esac )",
}

// walk calls visit for every command in steps, however deeply nested: inside
// compound commands and inside command substitutions.
func walk(steps []step, visit func(command)) {
	words := func(ws []word) {
		for _, w := range ws {
			for _, p := range w {
				if p.subst {
					walk(p.body, visit)
				}
			}
		}
	}
	for _, st := range steps {
		for _, c := range st.pipeline.cmds {
			visit(c)
			words(c.words)
			for _, r := range c.redirects {
				words([]word{r.target})
			}
			switch cc := c.compound.(type) {
			case *ifClause:
				for _, b := range cc.branches {
					walk(b.cond, visit)
					walk(b.body, visit)
				}
				walk(cc.elseBody, visit)
			case *forClause:
				words(cc.items)
				walk(cc.body, visit)
			case *whileClause:
				walk(cc.cond, visit)
				walk(cc.body, visit)
			case *caseClause:
				words([]word{cc.subject})
				for _, arm := range cc.arms {
					words(arm.patterns)
					walk(arm.body, visit)
				}
			}
		}
	}
}

// FuzzParse checks that no input makes the lexer or parser panic, and that
// whatever it accepts is well formed: every pipeline has a command, and every
// command is either compound or has a word.
func FuzzParse(f *testing.F) {
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, line string) {
		steps, err := parse(line)
		if err != nil {
			return
		}
		for i, st := range steps {
			if len(st.pipeline.cmds) == 0 {
				t.Fatalf("parse(%q): step %d has an empty pipeline", line, i)
			}
			switch st.connector {
			case "", ";", "&&", "||":
			default:
				t.Fatalf("parse(%q): step %d has connector %q", line, i, st.connector)
			}
		}
		walk(steps, func(c command) {
			if len(c.words) == 0 && c.compound == nil {
				t.Fatalf("parse(%q): a command has neither words nor a body", line)
			}
			for _, r := range c.redirects {
				if len(r.target) == 0 {
					t.Fatalf("parse(%q): redirect %q has no target", line, r.op)
				}
			}
		})
	})
}

// fakeFiles is the filesystem FuzzExpand globs against. A pattern such as
// /*/*/*/* against the real one would walk all of /proc on a Linux runner and
// stall the fuzzer, so matching runs filepath.Match over this list instead.
var fakeFiles = []string{"a", "b c", "dir/x.go", "dir/y.txt", "/etc/hosts", "[x]", "*"}

func fakeGlob(pattern string) ([]string, error) {
	var out []string
	for _, f := range fakeFiles {
		ok, err := filepath.Match(pattern, f)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, f)
		}
	}
	return out, nil
}

// FuzzRun checks that running whatever parses - expansion, control flow and
// command substitution included - never panics and always ends. Globbing uses
// fakeGlob, never the disk, and LoopLimit stops a loop that would not.
func FuzzRun(f *testing.F) {
	for _, s := range seeds {
		f.Add(s)
	}
	exec := func(name string, _ []string, _ core.Streams) (int, bool) {
		if name == "falsex" {
			return 1, true
		}
		return 0, true
	}
	f.Fuzz(func(t *testing.T, line string) {
		// Redirection and cd would reach the real disk.
		if strings.ContainsAny(line, "<>") || strings.Contains(line, "cd") {
			return
		}
		if _, err := parse(line); err != nil {
			return
		}
		sh := New(Options{Interactive: true}, exec, core.Discard())
		sh.Getenv = func(string) string { return "" }
		sh.HomeDir = func() (string, error) { return "/home/fuzz", nil }
		sh.Glob = fakeGlob
		sh.LoopLimit = 20
		sh.Set("NAME", "a b*")
		sh.Set("A", strings.Repeat("x", 8))
		sh.SetArgs("fuzz", []string{"one", "two words"})
		sh.Eval(line)
	})
}
