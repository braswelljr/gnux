//go:build unix

package ownership

import (
	"errors"
	"os"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"
)

// errNotSupported is never returned on Unix, where every file has an owner.
var errNotSupported = errors.New("operation not supported")

// ownerOf returns the numeric owner and group behind info.
func ownerOf(info os.FileInfo) (uid, gid int, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return int(st.Uid), int(st.Gid), true
}

// chown changes path's owner and group, following a symbolic link or not.
func chown(path string, uid, gid int, follow bool) error {
	if follow {
		return fsys.Chown(path, uid, gid)
	}
	return fsys.Lchown(path, uid, gid)
}
