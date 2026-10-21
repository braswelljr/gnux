//go:build !unix

package special

// Mkfifo fails: Windows has no named pipes in the file system, and the
// browser and WASI have no special files at all.
func Mkfifo(string, uint32) error { return ErrUnsupported }

// Mknod fails for the same reason.
func Mknod(string, uint32, bool, uint32, uint32) error { return ErrUnsupported }
