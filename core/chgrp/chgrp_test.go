package chgrp_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chgrp"
)

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts chgrp.Options
		args []string
		want string
	}{
		{"no operands", chgrp.Options{}, nil, "missing operand"},
		{"group only", chgrp.Options{}, []string{"0"}, "missing operand after '0'"},
		{"reference without files", chgrp.Options{Reference: "."}, nil, "missing operand"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chgrp.New(tt.opts, core.Discard()).Run(tt.args)
			var usage *core.UsageError
			if !errors.As(err, &usage) || usage.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", err, tt.want)
			}
		})
	}
}

func TestInvalidGroup(t *testing.T) {
	var errb bytes.Buffer
	err := chgrp.New(chgrp.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb}).
		Run([]string{"no-such-group-gnux", "f"})
	if !errors.Is(err, core.ErrFailed) || errb.String() != "chgrpx: invalid group: 'no-such-group-gnux'\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestReferenceMissing(t *testing.T) {
	var errb bytes.Buffer
	err := chgrp.New(chgrp.Options{Reference: "no-such-file-gnux"}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb}).
		Run([]string{"f"})
	if !errors.Is(err, core.ErrFailed) || errb.String() != "chgrpx: failed to get attributes of 'no-such-file-gnux': No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestRecursiveDereferenceNeedsTraversal(t *testing.T) {
	var errb bytes.Buffer
	err := chgrp.New(chgrp.Options{Recursive: true, Dereference: true}, core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: &errb}).Run([]string{"0", "f"})
	if !errors.Is(err, core.ErrFailed) || errb.String() != "chgrpx: -R --dereference requires either -H or -L\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}
