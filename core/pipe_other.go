//go:build !windows

package core

import (
	"errors"
	"io"
	"syscall"
)

// IsClosedPipe reports whether err means the reading end of an output pipe has
// gone away. A tool treats that as a quiet reason to stop, as the GNU tools do
// when SIGPIPE ends them.
func IsClosedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
