//go:build unix

package kill_test

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core/kill"
)

// TestExitStatusSignaled checks the 128+N status of a process killed by a
// signal, and that signal 0 only checks that a process exists.
func TestExitStatusSignaled(t *testing.T) {
	cmd := exec.Command(os.Args[0], "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := kill.Send(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("signal 0 to a live process: %v", err)
	}
	if err := kill.Send(cmd.Process.Pid, kill.TermSignal); err != nil {
		t.Fatal(err)
	}
	err := cmd.Wait()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Wait = %v", err)
	}
	if got := kill.ExitStatus(exitErr.ProcessState); got != 128+int(syscall.SIGTERM) {
		t.Fatalf("ExitStatus = %d, want %d", got, 128+int(syscall.SIGTERM))
	}
	if err := kill.Send(cmd.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("signal 0 to a reaped process = %v, want ESRCH", err)
	}
}
