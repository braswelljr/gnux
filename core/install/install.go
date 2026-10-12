// Package install implements the engine behind installx, a GNU-compatible
// implementation of the GNU/UNIX install command: copy files into place and
// set their permission mode, owner and group, or create directories.
package install

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "installx"

// defaultMode is what files and directories get without -m: rwxr-xr-x.
const defaultMode = 0o755

// Options mirrors the GNU install flag surface.
type Options struct {
	// Directory (-d) treats every operand as a directory to create.
	Directory bool
	// CreateLeading (-D) creates all leading components of DEST, or of the
	// -t directory, before copying.
	CreateLeading bool
	// Mode (-m) is the permission mode, octal or symbolic; empty means 755.
	Mode string
	// Owner (-o) and Group (-g) set ownership, by name or numeric ID.
	Owner string
	Group string
	// PreserveTimestamps (-p) gives each copy its source's modification time.
	PreserveTimestamps bool
	// Strip (-s) asks for symbol tables to be stripped; not supported.
	Strip bool
	// TargetDirectory (-t) copies every SOURCE into this directory.
	TargetDirectory string
	// NoTargetDirectory (-T) treats DEST as a normal file.
	NoTargetDirectory bool
	// Verbose (-v) prints the name of each file or directory as it is made.
	Verbose bool
	// Compare (-C) leaves a destination untouched when it already has the
	// same content, mode, owner and group.
	Compare bool
}

// Install is the engine. Construct it with New.
type Install struct {
	Options

	diag     *core.Diag
	mode     os.FileMode
	uid, gid int
}

// New builds an install engine writing to s.
func New(opts Options, s core.Streams) *Install {
	return &Install{Options: opts, diag: core.NewDiag(Name, s), uid: -1, gid: -1}
}

// Run installs the operands. A failing operand is reported and the rest are
// still processed; the returned error is core.ErrFailed if any failed.
func (in *Install) Run(args []string) error {
	if err := in.prepare(); err != nil {
		return err
	}
	if in.diag.Failed() {
		return core.ErrFailed
	}
	if len(args) == 0 {
		return core.Usagef("missing file operand")
	}
	if in.Strip {
		in.diag.Warnf("warning: stripping is not supported; files are installed unstripped")
	}

	if in.Directory {
		for _, dir := range args {
			in.makeDir(dir)
		}
		return in.diag.Result()
	}

	if in.TargetDirectory != "" {
		if !in.targetDir(in.TargetDirectory) {
			return in.diag.Result()
		}
		for _, src := range args {
			in.installFile(src, filepath.Join(in.TargetDirectory, filepath.Base(src)))
		}
		return in.diag.Result()
	}

	if len(args) == 1 {
		return core.Usagef("missing destination file operand after '%s'", args[0])
	}
	dest := args[len(args)-1]
	srcs := args[:len(args)-1]

	if in.NoTargetDirectory {
		if len(args) > 2 {
			return core.Usagef("extra operand '%s'", args[2])
		}
		in.toFile(srcs[0], dest)
		return in.diag.Result()
	}

	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		for _, src := range srcs {
			in.installFile(src, filepath.Join(dest, filepath.Base(src)))
		}
		return in.diag.Result()
	}
	if len(srcs) > 1 {
		in.diag.Errorf("target '%s' is not a directory", dest)
		return core.ErrFailed
	}
	in.toFile(srcs[0], dest)
	return in.diag.Result()
}

// prepare checks the option combinations and resolves mode and ownership.
func (in *Install) prepare() error {
	switch {
	case in.TargetDirectory != "" && in.NoTargetDirectory:
		return core.Usagef("cannot combine --target-directory (-t) and --no-target-directory (-T)")
	case in.Directory && in.TargetDirectory != "":
		return core.Usagef("target directory not allowed when installing a directory")
	case in.Compare && in.PreserveTimestamps:
		return core.Usagef("options --compare (-C) and --preserve-timestamps are mutually exclusive")
	case in.Compare && in.Strip:
		return core.Usagef("options --compare (-C) and --strip are mutually exclusive")
	}

	bits := uint32(defaultMode)
	if in.Mode != "" {
		var err error
		if bits, err = parseMode(in.Mode); err != nil {
			return core.Usagef("invalid mode '%s'", in.Mode)
		}
	}
	in.mode = fileMode(bits)

	if in.Owner != "" {
		id, ok := lookupUser(in.Owner)
		if !ok {
			in.diag.Errorf("invalid user '%s'", in.Owner)
		}
		in.uid = id
	}
	if in.Group != "" {
		id, ok := lookupGroup(in.Group)
		if !ok {
			in.diag.Errorf("invalid group '%s'", in.Group)
		}
		in.gid = id
	}
	return nil
}

// makeDir creates dir and any missing parents, reporting each under -v, then
// gives dir the requested mode and ownership.
func (in *Install) makeDir(dir string) {
	if !in.mkdirAll(dir) {
		return
	}
	in.setAttrs(dir)
}

// mkdirAll is mkdir -p with -v reporting; parents get the default mode.
func (in *Install) mkdirAll(dir string) bool {
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			in.diag.Errorf("cannot create directory '%s': %s", dir, "File exists")
			return false
		}
		return true
	}
	if parent := filepath.Dir(dir); parent != dir {
		if !in.mkdirAll(parent) {
			return false
		}
	}
	if err := os.Mkdir(dir, defaultMode); err != nil && !errors.Is(err, os.ErrExist) {
		in.diag.Errorf("cannot create directory '%s': %s", dir, core.ErrText(err))
		return false
	}
	if in.Verbose {
		in.diag.Printf("%s: creating directory '%s'", Name, dir)
	}
	return true
}

// targetDir checks (or, with -D, creates) the -t directory.
func (in *Install) targetDir(dir string) bool {
	if in.CreateLeading && !in.mkdirAll(dir) {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil {
		in.diag.Errorf("failed to access '%s': %s", dir, core.ErrText(err))
		return false
	}
	if !info.IsDir() {
		in.diag.Errorf("target '%s' is not a directory", dir)
		return false
	}
	return true
}

// toFile installs src as the file dest, creating dest's parents under -D.
func (in *Install) toFile(src, dest string) {
	if in.CreateLeading {
		if parent := filepath.Dir(dest); !in.mkdirAll(parent) {
			return
		}
	}
	in.installFile(src, dest)
}

// installFile copies one regular file into place and sets its attributes.
func (in *Install) installFile(src, dest string) {
	sinfo, err := os.Stat(src)
	if err != nil {
		in.diag.Errorf("cannot stat '%s': %s", src, core.ErrText(err))
		return
	}
	if sinfo.IsDir() {
		in.diag.Errorf("omitting directory '%s'", src)
		return
	}

	dinfo, derr := os.Lstat(dest)
	if derr == nil {
		if dinfo.IsDir() {
			in.diag.Errorf("cannot overwrite directory '%s' with non-directory", dest)
			return
		}
		if same, err := os.Stat(dest); err == nil && os.SameFile(sinfo, same) {
			in.diag.Errorf("'%s' and '%s' are the same file", src, dest)
			return
		}
		if in.Compare && in.unchanged(src, dest, sinfo, dinfo) {
			return
		}
		// Replace rather than rewrite, so a running program keeps its copy.
		if err := os.Remove(dest); err != nil {
			in.diag.Errorf("cannot remove '%s': %s", dest, core.ErrText(err))
			return
		}
	}

	if !in.copyFile(src, dest) {
		return
	}
	if in.Verbose {
		in.diag.Printf("'%s' -> '%s'", src, dest)
	}
	if !in.setAttrs(dest) {
		return
	}
	if in.PreserveTimestamps {
		if err := os.Chtimes(dest, sinfo.ModTime(), sinfo.ModTime()); err != nil {
			in.diag.Errorf("cannot set timestamps for '%s': %s", dest, core.ErrText(err))
		}
	}
}

// copyFile writes src's content to a new file at dest.
func (in *Install) copyFile(src, dest string) bool {
	r, err := os.Open(src)
	if err != nil {
		in.diag.Errorf("cannot open '%s' for reading: %s", src, core.ErrText(err))
		return false
	}
	defer r.Close()
	w, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		in.diag.Errorf("cannot create regular file '%s': %s", dest, core.ErrText(err))
		return false
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		in.diag.Errorf("error copying '%s' to '%s': %s", src, dest, core.ErrText(err))
		return false
	}
	if err := w.Close(); err != nil {
		in.diag.Errorf("error writing '%s': %s", dest, core.ErrText(err))
		return false
	}
	return true
}

// setAttrs applies the mode, then the owner and group.
func (in *Install) setAttrs(path string) bool {
	if err := os.Chmod(path, in.mode); err != nil {
		in.diag.Errorf("cannot change permissions of '%s': %s", path, core.ErrText(err))
		return false
	}
	if in.uid >= 0 || in.gid >= 0 {
		if err := chown(path, in.uid, in.gid); err != nil {
			in.diag.Errorf("cannot change ownership of '%s': %s", path, core.ErrText(err))
			return false
		}
	}
	return true
}

// unchanged reports whether -C may leave dest alone: a regular file with the
// same content, the requested mode and the requested ownership.
func (in *Install) unchanged(src, dest string, sinfo, dinfo os.FileInfo) bool {
	if !dinfo.Mode().IsRegular() || dinfo.Size() != sinfo.Size() {
		return false
	}
	if dinfo.Mode()&(os.ModePerm|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != in.mode {
		return false
	}
	if !sameOwner(dinfo, in.uid, in.gid) {
		return false
	}
	same, err := sameContent(src, dest)
	return err == nil && same
}

// sameContent compares two files byte for byte.
func sameContent(a, b string) (bool, error) {
	fa, err := os.Open(a)
	if err != nil {
		return false, fmt.Errorf("open: %w", err)
	}
	defer fa.Close()
	fb, err := os.Open(b)
	if err != nil {
		return false, fmt.Errorf("open: %w", err)
	}
	defer fb.Close()

	ba, bb := make([]byte, 32<<10), make([]byte, 32<<10)
	for {
		na, ea := io.ReadFull(fa, ba)
		nb, eb := io.ReadFull(fb, bb)
		if na != nb || !bytes.Equal(ba[:na], bb[:nb]) {
			return false, nil
		}
		if ea != nil || eb != nil {
			done := errors.Is(ea, io.EOF) || errors.Is(ea, io.ErrUnexpectedEOF)
			return done && (errors.Is(eb, io.EOF) || errors.Is(eb, io.ErrUnexpectedEOF)), nil
		}
	}
}
