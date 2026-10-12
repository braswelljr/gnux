package nlx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nl"
)

func options(t *testing.T, args ...string) (nl.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(nl.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want nl.Options
	}{
		{"defaults", nil, nl.Options{}},
		{"styles", []string{"-b", "a", "-h", "t", "-f", "p^x"}, nl.Options{BodyStyle: "a", HeaderStyle: "t", FooterStyle: "p^x"}},
		{"delimiter", []string{"-d", "@@"}, nl.Options{Delimiter: "@@", DelimiterSet: true}},
		{"empty delimiter", []string{"--section-delimiter="}, nl.Options{DelimiterSet: true}},
		{"increment", []string{"-i", "-2"}, nl.Options{Increment: -2, IncrementSet: true}},
		{"zero increment", []string{"-i", "0"}, nl.Options{IncrementSet: true}},
		{"join", []string{"-l", "3"}, nl.Options{JoinBlank: 3}},
		{"format", []string{"-n", "rz"}, nl.Options{Format: "rz"}},
		{"no renumber", []string{"-p"}, nl.Options{NoRenumber: true}},
		{"separator", []string{"-s", ": "}, nl.Options{Separator: ": ", SeparatorSet: true}},
		{"start", []string{"-v", "0"}, nl.Options{StartingLineSet: true}},
		{"width", []string{"-w", "3"}, nl.Options{Width: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-i", "x"}, "invalid line number increment: 'x'"},
		{[]string{"-v", "1.5"}, "invalid starting line number: '1.5'"},
		{[]string{"-l", "0"}, "invalid line number of blank lines: '0'"},
		{[]string{"-w", "0"}, "invalid line number field width: '0'"},
		{[]string{"-w", "x"}, "invalid line number field width: 'x'"},
	} {
		_, err := options(t, tt.args...)
		var fatal *fatalError
		if !errors.As(err, &fatal) || fatal.msg != tt.msg {
			t.Errorf("%v: %v, want %q", tt.args, err, tt.msg)
		}
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("a\nb\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-w", "2", "-s", ". "})
	if err := c.Execute(); err != nil || out.String() != " 1. a\n 2. b\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != nl.Name || !strings.HasPrefix(m.Use, nl.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example %+v", e)
		}
	}
}
