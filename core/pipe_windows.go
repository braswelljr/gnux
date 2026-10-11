//go:build windows

package core

import (
	"errors"
	"io"
	"syscall"
)

// Windows reports a vanished pipe reader with one of two codes, depending on
// whether the pipe was already closed or is in the middle of closing. Go maps
// neither of them to syscall.EPIPE.
const (
	errorBrokenPipe syscall.Errno = 109 // ERROR_BROKEN_PIPE
	errorNoData     syscall.Errno = 232 // ERROR_NO_DATA: "The pipe is being closed."
)

// IsClosedPipe reports whether err means the reading end of an output pipe has
// gone away. A tool treats that as a quiet reason to stop, as the GNU tools do
// when SIGPIPE ends them.
func IsClosedPipe(err error) bool {
	return errors.Is(err, errorBrokenPipe) || errors.Is(err, errorNoData) ||
		errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
