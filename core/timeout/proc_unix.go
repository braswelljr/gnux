//go:build unix

package timeout

import (
	"os"
	"os/signal"
	"syscall"
)

// procAttr puts the command in a process group of its own when group is set,
// so that a signal sent to the group reaches everything it started.
func procAttr(group bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: group}
}

// forwardSignals relays the signals that would stop timeoutx (INT, TERM, HUP,
// QUIT) to the command instead, as GNU timeout does. The returned func stops
// relaying.
func forwardSignals() (<-chan int, func()) {
	in := make(chan os.Signal, 4)
	out := make(chan int, 4)
	stop := make(chan struct{})
	signal.Notify(in, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for {
			select {
			case s := <-in:
				if sig, ok := s.(syscall.Signal); ok {
					select {
					case out <- int(sig):
					case <-stop:
						return
					}
				}
			case <-stop:
				return
			}
		}
	}()
	return out, func() {
		signal.Stop(in)
		close(stop)
	}
}
