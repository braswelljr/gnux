package ptx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func run(opts Options, stdin string, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	err := New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestUnescape(t *testing.T) {
	for in, want := range map[string]string{
		`a\tb`: "a\tb", `\n`: "\n", `\x41\x4a`: "AJ", `\0101`: "A", `\x`: `\x`, `\q`: `\q`, `ab\cdef`: "ab", `end\`: "end",
	} {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslateEmacs(t *testing.T) {
	tests := []struct{ in, want string }{
		{`a\(b\|c\)d`, `a(b|c)d`},
		{`(x)|{y}`, `\(x\)\|\{y\}`},
		{`*a+b?`, `\*a+b?`},
		{`^a$`, `^a$`},
		{`a^b$c`, `a\^b\$c`},
		{`\(^a\|b$\)`, `(^a|b$)`},
		{`[]"')}]*`, `[\]\"\'\)\}]*`},
		{`[^a-z]\w`, `[^a-z][0-9A-Za-z]`},
	}
	for _, tt := range tests {
		if got, err := translateEmacs(tt.in); err != nil || got != tt.want {
			t.Errorf("translateEmacs(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
	if _, err := translateEmacs(`\(a\)\1`); err == nil {
		t.Error("a back reference was accepted")
	}
}

func TestDumbOutput(t *testing.T) {
	out, errb, err := run(Options{Width: 40}, "one two three.\n")
	if err != nil {
		t.Fatalf("%v: %s", err, errb)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("output %q, want one line per word", out)
	}
	// Sorted by keyword, each keyword starting in the middle: after the
	// reference gap every line starts with (3) and half the width (20).
	for i, word := range []string{"one", "three.", "two"} {
		if !strings.HasPrefix(lines[i][23:], word) {
			t.Errorf("line %d = %q, want %q at column 23", i, lines[i], word)
		}
	}
}

func TestRoffAndTeX(t *testing.T) {
	out, _, err := run(Options{Format: Roff}, "say \"hi\" now\n")
	if err != nil || !strings.Contains(out, `.xx "" "say ""hi""" "now" ""`) {
		t.Fatalf("roff: %v %q", err, out)
	}
	out, _, err = run(Options{Format: TeX, MacroName: "idx"}, "a_b c\n")
	if err != nil || !strings.HasPrefix(out, `\idx {}{}{a}{\_b c}{}`) {
		t.Fatalf("tex: %v %q", err, out)
	}
}

func TestWordLists(t *testing.T) {
	dir := t.TempDir()
	ignore := filepath.Join(dir, "ignore")
	only := filepath.Join(dir, "only")
	if err := os.WriteFile(ignore, []byte("the\nand\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(only, []byte("cat\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, _ := run(Options{IgnoreFile: ignore}, "the cat and the dog\n")
	if n := strings.Count(out, "\n"); n != 2 {
		t.Fatalf("-i kept %d words: %q", n, out)
	}
	out, _, _ = run(Options{OnlyFile: only}, "the cat and the dog\n")
	if n := strings.Count(out, "\n"); n != 1 || !strings.Contains(out, "cat and the dog") {
		t.Fatalf("-o: %q", out)
	}
}

func TestAutoReference(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("alpha\nbeta\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(Options{AutoReference: true}, "", f)
	if err != nil || !strings.HasPrefix(out, f+":1:") || !strings.Contains(out, "\n"+f+":2:") {
		t.Fatalf("%v %q", err, out)
	}
}

func TestErrors(t *testing.T) {
	_, errb, err := run(Options{WordRegexp: `\(`}, "x")
	if !errors.Is(err, core.ErrFailed) || errb != "ptxx: Unmatched ( or \\( (for regexp '\\\\(')\n" {
		t.Fatalf("bad regexp: %v %q", err, errb)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	_, errb, _ = run(Options{}, "", missing)
	if errb != "ptxx: "+missing+": No such file or directory\n" {
		t.Fatalf("missing file: %q", errb)
	}
	var usage *core.UsageError
	if _, _, err := run(Options{Traditional: true}, "", "a", "b", "c"); !errors.As(err, &usage) || usage.Msg != "extra operand 'c'" {
		t.Fatalf("-G operands: %v", err)
	}
}
