//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package core

import "os"

// fileWidth is unavailable on this platform; COLUMNS or the 80-column
// default is used instead.
func fileWidth(_ *os.File) int {
	return 0
}
