//go:build windows

package config

import "os"

// userConfigDir is %AppData%, where Windows keeps a user's roaming
// settings.
func userConfigDir(getenv func(string) string) (string, error) {
	if d := getenv("APPDATA"); d != "" {
		return d, nil
	}
	return "", ErrNoDir
}

// openNonBlocking opens name for reading: a Windows named pipe is not a file
// in the configuration directory, and the regular-file check refuses one.
func openNonBlocking(name string) (*os.File, error) { return os.Open(name) }
