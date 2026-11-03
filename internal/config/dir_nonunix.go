//go:build !unix && !windows

package config

import (
	"os"
	"path/filepath"
)

// userConfigDir is $XDG_CONFIG_HOME, else ~/.config, as on Unix.
func userConfigDir(getenv func(string) string) (string, error) {
	if d := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(d) {
		return d, nil
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".config"), nil
	}
	return "", ErrNoDir
}

// openNonBlocking opens name for reading: there are no FIFOs to wait on here.
func openNonBlocking(name string) (*os.File, error) { return os.Open(name) }
