//go:build windows

package nohup

import "syscall"

// detachedProcess is DETACHED_PROCESS: the new process has no console.
const detachedProcess = 0x00000008

// ignoreHangup does nothing on Windows: there is no hangup signal. What ends
// a program there when its console window closes is the console itself,
// which hangupAttr takes care of.
func ignoreHangup() func() {
	return func() {}
}

// hangupAttr starts the command without a console and in a process group of
// its own, so closing the console window (or pressing Ctrl-C in it) does not
// end it: the Windows counterpart of ignoring SIGHUP. The command never needs
// the console, since every stream that was a terminal has been replaced.
func hangupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess}
}
