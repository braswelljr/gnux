// Package sysstat exposes the parts of a file's metadata that os.FileInfo
// does not: inode, link count, owner, blocks, device numbers and the access,
// change and birth times. Each platform fills in what it has; the rest is
// marked unknown rather than invented.
package sysstat

import (
	"os"
	"time"
)

// Stat is a file's metadata beyond os.FileInfo.
type Stat struct {
	Dev, Ino, Nlink, Rdev uint64
	// UID and GID are -1 where files have no numeric owner (Windows).
	UID, GID int64
	// Blocks is the space allocated, in 512-byte units.
	Blocks int64
	// BlockSize is the preferred I/O size (st_blksize).
	BlockSize int64
	// RawMode is st_mode as the system reports it: file type and
	// permission bits in the Unix layout.
	RawMode uint32
	// Atime, Ctime and Btime are zero when the platform does not record
	// them.
	Atime, Ctime, Btime time.Time
}

// Unix file type bits, for RawMode where the platform has none.
const (
	typeFIFO = 0o010000
	typeChar = 0o020000
	typeDir  = 0o040000
	typeBlk  = 0o060000
	typeReg  = 0o100000
	typeLnk  = 0o120000
	typeSock = 0o140000
)

// rawMode rebuilds a Unix st_mode from an os.FileMode.
func rawMode(m os.FileMode) uint32 {
	t := uint32(typeReg)
	switch {
	case m&os.ModeDir != 0:
		t = typeDir
	case m&os.ModeSymlink != 0:
		t = typeLnk
	case m&os.ModeNamedPipe != 0:
		t = typeFIFO
	case m&os.ModeSocket != 0:
		t = typeSock
	case m&os.ModeCharDevice != 0:
		t = typeChar
	case m&os.ModeDevice != 0:
		t = typeBlk
	}
	bits := uint32(m.Perm())
	if m&os.ModeSetuid != 0 {
		bits |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		bits |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		bits |= 0o1000
	}
	return t | bits
}
