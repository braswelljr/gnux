package whoami_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/whoami"
)

func run(name string, nameErr error, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	w := whoami.New(whoami.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	w.Username = func() (string, error) { return name, nameErr }
	w.UID = func() int { return 1001 }
	err := w.Run(args)
	return out.String(), errb.String(), err
}

func TestWhoami(t *testing.T) {
	tests := map[string]string{"alice": "alice\n", `CORP\bob`: "bob\n", "user.name": "user.name\n"}
	for name, want := range tests {
		if out, _, err := run(name, nil); err != nil || out != want {
			t.Fatalf("%q: out = %q, err = %v, want %q", name, out, err, want)
		}
	}
}

func TestUnknownUser(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{{"", errors.New("unknown userid")}, {"", nil}} {
		out, errb, err := run(tc.name, tc.err)
		if !errors.Is(err, core.ErrFailed) || out != "" || errb != "whoamix: cannot find name for user ID 1001\n" {
			t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
		}
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run("alice", nil, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}

func TestRealUser(t *testing.T) {
	var out, errb bytes.Buffer
	err := whoami.New(whoami.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(nil)
	if err != nil {
		t.Skipf("no user name available here: %v (%s)", err, errb.String())
	}
	if !strings.HasSuffix(out.String(), "\n") || len(out.String()) < 2 {
		t.Fatalf("stdout = %q", out.String())
	}
}
