//go:build unix

package cmd

import (
	"os/exec"
	"syscall"
)

// stopGroup makes a call its own process group and, when it is stopped,
// kills the whole group, so that a program it started dies with it.
func stopGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}
