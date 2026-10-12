//go:build unix

package chmod

import "syscall"

// processUmask returns the process file-creation mask. umask(2) can only be
// read by setting it, so the old value is put straight back.
func processUmask() uint32 {
	old := syscall.Umask(0)
	syscall.Umask(old)
	return uint32(old)
}
