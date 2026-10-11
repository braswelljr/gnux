//go:build windows

package mv

import (
	"errors"
	"syscall"
)

// errNotSameDevice is ERROR_NOT_SAME_DEVICE, what MoveFileEx returns for a
// move between volumes.
const errNotSameDevice = syscall.Errno(17)

// errDirNotEmpty is ERROR_DIR_NOT_EMPTY.
const errDirNotEmpty = syscall.Errno(145)

// isCrossDevice reports whether err is a rename refusing to move between
// volumes, the cue to fall back to copy-then-remove.
func isCrossDevice(err error) bool {
	return errors.Is(err, errNotSameDevice) || errors.Is(err, syscall.EXDEV)
}

// isNotEmpty reports whether err is a rename or rmdir refused because the
// directory still has entries.
func isNotEmpty(err error) bool {
	return errors.Is(err, errDirNotEmpty) || errors.Is(err, syscall.ENOTEMPTY)
}
