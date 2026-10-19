//go:build !windows && !wasip1

package sysstat

import (
	"os"
	"syscall"
	"time"
)

// Of returns the metadata behind info.
func Of(info os.FileInfo) Stat {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Stat{Nlink: 1, UID: -1, GID: -1, RawMode: rawMode(info.Mode()), BlockSize: 4096}
	}
	// The widths and signedness of these fields differ between Linux,
	// darwin, the BSDs and js, so each conversion is needed somewhere.
	s := Stat{
		Dev:       uint64(st.Dev),    //nolint:unconvert
		Ino:       uint64(st.Ino),    //nolint:unconvert
		Nlink:     uint64(st.Nlink),  //nolint:unconvert
		Rdev:      uint64(st.Rdev),   //nolint:unconvert
		UID:       int64(st.Uid),     //nolint:unconvert
		GID:       int64(st.Gid),     //nolint:unconvert
		Blocks:    int64(st.Blocks),  //nolint:unconvert
		BlockSize: int64(st.Blksize), //nolint:unconvert
		RawMode:   uint32(st.Mode),   //nolint:unconvert
	}
	s.Atime, s.Ctime, s.Btime = times(st)
	return s
}

func timespec(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec)) //nolint:unconvert // int32 on 32-bit platforms
}
