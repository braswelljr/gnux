//go:build unix || windows

package themepick

import (
	"os"
	"syscall"
)

// exitSignals end the picker from outside; it restores the terminal first.
var exitSignals = []os.Signal{syscall.SIGTERM, syscall.SIGHUP}
