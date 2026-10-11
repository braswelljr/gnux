//go:build !windows

package mv

import (
	"errors"
	"syscall"
)

// isCrossDevice reports whether err is rename(2) refusing to move between
// filesystems, the cue to fall back to copy-then-remove.
func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}

// isNotEmpty reports whether err is a rename or rmdir refused because the
// directory still has entries.
func isNotEmpty(err error) bool {
	return errors.Is(err, syscall.ENOTEMPTY)
}
