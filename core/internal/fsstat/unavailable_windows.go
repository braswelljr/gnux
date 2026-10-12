//go:build windows

package fsstat

import (
	"errors"
	"io/fs"

	"golang.org/x/sys/windows"
)

// Unavailable reports an error that means a drive cannot be examined right
// now: gone, not ours to look at, or a removable drive with no medium (an
// empty card reader or DVD drive), which df skips quietly.
func Unavailable(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) ||
		errors.Is(err, windows.ERROR_NOT_READY) || errors.Is(err, windows.ERROR_INVALID_DRIVE)
}
