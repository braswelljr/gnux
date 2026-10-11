//go:build !unix

package core

import "os"

// Writable reports whether path is writable. Off Unix (Windows, WebAssembly)
// there is no access(2); approximate with the owner write bit, which is how
// the read-only attribute surfaces.
func Writable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return info.Mode().Perm()&0o200 != 0
}

// DeviceID is unsupported off Unix, so --one-file-system is a no-op there.
func DeviceID(_ os.FileInfo) (uint64, bool) {
	return 0, false
}
