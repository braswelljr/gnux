//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package nice

import "syscall"

func getNiceness() (int, error) {
	return syscall.Getpriority(syscall.PRIO_PROCESS, 0)
}

func setNiceness(pid, niceness int) error {
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, niceness)
}
