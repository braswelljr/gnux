package paste_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/paste"
)

func run(opts paste.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = paste.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParallel(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1\n2\n3\n")
	b := write(t, dir, "b", "x\ny\n")
	c := write(t, dir, "c", "p\n")
	empty := write(t, dir, "empty", "")
	noeol := write(t, dir, "noeol", "u\nv")

	d := func(list string) paste.Options { return paste.Options{Delimiters: list, DelimitersSet: true} }
	tests := []struct {
		name string
		opts paste.Options
		args []string
		want string
	}{
		{"two files", paste.Options{}, []string{a, b}, "1\tx\n2\ty\n3\t\n"},
		{"shorter first", paste.Options{}, []string{b, a}, "x\t1\ny\t2\n\t3\n"},
		{"three files", paste.Options{}, []string{a, b, c}, "1\tx\tp\n2\ty\t\n3\t\t\n"},
		{"one file", paste.Options{}, []string{a}, "1\n2\n3\n"},
		{"empty file", paste.Options{}, []string{empty}, ""},
		{"empty and data", paste.Options{}, []string{empty, b}, "\tx\n\ty\n"},
		{"missing last newline", paste.Options{}, []string{noeol, b}, "u\tx\nv\ty\n"},
		{"delimiter", d(","), []string{a, b}, "1,x\n2,y\n3,\n"},
		{"delimiters cycle", d(",;"), []string{a, b, c}, "1,x;p\n2,y;\n3,;\n"},
		{"delimiters restart each line", d("abc"), []string{a, b, c}, "1axbp\n2ayb\n3ab\n"},
		{"empty delimiter escape", d(`\0`), []string{a, b}, "1x\n2y\n3\n"},
		{"empty list", d(""), []string{a, b}, "1x\n2y\n3\n"},
		{"escapes", d(`\t\n\\`), []string{a, b, c}, "1\tx\np\n2\ty\n\n3\t\n\n"},
		{"unknown escape keeps the character", d(`\q`), []string{a, b}, "1qx\n2qy\n3q\n"},
		{"zero terminated", paste.Options{ZeroTerminated: true}, []string{write(t, dir, "z1", "a\x00b\x00"), write(t, dir, "z2", "c\x00")}, "a\tc\x00b\t\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "", tt.args...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestSharedStdin checks that every "-" reads from the one standard input,
// so `paste - -` joins pairs of lines.
func TestSharedStdin(t *testing.T) {
	tests := []struct {
		name  string
		opts  paste.Options
		input string
		args  []string
		want  string
	}{
		{"no operands", paste.Options{}, "a\nb\n", nil, "a\nb\n"},
		{"pairs", paste.Options{}, "1\n2\n3\n4\n", []string{"-", "-"}, "1\t2\n3\t4\n"},
		{"odd pairs", paste.Options{}, "1\n2\n3\n", []string{"-", "-"}, "1\t2\n3\t\n"},
		{"triples", paste.Options{Delimiters: ",", DelimitersSet: true}, "1\n2\n", []string{"-", "-", "-"}, "1,2,\n"},
		{"serial reads it once", paste.Options{Serial: true}, "a\nb\n", []string{"-", "-"}, "a\tb\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, tt.input, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestSerial(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1\n2\n3\n")
	b := write(t, dir, "b", "x\ny")
	empty := write(t, dir, "empty", "")
	s := paste.Options{Serial: true}
	tests := []struct {
		name string
		opts paste.Options
		args []string
		want string
	}{
		{"two files", s, []string{a, b}, "1\t2\t3\nx\ty\n"},
		{"empty file is an empty line", s, []string{empty, a}, "\n1\t2\t3\n"},
		{"delimiters cycle and restart per file", paste.Options{Serial: true, Delimiters: ",;", DelimitersSet: true}, []string{a, a}, "1,2;3\n1,2;3\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, "", tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1\n")
	missing := filepath.Join(dir, "missing")

	// Parallel mode opens every file first and stops at the first failure.
	out, errb, err := run(paste.Options{}, "", a, missing)
	if !errors.Is(err, core.ErrFailed) || out != "" {
		t.Fatalf("parallel: err = %v, stdout = %q", err, out)
	}
	if want := "pastex: " + missing + ": No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}

	// Serial mode reports the file and carries on.
	out, errb, err = run(paste.Options{Serial: true}, "", missing, a)
	if !errors.Is(err, core.ErrFailed) || out != "1\n" || !strings.Contains(errb, "No such file or directory") {
		t.Fatalf("serial: err = %v, stdout = %q, stderr = %q", err, out, errb)
	}

	_, errb, err = run(paste.Options{Delimiters: `a\`, DelimitersSet: true}, "", a)
	if !errors.Is(err, core.ErrFailed) || errb != "pastex: delimiter list ends with an unescaped backslash: a\\\n" {
		t.Fatalf("backslash: err = %v, stderr = %q", err, errb)
	}

	_, errb, err = run(paste.Options{Serial: true}, "", dir)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, dir+": ") {
		t.Fatalf("directory: err = %v, stderr = %q", err, errb)
	}
}
