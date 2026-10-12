//go:build unix

package test

import (
	"os"
	"syscall"
)

// access answers -r/-w/-x with access(2), which accounts for the caller's
// identity and supplementary groups as GNU test does.
func access(path string, mode uint32) bool {
	return syscall.Access(path, mode) == nil
}

// owner returns the file's numeric owner and group.
func owner(info os.FileInfo) (uid, gid int, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return int(st.Uid), int(st.Gid), true
}

func euid() int { return os.Geteuid() }
func egid() int { return os.Getegid() }
