//go:build !unix

package timeout

import "syscall"

// procAttr has nothing to set off Unix: there are no process groups to
// signal, so the command alone is terminated at the limit.
func procAttr(bool) *syscall.SysProcAttr {
	return nil
}

// forwardSignals relays nothing off Unix, where timeoutx cannot pass a
// signal on except by terminating the command.
func forwardSignals() (<-chan int, func()) {
	return nil, func() {}
}
