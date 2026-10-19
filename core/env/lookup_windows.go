//go:build windows

package env

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// defaultPathExt is used when PATHEXT is unset or empty.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// sameName reports whether two variable names denote the same variable.
// Windows environment variable names are case-insensitive.
func sameName(a, b string) bool {
	return strings.EqualFold(a, b)
}

// lookPath finds the program file using the PATH and PATHEXT returned by
// getenv, resolving relative names against dir. As Go's own lookup does, and
// unlike cmd.exe, it does not search the current directory implicitly. The
// error is fs.ErrNotExist when nothing by that name exists.
func lookPath(file, dir string, getenv func(string) (string, bool)) (string, error) {
	exts := pathExts(getenv)
	if strings.ContainsAny(file, `:\/`) {
		return withExt(resolve(dir, file), exts)
	}

	path, _ := getenv("PATH")
	for _, d := range filepath.SplitList(path) {
		if d == "" {
			continue
		}
		if p, err := withExt(resolve(dir, filepath.Join(d, file)), exts); err == nil {
			return p, nil
		}
	}
	return "", fs.ErrNotExist
}

// pathExts returns the extensions that mark a file as runnable.
func pathExts(getenv func(string) (string, bool)) []string {
	list, _ := getenv("PATHEXT")
	if list == "" {
		list = defaultPathExt
	}
	var exts []string
	for _, ext := range strings.Split(list, ";") {
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		exts = append(exts, ext)
	}
	return exts
}

// withExt returns p itself when it already names a file with an extension,
// otherwise p completed with the first extension that yields an existing file.
func withExt(p string, exts []string) (string, error) {
	if filepath.Ext(p) != "" && isFile(p) {
		return p, nil
	}
	for _, ext := range exts {
		if isFile(p + ext) {
			return p + ext, nil
		}
	}
	if _, err := fsys.Stat(p); err == nil {
		// It exists, but it is a directory or has no runnable extension.
		return "", fs.ErrPermission
	}
	return "", fs.ErrNotExist
}

// isFile reports whether p exists and is not a directory.
func isFile(p string) bool {
	info, err := fsys.Stat(p)
	return err == nil && !info.IsDir()
}

// exitStatus converts a finished command's state to the status envx exits
// with. Windows has no signals, so the exit code is the whole story.
func exitStatus(state *os.ProcessState) int {
	return state.ExitCode()
}
