//go:build windows

package cp

import (
	"os"
)

// inode is unavailable through os.FileInfo on Windows, so hard links between
// sources are copied as independent files there.
func inode(_ os.FileInfo) (uint64, bool) {
	return 0, false
}

// chown is a no-op on Windows, which has no POSIX owner/group to carry over.
func chown(_ string, _ os.FileInfo) error {
	return nil
}

// createdDirMode is the mode a directory made by fsys.Mkdir(path, perm) is meant
// to have. Windows ignores the mode given to Mkdir and has no umask, so it is
// perm itself; cp applies it once the contents are in, which is how a
// read-only source directory gives a read-only copy.
func createdDirMode(_ string, perm os.FileMode) os.FileMode {
	return perm
}
