package sum_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sum"
)

func run(opts sum.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = sum.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestStdin(t *testing.T) {
	tests := []struct {
		name  string
		opts  sum.Options
		input string
		want  string
	}{
		{"bsd", sum.Options{}, "hello\n", "36979     1\n"},
		{"bsd pads the checksum with zeros", sum.Options{}, "a", "00097     1\n"},
		{"bsd empty", sum.Options{}, "", "00000     0\n"},
		{"bsd counts 1024-byte blocks", sum.Options{}, strings.Repeat("x", 1025), "00056     2\n"},
		{"sysv", sum.Options{SysV: true}, "hello\n", "542 1\n"},
		{"sysv empty", sum.Options{SysV: true}, "", "0 0\n"},
		{"sysv counts 512-byte blocks", sum.Options{SysV: true}, strings.Repeat("x", 513), "61560 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(tt.opts, tt.input)
			if err != nil || out != tt.want || errOut != "" {
				t.Fatalf("out = %q, stderr = %q, err = %v; want %q", out, errOut, err, tt.want)
			}
		})
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	if err := os.WriteFile(a, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(sum.Options{}, "hello\n", a, "-")
	if want := "36979     1 " + a + "\n36979     1 -\n"; err != nil || out != want {
		t.Fatalf("out = %q, err = %v; want %q", out, err, want)
	}
	out, _, err = run(sum.Options{SysV: true}, "", a)
	if want := "542 1 " + a + "\n"; err != nil || out != want {
		t.Fatalf("sysv out = %q, err = %v; want %q", out, err, want)
	}
}

func TestFailingOperandDoesNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	out, errOut, err := run(sum.Options{}, "", missing, dir, a)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "00097     1 " + a + "\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	want := "sumx: " + missing + ": No such file or directory\nsumx: " + dir + ": Is a directory\n"
	if errOut != want {
		t.Fatalf("stderr = %q; want %q", errOut, want)
	}
}
