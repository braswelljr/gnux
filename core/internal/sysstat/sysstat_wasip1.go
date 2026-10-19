//go:build wasip1

package sysstat

import (
	"os"
	"syscall"
	"time"
)

// Of returns the metadata behind info. WASI's stat reports no owner (always
// 0 there), block count, block size or device number, and keeps times as
// nanoseconds.
func Of(info os.FileInfo) Stat {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Stat{Nlink: 1, UID: -1, GID: -1, RawMode: rawMode(info.Mode()), BlockSize: 4096}
	}
	return Stat{
		Dev:       st.Dev,
		Ino:       st.Ino,
		Nlink:     st.Nlink,
		UID:       int64(st.Uid),
		GID:       int64(st.Gid),
		BlockSize: 4096,
		RawMode:   rawMode(info.Mode()),
		Atime:     time.Unix(0, int64(st.Atime)),
		Ctime:     time.Unix(0, int64(st.Ctime)),
	}
}
