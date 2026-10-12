package fsstat

import (
	"errors"
	"testing"
)

func TestStatfs(t *testing.T) {
	fs, err := Statfs(t.TempDir())
	if errors.Is(err, ErrUnsupported) {
		t.Skip("no file system statistics here")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fs.BlockSize == 0 || fs.Blocks == 0 || fs.TypeName == "" {
		t.Fatalf("Statfs = %+v", fs)
	}
	if fs.Bfree > fs.Blocks {
		t.Fatalf("more free blocks than blocks: %+v", fs)
	}
}

func TestMounts(t *testing.T) {
	ms, err := Mounts()
	if errors.Is(err, ErrUnsupported) {
		t.Skip("no mount table here")
	}
	if err != nil || len(ms) == 0 {
		t.Fatalf("Mounts = %v, %v", ms, err)
	}
	for _, m := range ms {
		if m.Target == "" {
			t.Fatalf("mount without a target: %+v", m)
		}
	}
}

func TestCString(t *testing.T) {
	if got := cString([]int8{'a', 'p', 'f', 's', 0, 'x'}); got != "apfs" {
		t.Fatalf("cString = %q", got)
	}
}
