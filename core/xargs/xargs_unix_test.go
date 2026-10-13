//go:build unix

package xargs_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/xargs"
)

// helperUnix adds the helper command that needs Unix signals: "kill-self"
// ends the helper with SIGKILL.
func helperUnix(args []string) {
	if args[0] == "kill-self" {
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		time.Sleep(5 * time.Second)
	}
}

func TestProgramEndedBySignal(t *testing.T) {
	self := testBinary(t)
	res := drive(t, xargs.Options{MaxArgs: 1}, "a\nb\n", []string{self, "kill-self"}, nil)
	if code := status(res.err); code != xargs.ExitSignaled {
		t.Fatalf("err = %v, want status %d", res.err, xargs.ExitSignaled)
	}
	if want := "xargsx: " + self + ": terminated by signal 9\n"; res.stderr != want {
		t.Errorf("stderr = %q, want %q", res.stderr, want)
	}
}

func TestProgramCannotBeRun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root may execute any file")
	}
	file := filepath.Join(t.TempDir(), "not-executable")
	if err := os.WriteFile(file, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := drive(t, xargs.Options{}, "a\n", []string{file}, nil)
	if code := status(res.err); code != xargs.ExitCannotRun {
		t.Fatalf("err = %v, want status %d", res.err, xargs.ExitCannotRun)
	}
	if want := "xargsx: " + file + ": Permission denied\n"; res.stderr != want {
		t.Errorf("stderr = %q, want %q", res.stderr, want)
	}
}
