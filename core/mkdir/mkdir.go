// Package mkdir implements the engine behind mkdirx, a GNU-compatible implementation of
// the GNU/UNIX mkdir command: create directories, optionally with their
// parents and an explicit mode.
package mkdir

import (
	"os"

	"github.com/braswelljr/gnux/core/fsys"
	filemode "github.com/braswelljr/gnux/core/internal/mode"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "mkdirx"

// Options mirrors the GNU mkdir flag surface.
type Options struct {
	// Parents (-p) creates missing parent directories and does not complain
	// about directories that already exist.
	Parents bool
	// Mode (-m) is the permission mode for the new directories, octal ("755")
	// or symbolic ("u=rwx,go=rx"). Empty means a=rwx minus the umask.
	Mode string
	// Verbose (-v) reports every directory created.
	Verbose bool
}

// Mkdir is the engine. Construct it with New.
type Mkdir struct {
	Options

	// Umask returns the process file-creation mask. It is only consulted for
	// symbolic modes that name no user class ("+x"), and is the injection
	// point tests use to pin that case down.
	Umask func() uint32

	diag *core.Diag
}

// New builds a mkdir engine writing to s.
func New(opts Options, s core.Streams) *Mkdir {
	return &Mkdir{
		Options: opts,
		Umask:   filemode.Umask,
		diag:    core.NewDiag(Name, s),
	}
}

// Run creates each directory in args. Diagnostics are written as they occur;
// the returned error is core.ErrFailed if any directory could not be created.
func (m *Mkdir) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	mode, explicit := os.FileMode(0o777), false
	if m.Mode != "" {
		bits, err := parseMode(m.Mode, m.Umask)
		if err != nil {
			return core.Usagef("invalid mode '%s'", m.Mode)
		}
		mode, explicit = fileMode(bits), true
	}

	for _, path := range args {
		m.create(path, mode, explicit)
	}
	return m.diag.Result()
}

// create makes one operand, with its parents first under -p.
func (m *Mkdir) create(path string, mode os.FileMode, explicit bool) {
	// A trailing separator names the same directory; dropping it keeps the
	// parent walk from treating the operand itself as an intermediate.
	target := path
	for len(target) > 1 && os.IsPathSeparator(target[len(target)-1]) {
		target = target[:len(target)-1]
	}

	if m.Parents && !m.createParents(target) {
		return
	}

	if err := fsys.Mkdir(target, mode.Perm()); err != nil {
		if m.Parents && isDir(target) {
			return
		}
		m.diag.Errorf("cannot create directory '%s': %s", path, mkdirText(target, err))
		return
	}

	// The kernel masks the mode passed to mkdir(2) with the umask and ignores
	// the special bits, so an explicit -m has to be applied afterwards.
	if explicit {
		if err := fsys.Chmod(target, mode); err != nil {
			m.diag.Errorf("cannot change permissions of '%s': %s", path, core.ErrText(err))
			return
		}
	}
	m.created(path)
}

// createParents makes every missing ancestor of target, reporting whether the
// operand itself can now be created.
func (m *Mkdir) createParents(target string) bool {
	for i := 1; i < len(target); i++ {
		if !os.IsPathSeparator(target[i]) || os.IsPathSeparator(target[i-1]) {
			continue
		}
		parent := target[:i]

		if err := fsys.Mkdir(parent, 0o777); err != nil {
			// Already there (or created by someone else meanwhile) is fine,
			// and so is a prefix that is not creatable at all, like "C:".
			if isDir(parent) {
				continue
			}
			m.diag.Errorf("cannot create directory '%s': %s", parent, mkdirText(parent, err))
			return false
		}
		ensureTraversable(parent)
		m.created(parent)
	}
	return true
}

func (m *Mkdir) created(path string) {
	if m.Verbose {
		m.diag.Printf("%s: created directory '%s'", Name, path)
	}
}

// ensureTraversable gives the owner write and search permission on a freshly
// made intermediate directory. A restrictive umask could otherwise leave it
// unusable for creating the next component inside it.
func ensureTraversable(path string) {
	info, err := fsys.Stat(path)
	if err != nil {
		return
	}
	const special = os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if mode := info.Mode() & (os.ModePerm | special); mode&0o300 != 0o300 {
		_ = fsys.Chmod(path, mode|0o300)
	}
}

// isDir reports whether path is a directory, following symbolic links so a
// link to a directory counts as an existing parent.
func isDir(path string) bool {
	info, err := fsys.Stat(path)
	return err == nil && info.IsDir()
}

// mkdirText renders a mkdir failure. Platforms disagree on the errno for a
// path that is already there (EEXIST, EISDIR for a root, access denied on
// Windows), so anything that exists is reported the way GNU does.
func mkdirText(path string, err error) string {
	if _, statErr := fsys.Lstat(path); statErr == nil {
		return "File exists"
	}
	return core.ErrText(err)
}
