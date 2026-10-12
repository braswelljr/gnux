// Package readlink implements the engine behind readlinkx, a GNU-compatible
// implementation of the GNU readlink command: print the value of a symbolic
// link, or the canonical name of a file.
package readlink

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
const Name = "readlinkx"

// maxLinks bounds how many symbolic links one name may expand through before
// it is treated as a loop.
const maxLinks = 40

// These failures are detected by the engine, so they carry GNU's wording
// (core.ErrText capitalizes it) rather than platform-specific errno text.
var (
	errNotDir = errors.New("not a directory")
	errLoop   = errors.New("too many levels of symbolic links")
)

// Mode says what readlink prints for each operand.
type Mode int

const (
	// ModeLink prints the target of a symbolic link, unresolved. An operand
	// that is not a symbolic link fails.
	ModeLink Mode = iota
	// ModeFollow (-f) canonicalizes; every component but the last must exist.
	ModeFollow
	// ModeExisting (-e) canonicalizes; every component must exist.
	ModeExisting
	// ModeMissing (-m) canonicalizes; nothing needs to exist.
	ModeMissing
)

// Options mirrors the GNU readlink flag surface.
type Options struct {
	// Mode selects between printing the link value and -f, -e, -m.
	Mode Mode
	// NoNewline (-n) omits the trailing delimiter. It is ignored, with a
	// warning, when there is more than one operand.
	NoNewline bool
	// Verbose (-v) reports errors. GNU readlink is quiet by default (-q, -s).
	Verbose bool
	// Zero (-z) ends each output item with NUL instead of a newline.
	Zero bool
}

// Readlink is the engine. Construct it with New.
type Readlink struct {
	Options

	// Getwd anchors relative names; it defaults to os.Getwd.
	Getwd func() (string, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a readlink engine writing to s.
func New(opts Options, s core.Streams) *Readlink {
	return &Readlink{
		Options: opts,
		Getwd:   os.Getwd,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the value or canonical name of each operand. A failing operand
// does not stop the rest; the returned error is core.ErrFailed if any failed.
func (r *Readlink) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	noNewline := r.NoNewline
	if noNewline && len(args) > 1 {
		r.diag.Warnf("ignoring --no-newline with multiple arguments")
		noNewline = false
	}
	end := "\n"
	switch {
	case noNewline:
		end = ""
	case r.Zero:
		end = "\x00"
	}

	failed := false
	for _, name := range args {
		var (
			value string
			err   error
		)
		if r.Mode == ModeLink {
			value, err = os.Readlink(name)
		} else {
			value, err = r.canonicalize(name)
		}
		if err != nil {
			failed = true
			if r.Verbose {
				r.diag.Errorf("%s: %s", name, core.ErrText(err))
			}
			continue
		}
		fmt.Fprint(r.streams.Out, value+end)
	}
	if failed {
		return core.ErrFailed
	}
	return nil
}

// canonicalize resolves every symbolic link and "."/".." component of name,
// requiring as much of it to exist as the mode demands.
func (r *Readlink) canonicalize(name string) (string, error) {
	if name == "" {
		return "", fs.ErrNotExist
	}
	abs := name
	switch {
	case filepath.IsAbs(abs):
	case filepath.VolumeName(abs) != "" || os.IsPathSeparator(abs[0]):
		// Windows names that are neither absolute nor relative to the working
		// directory ("C:dir", "\dir"); filepath.Abs knows how to anchor them.
		a, err := filepath.Abs(abs)
		if err != nil {
			return "", err
		}
		abs = a
	default:
		// Not cleaned: "a/.." must survive until the links in "a" are resolved.
		wd, err := r.Getwd()
		if err != nil {
			return "", err
		}
		abs = wd + string(filepath.Separator) + abs
	}

	root := filepath.VolumeName(abs) + string(filepath.Separator)
	pending := split(abs[len(filepath.VolumeName(abs)):])
	result := root
	links := 0

	for len(pending) > 0 {
		comp := pending[0]
		pending = pending[1:]
		switch comp {
		case ".":
			continue
		case "..":
			result = filepath.Dir(result)
			continue
		}
		next := filepath.Join(result, comp)
		info, err := os.Lstat(next)
		if err != nil {
			if r.Mode == ModeMissing || (r.Mode == ModeFollow && len(pending) == 0 && errors.Is(err, fs.ErrNotExist)) {
				result = next
				continue
			}
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			links++
			if links > maxLinks {
				return "", errLoop
			}
			target, err := os.Readlink(next)
			if err != nil {
				return "", err
			}
			if filepath.IsAbs(target) {
				result = filepath.VolumeName(target) + string(filepath.Separator)
				target = target[len(filepath.VolumeName(target)):]
			}
			pending = append(split(target), pending...)
			continue
		}
		if !info.IsDir() && len(pending) > 0 && r.Mode != ModeMissing {
			return "", errNotDir
		}
		result = next
	}
	return result, nil
}

// split breaks a name into its components, dropping empty ones.
func split(name string) []string {
	return strings.FieldsFunc(name, func(c rune) bool { return c < 128 && os.IsPathSeparator(uint8(c)) })
}
