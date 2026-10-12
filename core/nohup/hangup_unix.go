//go:build unix

package nohup

import (
	"os/signal"
	"syscall"
)

// ignoreHangup sets SIGHUP to be ignored while the command runs. An ignored
// signal stays ignored across exec, so the command inherits the setting, as
// it would from GNU nohup. The returned func restores the previous handling.
func ignoreHangup() func() {
	signal.Ignore(syscall.SIGHUP)
	return func() { signal.Reset(syscall.SIGHUP) }
}

// hangupAttr adds nothing to how the command is started: the ignored SIGHUP
// is inherited.
func hangupAttr() *syscall.SysProcAttr { return nil }
