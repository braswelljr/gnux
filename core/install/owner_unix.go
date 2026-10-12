//go:build unix

package install

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// lookupUser resolves a user name or numeric ID.
func lookupUser(name string) (int, bool) {
	if u, err := user.Lookup(name); err == nil {
		if id, err := strconv.Atoi(u.Uid); err == nil {
			return id, true
		}
	}
	id, err := strconv.Atoi(name)
	return id, err == nil && id >= 0
}

// lookupGroup resolves a group name or numeric ID.
func lookupGroup(name string) (int, bool) {
	if g, err := user.LookupGroup(name); err == nil {
		if id, err := strconv.Atoi(g.Gid); err == nil {
			return id, true
		}
	}
	id, err := strconv.Atoi(name)
	return id, err == nil && id >= 0
}

// chown changes the owner and group of path; -1 leaves one unchanged.
func chown(path string, uid, gid int) error {
	return os.Lchown(path, uid, gid)
}

// sameOwner reports whether info already has the requested owner and group.
func sameOwner(info os.FileInfo, uid, gid int) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	return (uid < 0 || int(st.Uid) == uid) && (gid < 0 || int(st.Gid) == gid)
}
