//go:build unix

package core

import (
	"os"
	"syscall"
)

// wOK is the write-permission bit for access(2).
const wOK = 0x2

// Writable reports whether the calling process may write to path. rm uses this
// to decide whether a file is "write-protected" and warrants a prompt.
func Writable(path string) bool {
	return syscall.Access(path, wOK) == nil
}

// DeviceID returns the device number backing info, used by --one-file-system.
func DeviceID(info os.FileInfo) (uint64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	// Stat_t.Dev is uint64 on Linux but int32 on darwin/BSD, so this conversion
	// is required off-Linux even though unconvert (run on Linux CI) flags it.
	return uint64(st.Dev), true //nolint:unconvert
}
