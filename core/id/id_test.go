package id_test

import (
	"bytes"
	"errors"
	"os/user"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/id"
)

var (
	users  = map[string]string{"501": "ann", "0": "root"}
	groups = map[string]string{"20": "staff", "12": "everyone", "80": "admin", "0": "wheel"}
	self   = id.Identity{UID: "501", GID: "20", EUID: "501", EGID: "20", Groups: []string{"20", "12", "80", "77"}}
	setuid = id.Identity{UID: "501", GID: "20", EUID: "0", EGID: "0", Groups: []string{"20", "12"}}
)

func run(t *testing.T, opts id.Options, me id.Identity, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	e := id.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	e.Self = func() (id.Identity, error) { return me, nil }
	e.Lookup = func(name string) (id.Identity, error) {
		switch name {
		case "root", "0":
			return id.Identity{UID: "0", GID: "0", EUID: "0", EGID: "0", Groups: []string{"0", "80"}}, nil
		case "ann":
			return self, nil
		}
		return id.Identity{}, errors.New("unknown")
	}
	e.UserName = func(s string) (string, bool) { n, ok := users[s]; return n, ok }
	e.GroupName = func(s string) (string, bool) { n, ok := groups[s]; return n, ok }
	err := e.Run(args)
	return out.String(), errb.String(), err
}

func TestDefaultFormat(t *testing.T) {
	out, errb, err := run(t, id.Options{}, self)
	if err != nil || errb != "" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if want := "uid=501(ann) gid=20(staff) groups=20(staff),12(everyone),80(admin),77\n"; out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}

	out, _, _ = run(t, id.Options{}, setuid)
	if want := "uid=501(ann) gid=20(staff) euid=0(root) egid=0(wheel) groups=20(staff),0(wheel),12(everyone)\n"; out != want {
		t.Fatalf("setuid: out = %q, want %q", out, want)
	}
}

func TestSingleIDs(t *testing.T) {
	tests := []struct {
		name string
		opts id.Options
		me   id.Identity
		want string
	}{
		{"user", id.Options{User: true}, self, "501\n"},
		{"user name", id.Options{User: true, Name: true}, self, "ann\n"},
		{"effective user", id.Options{User: true}, setuid, "0\n"},
		{"real user", id.Options{User: true, Real: true}, setuid, "501\n"},
		{"group", id.Options{Group: true}, self, "20\n"},
		{"group name", id.Options{Group: true, Name: true}, self, "staff\n"},
		{"effective group", id.Options{Group: true, Name: true}, setuid, "wheel\n"},
		{"real group", id.Options{Group: true, Real: true}, setuid, "20\n"},
		{"groups", id.Options{Groups: true}, self, "20 12 80 77\n"},
		{"groups with effective", id.Options{Groups: true}, setuid, "20 0 12\n"},
		{"zero user", id.Options{User: true, Zero: true}, self, "501\x00"},
		{"zero groups", id.Options{Groups: true, Zero: true, Name: true}, setuid, "staff\x00wheel\x00everyone\x00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(t, tt.opts, tt.me)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v (stderr %q), want %q", out, err, errb, tt.want)
			}
		})
	}
}

func TestMissingNames(t *testing.T) {
	out, errb, err := run(t, id.Options{Groups: true, Name: true}, self)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if out != "staff everyone admin 77\n" || errb != "idx: cannot find name for group ID 77\n" {
		t.Fatalf("out = %q, stderr = %q", out, errb)
	}

	nobody := id.Identity{UID: "999", GID: "20", EUID: "999", EGID: "20"}
	out, errb, err = run(t, id.Options{User: true, Name: true}, nobody)
	if !errors.Is(err, core.ErrFailed) || out != "999\n" || errb != "idx: cannot find name for user ID 999\n" {
		t.Fatalf("user: out = %q, stderr = %q, err = %v", out, errb, err)
	}

	out, _, err = run(t, id.Options{}, nobody)
	if err != nil || out != "uid=999 gid=20(staff) groups=20(staff)\n" {
		t.Fatalf("default format: out = %q, err = %v", out, err)
	}
}

func TestUserOperands(t *testing.T) {
	out, errb, err := run(t, id.Options{}, self, "root", "nosuch", "ann")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	want := "uid=0(root) gid=0(wheel) groups=0(wheel),80(admin)\n" +
		"uid=501(ann) gid=20(staff) groups=20(staff),12(everyone),80(admin),77\n"
	if out != want || errb != "idx: 'nosuch': no such user\n" {
		t.Fatalf("out = %q, stderr = %q", out, errb)
	}

	out, _, err = run(t, id.Options{Groups: true, Name: true}, self, "root")
	if err != nil || out != "wheel admin\n" {
		t.Fatalf("-Gn root: out = %q, err = %v", out, err)
	}
}

func TestSelfError(t *testing.T) {
	var out, errb bytes.Buffer
	e := id.New(id.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	e.Self = func() (id.Identity, error) { return id.Identity{}, errors.New("no account database") }
	if err := e.Run(nil); !errors.Is(err, core.ErrFailed) || errb.String() != "idx: cannot get the current user's IDs: No account database\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		opts id.Options
		want string
	}{
		{id.Options{User: true, Group: true}, "cannot print \"only\" of more than one choice"},
		{id.Options{Groups: true, User: true}, "cannot print \"only\" of more than one choice"},
		{id.Options{Name: true}, "cannot print only names or real IDs in default format"},
		{id.Options{Real: true}, "cannot print only names or real IDs in default format"},
		{id.Options{Zero: true}, "option --zero not permitted in default format"},
	}
	for _, tt := range tests {
		_, _, err := run(t, tt.opts, self)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Errorf("%+v: err = %v, want %q", tt.opts, err, tt.want)
		}
	}
}

// TestRealProcess runs the engine with its default sources against the
// running process, which must at least print a uid.
func TestRealProcess(t *testing.T) {
	if _, err := user.Current(); err != nil {
		t.Skipf("no current user: %v", err)
	}
	var out, errb bytes.Buffer
	if err := id.New(id.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(nil); err != nil {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
	if !strings.HasPrefix(out.String(), "uid=") {
		t.Fatalf("out = %q", out.String())
	}
	out.Reset()
	if err := id.New(id.Options{User: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(nil); err != nil || strings.TrimSpace(out.String()) == "" {
		t.Fatalf("-u: out = %q, err = %v", out.String(), err)
	}
}
