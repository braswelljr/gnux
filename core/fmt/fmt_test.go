package fmt_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fmt"
)

func run(opts fmt.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = fmt.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestFmt(t *testing.T) {
	tests := []struct {
		name string
		opts fmt.Options
		in   string
		want string
	}{
		{"one line", fmt.Options{}, "a b c\n", "a b c\n"},
		{"joins lines", fmt.Options{}, "hello\nworld\n", "hello world\n"},
		{"keeps paragraphs", fmt.Options{}, "a\n\nb\n", "a\n\nb\n"},
		{"keeps indentation", fmt.Options{}, "  a\n  b\n", "  a b\n"},
		{"wraps", fmt.Options{Width: 10, WidthSet: true}, "aaa bbb ccc ddd\n", "aaa bbb\nccc ddd\n"},
		{"split only", fmt.Options{Split: true}, "a\nb\n", "a\nb\n"},
		{"split only wraps", fmt.Options{Split: true, Width: 10, WidthSet: true}, "aaa bbb ccc ddd\nx\n", "aaa bbb\nccc ddd\nx\n"},
		{"uniform", fmt.Options{Uniform: true}, "a   b\n", "a b\n"},
		{"prefix", fmt.Options{Prefix: "# "}, "# a\n# b\nc\n", "# a b\nc\n"},
		{"no final newline", fmt.Options{}, "a\nb", "a b\n"},
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

func TestWidths(t *testing.T) {
	tests := []struct {
		opts        fmt.Options
		width, goal int
	}{
		{fmt.Options{}, 75, 70},
		{fmt.Options{Width: 100, WidthSet: true}, 100, 93},
		{fmt.Options{Goal: 50, GoalSet: true}, 60, 50},
		{fmt.Options{Width: 80, WidthSet: true, Goal: 50, GoalSet: true}, 80, 50},
	}
	for _, tt := range tests {
		if w, g := tt.opts.Widths(); w != tt.width || g != tt.goal {
			t.Errorf("%+v.Widths() = %d, %d; want %d, %d", tt.opts, w, g, tt.width, tt.goal)
		}
	}
}

func TestMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	out, errb, err := run(fmt.Options{}, "x\n", missing, "-")
	if !errors.Is(err, core.ErrFailed) || errb != "fmtx: cannot open '"+missing+"' for reading: No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "x\n" {
		t.Fatalf("the remaining operand was not formatted: %q", out)
	}
}
