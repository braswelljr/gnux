//go:build unix

package special

import "golang.org/x/sys/unix"

// Mkfifo creates a named pipe with the given permission bits, less the umask.
func Mkfifo(path string, perm uint32) error {
	return unix.Mkfifo(path, perm)
}

// Mknod creates a block or character device file with the given permission
// bits, less the umask.
func Mknod(path string, perm uint32, block bool, major, minor uint32) error {
	mode := perm | unix.S_IFCHR
	if block {
		mode = perm | unix.S_IFBLK
	}
	return mknod(path, mode, unix.Mkdev(major, minor))
}
