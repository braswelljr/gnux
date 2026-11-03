//go:build windows

package ls

import (
	"os"
	"syscall"
	"time"
)

// statSys fills in what Windows offers. There are no POSIX inode numbers, link
// counts, IDs or block counts behind os.FileInfo, so those are placeholders: a
// link count of 1 and an allocation rounded up from the file size.
func statSys(info os.FileInfo) sysInfo {
	s := sysInfo{nlink: 1}
	if info.Mode().IsRegular() {
		s.blocks = (info.Size() + 511) / 512
	}
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		s.atime = time.Unix(0, d.LastAccessTime.Nanoseconds())
		s.btime = time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return s
}

// deviceNumbers is a placeholder: Windows has no device special files.
func deviceNumbers(_ uint64) (major, minor uint64) {
	return 0, 0
}

// idCache is empty on Windows, where files have no numeric owner or group.
type idCache struct{}

// user returns a placeholder owner name.
func (*idCache) user(_ uint32) (string, bool) {
	return "-", true
}

// group returns a placeholder group name.
func (*idCache) group(_ uint32) (string, bool) {
	return "-", true
}
