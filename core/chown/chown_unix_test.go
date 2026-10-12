//go:build unix

package chown_test

import (
	"bytes"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chown"
)

// nobody is an ID no account has, so it prints as a number.
const nobody = 4242424

type call struct {
	path     string
	uid, gid int
	follow   bool
}

// run executes chown with a recording ChownFn that changes nothing.
func run(t *testing.T, opts chown.Options, args ...string) (calls []call, stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := chown.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.ChownFn = func(p string, uid, gid int, follow bool) error {
		calls = append(calls, call{p, uid, gid, follow})
		return nil
	}
	err = c.Run(args)
	return calls, out.String(), errb.String(), err
}

func names(t *testing.T) (uname, gname string) {
	t.Helper()
	uname, gname = strconv.Itoa(os.Getuid()), strconv.Itoa(os.Getgid())
	if u, err := user.LookupId(uname); err == nil {
		uname = u.Username
	}
	if g, err := user.LookupGroupId(gname); err == nil {
		gname = g.Name
	}
	return uname, gname
}

func file(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOwnerAndGroup(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	uname, gname := names(t)
	spec := strconv.Itoa(nobody) + ":" + strconv.Itoa(nobody)

	calls, out, _, err := run(t, chown.Options{Verbose: true}, spec, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0] != (call{f, nobody, nobody, true}) {
		t.Fatalf("calls = %+v", calls)
	}
	want := "changed ownership of '" + f + "' from " + uname + ":" + gname + " to " + spec + "\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestUserOnlyAndGroupOnly(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	uname, gname := names(t)

	calls, out, _, _ := run(t, chown.Options{Changes: true}, strconv.Itoa(nobody), f)
	if calls[0].uid != nobody || calls[0].gid != -1 {
		t.Fatalf("calls = %+v", calls)
	}
	if want := "changed ownership of '" + f + "' from " + uname + " to " + strconv.Itoa(nobody) + "\n"; out != want {
		t.Fatalf("user only: %q, want %q", out, want)
	}

	calls, out, _, _ = run(t, chown.Options{Changes: true}, ":"+strconv.Itoa(nobody), f)
	if calls[0].uid != -1 || calls[0].gid != nobody {
		t.Fatalf("calls = %+v", calls)
	}
	if want := "changed group of '" + f + "' from " + gname + " to " + strconv.Itoa(nobody) + "\n"; out != want {
		t.Fatalf("group only: %q, want %q", out, want)
	}
}

func TestRetained(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	uname, gname := names(t)
	spec := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())

	_, out, _, _ := run(t, chown.Options{Verbose: true}, spec, f)
	if want := "ownership of '" + f + "' retained as " + uname + ":" + gname + "\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	_, out, _, _ = run(t, chown.Options{Changes: true}, spec, f)
	if out != "" {
		t.Fatalf("-c printed %q for an unchanged file", out)
	}
}

func TestLoginGroup(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skip("no current user")
	}
	f := file(t, t.TempDir(), "f")
	calls, _, _, err := run(t, chown.Options{}, u.Username+":", f)
	if err != nil {
		t.Fatal(err)
	}
	gid, _ := strconv.Atoi(u.Gid)
	if calls[0].gid != gid {
		t.Fatalf("OWNER: set gid %d, want the login group %d", calls[0].gid, gid)
	}
}

func TestFrom(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	calls, out, _, err := run(t, chown.Options{Verbose: true, From: strconv.Itoa(nobody)}, "0", f)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("--from mismatch still changed: %+v", calls)
	}
	if want := "ownership of '" + f + "' retained\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	calls, _, _, _ = run(t, chown.Options{From: strconv.Itoa(os.Getuid())}, strconv.Itoa(nobody), f)
	if len(calls) != 1 {
		t.Fatalf("--from match did not change: %+v", calls)
	}
}

func TestSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := file(t, dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		opts   chown.Options
		follow bool
	}{
		{"default follows", chown.Options{}, true},
		{"-h changes the link", chown.Options{NoDereference: true}, false},
		{"-R defaults to -P", chown.Options{Recursive: true}, false},
		{"-R -H follows an operand", chown.Options{Recursive: true, Traversal: chown.TraverseCommandLine}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, _, _, err := run(t, tt.opts, strconv.Itoa(nobody), link)
			if err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 || calls[0].follow != tt.follow {
				t.Fatalf("calls = %+v, want follow=%v", calls, tt.follow)
			}
		})
	}
}

func TestRecursive(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "d")
	if err := os.MkdirAll(filepath.Join(d, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	file(t, d, "a")
	file(t, filepath.Join(d, "sub"), "b")
	calls, _, _, err := run(t, chown.Options{Recursive: true}, strconv.Itoa(nobody), d)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range calls {
		rel, _ := filepath.Rel(root, c.path)
		got = append(got, rel)
	}
	if want := "d d/a d/sub d/sub/b"; strings.Join(got, " ") != filepath.FromSlash(want) {
		t.Fatalf("visited %v, want %s", got, want)
	}
}

func TestFailureAndMissing(t *testing.T) {
	dir := t.TempDir()
	f := file(t, dir, "f")
	missing := filepath.Join(dir, "missing")
	var errb bytes.Buffer
	c := chown.New(chown.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	c.ChownFn = func(p string, _, _ int, _ bool) error {
		return &os.PathError{Op: "chown", Path: p, Err: syscall.EPERM}
	}
	if err := c.Run([]string{"0", missing, f}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	want := "chownx: cannot access '" + missing + "': No such file or directory\n" +
		"chownx: changing ownership of '" + f + "': Operation not permitted\n"
	if errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestReference(t *testing.T) {
	dir := t.TempDir()
	ref, f := file(t, dir, "ref"), file(t, dir, "f")
	calls, _, _, err := run(t, chown.Options{Reference: ref}, f)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].uid != os.Getuid() || calls[0].gid != os.Getgid() {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestPreserveRoot(t *testing.T) {
	calls, _, _, err := run(t, chown.Options{Recursive: true}, "0", "/")
	if !errors.Is(err, core.ErrFailed) || len(calls) != 0 {
		t.Fatalf("err = %v, calls = %d", err, len(calls))
	}
}
