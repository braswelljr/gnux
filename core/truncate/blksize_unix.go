//go:build unix

package truncate

import (
	"os"
	"syscall"
)

// blockSize is the file's preferred I/O block size, used by -o.
func blockSize(info os.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Blksize > 0 {
		// Blksize is int64 on some platforms and int32 on others.
		return int64(st.Blksize) //nolint:unconvert
	}
	return 512
}
