//go:build linux

package sync

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func syncAll() error {
	unix.Sync()
	return nil
}

func syncFile(f *os.File, m Mode) error {
	switch m {
	case Data:
		return unix.Fdatasync(int(f.Fd()))
	case FileSystem:
		return unix.Syncfs(int(f.Fd()))
	}
	return f.Sync()
}

// nonblock keeps opening a FIFO from waiting for a writer.
const nonblock = syscall.O_NONBLOCK
