//go:build !windows

package tail

// busy reports whether err only means that another process has the file for
// now. Only Windows refuses to open a file another process holds.
func busy(error) bool { return false }
