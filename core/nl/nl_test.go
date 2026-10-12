package nl_test

import (
	"bytes"
	"errors"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nl"
)

func run(opts nl.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = nl.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestNumbering(t *testing.T) {
	tests := []struct {
		name string
		opts nl.Options
		in   string
		want string
	}{
		{"default skips empty lines", nl.Options{}, "a\n\nb\n", "     1\ta\n       \n     2\tb\n"},
		{"no final newline", nl.Options{}, "a", "     1\ta\n"},
		{"all", nl.Options{BodyStyle: "a"}, "a\n\nb\n", "     1\ta\n     2\t\n     3\tb\n"},
		{"none", nl.Options{BodyStyle: "n"}, "a\n", "       a\n"},
		{"regex", nl.Options{BodyStyle: "p^#"}, "# x\ny\n# z\n", "     1\t# x\n       y\n     2\t# z\n"},
		{"left", nl.Options{Format: "ln"}, "a\n", "1     \ta\n"},
		{"zeros", nl.Options{Format: "rz"}, "a\n", "000001\ta\n"},
		{"width and separator", nl.Options{Width: 3, Separator: ": ", SeparatorSet: true}, "a\n\n", "  1: a\n     \n"},
		{"empty separator", nl.Options{Width: 2, SeparatorSet: true}, "a\n", " 1a\n"},
		{"start and increment", nl.Options{StartingLine: 10, StartingLineSet: true, Increment: 5, IncrementSet: true, Width: 2}, "a\nb\n", "10\ta\n15\tb\n"},
		{"zero start", nl.Options{StartingLineSet: true, Width: 1}, "a\n", "0\ta\n"},
		{"negative", nl.Options{StartingLine: -1, StartingLineSet: true, Width: 2}, "a\nb\n", "-1\ta\n 0\tb\n"},
		{"join blank lines", nl.Options{BodyStyle: "a", JoinBlank: 2, Width: 1}, "\n\n\n\nx\n", "  \n1\t\n  \n2\t\n3\tx\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.in)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v (stderr %q), want %q", out, err, errb, tt.want)
			}
		})
	}
}

func TestSections(t *testing.T) {
	in := "\\:\\:\\:\nhead\n\\:\\:\nb1\nb2\n\\:\nfoot\n\\:\\:\\:\n\\:\\:\nb3\n"
	want := "\n       head\n\n     1\tb1\n     2\tb2\n\n       foot\n\n\n     1\tb3\n"
	if out, _, err := run(nl.Options{}, in); err != nil || out != want {
		t.Fatalf("sections: %q, %v; want %q", out, err, want)
	}
	// -p keeps counting across pages.
	want = "\n       head\n\n     1\tb1\n     2\tb2\n\n       foot\n\n\n     3\tb3\n"
	if out, _, _ := run(nl.Options{NoRenumber: true}, in); out != want {
		t.Fatalf("-p: %q, want %q", out, want)
	}
	// Header and footer styles.
	want = "\n     1\thead\n\n     2\tb1\n     3\tb2\n\n     4\tfoot\n\n\n     1\tb3\n"
	if out, _, _ := run(nl.Options{HeaderStyle: "a", FooterStyle: "t"}, in); out != want {
		t.Fatalf("-h a -f t: %q, want %q", out, want)
	}
}

func TestDelimiter(t *testing.T) {
	// One character replaces the first: "@:".
	out, _, _ := run(nl.Options{Delimiter: "@", DelimiterSet: true}, "@:@:\nx\n")
	if out != "\n     1\tx\n" {
		t.Fatalf("-d @: %q", out)
	}
	// An empty delimiter turns sections off.
	out, _, _ = run(nl.Options{Delimiter: "", DelimiterSet: true}, "\\:\\:\n")
	if out != "     1\t\\:\\:\n" {
		t.Fatalf("-d '': %q", out)
	}
}

func TestFilesAreOneStream(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	out, errb, err := run(nl.Options{Width: 1}, "a\n", "-", missing, "-")
	if !errors.Is(err, core.ErrFailed) || errb != "nlx: "+missing+": No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "1\ta\n" {
		t.Fatalf("out = %q", out)
	}
}

func TestInvalid(t *testing.T) {
	tests := []struct {
		opts nl.Options
		msg  string
	}{
		{nl.Options{BodyStyle: "x"}, "invalid body numbering style: 'x'"},
		{nl.Options{HeaderStyle: "q"}, "invalid header numbering style: 'q'"},
		{nl.Options{FooterStyle: "z"}, "invalid footer numbering style: 'z'"},
		{nl.Options{BodyStyle: "p["}, "Unmatched [, [^, [:, [., or [="},
		{nl.Options{Format: "xx"}, "invalid line numbering format: 'xx'"},
	}
	for _, tt := range tests {
		_, errb, err := run(tt.opts, "a\n")
		if !errors.Is(err, core.ErrFailed) || errb != "nlx: "+tt.msg+"\n" {
			t.Errorf("%+v: err = %v, stderr = %q", tt.opts, err, errb)
		}
	}
}

func TestOverflow(t *testing.T) {
	out, errb, err := run(nl.Options{StartingLine: math.MaxInt64, StartingLineSet: true, Width: 1}, "a\nb\n")
	if !errors.Is(err, core.ErrFailed) || errb != "nlx: line number overflow\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "9223372036854775807\ta\n" {
		t.Fatalf("out = %q", out)
	}
}
