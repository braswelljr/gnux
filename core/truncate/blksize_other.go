//go:build !unix

package truncate

import "os"

// blockSize has no stat(2) block size to report off Unix; -o uses 512 bytes.
func blockSize(os.FileInfo) int64 {
	return 512
}
