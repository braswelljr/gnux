//go:build unix

package sysstat

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, make([]byte, 5000), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, filepath.Join(dir, "hard")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := Of(info)
	if st.RawMode != typeReg|0o640 {
		t.Errorf("RawMode = %#o, want %#o", st.RawMode, typeReg|0o640)
	}
	if st.Nlink != 2 {
		t.Errorf("Nlink = %d, want 2", st.Nlink)
	}
	if st.UID != int64(os.Getuid()) {
		t.Errorf("UID = %d, want %d", st.UID, os.Getuid())
	}
	if st.Ino == 0 || st.BlockSize <= 0 || st.Blocks*512 < 4096 {
		t.Errorf("Ino=%d BlockSize=%d Blocks=%d", st.Ino, st.BlockSize, st.Blocks)
	}
	if since := time.Since(st.Atime); since < 0 || since > time.Hour {
		t.Errorf("Atime = %v", st.Atime)
	}

	dinfo, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := Of(dinfo).RawMode &^ 0o7777; got != typeDir {
		t.Errorf("directory type = %#o, want %#o", got, typeDir)
	}
}
