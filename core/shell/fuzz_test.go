package shell

import (
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
}

// FuzzParse checks that no input makes the lexer or parser panic, and that
// whatever it accepts is well formed: every pipeline has a command and every
// command a word.
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
			if len(st.pipeline) == 0 {
				t.Fatalf("parse(%q): step %d has an empty pipeline", line, i)
			}
			switch st.connector {
			case "", ";", "&&", "||":
			default:
				t.Fatalf("parse(%q): step %d has connector %q", line, i, st.connector)
			}
			for j, c := range st.pipeline {
				if len(c.words) == 0 {
					t.Fatalf("parse(%q): step %d command %d has no words", line, i, j)
				}
				for _, r := range c.redirects {
					if len(r.target) == 0 {
						t.Fatalf("parse(%q): redirect %q has no target", line, r.op)
					}
				}
			}
		}
	})
}

// FuzzExpand checks that expanding any parsed word, with variables set and
// unset, never panics. Expansion only reads the filesystem (globbing), so it
// is safe to run in parallel fuzz workers.
func FuzzExpand(f *testing.F) {
	for _, s := range seeds {
		f.Add(s)
	}
	exec := func(string, []string, core.Streams) (int, bool) { return 0, true }
	f.Fuzz(func(t *testing.T, line string) {
		steps, err := parse(line)
		if err != nil {
			return
		}
		sh := New(Options{}, exec, core.Discard())
		sh.Getenv = func(string) string { return "" }
		sh.HomeDir = func() (string, error) { return "/home/fuzz", nil }
		sh.Set("NAME", "a b*")
		sh.Set("A", strings.Repeat("x", 8))
		for _, st := range steps {
			for _, c := range st.pipeline {
				for _, w := range c.words {
					_ = sh.expand(w)
				}
				for _, r := range c.redirects {
					_ = sh.expand(r.target)
				}
			}
		}
	})
}
