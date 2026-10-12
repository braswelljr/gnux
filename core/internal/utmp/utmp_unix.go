//go:build unix

package utmp

import (
	"errors"
	"os"
	"syscall"
)

// ReadFile reads a utmp-format file in this platform's layout.
func ReadFile(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decode(data), nil
}

// Current returns the records of the users logged in now: the system's
// file, without the user records of processes that have since exited, and
// with a boot record.
func Current() ([]Entry, error) {
	entries, err := ReadFile(DefaultFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return withBoot(nil), nil
		}
		return nil, err
	}
	live := entries[:0]
	for _, e := range entries {
		if e.Type == UserProcess && e.PID > 0 && errors.Is(syscall.Kill(e.PID, 0), syscall.ESRCH) {
			continue
		}
		live = append(live, e)
	}
	return withBoot(live), nil
}
