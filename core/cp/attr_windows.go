//go:build windows

package cp

import "os"

// inode is unavailable through os.FileInfo on Windows, so hard links between
// sources are copied as independent files there.
func inode(_ os.FileInfo) (uint64, bool) {
	return 0, false
}

// chown is a no-op on Windows, which has no POSIX owner/group to carry over.
func chown(_ string, _ os.FileInfo) error {
	return nil
}
