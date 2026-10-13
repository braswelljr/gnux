//go:build unix

package find

import "syscall"

// The mode bits of access(2).
const (
	accessExec  = 0x1
	accessWrite = 0x2
	accessRead  = 0x4
)

// access answers -readable, -writable and -executable with access(2), which
// accounts for the caller's identity, supplementary groups and (unlike the
// permission bits -perm looks at) access control lists and read-only mounts.
func access(path string, mode uint32) bool {
	return syscall.Access(path, mode) == nil
}
