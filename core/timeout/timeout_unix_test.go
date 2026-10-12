//go:build unix

package timeout_test

import (
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/timeout"
)

// TestPreserveStatus checks that --preserve-status reports the command's
// own status, 128+TERM, instead of 124.
func TestPreserveStatus(t *testing.T) {
	_, _, status := run(t, timeout.Options{PreserveStatus: true}, "", "0.2", self, "sleep", "30")
	if want := 128 + int(syscall.SIGTERM); status != want {
		t.Fatalf("status = %d, want %d", status, want)
	}
}

// TestKillAfter checks that a command ignoring TERM is killed after -k.
func TestKillAfter(t *testing.T) {
	start := time.Now()
	_, errb, status := run(t, timeout.Options{KillAfter: 300 * time.Millisecond, Verbose: true},
		"", "0.3", self, "ignore-term", "30")
	if status != 137 {
		t.Fatalf("status = %d, want 137 (stderr %q)", status, errb)
	}
	want := "timeoutx: sending signal TERM to command '" + self + "'\n" +
		"timeoutx: sending signal KILL to command '" + self + "'\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("took %v", d)
	}
}

// TestKilledBySignal checks a command that dies of a signal not sent by
// timeoutx: its status is 128+N, not 124.
func TestKilledBySignal(t *testing.T) {
	_, _, status := run(t, timeout.Options{}, "", "30", "sh", "-c", "kill -INT $$")
	if want := 128 + int(syscall.SIGINT); status != want {
		t.Fatalf("status = %d, want %d", status, want)
	}
}
