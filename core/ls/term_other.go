//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package ls

import "os"

// terminalWidth is unavailable on this platform; COLUMNS or the 80-column
// default is used instead.
func terminalWidth(_ *os.File) int {
	return 0
}
