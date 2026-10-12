//go:build windows

package sysstat

import (
	"os"
	"syscall"
	"time"
)

// Of returns what Windows offers: there are no inode numbers, numeric owners
// or device numbers behind os.FileInfo, so those are unknown, the link count
// is 1 and the allocation is the size rounded up to 512-byte units.
func Of(info os.FileInfo) Stat {
	s := Stat{Nlink: 1, UID: -1, GID: -1, BlockSize: 4096, RawMode: rawMode(info.Mode())}
	if info.Mode().IsRegular() {
		s.Blocks = (info.Size() + 511) / 512
	}
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		s.Atime = time.Unix(0, d.LastAccessTime.Nanoseconds())
		s.Btime = time.Unix(0, d.CreationTime.Nanoseconds())
	}
	return s
}
