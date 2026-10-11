//go:build !windows

package yes

import (
	"errors"
	"syscall"
)

// isBrokenPipe reports whether err means the reading end of the output pipe
// (or socket) has gone away.
func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
