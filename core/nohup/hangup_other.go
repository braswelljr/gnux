//go:build !unix && !windows

package nohup

import "syscall"

// ignoreHangup does nothing here: there is no hangup signal.
func ignoreHangup() func() {
	return func() {}
}

// hangupAttr adds nothing to how the command is started.
func hangupAttr() *syscall.SysProcAttr { return nil }
