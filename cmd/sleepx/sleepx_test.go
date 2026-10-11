package sleepx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sleep"
)

func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	// cobra falls back to os.Args for a nil slice.
	c.SetArgs(append([]string{}, args...))
	err = c.Execute()
	return out.String(), errb.String(), err
}

func TestCommandSleeps(t *testing.T) {
	// Zero-length intervals exercise the whole path without slowing the suite.
	for _, args := range [][]string{{"0"}, {"0s", "0m", "0.0h"}, {"--", "0"}} {
		stdout, stderr, err := execute(t, args...)
		if err != nil {
			t.Errorf("Execute(%q): %v", args, err)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("Execute(%q): stdout = %q, stderr = %q, want both empty", args, stdout, stderr)
		}
	}
}

func TestCommandUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing operand", nil, "missing operand"},
		{"invalid interval", []string{"1", "soon"}, "invalid time interval 'soon'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := execute(t, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) || usage.Msg != tt.want {
				t.Fatalf("Execute(%q) = %v, want usage error %q", tt.args, err, tt.want)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	stdout, _, err := execute(t, "--version")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(stdout, sleep.Name+" ") {
		t.Fatalf("stdout = %q, want a version line for %s", stdout, sleep.Name)
	}
}
