//go:build unix

package chmod_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chmod"
)

func perm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
}

func setMode(t *testing.T, path string, m os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, m); err != nil {
		t.Fatal(err)
	}
}

func TestModes(t *testing.T) {
	tests := []struct {
		mode  string
		from  os.FileMode
		umask uint32
		want  os.FileMode
	}{
		{"755", 0o644, 0, 0o755},
		{"0", 0o644, 0, 0},
		{"u+x", 0o644, 0, 0o744},
		{"go-r", 0o644, 0, 0o600},
		{"a=r", 0o777, 0, 0o444},
		{"u=rwx,g=rx,o=", 0o000, 0, 0o750},
		{"+x", 0o644, 0, 0o755},
		{"+x", 0o644, 0o022, 0o755},
		{"+w", 0o444, 0o022, 0o644},
		{"a+X", 0o644, 0, 0o644},
		{"a+X", 0o744, 0, 0o755},
		{"g=u", 0o640, 0, 0o660},
		{"o=g", 0o654, 0, 0o655},
		{"u-w+x", 0o644, 0, 0o544},
		{"u+s", 0o755, 0, 0o755 | os.ModeSetuid},
		{"=", 0o755, 0, 0},
		{"=", 0o777, 0o022, 0o022},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			f := file(t, t.TempDir(), "f")
			setMode(t, f, tt.from)
			_, errb, err := run(t, chmod.Options{}, tt.umask, tt.mode, f)
			// "=" with a 022 umask is exercised below; here only the result.
			if err != nil && tt.umask == 0 {
				t.Fatalf("err = %v, stderr %q", err, errb)
			}
			if got := perm(t, f); got != tt.want {
				t.Fatalf("chmod %s on %o: got %o, want %o", tt.mode, tt.from, got, tt.want)
			}
		})
	}
}

func TestDirectorySearchBit(t *testing.T) {
	d := filepath.Join(t.TempDir(), "d")
	if err := os.Mkdir(d, 0o644); err != nil {
		t.Fatal(err)
	}
	setMode(t, d, 0o644)
	if _, _, err := run(t, chmod.Options{}, 0, "a+X", d); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, d); got != 0o755 {
		t.Fatalf("a+X on a directory: got %o, want 755", got)
	}
}

func TestUmaskWarning(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	setMode(t, f, 0o666)
	_, errb, err := run(t, chmod.Options{DiagnoseSurprises: true}, 0o022, "-w", f)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	want := "chmodx: " + f + ": new permissions are r--rw-rw-, not r--r--r--\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if got := perm(t, f); got != 0o466 {
		t.Fatalf("mode = %o, want 466 (the change is still made)", got)
	}

	// Bits the umask kept from being added are no surprise.
	setMode(t, f, 0o600)
	if _, errb, err := run(t, chmod.Options{DiagnoseSurprises: true}, 0o022, "=rw", f); err != nil || errb != "" {
		t.Fatalf("=rw: err = %v, stderr = %q", err, errb)
	}
	// Without the option form, nothing is diagnosed.
	setMode(t, f, 0o666)
	if _, errb, err := run(t, chmod.Options{}, 0o022, "-w", f); err != nil || errb != "" {
		t.Fatalf("-w as an operand: err = %v, stderr = %q", err, errb)
	}
}

func TestVerboseAndChanges(t *testing.T) {
	dir := t.TempDir()
	f := file(t, dir, "f")
	setMode(t, f, 0o644)

	out, _, err := run(t, chmod.Options{Verbose: true}, 0, "755", f)
	if err != nil {
		t.Fatal(err)
	}
	if want := "mode of '" + f + "' changed from 0644 (rw-r--r--) to 0755 (rwxr-xr-x)\n"; out != want {
		t.Fatalf("verbose = %q, want %q", out, want)
	}

	out, _, _ = run(t, chmod.Options{Verbose: true}, 0, "755", f)
	if want := "mode of '" + f + "' retained as 0755 (rwxr-xr-x)\n"; out != want {
		t.Fatalf("verbose unchanged = %q, want %q", out, want)
	}

	out, _, _ = run(t, chmod.Options{Changes: true}, 0, "755", f)
	if out != "" {
		t.Fatalf("-c on an unchanged file printed %q", out)
	}
	out, _, _ = run(t, chmod.Options{Changes: true}, 0, "u+s", f)
	if want := "mode of '" + f + "' changed from 0755 (rwxr-xr-x) to 4755 (rwsr-xr-x)\n"; out != want {
		t.Fatalf("-c = %q, want %q", out, want)
	}
}

func TestReference(t *testing.T) {
	dir := t.TempDir()
	ref, f := file(t, dir, "ref"), file(t, dir, "f")
	setMode(t, ref, 0o640)
	if _, _, err := run(t, chmod.Options{Reference: ref}, 0, f); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, f); got != 0o640 {
		t.Fatalf("mode = %o, want 640", got)
	}
}

func TestRecursive(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "d")
	sub := filepath.Join(d, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	a, b := file(t, d, "a"), file(t, sub, "b")
	outside := file(t, root, "outside")
	setMode(t, outside, 0o600)
	if err := os.Symlink(outside, filepath.Join(d, "link")); err != nil {
		t.Fatal(err)
	}

	if _, _, err := run(t, chmod.Options{Recursive: true}, 0, "go-rx", d); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{d, sub} {
		if got := perm(t, p); got != 0o700 {
			t.Fatalf("%s = %o, want 700", p, got)
		}
	}
	for _, p := range []string{a, b} {
		if got := perm(t, p); got != 0o600 {
			t.Fatalf("%s = %o, want 600", p, got)
		}
	}
	if got := perm(t, outside); got != 0o600 {
		t.Fatalf("a link found while recursing was followed: %o", got)
	}

	// -L follows it.
	if _, _, err := run(t, chmod.Options{Recursive: true, Traversal: chmod.TraverseLogical}, 0, "o+r", d); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, outside); got != 0o604 {
		t.Fatalf("-L did not follow the link: %o", got)
	}
}

func TestOperandSymlinkIsFollowed(t *testing.T) {
	dir := t.TempDir()
	target := file(t, dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(t, chmod.Options{}, 0, "600", link); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, target); got != 0o600 {
		t.Fatalf("target = %o, want 600", got)
	}

	// -R -P does not follow even an operand.
	out, _, err := runNoLinkModes(t, chmod.Options{Recursive: true, Traversal: chmod.TraversePhysical, Verbose: true}, "644", link)
	if err != nil {
		t.Fatal(err)
	}
	if want := "neither symbolic link '" + link + "' nor referent has been changed\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if got := perm(t, target); got != 0o600 {
		t.Fatalf("-R -P followed the operand: %o", got)
	}
}

func TestPreserveRoot(t *testing.T) {
	var changed []string
	c := chmod.New(chmod.Options{Recursive: true}, core.Discard())
	c.ChmodFn = func(p string, _ os.FileMode) error { changed = append(changed, p); return nil }
	if err := c.Run([]string{"u+w", "/"}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if len(changed) != 0 {
		t.Fatalf("changed %v under the failsafe", changed)
	}
}

func TestDirectoryKeepsSetGID(t *testing.T) {
	d := filepath.Join(t.TempDir(), "d")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0o755|os.ModeSetgid); err != nil || perm(t, d)&os.ModeSetgid == 0 {
		t.Skip("cannot set the set-group-ID bit here")
	}
	if _, _, err := run(t, chmod.Options{}, 0, "750", d); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, d); got != 0o750|os.ModeSetgid {
		t.Fatalf("750 on a setgid directory = %o, want 2750", got)
	}
	if _, _, err := run(t, chmod.Options{}, 0, "00750", d); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, d); got != 0o750 {
		t.Fatalf("00750 = %o, want 750", got)
	}
}

func TestChmodFailure(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	var out, errb bytes.Buffer
	c := chmod.New(chmod.Options{Verbose: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.ChmodFn = func(string, os.FileMode) error {
		return &os.PathError{Op: "chmod", Path: f, Err: syscall.EPERM}
	}
	c.Umask = func() uint32 { return 0 }
	setMode(t, f, 0o644)
	if err := c.Run([]string{"600", f}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "chmodx: changing permissions of '" + f + "': Operation not permitted\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	if want := "failed to change mode of '" + f + "' from 0644 (rw-r--r--) to 0600 (rw-------)\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

// runNoLinkModes runs chmod as on Linux, where a symbolic link has no mode of
// its own and changing one fails with EOPNOTSUPP.
func runNoLinkModes(t *testing.T, opts chmod.Options, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := chmod.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.Umask = func() uint32 { return 0 }
	c.LchmodFn = func(path string, _ os.FileMode) error {
		return &os.PathError{Op: "lchmod", Path: path, Err: syscall.EOPNOTSUPP}
	}
	err = c.Run(args)
	return out.String(), errb.String(), err
}

func TestNoDereference(t *testing.T) {
	dir := t.TempDir()
	target := file(t, dir, "target")
	link := filepath.Join(dir, "link")
	dangling := filepath.Join(dir, "dangling")
	for old, l := range map[string]string{target: link, filepath.Join(dir, "missing"): dangling} {
		if err := os.Symlink(old, l); err != nil {
			t.Fatal(err)
		}
	}

	// Linux: the link has no mode, so nothing changes and nothing fails.
	out, errs, err := runNoLinkModes(t, chmod.Options{NoDereference: true, Verbose: true}, "600", link, dangling)
	if err != nil || errs != "" {
		t.Fatalf("err = %v, stderr = %q", err, errs)
	}
	want := "neither symbolic link '" + link + "' nor referent has been changed\n" +
		"neither symbolic link '" + dangling + "' nor referent has been changed\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if got := perm(t, target); got != 0o644 {
		t.Fatalf("-h changed the referent: %o", got)
	}

	// Where links have modes, the link's own mode is the one changed.
	var changed []string
	var buf bytes.Buffer
	c := chmod.New(chmod.Options{NoDereference: true, Changes: true}, core.Streams{In: strings.NewReader(""), Out: &buf, Err: &buf})
	c.Umask = func() uint32 { return 0 }
	c.LchmodFn = func(path string, mode os.FileMode) error {
		changed = append(changed, path+" "+mode.String())
		return nil
	}
	if err := c.Run([]string{"600", link}); err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != link+" -rw-------" {
		t.Fatalf("lchmod calls = %q", changed)
	}
	if perm(t, target) != 0o644 {
		t.Fatal("-h changed the referent")
	}
}

func TestDereferenceWhileRecursing(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "d")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := file(t, root, "outside")
	other := filepath.Join(root, "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	inner := file(t, other, "inner")
	for old, l := range map[string]string{outside: "lf", other: "ld"} {
		if err := os.Symlink(old, filepath.Join(d, l)); err != nil {
			t.Fatal(err)
		}
	}

	// The default leaves the links (and so their referents) alone.
	out, _, err := runNoLinkModes(t, chmod.Options{Recursive: true, Verbose: true}, "700", d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "neither symbolic link '"+filepath.Join(d, "lf")+"'") || perm(t, outside) != 0o644 {
		t.Fatalf("default -R: stdout %q, outside %o", out, perm(t, outside))
	}

	// --dereference changes each referent but does not descend into one.
	if _, _, err := runNoLinkModes(t, chmod.Options{Recursive: true, Dereference: true}, "700", d); err != nil {
		t.Fatal(err)
	}
	if perm(t, outside) != 0o700 || perm(t, other) != 0o700 {
		t.Fatalf("referents: %o %o, want 700", perm(t, outside), perm(t, other))
	}
	if perm(t, inner) != 0o644 {
		t.Fatalf("--dereference descended into a linked directory: %o", perm(t, inner))
	}

	// -h wins over --dereference.
	if _, _, err := runNoLinkModes(t, chmod.Options{Recursive: true, Dereference: true, NoDereference: true}, "755", d); err != nil {
		t.Fatal(err)
	}
	if perm(t, outside) != 0o700 {
		t.Fatalf("-h did not win: %o", perm(t, outside))
	}

	// With -P, --dereference is refused.
	_, errs, err := runNoLinkModes(t, chmod.Options{Recursive: true, Dereference: true, Traversal: chmod.TraversePhysical}, "700", d)
	if !errors.Is(err, core.ErrFailed) || errs != "chmodx: -R --dereference requires either -H or -L\n" {
		t.Fatalf("err = %v, stderr = %q", err, errs)
	}
}
