package trx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tr"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) tr.Options {
	t.Helper()
	fs := pflag.NewFlagSet(tr.Name, pflag.ContinueOnError)
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
		want tr.Options
	}{
		{"defaults", nil, tr.Options{}},
		{"complement", []string{"-c"}, tr.Options{Complement: true}},
		{"complement upper", []string{"-C"}, tr.Options{Complement: true}},
		{"complement long", []string{"--complement"}, tr.Options{Complement: true}},
		{"both complements", []string{"-cC"}, tr.Options{Complement: true}},
		{"delete", []string{"-d"}, tr.Options{Delete: true}},
		{"delete long", []string{"--delete"}, tr.Options{Delete: true}},
		{"squeeze", []string{"-s"}, tr.Options{Squeeze: true}},
		{"squeeze long", []string{"--squeeze-repeats"}, tr.Options{Squeeze: true}},
		{"truncate", []string{"-t"}, tr.Options{Truncate: true}},
		{"truncate long", []string{"--truncate-set1"}, tr.Options{Truncate: true}},
		{"combined", []string{"-cds"}, tr.Options{Complement: true, Delete: true, Squeeze: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options(t, tt.args...); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"translate", []string{"a-z", "A-Z"}, "HELLO, WORLD\n"},
		{"delete", []string{"-d", "lo"}, "he, wrd\n"},
		{"squeeze", []string{"-s", "l"}, "helo, world\n"},
		{"words per line", []string{"-cs", "[:alpha:]", `\n`}, "hello\nworld\n"},
		// A SET that starts with '-' follows the end of options.
		{"dash set", []string{"--", "-,", "_"}, "hello_ world\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("hello, world\n"), Out: &out, Err: &errb})
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
	if m.Name != tr.Name || !strings.HasPrefix(m.Use, tr.Name+" ") {
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
