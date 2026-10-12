package printfx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printf"
)

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"text", []string{`hi\n`}, "hi\n"},
		{"conversions", []string{"%s=%d\n", "a", "1"}, "a=1\n"},
		{"negative argument is not an option", []string{"%d\n", "-5"}, "-5\n"},
		{"format that looks like an option", []string{"-x"}, "-x"},
		{"double dash", []string{"--", "%s", "x"}, "x"},
		{"--help after a format is an argument", []string{"%s", "--help"}, "--help"},
		{"reused format", []string{"%s,", "a", "b"}, "a,b,"},
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

func TestLoneHelpAndVersion(t *testing.T) {
	for _, arg := range []string{"--help", "--version"} {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
		c.SetArgs([]string{arg})
		if err := c.Execute(); err != nil {
			t.Fatalf("Execute(%s): %v", arg, err)
		}
		if !strings.Contains(out.String(), printf.Name) {
			t.Fatalf("Execute(%s) stdout = %q", arg, out.String())
		}
	}
}

func TestMissingOperand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{})
	var usage *core.UsageError
	if err := c.Execute(); !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("Execute() = %v, want the usage error 'missing operand'", err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != printf.Name || !strings.HasPrefix(m.Use, printf.Name+" ") {
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
