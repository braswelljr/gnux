package users_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/users"
)

func run(recs []users.Record, readErr error, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	u := users.New(users.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	u.Current = func() ([]users.Record, error) { return recs, readErr }
	u.ReadFile = func(path string) ([]users.Record, error) {
		if path != "wtmp" {
			return nil, os.ErrNotExist
		}
		return []users.Record{{User: "zed", Login: true}, {User: "amy", Login: true}}, nil
	}
	err = u.Run(args)
	return out.String(), errb.String(), err
}

func TestUsers(t *testing.T) {
	recs := []users.Record{
		{User: "reboot"},
		{User: "carol", Login: true},
		{User: "alice", Login: true},
		{User: "LOGIN"},
		{User: "", Login: true},
		{User: "alice  ", Login: true},
		{User: "Bob", Login: true},
	}
	out, _, err := run(recs, nil)
	if want := "Bob alice alice carol\n"; err != nil || out != want {
		t.Fatalf("out = %q, err = %v; want %q", out, err, want)
	}
}

func TestNobody(t *testing.T) {
	out, errOut, err := run(nil, nil)
	if out != "" || errOut != "" || err != nil {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errOut, err)
	}
}

func TestFile(t *testing.T) {
	if out, _, err := run(nil, nil, "wtmp"); err != nil || out != "amy zed\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	_, errOut, err := run(nil, nil, "missing")
	if errOut != "usersx: missing: No such file or directory\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
}

func TestRecordsFail(t *testing.T) {
	_, errOut, err := run(nil, os.ErrPermission)
	if errOut != "usersx: cannot read the login records: Permission denied\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
}

func TestExtraOperand(t *testing.T) {
	_, _, err := run(nil, nil, "a", "b")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("err = %v", err)
	}
}

func TestReal(t *testing.T) {
	var out bytes.Buffer
	if err := users.New(users.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}}).Run(nil); err != nil {
		t.Skipf("no login records here: %v", err)
	}
	if s := out.String(); s != "" && !strings.HasSuffix(s, "\n") {
		t.Fatalf("stdout = %q", s)
	}
}

func TestSelfColored(t *testing.T) {
	var out bytes.Buffer
	u := users.New(users.Options{Pretty: true, Color: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	u.Current = func() ([]users.Record, error) {
		return []users.Record{{User: "zed", Login: true}, {User: "amy", Login: true}}, nil
	}
	u.Self = func() string { return "zed" }
	u.Getenv = func(string) string { return "" }
	if err := u.Run(nil); err != nil {
		t.Fatal(err)
	}
	// The text is GNU's: sorted, one space apart; only the caller's name
	// is colored (the default theme gives others none).
	if want := "amy \x1b[1;33mzed\x1b[0m\n"; out.String() != want {
		t.Fatalf("out = %q; want %q", out.String(), want)
	}
}
