// Package rmdir implements the engine behind rmdirx, a GNU-compatible implementation of
// the GNU/UNIX rmdir command: remove directories, but only if they are empty.
package rmdir

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "rmdirx"

// Options mirrors the GNU rmdir flag surface.
type Options struct {
	// Parents (-p) removes the operand and then each of its ancestors, so
	// "a/b/c" is treated like "a/b/c a/b a".
	Parents bool
	// IgnoreNonEmpty (--ignore-fail-on-non-empty) silences failures that are
	// solely because a directory is not empty.
	IgnoreNonEmpty bool
	// Verbose (-v) reports every directory processed.
	Verbose bool
}

// Rmdir is the engine. Construct it with New.
type Rmdir struct {
	Options

	diag *core.Diag
}

// New builds a rmdir engine writing to s.
func New(opts Options, s core.Streams) *Rmdir {
	return &Rmdir{Options: opts, diag: core.NewDiag(Name, s)}
}

// Run removes each directory in args. Diagnostics are written as they occur;
// the returned error is core.ErrFailed if any removal failed.
func (r *Rmdir) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	for _, path := range args {
		if !r.remove(path, "failed to remove") || !r.Parents {
			continue
		}
		for dir := parent(path); dir != ""; dir = parent(dir) {
			if !r.remove(dir, "failed to remove directory") {
				break
			}
		}
	}
	return r.diag.Result()
}

// remove deletes one empty directory, reporting whether it is gone. what is
// the lead-in of the diagnostic, which GNU words differently for operands and
// for the ancestors -p walks up to.
func (r *Rmdir) remove(path, what string) bool {
	if r.Verbose {
		r.diag.Printf("%s: removing directory, '%s'", Name, path)
	}

	// fsys.Remove would happily unlink a file, so the type is checked first.
	info, err := fsys.Lstat(path)
	if err != nil {
		r.diag.Errorf("%s '%s': %s", what, path, core.ErrText(err))
		return false
	}
	if !info.IsDir() {
		r.diag.Errorf("%s '%s': Not a directory", what, path)
		return false
	}

	err = fsys.Remove(path)
	switch {
	case err == nil:
		return true
	case nonEmpty(path, err):
		if !r.IgnoreNonEmpty {
			r.diag.Errorf("%s '%s': Directory not empty", what, path)
		}
	default:
		r.diag.Errorf("%s '%s': %s", what, path, core.ErrText(err))
	}
	return false
}

// nonEmpty reports whether removing path failed because it still has entries.
// The errno for that differs by platform (ENOTEMPTY, EEXIST, a Windows error
// code), so the directory is inspected directly where it can be read.
func nonEmpty(path string, err error) bool {
	if errors.Is(err, syscall.ENOTEMPTY) {
		return true
	}
	f, openErr := fsys.Open(path)
	if openErr != nil {
		return false
	}
	defer f.Close()
	names, _ := f.Readdirnames(1)
	return len(names) > 0
}

// parent returns path with its last component removed, or "" once nothing
// removable is left. The path is taken as written - no cleaning - so the
// names reported match what the user typed.
func parent(path string) string {
	end := len(path)
	for end > 0 && os.IsPathSeparator(path[end-1]) {
		end--
	}
	for end > 0 && !os.IsPathSeparator(path[end-1]) {
		end--
	}
	for end > 0 && os.IsPathSeparator(path[end-1]) {
		end--
	}

	dir := path[:end]
	// A bare volume ("C:") is a root, not a directory to remove.
	if dir == filepath.VolumeName(dir) {
		return ""
	}
	return dir
}
