//go:build wasip1

package ls

import (
	"os"
	"syscall"
	"time"
)

// statSys extracts what WASI's stat reports behind info: no owner (always 0
// there), block count or device number, and times as nanoseconds.
func statSys(info os.FileInfo) sysInfo {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return sysInfo{nlink: 1}
	}
	return sysInfo{
		ino:   st.Ino,
		nlink: st.Nlink,
		uid:   st.Uid,
		gid:   st.Gid,
		atime: time.Unix(0, int64(st.Atime)),
		ctime: time.Unix(0, int64(st.Ctime)),
	}
}
