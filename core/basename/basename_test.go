package basename_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basename"
)

func run(opts basename.Options, args ...string) (string, error) {
	var out, errb bytes.Buffer
	b := basename.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	err := b.Run(args)
	return out.String(), err
}

func TestBase(t *testing.T) {
	tests := []struct {
		name, suffix, want string
	}{
		{"/usr/bin/sort", "", "sort"},
		{"include/stdio.h", ".h", "stdio"},
		{"include/stdio.h", "", "stdio.h"},
		{"dir/", "", "dir"},
		{"dir///", "", "dir"},
		{"/", "", "/"},
		{"///", "", "/"},
		{"", "", ""},
		{"a", "", "a"},
		{"a//b", "", "b"},
		{".h", ".h", ".h"},
		{"dir/.h", ".h", ".h"},
		{"stdio.h", "x", "stdio.h"},
		{"stdio.h", "stdio.h.long", "stdio.h"},
		{"/", "/", "/"},
	}
	for _, tt := range tests {
		if got := basename.Base(tt.name, tt.suffix); got != tt.want {
			t.Errorf("Base(%q, %q) = %q, want %q", tt.name, tt.suffix, got, tt.want)
		}
	}
}

func TestBaseNativeSeparator(t *testing.T) {
	sep := string(os.PathSeparator)
	if got := basename.Base("a"+sep+"b"+sep, ""); got != "b" {
		t.Fatalf("Base with native separator = %q, want %q", got, "b")
	}
	if got := basename.Base(sep+sep, ""); got != sep {
		t.Fatalf("Base of native root = %q, want %q", got, sep)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		opts basename.Options
		args []string
		want string
	}{
		{"single name", basename.Options{}, []string{"/usr/bin/sort"}, "sort\n"},
		{"name and suffix", basename.Options{}, []string{"include/stdio.h", ".h"}, "stdio\n"},
		{"empty name", basename.Options{}, []string{""}, "\n"},
		{"multiple", basename.Options{Multiple: true}, []string{"a/b", "c/d", "e"}, "b\nd\ne\n"},
		{"suffix implies multiple", basename.Options{Suffix: ".h"}, []string{"a/x.h", "b/y.h"}, "x\ny\n"},
		{"multiple with suffix", basename.Options{Multiple: true, Suffix: ".go"}, []string{"a.go", ".go"}, "a\n.go\n"},
		{"zero", basename.Options{Multiple: true, Zero: true}, []string{"a/b", "c"}, "b\x00c\x00"},
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

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing operand", nil, "missing operand"},
		{"extra operand", []string{"a", "b", "c"}, "extra operand 'c'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(basename.Options{}, tt.args...)
			var uerr *core.UsageError
			if !errors.As(err, &uerr) {
				t.Fatalf("err = %v, want UsageError", err)
			}
			if uerr.Msg != tt.want {
				t.Fatalf("usage = %q, want %q", uerr.Msg, tt.want)
			}
			if out != "" {
				t.Fatalf("stdout = %q, want empty", out)
			}
		})
	}
}
