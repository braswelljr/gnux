package logname_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/logname"
)

func run(name string, ok bool, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	l := logname.New(logname.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	l.Login = func(any) (string, bool) { return name, ok }
	err = l.Run(args)
	return out.String(), errb.String(), err
}

func TestLogin(t *testing.T) {
	if out, _, err := run("alice", true); err != nil || out != "alice\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestNoLogin(t *testing.T) {
	for _, tc := range []struct {
		name string
		ok   bool
	}{{"", false}, {"", true}} {
		out, errOut, err := run(tc.name, tc.ok)
		if out != "" || errOut != "lognamex: no login name\n" || !errors.Is(err, core.ErrFailed) {
			t.Fatalf("out = %q, stderr = %q, err = %v", out, errOut, err)
		}
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run("alice", true, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}

// TestNoTerminal runs the real lookup with standard input that is not a
// terminal: it may find a login name (Linux records one per login) or
// report none, but it must not fail any other way.
func TestNoTerminal(t *testing.T) {
	var out, errb bytes.Buffer
	err := logname.New(logname.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(nil)
	switch {
	case err == nil && strings.HasSuffix(out.String(), "\n") && len(out.String()) > 1:
	case errors.Is(err, core.ErrFailed) && errb.String() == "lognamex: no login name\n":
	default:
		t.Fatalf("out = %q, stderr = %q, err = %v", out.String(), errb.String(), err)
	}
}
