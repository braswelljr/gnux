// Package realpath implements the engine behind realpathx, a GNU-compatible
// implementation of the GNU realpath command: print the resolved absolute name
// of each file.
package realpath

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "realpathx"

// maxLinks bounds how many symbolic links one name may expand through before
// it is treated as a loop.
const maxLinks = 40

// These failures are detected by the engine rather than reported by the
// kernel, so they carry GNU's wording (core.ErrText capitalizes it) instead of
// an errno whose text would vary by platform.
var (
	errNotDir = errors.New("not a directory")
	errLoop   = errors.New("too many levels of symbolic links")
)

// Mode says how much of a name has to exist for it to be canonicalized.
type Mode int

const (
	// ModeDefault requires every component but the last to exist.
	ModeDefault Mode = iota
	// ModeExisting (-e) requires every component to exist.
	ModeExisting
	// ModeMissing (-m) requires nothing to exist.
	ModeMissing
)

// Options mirrors the GNU realpath flag surface.
type Options struct {
	// Mode selects between the default, -e and -m existence requirements.
	Mode Mode
	// Logical (-L) resolves ".." components before symbolic links. When false
	// (-P, the default) links are resolved as they are encountered.
	Logical bool
	// Quiet (-q) suppresses most error messages; the exit status still
	// reports them.
	Quiet bool
	// Strip (-s, --no-symlinks) does not expand symbolic links at all.
	Strip bool
	// Zero (-z) ends each output line with NUL instead of a newline.
	Zero bool
	// RelativeTo (--relative-to) prints names relative to this directory.
	RelativeTo string
	// RelativeBase (--relative-base) prints absolute names unless they are
	// below this directory.
	RelativeBase string
}

// Realpath is the engine. Construct it with New.
type Realpath struct {
	Options

	// Getwd is the injection point tests use to control the directory
	// relative names are resolved against; it defaults to os.Getwd.
	Getwd func() (string, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a realpath engine writing to s.
func New(opts Options, s core.Streams) *Realpath {
	return &Realpath{
		Options: opts,
		Getwd:   os.Getwd,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the canonical name of each file in args. The returned error is
// core.ErrFailed if any name could not be resolved.
func (r *Realpath) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	relTo, relBase, err := r.relativeDirs()
	if err != nil {
		return err
	}

	end := "\n"
	if r.Zero {
		end = "\x00"
	}
	failed := false
	for _, path := range args {
		resolved, err := r.canonicalize(path)
		if err != nil {
			failed = true
			if !r.Quiet {
				r.diag.Errorf("%s: %s", quoteEmpty(path), core.ErrText(err))
			}
			continue
		}
		fmt.Fprint(r.streams.Out, relative(resolved, relTo, relBase)+end)
	}
	if failed {
		return core.ErrFailed
	}
	return nil
}

// relativeDirs canonicalizes --relative-to and --relative-base. A directory
// that cannot be resolved is fatal: every result would be printed relative to
// the wrong place.
func (r *Realpath) relativeDirs() (relTo, relBase string, err error) {
	resolve := func(dir string) (string, error) {
		resolved, err := r.canonicalize(dir)
		if err == nil && r.Mode == ModeExisting && !isDir(resolved) {
			err = errNotDir
		}
		if err != nil {
			r.diag.Errorf("%s: %s", dir, core.ErrText(err))
			return "", core.ErrFailed
		}
		return resolved, nil
	}

	// --relative-base alone also serves as the directory to be relative to.
	to := r.RelativeTo
	if to == "" {
		to = r.RelativeBase
	}
	if to == "" {
		return "", "", nil
	}
	if relTo, err = resolve(to); err != nil {
		return "", "", err
	}
	switch r.RelativeBase {
	case "":
		return relTo, "", nil
	case to:
		return relTo, relTo, nil
	}

	if relBase, err = resolve(r.RelativeBase); err != nil {
		return "", "", err
	}
	// A --relative-to outside the base cannot give relative names that stay
	// under the base, so everything is printed absolute.
	if !hasPrefix(relBase, relTo) {
		return "", "", nil
	}
	return relTo, relBase, nil
}

// canonicalize returns the absolute name of path with symbolic links and
// "."/".." components resolved according to the options.
func (r *Realpath) canonicalize(path string) (string, error) {
	if path == "" {
		return "", fs.ErrNotExist
	}
	abs, err := r.absolute(path)
	if err != nil {
		return "", err
	}

	if r.Strip {
		return r.lexical(abs)
	}
	if r.Logical {
		// Collapsing ".." first is what makes the result logical. Clean also
		// drops a trailing separator, which still has to demand a directory.
		trailing := os.IsPathSeparator(abs[len(abs)-1])
		abs = filepath.Clean(abs)
		if trailing {
			abs += string(filepath.Separator)
		}
	}
	return r.resolve(abs)
}

// absolute anchors path at the working directory without cleaning it: "a/.."
// must survive until the links in "a" have been resolved.
func (r *Realpath) absolute(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	// Windows has names that are neither absolute nor relative to the working
	// directory ("\dir", "C:dir"); filepath.Abs knows how to anchor those.
	// Neither condition can hold for a non-absolute name on Unix.
	if filepath.VolumeName(path) != "" || os.IsPathSeparator(path[0]) {
		return filepath.Abs(path)
	}
	wd, err := r.Getwd()
	if err != nil {
		return "", err
	}
	return wd + string(filepath.Separator) + path, nil
}

// lexical canonicalizes abs without expanding symbolic links (-s), applying
// only the mode's existence requirement.
func (r *Realpath) lexical(abs string) (string, error) {
	trailing := os.IsPathSeparator(abs[len(abs)-1])
	clean := filepath.Clean(abs)

	switch r.Mode {
	case ModeExisting:
		info, err := os.Stat(clean)
		if err != nil {
			return "", err
		}
		if trailing && !info.IsDir() {
			return "", errNotDir
		}
	case ModeDefault:
		info, err := os.Stat(filepath.Dir(clean))
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", errNotDir
		}
	case ModeMissing:
	}
	return clean, nil
}

// resolve walks abs one component at a time, expanding each symbolic link as
// it is reached so that a following ".." applies to the link's target.
func (r *Realpath) resolve(abs string) (string, error) {
	vol := filepath.VolumeName(abs)
	dest := vol + string(filepath.Separator)
	todo := split(abs[len(vol):])

	destIsDir := true // dest starts at the root
	missing := false  // set once a component is absent: nothing below it can be looked up
	links := 0

	for len(todo) > 0 {
		comp := todo[0]
		todo = todo[1:]

		if comp == "." || comp == ".." {
			// Both only make sense inside a directory; "file/.." is an error
			// even though dropping the last component would "work".
			if !destIsDir && !missing && r.Mode != ModeMissing {
				return "", errNotDir
			}
			if comp == ".." {
				dest = filepath.Dir(dest)
				destIsDir = true
			}
			continue
		}

		next := filepath.Join(dest, comp)
		if missing {
			dest = next
			continue
		}

		info, err := os.Lstat(next)
		if err != nil {
			if !r.tolerates(err, todo) {
				return "", err
			}
			missing, dest = true, next
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			dest, destIsDir = next, info.IsDir()
			continue
		}

		if links++; links > maxLinks {
			return "", errLoop
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(target) {
			targetVol := filepath.VolumeName(target)
			dest, destIsDir = targetVol+string(filepath.Separator), true
			target = target[len(targetVol):]
		}
		todo = append(split(target), todo...)
	}
	return dest, nil
}

// tolerates reports whether failing to look up a component is acceptable in
// the current mode. rest is what remains of the name after that component.
func (r *Realpath) tolerates(err error, rest []string) bool {
	switch r.Mode {
	case ModeMissing:
		return true
	case ModeDefault:
		// Only the final component may be absent.
		if !errors.Is(err, fs.ErrNotExist) {
			return false
		}
		for _, comp := range rest {
			if comp != "." {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// split breaks a path into its components. A trailing separator becomes a
// final "." so the walk still checks that it follows a directory.
func split(path string) []string {
	comps := strings.FieldsFunc(path, func(c rune) bool {
		return c < 0x80 && os.IsPathSeparator(uint8(c))
	})
	if path != "" && os.IsPathSeparator(path[len(path)-1]) {
		comps = append(comps, ".")
	}
	return comps
}

// relative renders path against the canonical --relative-to and
// --relative-base directories, either of which may be empty.
func relative(path, relTo, relBase string) string {
	if relTo == "" || (relBase != "" && !hasPrefix(relBase, path)) {
		return path
	}
	// Names on different Windows volumes have no relative form.
	rel, err := filepath.Rel(relTo, path)
	if err != nil {
		return path
	}
	return rel
}

// hasPrefix reports whether path is dir or lies below it. Both must be clean.
func hasPrefix(dir, path string) bool {
	sep := string(filepath.Separator)
	return path == dir || strings.HasPrefix(path, strings.TrimSuffix(dir, sep)+sep)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// quoteEmpty makes an empty operand visible in a diagnostic, as GNU does.
func quoteEmpty(path string) string {
	if path == "" {
		return "''"
	}
	return path
}
