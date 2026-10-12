package unlink_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unlink"
)

func run(args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	err := unlink.New(unlink.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestUnlink(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errb, err := run(f); err != nil || errb != "" {
		t.Fatalf("err = %v, stderr %q", err, errb)
	}
	if _, err := os.Lstat(f); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
}

func TestUnlinkSymlinkNotTarget(t *testing.T) {
	dir := t.TempDir()
	target, l := filepath.Join(dir, "t"), filepath.Join(dir, "l")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, l); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if _, _, err := run(l); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target removed: %v", err)
	}
}

func TestUnlinkRefusesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "d")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, errb, err := run(dir)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "unlinkx: cannot unlink '" + dir + "': Is a directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("empty directory removed: %v", err)
	}
}

func TestUnlinkMissing(t *testing.T) {
	f := filepath.Join(t.TempDir(), "nope")
	_, errb, err := run(f)
	if !errors.Is(err, core.ErrFailed) || errb != "unlinkx: cannot unlink '"+f+"': No such file or directory\n" {
		t.Fatalf("err = %v, stderr %q", err, errb)
	}
}

func TestInjectedRemove(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	u := unlink.New(unlink.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	u.RemoveFn = func(string) error { return os.ErrPermission }
	if err := u.Run([]string{f}); !errors.Is(err, core.ErrFailed) || !strings.HasSuffix(errb.String(), ": Permission denied\n") {
		t.Fatalf("err = %v, stderr %q", err, errb.String())
	}
}

func TestOperands(t *testing.T) {
	for args, want := range map[string]string{"": "missing operand", "a b": "extra operand 'b'"} {
		_, _, err := run(strings.Fields(args)...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != want {
			t.Fatalf("%q: err = %v, want %q", args, err, want)
		}
	}
}
