//go:build unix && !linux

package sync

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func syncAll() error {
	return unix.Sync()
}

// Off Linux there is no per-file-system sync and fdatasync is not portable:
// -d flushes the file as fsync does, and -f flushes every file system.
func syncFile(f *os.File, m Mode) error {
	if m == FileSystem {
		return unix.Sync()
	}
	return f.Sync()
}

// nonblock keeps opening a FIFO from waiting for a writer.
const nonblock = syscall.O_NONBLOCK
