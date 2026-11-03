//go:build unix

package ownership

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	uid, gid, err := Owner(path)
	if err != nil || uid != os.Getuid() || gid < 0 {
		t.Fatalf("Owner = %d, %d, %v; want uid %d", uid, gid, err, os.Getuid())
	}
	if _, _, err := Owner(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Owner of a missing file succeeded")
	}
}
