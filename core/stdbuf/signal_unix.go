//go:build unix

package stdbuf

import (
	"os"
	"syscall"
)

func signalOf(ps *os.ProcessState) int {
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return int(ws.Signal())
	}
	return 0
}
