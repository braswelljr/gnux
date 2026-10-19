package tsort_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tsort"
)

func run(stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = tsort.New(tsort.Options{}, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestOrder(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"chain", "a b\nb c\n", "a\nb\nc\n"},
		{"free strings in byte order", "b a\nc a\nd c\n", "b\nd\nc\na\n"},
		{"a pair of one string is a node", "a a\n", "a\n"},
		{"tabs and blank lines separate", "x\ty\n\n  y   z", "x\ny\nz\n"},
		{"successors freed in reverse reading order",
			"3 8\n3 10\n5 11\n7 8\n7 11\n8 9\n11 2\n11 9\n11 10\n", "3\n5\n7\n11\n8\n10\n2\n9\n"},
		{"duplicate pairs", "a b\na b\n", "a\nb\n"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, err := run(tt.in)
			if err != nil || out != tt.want || errOut != "" {
				t.Fatalf("out = %q, stderr = %q, err = %v; want %q", out, errOut, err, tt.want)
			}
		})
	}
}

func TestLoop(t *testing.T) {
	out, errOut, err := run("a b\nb c\nc a\nd e\n")
	if want := "d\ne\na\nb\nc\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	if want := "tsortx: -: input contains a loop:\ntsortx: a\ntsortx: b\ntsortx: c\n"; errOut != want {
		t.Fatalf("stderr = %q; want %q", errOut, want)
	}
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}

	// A loop reached from outside: only its own strings are reported.
	out, errOut, _ = run("a b b c c d d b\n")
	if want := "a\nb\nc\nd\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	if want := "tsortx: -: input contains a loop:\ntsortx: b\ntsortx: c\ntsortx: d\n"; errOut != want {
		t.Fatalf("stderr = %q; want %q", errOut, want)
	}
}

func TestOddTokens(t *testing.T) {
	out, errOut, err := run("x y z\n")
	if out != "" || errOut != "tsortx: -: input contains an odd number of tokens\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errOut, err)
	}
}

func TestFileOperand(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deps")
	if err := os.WriteFile(f, []byte("lib app\nlib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, _ := run("", f)
	if want := "tsortx: " + f + ": input contains an odd number of tokens\n"; errOut != want {
		t.Fatalf("stderr = %q; want %q", errOut, want)
	}
	if err := os.WriteFile(f, []byte("lib app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, _, err := run("", f); err != nil || out != "lib\napp\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if out, _, err := run("x y", "-"); err != nil || out != "x\ny\n" {
		t.Fatalf("- out = %q, err = %v", out, err)
	}

	missing := filepath.Join(dir, "missing")
	if _, errOut, err := run("", missing); errOut != "tsortx: "+missing+": No such file or directory\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
	if _, errOut, _ := run("", dir); errOut != "tsortx: "+dir+": Is a directory\n" {
		t.Fatalf("stderr = %q", errOut)
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run("", "a", "b")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("err = %v", err)
	}
}
