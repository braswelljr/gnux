package ls

import "time"

// sysInfo is the part of a file's metadata that os.FileInfo does not expose
// portably. Each platform fills in what it can (stat_unix.go, stat_windows.go).
type sysInfo struct {
	ino    uint64
	nlink  uint64
	uid    uint32
	gid    uint32
	blocks int64  // allocated size in 512-byte units
	rdev   uint64 // device number of a character or block special file

	// Zero when the platform does not record the timestamp.
	atime, ctime, btime time.Time
}
