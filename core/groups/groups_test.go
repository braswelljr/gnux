package groups_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/groups"
	"github.com/braswelljr/gnux/core/id"
)

var names = map[string]string{"10": "staff", "20": "admin", "30": "docker"}

func run(args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	g := groups.New(groups.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	g.Self = func() (id.Identity, error) {
		return id.Identity{UID: "1", GID: "10", EUID: "1", EGID: "20", Groups: []string{"30", "10", "99"}}, nil
	}
	g.Lookup = func(name string) (id.Identity, error) {
		if name != "alice" {
			return id.Identity{}, errors.New("unknown user")
		}
		return id.Identity{UID: "2", GID: "30", EUID: "2", EGID: "30", Groups: []string{"10", "30"}}, nil
	}
	g.GroupName = func(gid string) (string, bool) {
		n, ok := names[gid]
		return n, ok
	}
	err = g.Run(args)
	return out.String(), errb.String(), err
}

func TestSelf(t *testing.T) {
	out, errOut, err := run()
	if want := "staff admin docker 99\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	if want := "groupsx: cannot find name for group ID 99\n"; errOut != want || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
}

func TestUsers(t *testing.T) {
	out, errOut, err := run("alice", "nobody", "alice")
	if want := "alice : docker staff\nalice : docker staff\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	if errOut != "groupsx: 'nobody': no such user\n" || !errors.Is(err, core.ErrFailed) {
		t.Fatalf("stderr = %q, err = %v", errOut, err)
	}
}

func TestSelfFails(t *testing.T) {
	var errb bytes.Buffer
	g := groups.New(groups.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	g.Self = func() (id.Identity, error) { return id.Identity{}, errors.New("boom") }
	if err := g.Run(nil); !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb.String(), "groupsx: cannot get the current user's groups: ") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

// TestReal runs against the real account database: the process is in at
// least its own group.
func TestReal(t *testing.T) {
	var out bytes.Buffer
	if err := groups.New(groups.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}}).Run(nil); err != nil {
		t.Skipf("no account database here: %v", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("no groups printed")
	}
}
