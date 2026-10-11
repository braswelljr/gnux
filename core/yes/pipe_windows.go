//go:build windows

package yes

import (
	"errors"
	"syscall"
)

// Windows reports a vanished pipe reader with one of two codes, depending on
// whether the pipe was already closed or is in the middle of closing. Go maps
// neither of them to syscall.EPIPE.
const (
	errorBrokenPipe syscall.Errno = 109 // ERROR_BROKEN_PIPE
	errorNoData     syscall.Errno = 232 // ERROR_NO_DATA: "The pipe is being closed."
)

// isBrokenPipe reports whether err means the reading end of the output pipe
// has gone away.
func isBrokenPipe(err error) bool {
	return errors.Is(err, errorBrokenPipe) || errors.Is(err, errorNoData) ||
		errors.Is(err, syscall.EPIPE)
}
