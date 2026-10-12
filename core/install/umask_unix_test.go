//go:build unix

package install_test

import "syscall"

func umask() uint32 {
	old := syscall.Umask(0)
	syscall.Umask(old)
	return uint32(old)
}
