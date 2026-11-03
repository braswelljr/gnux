// Package config keeps what gnux saves for a user between runs: the
// built-in theme chosen with gnux theme. The file holds one theme name and
// nothing else, so it can only pick among the themes gnux ships, never
// bring codes of its own to the terminal.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// maxThemeFile is the most a theme file may hold: the longest built-in
// name is far shorter, so anything larger is not a file gnux wrote.
const maxThemeFile = 64

// ErrNoDir is returned when no configuration directory can be found.
var ErrNoDir = errors.New("no configuration directory: set GNUX_CONFIG_DIR, XDG_CONFIG_HOME or HOME")

// Dir is gnux's configuration directory: GNUX_CONFIG_DIR when set, else
// the system's per-user configuration directory followed by "gnux"
// ($XDG_CONFIG_HOME/gnux or ~/.config/gnux; %AppData%\gnux on Windows).
func Dir(getenv func(string) string) (string, error) {
	if d := getenv("GNUX_CONFIG_DIR"); d != "" {
		return d, nil
	}
	base, err := userConfigDir(getenv)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "gnux"), nil
}

// ThemeFile is the file the saved theme lives in.
func ThemeFile(getenv func(string) string) (string, error) {
	d, err := Dir(getenv)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "theme"), nil
}

// Theme returns the saved theme's name, lower-cased, and the file it was
// read from. Nothing saved, or no configuration directory, is "" and no
// error. A file that is not a regular file, is too large or names no
// built-in theme is an error, and is never read past maxThemeFile bytes.
func Theme(getenv func(string) string) (name, path string, err error) {
	path, err = ThemeFile(getenv)
	if err != nil {
		return "", "", nil
	}
	// Opened without blocking (a FIFO put in its place would otherwise hang
	// every tool that looks for a theme), then checked as the file it is,
	// so nothing can be swapped in between the check and the read.
	f, err := openNonBlocking(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", path, nil
	case err != nil:
		return "", path, err
	}
	defer f.Close()
	info, err := f.Stat()
	switch {
	case err != nil:
		return "", path, err
	case !info.Mode().IsRegular():
		return "", path, errors.New("not a regular file")
	case info.Size() > maxThemeFile:
		return "", path, errors.New("too large to hold a theme name")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxThemeFile+1))
	if err != nil {
		return "", path, err
	}
	if len(data) > maxThemeFile {
		return "", path, errors.New("too large to hold a theme name")
	}
	name = strings.ToLower(strings.TrimSpace(string(data)))
	if name == "" {
		return "", path, nil
	}
	if !Valid(name) {
		return "", path, fmt.Errorf("unknown theme %q", name)
	}
	return name, path, nil
}

// Valid reports whether name is a built-in theme, in any case.
func Valid(name string) bool {
	for _, t := range core.ThemeNames() {
		if strings.EqualFold(t, name) {
			return true
		}
	}
	return false
}

// SaveTheme saves name, which must be a built-in theme, as the user's
// theme and returns the file. The file is replaced whole (written aside,
// then renamed), so a reader never sees half of it, and only its owner may
// read or write it.
func SaveTheme(getenv func(string) string, name string) (string, error) {
	if !Valid(name) {
		return "", fmt.Errorf("unknown theme %q", name)
	}
	path, err := ThemeFile(getenv)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return path, err
	}
	tmp, err := os.CreateTemp(dir, ".theme-*")
	if err != nil {
		return path, err
	}
	defer os.Remove(tmp.Name()) // gone already once renamed
	if _, err := io.WriteString(tmp, strings.ToLower(name)+"\n"); err != nil {
		tmp.Close()
		return path, err
	}
	if err := tmp.Close(); err != nil {
		return path, err
	}
	return path, os.Rename(tmp.Name(), path)
}

// ResetTheme removes the saved theme, so the tools go back to GNUX_THEME or
// the default. It reports whether there was one.
func ResetTheme(getenv func(string) string) (path string, removed bool, err error) {
	path, err = ThemeFile(getenv)
	if err != nil {
		return "", false, err
	}
	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, false, nil
	}
	return path, err == nil, err
}
