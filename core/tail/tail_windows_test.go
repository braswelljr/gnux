//go:build windows

package tail_test

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/tail"
)

// TestFollowNameWhileHeldByAnother holds the followed file without sharing
// it, as an editor saving it or a rename over it does, so that every poll in
// between is refused with a sharing violation: -F waits for it to be let go
// rather than giving the file up.
func TestFollowNameWhileHeldByAnother(t *testing.T) {
	path := write(t, t.TempDir(), "log", "before\n")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowName}, path)
	f.waitOut("before\n")

	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Long enough for many polls (one every 2ms) to meet the refusal; the
	// engine has nothing to show for them, so there is nothing to wait on.
	time.Sleep(100 * time.Millisecond)
	_ = syscall.CloseHandle(h)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteString("after\n")
	_ = file.Close()
	f.waitOut("before\nafter\n")
	if err := f.stop(); err != nil {
		t.Fatalf("RunContext = %v, want nil", err)
	}
	if got := f.errb.String(); got != "" {
		t.Fatalf("stderr = %q, want nothing", got)
	}
}
