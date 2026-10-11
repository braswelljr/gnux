//go:build windows

package tail

import "syscall"

const (
	// processQueryLimitedInformation is the least access that allows
	// GetExitCodeProcess.
	processQueryLimitedInformation = 0x1000
	// stillActive is the exit code reported for a process that has not exited.
	stillActive = 259
)

// processAlive reports whether pid names a running process.
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()

	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
