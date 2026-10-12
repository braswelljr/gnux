//go:build linux

package nice

import "syscall"

// The raw getpriority system call on Linux returns 20 minus the niceness, so
// that a valid result is never negative.
func getNiceness() (int, error) {
	p, err := syscall.Getpriority(syscall.PRIO_PROCESS, 0)
	if err != nil {
		return 0, err
	}
	return 20 - p, nil
}

func setNiceness(pid, niceness int) error {
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, niceness)
}
