package echox

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/echo"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", []string{}, "\n"},
		{"operands", []string{"a", "b"}, "a b\n"},
		{"-n", []string{"-n", "a"}, "a"},
		{"-e", []string{"-e", `a\tb`}, "a\tb\n"},
		{"-ne cluster", []string{"-ne", `a\n`}, "a\n"},
		{"-E", []string{"-e", "-E", `a\tb`}, `a\tb` + "\n"},
		{"unknown option is text", []string{"-x"}, "-x\n"},
		{"double dash is text", []string{"--"}, "--\n"},
		{"double dash after -n", []string{"-n", "--", "a"}, "-- a"},
		{"--help with operands is text", []string{"--help", "me"}, "--help me\n"},
		{"--version after an operand is text", []string{"a", "--version"}, "a --version\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if out.String() != tt.want {
				t.Errorf("stdout = %q, want %q", out.String(), tt.want)
			}
			if errb.Len() != 0 {
				t.Errorf("stderr = %q, want empty", errb.String())
			}
		})
	}
}

func TestLoneVersion(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"--version"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.String(), echo.Name+" ") {
		t.Fatalf("stdout = %q, want a version line for %s", out.String(), echo.Name)
	}
}
