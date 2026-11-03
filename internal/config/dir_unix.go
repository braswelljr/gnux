//go:build unix

package config

import (
	"os"
	"path/filepath"
	"syscall"
)

// userConfigDir is $XDG_CONFIG_HOME, else ~/.config: where command-line
// tools keep their settings on Linux, the BSDs and macOS alike.
func userConfigDir(getenv func(string) string) (string, error) {
	if d := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(d) {
		return d, nil
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".config"), nil
	}
	return "", ErrNoDir
}

// openNonBlocking opens name for reading with O_NONBLOCK, which makes opening
// a FIFO return at once instead of waiting for a writer.
func openNonBlocking(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
