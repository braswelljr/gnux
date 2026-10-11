//go:build !windows

package tail

import (
	"errors"
	"syscall"
)

// processAlive reports whether pid names a running process. Signal 0 performs
// the existence and permission checks without delivering anything; EPERM means
// the process exists but belongs to someone else.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
