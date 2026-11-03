//go:build unix

package special

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMkfifo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pipe")
	if err := Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("not a named pipe: %v %v", info, err)
	}
	if err := Mkfifo(path, 0o600); !errors.Is(err, syscall.EEXIST) {
		t.Fatalf("a second Mkfifo = %v, want EEXIST", err)
	}
}

func TestMknodNeedsPrivilege(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, where mknod succeeds")
	}
	err := Mknod(filepath.Join(t.TempDir(), "dev"), 0o600, false, 1, 3)
	if err == nil {
		t.Fatal("Mknod of a character device succeeded without privilege")
	}
}
