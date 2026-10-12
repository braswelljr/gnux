package hostname_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hostname"
)

func run(name string, lookupErr error, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	h := hostname.New(hostname.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	h.Lookup = func() (string, error) { return name, lookupErr }
	err := h.Run(args)
	return out.String(), errb.String(), err
}

func TestPrint(t *testing.T) {
	if out, _, err := run("build-01", nil); err != nil || out != "build-01\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestLookupFailure(t *testing.T) {
	out, errb, err := run("", errors.New("permission denied"))
	if !errors.Is(err, core.ErrFailed) || out != "" || errb != "hostnamex: cannot determine hostname: Permission denied\n" {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

func TestSetIsRefused(t *testing.T) {
	out, errb, err := run("old", nil, "new")
	if !errors.Is(err, core.ErrFailed) || out != "" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if want := "hostnamex: cannot set name to 'new': Operation not supported\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run("h", nil, "a", "b")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("err = %v", err)
	}
}

func TestRealHostname(t *testing.T) {
	var out bytes.Buffer
	err := hostname.New(hostname.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}}).Run(nil)
	if err != nil {
		t.Skipf("no host name here: %v", err)
	}
	if out.Len() < 2 {
		t.Fatalf("stdout = %q", out.String())
	}
}
