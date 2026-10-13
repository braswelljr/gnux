//go:build unix

package run

import (
	"os"
	"syscall"
)

// signalOf returns the number of the signal that ended the process.
func signalOf(state *os.ProcessState) int {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return int(ws.Signal())
	}
	return 0
}
