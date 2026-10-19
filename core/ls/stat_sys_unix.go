//go:build !windows && !wasip1

package ls

import (
	"os"
	"syscall"
)

// statSys extracts the stat(2) fields behind info.
func statSys(info os.FileInfo) sysInfo {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return sysInfo{nlink: 1}
	}
	// The widths (and signedness) of these fields differ between Linux, darwin
	// and the BSDs, so each conversion is required somewhere even where
	// unconvert, run on a single platform, flags it.
	s := sysInfo{
		ino:    uint64(st.Ino),   //nolint:unconvert
		nlink:  uint64(st.Nlink), //nolint:unconvert
		uid:    uint32(st.Uid),   //nolint:unconvert
		gid:    uint32(st.Gid),   //nolint:unconvert
		blocks: int64(st.Blocks), //nolint:unconvert
		rdev:   uint64(st.Rdev),  //nolint:unconvert
	}
	s.atime, s.ctime, s.btime = statTimes(st)
	return s
}
