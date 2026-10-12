package fold_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fold"
)

func run(opts fold.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = fold.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestFold(t *testing.T) {
	tests := []struct {
		name string
		opts fold.Options
		in   string
		want string
	}{
		{"short line", fold.Options{}, "hello\n", "hello\n"},
		{"default width", fold.Options{}, strings.Repeat("x", 85) + "\n", strings.Repeat("x", 80) + "\n" + "xxxxx\n"},
		{"width", fold.Options{Width: 3}, "abcdef\n", "abc\ndef\n"},
		{"exact width", fold.Options{Width: 3}, "abc\n", "abc\n"},
		{"no final newline", fold.Options{Width: 2}, "abcd", "ab\ncd"},
		{"spaces", fold.Options{Width: 8, Spaces: true}, "hello world foo\n", "hello \nworld \nfoo\n"},
		{"spaces long word", fold.Options{Width: 3, Spaces: true}, "abcdef\n", "abc\ndef\n"},
		{"tab counts to next stop", fold.Options{Width: 10}, "\tab\n", "\tab\n"},
		{"tab overflows", fold.Options{Width: 9}, "\tab\n", "\ta\nb\n"},
		{"bytes", fold.Options{Width: 2, Bytes: true}, "\tabc\n", "\ta\nbc\n"},
		{"empty lines", fold.Options{Width: 2}, "\n\n", "\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(tt.opts, tt.in)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v, want %q", out, err, tt.want)
			}
		})
	}
}

func TestMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	out, errb, err := run(fold.Options{Width: 2}, "abcd\n", missing, "-")
	if !errors.Is(err, core.ErrFailed) || errb != "foldx: "+missing+": No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "ab\ncd\n" {
		t.Fatalf("the remaining operand was not folded: %q", out)
	}
}
