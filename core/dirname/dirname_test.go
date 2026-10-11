package dirname_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dirname"
)

func run(opts dirname.Options, args ...string) (string, error) {
	var out, errb bytes.Buffer
	d := dirname.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	err := d.Run(args)
	return out.String(), err
}

func TestDir(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"/usr/bin/sort", "/usr/bin"},
		{"dir1/str", "dir1"},
		{"stdio.h", "."},
		{"a", "."},
		{"", "."},
		{".", "."},
		{"..", "."},
		{"/a", "/"},
		{"/", "/"},
		{"//", "/"},
		{"///a", "/"},
		{"a/b/", "a"},
		{"a/b//", "a"},
		{"a//b", "a"},
		{"a/", "."},
		{"a/./b", "a/."},
		{"/a/b/", "/a"},
	}
	for _, tt := range tests {
		if got := dirname.Dir(tt.name); got != tt.want {
			t.Errorf("Dir(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDirNativeSeparator(t *testing.T) {
	sep := string(os.PathSeparator)
	if got := dirname.Dir("a" + sep + "b" + sep + "c"); got != "a"+sep+"b" {
		t.Fatalf("Dir with native separator = %q, want %q", got, "a"+sep+"b")
	}
	if got := dirname.Dir(sep + "a"); got != sep {
		t.Fatalf("Dir under native root = %q, want %q", got, sep)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		opts dirname.Options
		args []string
		want string
	}{
		{"single", dirname.Options{}, []string{"/usr/bin/sort"}, "/usr/bin\n"},
		{"several", dirname.Options{}, []string{"dir1/str", "dir2/str", "stdio.h"}, "dir1\ndir2\n.\n"},
		{"zero", dirname.Options{Zero: true}, []string{"a/b", "c"}, "a\x00.\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := run(tt.opts, tt.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMissingOperand(t *testing.T) {
	out, err := run(dirname.Options{})
	var uerr *core.UsageError
	if !errors.As(err, &uerr) {
		t.Fatalf("err = %v, want UsageError", err)
	}
	if uerr.Msg != "missing operand" {
		t.Fatalf("usage = %q, want %q", uerr.Msg, "missing operand")
	}
	if out != "" {
		t.Fatalf("stdout = %q, want empty", out)
	}
}
