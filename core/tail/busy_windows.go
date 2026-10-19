//go:build windows

package tail

import (
	"errors"
	"syscall"
)

// Windows refuses to open a file while another process holds it without
// sharing it: an editor saving it, a program renaming another file over it.
const (
	errorSharingViolation syscall.Errno = 32
	errorLockViolation    syscall.Errno = 33
)

// busy reports whether err only means that another process has the file for
// now, so that following by name looks again at the next poll rather than
// giving the file up as inaccessible.
func busy(err error) bool {
	return errors.Is(err, errorSharingViolation) || errors.Is(err, errorLockViolation)
}
