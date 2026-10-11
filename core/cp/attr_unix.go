//go:build !windows

package cp

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// inode returns the inode number of info when the file has more than one hard
// link, which is the only case --preserve=links needs to track.
func inode(info os.FileInfo) (uint64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Nlink < 2 {
		return 0, false
	}
	// Stat_t.Ino is not uint64 on every Unix, so the conversion stays even
	// where unconvert considers it redundant.
	return uint64(st.Ino), true //nolint:unconvert
}

// chown gives path the owner and group recorded in info without following a
// symbolic link. Lacking the privilege to do so is not an error: like GNU cp,
// an ordinary user keeps their own ownership silently.
func chown(path string, info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	err := os.Lchown(path, int(st.Uid), int(st.Gid))
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EINVAL) {
		return nil
	}
	return err
}

// createdDirMode is the mode a directory made by os.Mkdir(path, perm) ended up
// with: perm as the umask left it, which is the mode cp gives a new directory
// without -p.
func createdDirMode(path string, perm os.FileMode) os.FileMode {
	if info, err := os.Lstat(path); err == nil {
		return info.Mode().Perm()
	}
	return perm
}
