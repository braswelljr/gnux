//go:build !windows

package fsstat

import (
	"errors"
	"io/fs"
)

// Unavailable reports an error that means a mount point cannot be examined
// right now (gone, or not ours to look at) rather than that something broke;
// df skips such entries quietly, as GNU df does with ENOENT and EACCES.
func Unavailable(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission)
}
