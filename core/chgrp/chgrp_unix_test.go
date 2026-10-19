//go:build unix

package chgrp_test

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chgrp"
)

// nobody is an ID no group has, so it prints as a number.
const nobody = 4242424

type call struct {
	path     string
	uid, gid int
	follow   bool
}

func run(t *testing.T, opts chgrp.Options, args ...string) (calls []call, stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := chgrp.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.ChownFn = func(p string, uid, gid int, follow bool) error {
		calls = append(calls, call{p, uid, gid, follow})
		return nil
	}
	err = c.Run(args)
	return calls, out.String(), errb.String(), err
}

func TestGroup(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gname := strconv.Itoa(os.Getgid())
	if g, err := user.LookupGroupId(gname); err == nil {
		gname = g.Name
	}

	calls, out, _, err := run(t, chgrp.Options{Changes: true}, strconv.Itoa(nobody), f)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0] != (call{f, -1, nobody, true}) {
		t.Fatalf("calls = %+v", calls)
	}
	if want := "changed group of '" + f + "' from " + gname + " to " + strconv.Itoa(nobody) + "\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}

	// The current group, with -v, is retained.
	_, out, _, _ = run(t, chgrp.Options{Verbose: true}, strconv.Itoa(os.Getgid()), f)
	if want := "group of '" + f + "' retained as " + gname + "\n"; out != want {
		t.Fatalf("retained: %q, want %q", out, want)
	}
}

func TestRecursiveAndNoDereference(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "d")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(sub, "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(f, link); err != nil {
		t.Fatal(err)
	}

	calls, _, _, err := run(t, chgrp.Options{Recursive: true}, strconv.Itoa(nobody), sub)
	if err != nil || len(calls) != 2 || calls[0].path != sub || calls[1].path != f {
		t.Fatalf("-R: calls = %+v, err = %v", calls, err)
	}

	calls, _, _, _ = run(t, chgrp.Options{NoDereference: true}, strconv.Itoa(nobody), link)
	if len(calls) != 1 || calls[0].follow {
		t.Fatalf("-h: calls = %+v", calls)
	}
	calls, _, _, _ = run(t, chgrp.Options{}, strconv.Itoa(nobody), link)
	if len(calls) != 1 || !calls[0].follow {
		t.Fatalf("default: calls = %+v", calls)
	}
}

func TestPreserveRoot(t *testing.T) {
	calls, _, errb, err := run(t, chgrp.Options{Recursive: true}, strconv.Itoa(nobody), "/")
	if err == nil || len(calls) != 0 {
		t.Fatalf("err = %v, calls = %+v", err, calls)
	}
	want := "chgrpx: it is dangerous to operate recursively on '/'\nchgrpx: use --no-preserve-root to override this failsafe\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestFrom(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	mine := strconv.Itoa(os.Getgid())

	// A file in another group is left alone, silently.
	calls, out, _, err := run(t, chgrp.Options{From: ":" + strconv.Itoa(nobody), Changes: true}, mine, f)
	if err != nil || len(calls) != 0 || out != "" {
		t.Fatalf("calls = %+v, stdout = %q, err = %v", calls, out, err)
	}
	// A file that matches is changed.
	calls, _, _, err = run(t, chgrp.Options{From: strconv.Itoa(os.Getuid()) + ":" + mine}, strconv.Itoa(nobody), f)
	if err != nil || len(calls) != 1 || calls[0].gid != nobody {
		t.Fatalf("calls = %+v, err = %v", calls, err)
	}
	// An unknown name is refused before anything changes.
	calls, _, errs, err := run(t, chgrp.Options{From: ":no-such-group-gnux"}, mine, f)
	if err == nil || len(calls) != 0 || errs != "chgrpx: invalid group: ':no-such-group-gnux'\n" {
		t.Fatalf("calls = %+v, stderr = %q, err = %v", calls, errs, err)
	}
}
