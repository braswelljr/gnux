// Package chmod implements the engine behind chmodx, a GNU-compatible
// implementation of the GNU/UNIX chmod command: change the permission bits of
// files.
package chmod

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "chmodx"

// Traversal says which symbolic links a recursive run follows (-H, -L, -P).
type Traversal int

const (
	// TraverseCommandLine (-H) follows a symbolic link given as an operand
	// but skips the ones met inside a directory tree. It is the default.
	TraverseCommandLine Traversal = iota
	// TraversePhysical (-P) follows no symbolic link while recursing, not
	// even an operand.
	TraversePhysical
	// TraverseLogical (-L) follows every symbolic link.
	TraverseLogical
)

// Options mirrors the GNU chmod flag surface.
type Options struct {
	// Changes (-c) reports only the files whose mode actually changed.
	Changes bool
	// Verbose (-v) reports every file processed.
	Verbose bool
	// Silent (-f) suppresses most error messages; the status still says
	// whether everything succeeded.
	Silent bool
	// Recursive (-R) changes directories and their contents.
	Recursive bool
	// Traversal selects the symbolic links a recursive run follows.
	Traversal Traversal
	// NoDereference (-h) changes a symbolic link itself rather than its
	// referent. Where links have no mode of their own (Linux), the link is
	// reported as unchanged, as GNU does.
	NoDereference bool
	// Dereference (--dereference) changes the referent of every symbolic
	// link, the ones met while recursing included, without descending
	// into them. With -R it needs -H or -L; NoDereference wins over it.
	Dereference bool
	// Reference (--reference=RFILE) takes the mode from RFILE instead of a
	// MODE operand.
	Reference string
	// NoPreserveRoot (--no-preserve-root) allows a recursive run on '/'.
	NoPreserveRoot bool
	// DiagnoseSurprises reports, and counts as a failure, a file left with
	// permission bits the mode would have cleared but for the umask. GNU does
	// this when the mode was written as an option ("chmod -w FILE"), so the
	// command line sets it for that form.
	DiagnoseSurprises bool
}

// Chmod is the engine. Construct it with New.
type Chmod struct {
	Options

	// ChmodFn changes a file's mode; it defaults to fsys.Chmod and is the
	// injection point for tests.
	ChmodFn func(path string, mode os.FileMode) error
	// LchmodFn changes the mode of a symbolic link itself; it fails with
	// EOPNOTSUPP where links have no mode. It is the injection point for -h.
	LchmodFn func(path string, mode os.FileMode) error
	// Umask returns the file-creation mask a symbolic mode without a user
	// class honors; it defaults to the process umask.
	Umask func() uint32

	diag   *core.Diag
	failed bool
	change change
	umask  uint32
}

// New builds a chmod engine writing to s.
func New(opts Options, s core.Streams) *Chmod {
	return &Chmod{
		Options:  opts,
		ChmodFn:  fsys.Chmod,
		LchmodFn: lchmod,
		Umask:    processUmask,
		diag:     core.NewDiag(Name, s),
	}
}

// Run applies the mode to each operand: "MODE FILE..." or, with Reference,
// "FILE...". A failing operand is reported and the rest still run. The
// returned error is core.ErrFailed if anything failed, or a *core.UsageError
// for a command-line mistake.
func (c *Chmod) Run(args []string) error {
	if c.Recursive && c.Traversal == TraversePhysical && c.Dereference && !c.NoDereference {
		c.diag.Errorf("-R --dereference requires either -H or -L")
		return core.ErrFailed
	}
	if c.Reference != "" {
		info, err := fsys.Stat(c.Reference)
		if err != nil {
			c.diag.Errorf("failed to get attributes of '%s': %s", c.Reference, core.ErrText(err))
			return core.ErrFailed
		}
		// Five digits: a reference mode sets a directory's set-ID bits too.
		bits := unixBits(info.Mode())
		c.change = change{text: "00000", octal: true, value: bits}
	} else {
		if len(args) == 0 {
			return core.Usagef("missing operand")
		}
		ch, err := parseChange(args[0])
		if err != nil {
			return core.Usagef("invalid mode: '%s'", args[0])
		}
		c.change = ch
		args = args[1:]
	}
	if len(args) == 0 {
		if c.Reference != "" {
			return core.Usagef("missing operand")
		}
		return core.Usagef("missing operand after '%s'", c.change.text)
	}

	c.umask = c.Umask()
	for _, arg := range args {
		c.operand(arg)
	}
	if c.failed || c.diag.Failed() {
		return core.ErrFailed
	}
	return nil
}

// errorf reports a failure, unless -f asked for silence.
func (c *Chmod) errorf(format string, a ...any) {
	c.failed = true
	if !c.Silent {
		c.diag.Errorf(format, a...)
	}
}

func (c *Chmod) operand(path string) {
	info, err := fsys.Lstat(path)
	if err != nil {
		c.errorf("cannot access '%s': %s", path, core.ErrText(err))
		c.inaccessible(path)
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// An operand is followed unless -h, or -R -P, says otherwise.
		if c.NoDereference || c.Recursive && c.Traversal == TraversePhysical {
			c.link(path, info)
			return
		}
		if info, err = fsys.Stat(path); err != nil {
			c.errorf("cannot operate on dangling symlink '%s'", path)
			c.inaccessible(path)
			return
		}
	}

	if c.Recursive && info.IsDir() && !c.NoPreserveRoot && core.IsRoot(path) {
		if filepath.Clean(path) == "/" {
			c.errorf("it is dangerous to operate recursively on '%s'", path)
		} else {
			c.errorf("it is dangerous to operate recursively on '%s' (same as '/')", path)
		}
		c.errorf("use --no-preserve-root to override this failsafe")
		return
	}
	c.visit(path, info, nil)
}

// link changes the mode of a symbolic link that is not followed. Where links
// have no mode of their own, nothing changes, which is no failure.
func (c *Chmod) link(path string, info os.FileInfo) {
	c.applyWith(path, info, true, func(path string, mode os.FileMode) error {
		err := c.LchmodFn(path, mode)
		if err != nil && linkModeUnsupported(err) {
			return errNotApplied
		}
		return err
	})
}

// errNotApplied says a symbolic link was left alone because links have no
// mode of their own here.
var errNotApplied = errors.New("not applied")

// inaccessible notes, for -v, a file whose mode could not be read.
func (c *Chmod) inaccessible(path string) {
	if c.Verbose {
		c.diag.Printf("'%s' could not be accessed", path)
	}
}

// visit changes path, then, when recursing, its contents (pre-order, as GNU
// chmod does: a directory made unreadable cannot then be descended into).
func (c *Chmod) visit(path string, info os.FileInfo, ancestors []os.FileInfo) {
	c.apply(path, info)
	if !c.Recursive || !info.IsDir() {
		return
	}
	for _, a := range ancestors {
		if os.SameFile(a, info) {
			c.errorf("WARNING: Circular directory structure.\nThis almost certainly means that you have a corrupted file system.\nNOTIFY YOUR SYSTEM MANAGER.\nThe following directory is part of the cycle:\n  '%s'", path)
			return
		}
	}

	entries, err := fsys.ReadDir(path)
	if err != nil {
		c.errorf("cannot read directory '%s': %s", path, core.ErrText(err))
		return
	}
	ancestors = append(ancestors, info)
	for _, e := range entries {
		child := filepath.Join(path, e.Name())
		ci, err := fsys.Lstat(child)
		if err != nil {
			c.errorf("cannot access '%s': %s", child, core.ErrText(err))
			c.inaccessible(child)
			continue
		}
		if ci.Mode()&os.ModeSymlink != 0 {
			// -L descends through every link; --dereference changes the
			// referent without descending; otherwise the link itself.
			follow := c.Traversal == TraverseLogical
			if !follow && (!c.Dereference || c.NoDereference) {
				c.link(child, ci)
				continue
			}
			if ci, err = fsys.Stat(child); err != nil {
				c.errorf("cannot operate on dangling symlink '%s'", child)
				c.inaccessible(child)
				continue
			}
			if !follow {
				c.apply(child, ci)
				continue
			}
		}
		c.visit(child, ci, ancestors)
	}
}

// apply changes one file's mode and reports it as -c and -v ask.
func (c *Chmod) apply(path string, info os.FileInfo) {
	c.applyWith(path, info, false, c.ChmodFn)
}

// applyWith is apply with the function that changes the mode, called even
// for an unchanged mode when always is set (a link may have no mode at all).
func (c *Chmod) applyWith(path string, info os.FileInfo, always bool, chmod func(string, os.FileMode) error) {
	old := unixBits(info.Mode())
	dir := info.IsDir()
	mode, err := c.change.apply(old, dir, c.umask)
	if err != nil {
		c.errorf("invalid mode: '%s'", c.change.text)
		return
	}

	if mode != old || always {
		if err := chmod(path, fileMode(mode)); err != nil {
			if errors.Is(err, errNotApplied) {
				if c.Verbose {
					c.diag.Printf("neither symbolic link '%s' nor referent has been changed", path)
				}
				return
			}
			if c.Verbose {
				c.diag.Printf("failed to change mode of '%s' from %s to %s", path, describe(old), describe(mode))
			}
			c.errorf("changing permissions of '%s': %s", path, errText(err))
			return
		}
	}
	if mode != old {
		if c.Verbose || c.Changes {
			c.diag.Printf("mode of '%s' changed from %s to %s", path, describe(old), describe(mode))
		}
	} else if c.Verbose {
		c.diag.Printf("mode of '%s' retained as %s", path, describe(old))
	}

	// A symbolic mode with no user class honors the umask; GNU reports when
	// that left bits set that the mode literally says to clear.
	if c.DiagnoseSurprises && !c.change.octal && c.umask != 0 {
		if naive, err := c.change.apply(old, dir, 0); err == nil && mode&^naive != 0 {
			c.errorf("%s: new permissions are %s, not %s", path, permString(mode), permString(naive))
		}
	}
}

// errText is core.ErrText, with EPERM spelled as GNU spells it.
func errText(err error) string {
	if errors.Is(err, fs.ErrPermission) {
		var perr *fs.PathError
		if errors.As(err, &perr) && errors.Is(perr.Err, syscall.EPERM) {
			return "Operation not permitted"
		}
	}
	return core.ErrText(err)
}
