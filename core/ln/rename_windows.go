//go:build windows

package ln

import (
	"os"

	"github.com/braswelljr/gnux/core/fsys"
)

// renameOver moves the new link at tmp onto dst. Windows refuses to rename
// over a symbolic link to a directory, so when that is what dst is, it is
// removed first and the rename tried again. Only there is the replacement not
// a single step: for a moment dst does not exist.
func renameOver(tmp, dst string) error {
	err := fsys.Rename(tmp, dst)
	if err == nil {
		return nil
	}
	if info, lerr := fsys.Lstat(dst); lerr != nil || info.Mode()&os.ModeSymlink == 0 {
		return err
	}
	if rerr := fsys.Remove(dst); rerr != nil {
		return err
	}
	return fsys.Rename(tmp, dst)
}
