package falsex

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no arguments", []string{}},
		{"operands are ignored", []string{"a", "b"}},
		{"unknown options are ignored", []string{"-x", "--nope"}},
		{"--help with other arguments is ignored", []string{"--help", "x"}},
		{"double dash is ignored", []string{"--"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			err := c.Execute()
			var exit *core.ExitError
			if !errors.As(err, &exit) || exit.Code != 1 {
				t.Fatalf("Execute(%q) = %v, want exit status 1", tt.args, err)
			}
			if out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
			}
		})
	}
}

func TestLoneVersion(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
	c.SetArgs([]string{"--version"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v, want nil (version exits 0)", err)
	}
	if !strings.HasPrefix(out.String(), Name+" ") {
		t.Fatalf("output = %q, want a version line for %s", out.String(), Name)
	}
}
