//go:build windows

package touch

import (
	"os"
	"syscall"
	"time"
)

// AccessTime returns the time info's file was last accessed. os.FileInfo only
// carries the modification time, so this reads the Win32 attribute record.
func AccessTime(info os.FileInfo) time.Time {
	attr, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return info.ModTime()
	}
	return time.Unix(0, attr.LastAccessTime.Nanoseconds())
}
