// Package mktemp implements the engine behind mktempx, a GNU-compatible
// implementation of the GNU/UNIX mktemp command: create a uniquely named
// temporary file or directory and print its name.
package mktemp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "mktempx"

// DefaultTemplate is used when no TEMPLATE is given; it implies --tmpdir.
const DefaultTemplate = "tmp.XXXXXXXXXX"

// letters are the characters an X is replaced with, as in GNU mktemp.
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// attempts bounds the retries when a candidate name already exists.
const attempts = 1 << 16

// Options mirrors the GNU mktemp flag surface.
type Options struct {
	// Directory (-d) creates a directory instead of a file.
	Directory bool
	// Quiet (-q) suppresses diagnostics about failing to create the file.
	Quiet bool
	// DryRun (-u) prints a name without creating anything (unsafe).
	DryRun bool
	// TmpDir (-p DIR, --tmpdir[=DIR]) creates the name under DIR, or under
	// $TMPDIR (else the system temporary directory) when DIR is empty.
	TmpDir string
	// UseTmpDir records that -p or --tmpdir was given, even without DIR.
	UseTmpDir bool
	// Suffix (--suffix=SUFF) is appended to the template; SuffixSet records
	// that it was given, so an empty suffix still counts.
	Suffix    string
	SuffixSet bool
	// Legacy (-t) interprets TEMPLATE as a single file name component
	// relative to $TMPDIR, else DIR, else the system temporary directory.
	Legacy bool
}

// Mktemp is the engine. Construct it with New.
type Mktemp struct {
	Options

	// Getenv reads TMPDIR; TempDir is the system default. Both are injection
	// points for tests.
	Getenv  func(string) string
	TempDir func() string
	// Random is the source of randomness for the X characters.
	Random io.Reader

	streams core.Streams
	diag    *core.Diag
}

// New builds a mktemp engine reading and writing s.
func New(opts Options, s core.Streams) *Mktemp {
	return &Mktemp{
		Options: opts,
		Getenv:  os.Getenv,
		TempDir: os.TempDir,
		Random:  rand.Reader,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run creates the temporary file or directory and prints its name.
func (m *Mktemp) Run(args []string) error {
	if len(args) > 1 {
		return core.Usagef("too many templates")
	}
	template := DefaultTemplate
	useDir := m.UseTmpDir || m.Legacy
	if len(args) == 1 {
		template = args[0]
	} else {
		useDir = true
	}

	suffix := ""
	if m.SuffixSet {
		suffix = m.Suffix
		if strings.ContainsRune(suffix, '/') || strings.ContainsRune(suffix, filepath.Separator) {
			return core.Usagef("invalid suffix '%s', contains directory separator", suffix)
		}
		if !strings.HasSuffix(template, "X") {
			return core.Usagef("with --suffix, template '%s' must end in X", template)
		}
	} else if i := strings.LastIndexByte(template, 'X'); i >= 0 {
		// Without --suffix, whatever follows the last X is the suffix.
		template, suffix = template[:i+1], template[i+1:]
	}

	x := 0
	for x < len(template) && template[len(template)-1-x] == 'X' {
		x++
	}
	if x < 3 {
		return core.Usagef("too few X's in template '%s'", template+suffix)
	}

	if useDir {
		dir, err := m.dir(template + suffix)
		if err != nil {
			return err
		}
		template = filepath.Join(dir, template)
	}

	name, err := m.create(template[:len(template)-x], x, suffix)
	if err != nil {
		if !m.Quiet {
			kind := "file"
			if m.Directory {
				kind = "directory"
			}
			m.diag.Errorf("failed to create %s via template '%s': %s", kind, template+suffix, core.ErrText(err))
		}
		return core.ErrFailed
	}
	fmt.Fprintln(m.streams.Out, name)
	return nil
}

// dir picks the directory a template is created in, checking the template
// against the rules of -t and --tmpdir.
func (m *Mktemp) dir(template string) (string, error) {
	env := m.Getenv("TMPDIR")
	if m.Legacy {
		if strings.ContainsRune(template, '/') || strings.ContainsRune(template, filepath.Separator) {
			return "", core.Usagef("invalid template, '%s', contains directory separator", template)
		}
		switch {
		case env != "":
			return env, nil
		case m.TmpDir != "":
			return m.TmpDir, nil
		}
		return m.TempDir(), nil
	}
	if filepath.IsAbs(template) || strings.HasPrefix(template, "/") {
		return "", core.Usagef("invalid template, '%s'; with --tmpdir, it may not be absolute", template)
	}
	switch {
	case m.TmpDir != "":
		return m.TmpDir, nil
	case env != "":
		return env, nil
	}
	return m.TempDir(), nil
}

// create tries random names for the x trailing X's until one can be made.
func (m *Mktemp) create(prefix string, x int, suffix string) (string, error) {
	buf := make([]byte, x)
	var last error
	for range attempts {
		if _, err := io.ReadFull(m.Random, buf); err != nil {
			return "", fmt.Errorf("read random: %w", err)
		}
		for i, b := range buf {
			buf[i] = letters[int(b)%len(letters)]
		}
		name := prefix + string(buf) + suffix

		var err error
		switch {
		case m.DryRun:
			if _, err = os.Lstat(name); err == nil {
				err = os.ErrExist
			} else if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		case m.Directory:
			err = os.Mkdir(name, 0o700)
		default:
			var f *os.File
			if f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600); err == nil {
				f.Close()
			}
		}
		if err == nil {
			return name, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
		last = err
	}
	return "", last
}
