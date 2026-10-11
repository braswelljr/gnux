//go:build linux || openbsd || dragonfly || solaris

package touch

import (
	"os"
	"syscall"
	"time"
)

// AccessTime returns the time info's file was last accessed. os.FileInfo only
// carries the modification time, so this reads the platform's stat record.
func AccessTime(info os.FileInfo) time.Time {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.ModTime()
	}
	// Timespec fields are int32 on 32-bit targets, so the conversions are
	// required there even though unconvert (run on 64-bit) flags them.
	return time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec)) //nolint:unconvert
}
