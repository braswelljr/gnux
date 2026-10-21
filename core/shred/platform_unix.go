//go:build unix

package shred

import (
	"os"

	"golang.org/x/sys/unix"

	"github.com/braswelljr/gnux/core/fsys"
)

// appendOnly reports whether f was opened for appending, where every write
// goes to the end and shredding it would only add data.
func appendOnly(f *os.File) bool {
	flags, err := unix.FcntlInt(f.Fd(), unix.F_GETFL, 0)
	return err == nil && flags&unix.O_APPEND != 0
}

// syncDirectory flushes a directory, so that a rename in it reaches the
// disk before the next one.
func syncDirectory(dir string) {
	if dir == "" {
		dir = "."
	}
	if d, err := fsys.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
}
