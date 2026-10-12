package seqx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/seq"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		opts     seq.Options
		operands []string
	}{
		{"defaults", []string{"5"}, seq.Options{}, []string{"5"}},
		{"no arguments", nil, seq.Options{}, []string{}},
		{"negative first", []string{"-5", "5"}, seq.Options{}, []string{"-5", "5"}},
		{"negative increment", []string{"10", "-2", "1"}, seq.Options{}, []string{"10", "-2", "1"}},
		{"negative decimal", []string{"-.5", "1"}, seq.Options{}, []string{"-.5", "1"}},
		{"option then negative", []string{"-w", "-5", "5"}, seq.Options{EqualWidth: true}, []string{"-5", "5"}},
		{"equal width", []string{"-w", "10"}, seq.Options{EqualWidth: true}, []string{"10"}},
		{"equal width long", []string{"--equal-width", "10"}, seq.Options{EqualWidth: true}, []string{"10"}},
		{"long prefix", []string{"--equal", "10"}, seq.Options{EqualWidth: true}, []string{"10"}},
		{"format", []string{"-f", "%.1f", "3"}, seq.Options{Format: "%.1f"}, []string{"3"}},
		{"format attached", []string{"-f%g", "3"}, seq.Options{Format: "%g"}, []string{"3"}},
		{"format long", []string{"--format=%e", "3"}, seq.Options{Format: "%e"}, []string{"3"}},
		{"format long separate", []string{"--format", "%e", "3"}, seq.Options{Format: "%e"}, []string{"3"}},
		{"separator", []string{"-s", ",", "3"}, seq.Options{Separator: ",", SeparatorSet: true}, []string{"3"}},
		{"separator attached", []string{"-s:", "3"}, seq.Options{Separator: ":", SeparatorSet: true}, []string{"3"}},
		{"empty separator", []string{"-s", "", "3"}, seq.Options{SeparatorSet: true}, []string{"3"}},
		{"separator long", []string{"--separator=;", "3"}, seq.Options{Separator: ";", SeparatorSet: true}, []string{"3"}},
		{"separator that looks negative", []string{"-s", "-1", "3"}, seq.Options{Separator: "-1", SeparatorSet: true}, []string{"3"}},
		{"cluster", []string{"-ws,", "3"}, seq.Options{EqualWidth: true, Separator: ",", SeparatorSet: true}, []string{"3"}},
		{"double dash", []string{"--", "3"}, seq.Options{}, []string{"3"}},
		{"options end at the first operand", []string{"1", "-w"}, seq.Options{}, []string{"1", "-w"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%q): %v", tt.args, err)
			}
			if p.opts != tt.opts {
				t.Fatalf("opts = %+v, want %+v", p.opts, tt.opts)
			}
			if len(p.operands) != len(tt.operands) || (len(tt.operands) > 0 && !reflect.DeepEqual(p.operands, tt.operands)) {
				t.Fatalf("operands = %q, want %q", p.operands, tt.operands)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-x", "1"}, "invalid option -- 'x'"},
		{[]string{"-f"}, "option requires an argument -- 'f'"},
		{[]string{"-s"}, "option requires an argument -- 's'"},
		{[]string{"--format"}, "option '--format' requires an argument"},
		{[]string{"--equal-width=yes", "1"}, "option '--equal-width' doesn't allow an argument"},
		{[]string{"--bogus", "1"}, "unrecognized option '--bogus'"},
		{[]string{"-inf"}, "invalid option -- 'i'"},
	}
	for _, tt := range tests {
		_, err := parseArgs(tt.args)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Errorf("parseArgs(%q) = %v, want the usage error %q", tt.args, err, tt.want)
		}
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"last", []string{"3"}, "1\n2\n3\n"},
		{"negative operands", []string{"-1", "1"}, "-1\n0\n1\n"},
		{"count down", []string{"3", "-1", "1"}, "3\n2\n1\n"},
		{"separator", []string{"-s", ",", "3"}, "1,2,3\n"},
		{"equal width", []string{"-w", "9", "10"}, "09\n10\n"},
		{"format", []string{"-f", "%.1f", "2"}, "1.0\n2.0\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v (stderr %q)", tt.args, err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-w", "--help"}, {"--version"}, {"--vers"}} {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatalf("Execute(%q): %v", args, err)
		}
		if !strings.Contains(out.String(), seq.Name) {
			t.Fatalf("Execute(%q) stdout = %q, want help or version text", args, out.String())
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != seq.Name || !strings.HasPrefix(m.Use, seq.Name+" ") {
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
