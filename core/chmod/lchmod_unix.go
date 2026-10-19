//go:build unix

package chmod

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// lchmod changes the mode of a symbolic link itself. Linux has no link
// modes and refuses with EOPNOTSUPP, which the engine reports as GNU does;
// the BSDs and macOS change the link.
func lchmod(path string, mode os.FileMode) error {
	if err := unix.Fchmodat(unix.AT_FDCWD, path, unixBits(mode), unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return &os.PathError{Op: "lchmod", Path: path, Err: err}
	}
	return nil
}

// linkModeUnsupported reports whether err says symbolic links have no mode
// of their own on this system.
func linkModeUnsupported(err error) bool {
	return errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.ENOTSUP)
}
