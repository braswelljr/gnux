package unexpand_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unexpand"
)

func run(opts unexpand.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = unexpand.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestUnexpand(t *testing.T) {
	tests := []struct {
		name string
		opts unexpand.Options
		in   string
		want string
	}{
		{"leading only", unexpand.Options{}, "        a       b\n", "\ta       b\n"},
		{"partial leading", unexpand.Options{}, "          a\n", "\t  a\n"},
		{"all", unexpand.Options{All: true}, "        a       b\n", "\ta\tb\n"},
		{"single blank kept", unexpand.Options{All: true}, "abcdefg b\n", "abcdefg b\n"},
		{"size", unexpand.Options{All: true, Tabs: []string{"4"}}, "    a   b\n", "\ta\tb\n"},
		{"first only wins", unexpand.Options{All: true, FirstOnly: true}, "        a       b\n", "\ta       b\n"},
		{"tabs kept", unexpand.Options{}, "\ta\n", "\ta\n"},
		{"no blanks", unexpand.Options{}, "abc\n", "abc\n"},
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

func TestInvalidTabs(t *testing.T) {
	_, errb, err := run(unexpand.Options{Tabs: []string{"0"}}, "x\n")
	if !errors.Is(err, core.ErrFailed) || errb != "unexpandx: tab size cannot be 0\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
}
