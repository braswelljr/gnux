//go:build !windows

package env

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"
)

// defaultPath is searched when PATH is not set at all, as execvp does. It is
// what keeps `envx -i ls` working.
const defaultPath = "/bin:/usr/bin"

// sameName reports whether two variable names denote the same variable.
func sameName(a, b string) bool {
	return a == b
}

// lookPath finds the program file the way execvp would, using the PATH
// returned by getenv and resolving relative names against dir. The error is
// fs.ErrNotExist when nothing by that name exists, and another error
// (typically fs.ErrPermission) when something exists but cannot be executed.
func lookPath(file, dir string, getenv func(string) (string, bool)) (string, error) {
	if strings.Contains(file, "/") {
		p := resolve(dir, file)
		return p, executable(p)
	}

	path, ok := getenv("PATH")
	if !ok {
		path = defaultPath
	}
	dirs := filepath.SplitList(path)
	if len(dirs) == 0 {
		// An empty PATH means the current directory.
		dirs = []string{"."}
	}

	denied := false
	for _, d := range dirs {
		if d == "" {
			d = "."
		}
		p := resolve(dir, filepath.Join(d, file))
		err := executable(p)
		if err == nil {
			return p, nil
		}
		// Keep looking, but remember it: a later directory may hold a usable
		// program, and if none does "Permission denied" is the better answer.
		if errors.Is(err, fs.ErrPermission) {
			denied = true
		}
	}
	if denied {
		return "", fs.ErrPermission
	}
	return "", fs.ErrNotExist
}

// executable reports whether p is a file with an execute bit set.
func executable(p string) error {
	info, err := fsys.Stat(p)
	if err != nil {
		return err
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return fs.ErrPermission
	}
	return nil
}

// exitStatus converts a finished command's state to the status envx exits
// with. A command killed by a signal has no exit status of its own; it is
// reported as 128 plus the signal number, the shell convention.
func exitStatus(state *os.ProcessState) int {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return state.ExitCode()
}
