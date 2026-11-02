//go:build !windows

package visual

// permissionBits is true: the mode's permission bits are the file's own, so
// they can mark a directory other-writable or sticky.
const permissionBits = true
