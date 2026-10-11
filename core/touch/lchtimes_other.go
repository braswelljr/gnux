//go:build !linux

package touch

import (
	"errors"
	"os"
	"time"
)

var errLinkTimes = errors.New("operation not supported")

// lchtimes sets the times of path without following a final symbolic link.
// Off Linux the standard library offers no way to do that, so a symbolic link
// is refused - silently touching its target instead would change the wrong
// file. Anything else has no link to follow and is handled normally.
func lchtimes(path string, atime, mtime time.Time) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &os.PathError{Op: "lutimes", Path: path, Err: errLinkTimes}
	}
	return os.Chtimes(path, atime, mtime)
}
