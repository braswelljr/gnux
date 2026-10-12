package csplitx

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/csplit"
)

func options(t *testing.T, args ...string) (csplit.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(csplit.Name, pflag.ContinueOnError)
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
		want csplit.Options
	}{
		{"defaults", nil, csplit.Options{}},
		{"-f", []string{"-f", "part"}, csplit.Options{Prefix: "part", PrefixSet: true}},
		{"empty -f", []string{"--prefix="}, csplit.Options{PrefixSet: true}},
		{"-b", []string{"-b", "%03d.txt"}, csplit.Options{SuffixFormat: "%03d.txt"}},
		{"-n", []string{"-n", "4"}, csplit.Options{Digits: 4}},
		{"-n 0", []string{"-n", "0"}, csplit.Options{Digits: 1}},
		{"-k -z", []string{"-kz"}, csplit.Options{KeepFiles: true, ElideEmpty: true}},
		{"-s", []string{"-s"}, csplit.Options{Quiet: true}},
		{"-q", []string{"-q"}, csplit.Options{Quiet: true}},
		{"--silent", []string{"--silent"}, csplit.Options{Quiet: true}},
		{"--suppress-matched", []string{"--suppress-matched"}, csplit.Options{SuppressMatched: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
	_, err := options(t, "-n", "x")
	var fatal *fatalError
	if !errors.As(err, &fatal) || fatal.msg != "invalid number: 'x'" {
		t.Fatalf("-n x: %v", err)
	}
}

func TestCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("a\n# one\nb\n# two\nc\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-z", "-f", "", "-", "/^#/", "{*}"})
	if err := c.Execute(); err != nil || out.String() != "2\n8\n8\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
	if data, _ := os.ReadFile("02"); string(data) != "# two\nc\n" {
		t.Fatalf("02 = %q", data)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != csplit.Name || !strings.HasPrefix(m.Use, csplit.Name+" ") {
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
