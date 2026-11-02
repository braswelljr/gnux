//go:build windows

package visual

// permissionBits is false: Windows has no POSIX permission bits, and Go
// reports every directory as rwxrwxrwx, so a mode's write and sticky bits
// say nothing about the directory and must not color it as other-writable.
const permissionBits = false
