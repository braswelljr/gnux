package tac_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tac"
)

func run(opts tac.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = tac.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestTac(t *testing.T) {
	tests := []struct {
		name string
		opts tac.Options
		in   string
		want string
	}{
		{"lines", tac.Options{}, "1\n2\n3\n", "3\n2\n1\n"},
		{"no final newline", tac.Options{}, "a\nb", "ba\n"},
		{"empty", tac.Options{}, "", ""},
		{"blank lines", tac.Options{}, "a\n\nb\n", "b\n\na\n"},
		{"separator", tac.Options{Separator: ",", SeparatorSet: true}, "a,b,c,", "c,b,a,"},
		{"multi-byte separator", tac.Options{Separator: "--", SeparatorSet: true}, "a--b--", "b--a--"},
		{"before", tac.Options{Before: true}, "1\n2\n3\n", "\n\n3\n21"},
		{"before with separator", tac.Options{Before: true, Separator: ",", SeparatorSet: true}, ",a,b", ",b,a"},
		{"regex", tac.Options{Regex: true, Separator: "[0-9]+", SeparatorSet: true}, "a1b22c333", "c333b22a1"},
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

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	if err := os.WriteFile(a, []byte("1\n2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	out, errb, err := run(tac.Options{}, "x\ny\n", a, missing, "-", dir)
	if out != "2\n1\ny\nx\n" {
		t.Fatalf("out = %q", out)
	}
	want := "tacx: failed to open '" + missing + "' for reading: No such file or directory\n" +
		"tacx: " + dir + ": read error: Is a directory\n"
	if !errors.Is(err, core.ErrFailed) || errb != want {
		t.Fatalf("err = %v, stderr = %q, want %q", err, errb, want)
	}
}

func TestErrors(t *testing.T) {
	_, errb, err := run(tac.Options{SeparatorSet: true}, "x")
	if !errors.Is(err, core.ErrFailed) || errb != "tacx: separator cannot be empty\n" {
		t.Fatalf("empty separator: %v %q", err, errb)
	}
	_, errb, err = run(tac.Options{Regex: true, Separator: "(", SeparatorSet: true}, "x")
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb, "tacx: ") {
		t.Fatalf("bad regex: %v %q", err, errb)
	}
}
