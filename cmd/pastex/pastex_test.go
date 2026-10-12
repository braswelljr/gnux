package pastex

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/paste"
)

func options(t *testing.T, args ...string) paste.Options {
	t.Helper()
	fs := pflag.NewFlagSet(paste.Name, pflag.ContinueOnError)
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
		want paste.Options
	}{
		{"defaults", nil, paste.Options{}},
		{"delimiters", []string{"-d", ","}, paste.Options{Delimiters: ",", DelimitersSet: true}},
		{"delimiters attached", []string{"-d,;"}, paste.Options{Delimiters: ",;", DelimitersSet: true}},
		{"delimiters long", []string{"--delimiters=x"}, paste.Options{Delimiters: "x", DelimitersSet: true}},
		{"empty delimiters", []string{"--delimiters="}, paste.Options{DelimitersSet: true}},
		{"serial", []string{"-s"}, paste.Options{Serial: true}},
		{"serial long", []string{"--serial"}, paste.Options{Serial: true}},
		{"zero terminated", []string{"-z"}, paste.Options{ZeroTerminated: true}},
		{"zero terminated long", []string{"--zero-terminated"}, paste.Options{ZeroTerminated: true}},
		{"combined", []string{"-sz", "-d", "+"}, paste.Options{Serial: true, ZeroTerminated: true, Delimiters: "+", DelimitersSet: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options(t, tt.args...); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "1\n2\n3\n"},
		{"pairs", []string{"-", "-"}, "1\t2\n3\t\n"},
		{"serial", []string{"-s", "-d", ","}, "1,2,3\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("1\n2\n3\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != paste.Name || !strings.HasPrefix(m.Use, paste.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatal("Meta is missing prose")
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
