//go:build linux

package touch

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	atSymlinkNofollow = 0x100
	// utimeOmit in a timespec's nanoseconds leaves that timestamp unchanged.
	utimeOmit = (1 << 30) - 2
)

// lchtimes sets the times of path without following a final symbolic link. A
// zero time leaves the corresponding timestamp unchanged, as os.Chtimes does.
// The syscall package exports no wrapper for utimensat(2), hence the raw call.
func lchtimes(path string, atime, mtime time.Time) error {
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		return &os.PathError{Op: "utimensat", Path: path, Err: err}
	}
	ts := [2]syscall.Timespec{timespec(atime), timespec(mtime)}
	atFDCWD := -100

	_, _, errno := syscall.Syscall6(
		syscall.SYS_UTIMENSAT,
		uintptr(atFDCWD),
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&ts)),
		atSymlinkNofollow,
		0, 0,
	)
	if errno != 0 {
		return &os.PathError{Op: "utimensat", Path: path, Err: errno}
	}
	return nil
}

func timespec(t time.Time) syscall.Timespec {
	if t.IsZero() {
		return syscall.Timespec{Nsec: utimeOmit}
	}
	return syscall.NsecToTimespec(t.UnixNano())
}
