// Package ln implements the engine behind lnx, a GNU-compatible implementation of the
// GNU/UNIX ln command: make hard or symbolic links between files.
package ln

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic and prompt.
const Name = "lnx"

// Options mirrors the GNU ln flag surface.
type Options struct {
	Symbolic          bool   // -s: make symbolic links instead of hard links
	Force             bool   // -f: remove existing destination files
	Interactive       bool   // -i: prompt before removing a destination
	NoDereference     bool   // -n: treat a LINK_NAME that is a symlink to a directory as a file
	Verbose           bool   // -v: print the name of each linked file
	Relative          bool   // -r: with -s, make the link relative to its own location
	TargetDirectory   string // -t: create the links in this directory
	NoTargetDirectory bool   // -T: always treat LINK_NAME as a normal file
	Logical           bool   // -L: hard link to the referent of a symlink TARGET
	Physical          bool   // -P: hard link to a symlink TARGET itself (the default)
}

// Ln is the link engine. Construct it with New.
type Ln struct {
	Options

	// PromptFn, when non-nil, is consulted instead of reading from standard
	// input. It is the injection point tests use to answer replace prompts.
	PromptFn func(question string) bool

	diag   *core.Diag
	prompt *core.Prompter
}

// New builds a link engine writing to s.
func New(opts Options, s core.Streams) *Ln {
	return &Ln{
		Options: opts,
		diag:    core.NewDiag(Name, s),
		prompt:  core.NewPrompter(Name, s),
	}
}

// Run creates links according to args, which take one of ln's four forms:
// TARGET LINK_NAME, TARGET (link in the current directory), TARGET...
// DIRECTORY, or (with -t) TARGET.... Diagnostics are written as they occur;
// the returned error is core.ErrFailed if any link failed, a *core.UsageError
// for a malformed command line, otherwise nil.
func (l *Ln) Run(args []string) error {
	switch {
	case l.Relative && !l.Symbolic:
		return core.Usagef("cannot do --relative without --symbolic")
	case l.TargetDirectory != "" && l.NoTargetDirectory:
		return core.Usagef("cannot combine --target-directory and --no-target-directory")
	case len(args) == 0:
		return core.Usagef("missing file operand")
	}

	var (
		targets []string
		dir     string
	)
	switch {
	case l.TargetDirectory != "":
		targets, dir = args, l.TargetDirectory
		info, err := os.Stat(dir)
		if err == nil && !info.IsDir() {
			l.diag.Errorf("target directory '%s': Not a directory", dir)
			return l.diag.Result()
		}
		if err != nil {
			l.diag.Errorf("target directory '%s': %s", dir, core.ErrText(err))
			return l.diag.Result()
		}
	case l.NoTargetDirectory:
		switch {
		case len(args) < 2:
			return core.Usagef("missing destination file operand after '%s'", args[0])
		case len(args) > 2:
			return core.Usagef("extra operand '%s'", args[2])
		}
		l.link(args[0], args[1])
		return l.diag.Result()
	case len(args) == 1:
		// "ln TARGET": the link takes TARGET's name in the current directory.
		// GNU spells that destination "./NAME" in its messages.
		l.link(args[0], "."+string(filepath.Separator)+baseName(args[0]))
		return l.diag.Result()
	case len(args) == 2 && !l.isDir(args[1]):
		l.link(args[0], args[1])
		return l.diag.Result()
	default:
		targets, dir = args[:len(args)-1], args[len(args)-1]
		if !l.isDir(dir) {
			return core.Usagef("target '%s' is not a directory", dir)
		}
	}

	for _, target := range targets {
		l.link(target, filepath.Join(dir, baseName(target)))
	}
	return l.diag.Result()
}

// isDir reports whether path should be treated as a directory to create links
// in. Under -n a symbolic link to a directory counts as a plain file, so it
// can be replaced rather than descended into.
func (l *Ln) isDir(path string) bool {
	stat := os.Stat
	if l.NoDereference {
		stat = os.Lstat
	}
	info, err := stat(path)
	return err == nil && info.IsDir()
}

// link creates the link dst pointing at target.
func (l *Ln) link(target, dst string) {
	source := target
	var ti os.FileInfo
	if !l.Symbolic {
		var err error
		if l.Logical {
			ti, err = os.Stat(target)
		} else {
			ti, err = os.Lstat(target)
		}
		if err != nil {
			l.diag.Errorf("failed to access '%s': %s", target, core.ErrText(err))
			return
		}
		if ti.IsDir() {
			l.diag.Errorf("'%s': hard link not allowed for directory", target)
			return
		}
		if l.Logical {
			// link(2) does not follow a symlink everywhere; resolve it so -L
			// links to the referent on every platform.
			if resolved, err := filepath.EvalSymlinks(target); err == nil {
				source = resolved
			}
		}
	}

	replace := false
	if di, err := os.Lstat(dst); err == nil && (l.Force || l.Interactive) {
		if di.IsDir() {
			l.diag.Errorf("'%s': cannot overwrite directory", dst)
			return
		}
		if ti == nil {
			ti, _ = os.Stat(target)
		}
		if ti != nil && os.SameFile(ti, di) {
			// Replacing an entry with a link to itself would destroy the
			// only name the data has.
			if sameEntry(target, dst) {
				l.diag.Errorf("'%s' and '%s' are the same file", target, dst)
				return
			}
			// A different name for the same file is the hard link that was
			// asked for; re-running "ln -f a b" must stay a no-op.
			if !l.Symbolic {
				if l.Verbose {
					l.diag.Printf("'%s' => '%s'", dst, target)
				}
				return
			}
		}
		if l.Interactive && !l.ask(fmt.Sprintf("replace '%s'?", dst)) {
			return
		}
		replace = true
	}

	value := target
	if l.Relative {
		value = relativeTarget(dst, target)
	}
	create := func(name string) error {
		if l.Symbolic {
			return os.Symlink(value, name)
		}
		return os.Link(source, name)
	}

	var err error
	if replace {
		err = replaceWith(dst, create)
	} else {
		err = create(dst)
	}
	if err != nil {
		l.reportFailure(target, dst, err)
		return
	}

	if l.Verbose {
		arrow := "=>"
		if l.Symbolic {
			arrow = "->"
		}
		l.diag.Printf("'%s' %s '%s'", dst, arrow, value)
	}
}

// reportFailure words a failed link the way GNU ln does: the target is only
// named for hard links, and not when the destination itself is the problem.
func (l *Ln) reportFailure(target, dst string, err error) {
	switch {
	case l.Symbolic:
		l.diag.Errorf("failed to create symbolic link '%s': %s", dst, core.ErrText(err))
	case errors.Is(err, fs.ErrExist):
		l.diag.Errorf("failed to create hard link '%s': %s", dst, core.ErrText(err))
	default:
		l.diag.Errorf("failed to create hard link '%s' => '%s': %s", dst, target, core.ErrText(err))
	}
}

// ask poses a yes/no question, returning true on an affirmative answer.
func (l *Ln) ask(question string) bool {
	if l.PromptFn != nil {
		return l.PromptFn(question)
	}
	return l.prompt.Ask(question)
}

// replaceWith atomically replaces dst: the new link is created under a
// temporary name beside it and renamed into place, so dst is never missing
// and survives a failure to create its replacement.
func replaceWith(dst string, create func(name string) error) error {
	dir := filepath.Dir(dst)
	var (
		tmp string
		err error
	)
	for range 16 {
		tmp = filepath.Join(dir, fmt.Sprintf(".%s-%08x", Name, rand.Uint32()))
		if err = create(tmp); !errors.Is(err, fs.ErrExist) {
			break
		}
	}
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace %s: %w", dst, err)
	}
	return nil
}

// relativeTarget returns target expressed relative to the directory holding
// the link dst, for -r. Symbolic links in both are resolved first, as GNU ln
// does; if no relative path exists (different volumes) the absolute, resolved
// target is used.
func relativeTarget(dst, target string) string {
	abs := realPath(target)
	rel, err := filepath.Rel(realPath(filepath.Dir(dst)), abs)
	if err != nil {
		return abs
	}
	return rel
}

// realPath returns the absolute, symlink-free form of path, resolving as much
// of it as exists.
func realPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return abs
	}
	return filepath.Join(realPath(parent), filepath.Base(abs))
}

// sameEntry reports whether a and b are the same name in the same directory,
// as opposed to two hard links to one file.
func sameEntry(a, b string) bool {
	if baseName(a) != baseName(b) {
		return false
	}
	da, err := os.Stat(filepath.Dir(filepath.Clean(a)))
	if err != nil {
		return false
	}
	db, err := os.Stat(filepath.Dir(filepath.Clean(b)))
	if err != nil {
		return false
	}
	return os.SameFile(da, db)
}

// baseName returns the last component of path, ignoring trailing separators.
func baseName(path string) string {
	return filepath.Base(filepath.Clean(path))
}
