package chown_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chown"
)

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts chown.Options
		args []string
		want string
	}{
		{"no operands", chown.Options{}, nil, "missing operand"},
		{"spec only", chown.Options{}, []string{"0:0"}, "missing operand after '0:0'"},
		{"recursive dereference without -H or -L", chown.Options{Recursive: true, Dereference: true}, []string{"0", "f"}, "-R --dereference requires either -H or -L"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chown.New(tt.opts, core.Discard()).Run(tt.args)
			var usage *core.UsageError
			if !errors.As(err, &usage) || usage.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", err, tt.want)
			}
		})
	}
}

func TestInvalidSpec(t *testing.T) {
	for spec, want := range map[string]string{
		"no-such-user-gnux:0":  "chownx: invalid user: 'no-such-user-gnux:0'\n",
		"0:no-such-group-gnux": "chownx: invalid group: '0:no-such-group-gnux'\n",
	} {
		var errb bytes.Buffer
		err := chown.New(chown.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb}).Run([]string{spec, "f"})
		if !errors.Is(err, core.ErrFailed) || errb.String() != want {
			t.Fatalf("%s: err = %v, stderr = %q, want %q", spec, err, errb.String(), want)
		}
	}
}
