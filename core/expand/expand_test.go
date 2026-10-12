package expand_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expand"
)

func run(opts expand.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = expand.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name string
		opts expand.Options
		in   string
		want string
	}{
		{"default", expand.Options{}, "a\tb\n", "a       b\n"},
		{"leading tab", expand.Options{}, "\tx\n", "        x\n"},
		{"size", expand.Options{Tabs: []string{"4"}}, "a\tb\n", "a   b\n"},
		{"list", expand.Options{Tabs: []string{"2,5"}}, "\ta\tb\n", "  a  b\n"},
		{"list in two specs", expand.Options{Tabs: []string{"2", "5"}}, "\ta\tb\n", "  a  b\n"},
		{"blank separated list", expand.Options{Tabs: []string{"2 5"}}, "\ta\tb\n", "  a  b\n"},
		{"past the last stop", expand.Options{Tabs: []string{"2"}}, "abc\td\n", "abc d\n"},
		{"past the list", expand.Options{Tabs: []string{"2,4"}}, "abcd\te\n", "abcd e\n"},
		{"extend", expand.Options{Tabs: []string{"2,/5"}}, "\ta\tb\n", "  a  b\n"},
		{"increment", expand.Options{Tabs: []string{"2,+5"}}, "\ta\tb\n", "  a    b\n"},
		{"initial", expand.Options{Initial: true}, "\ta\tb\n", "        a\tb\n"},
		{"backspace", expand.Options{Tabs: []string{"4"}}, "ab\b\tc\n", "ab\b   c\n"},
		{"no final newline", expand.Options{Tabs: []string{"2"}}, "\t", "  "},
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
	tests := []struct {
		tabs []string
		msg  string
	}{
		{[]string{"0"}, "tab size cannot be 0"},
		{[]string{"4,2"}, "tab sizes must be ascending"},
		{[]string{"a"}, "tab size contains invalid character(s): 'a'"},
		{[]string{"2,/4,/8"}, "'/' specifier only allowed with the last value"},
		{[]string{"4/"}, "'/' specifier not at start of number: '/'"},
		{[]string{"2,/4", "+8"}, "'/' specifier is mutually exclusive with '+'"},
		{[]string{"99999999999999999999"}, "tab stop is too large '99999999999999999999'"},
	}
	for _, tt := range tests {
		out, errb, err := run(expand.Options{Tabs: tt.tabs}, "\tx\n")
		if !errors.Is(err, core.ErrFailed) || errb != "expandx: "+tt.msg+"\n" || out != "" {
			t.Errorf("%q: err = %v, stderr = %q, out = %q", tt.tabs, err, errb, out)
		}
	}
}

func TestMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	out, errb, err := run(expand.Options{}, "\tx\n", missing, "-")
	if !errors.Is(err, core.ErrFailed) || errb != "expandx: "+missing+": No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "        x\n" {
		t.Fatalf("the remaining operand was not expanded: %q", out)
	}
}
