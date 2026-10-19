// Package cp implements the copy engine behind cpx, a GNU-compatible implementation of
// the GNU/UNIX cp command. It mirrors cp's flags, overwrite rules, symlink
// handling and diagnostics. mvx reuses it for moves that cross filesystems.
package cp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic and prompt.
const Name = "cpx"

// Deref controls when symbolic links in the sources are followed.
type Deref int

const (
	// DerefDefault follows symbolic links unless copying recursively, as cp
	// does when none of -L, -H, -P or -d is given.
	DerefDefault Deref = iota
	// DerefNever copies symbolic links as symbolic links (-P, -d).
	DerefNever
	// DerefCommandLine follows only the links named on the command line (-H).
	DerefCommandLine
	// DerefAlways follows every symbolic link (-L).
	DerefAlways
)

// Preserve selects the attributes carried over from source to destination.
type Preserve struct {
	Mode       bool // permission bits, including setuid/setgid/sticky
	Ownership  bool // owner and group (best effort; a no-op on Windows)
	Timestamps bool // modification time
	Links      bool // hard links between the copied sources
}

// PreserveAll returns the attribute set selected by --preserve=all (and -a).
func PreserveAll() Preserve {
	return Preserve{Mode: true, Ownership: true, Timestamps: true, Links: true}
}

// Options mirrors the GNU cp flag surface.
type Options struct {
	Recursive            bool     // -r/-R: copy directories recursively
	Force                bool     // -f: remove a destination that cannot be opened, then retry
	Interactive          bool     // -i: prompt before overwriting
	NoClobber            bool     // -n: never overwrite an existing file
	Update               bool     // -u: copy only when the source is newer or the destination is missing
	Verbose              bool     // -v: explain what is being done
	Preserve             Preserve // -p/--preserve: attributes to carry over
	Dereference          Deref    // -L/-H/-P/-d: symbolic link policy
	Link                 bool     // -l: hard link files instead of copying
	SymbolicLink         bool     // -s: make symbolic links instead of copying
	TargetDirectory      string   // -t: copy every source into this directory
	NoTargetDirectory    bool     // -T: treat the destination as a normal file
	OneFileSystem        bool     // -x: stay on the source's filesystem
	Parents              bool     // --parents: recreate the source path under the directory
	StripTrailingSlashes bool     // --strip-trailing-slashes: trim '/' from each source
	RemoveDestination    bool     // --remove-destination: remove an existing destination before opening it
	// Backup (--backup, -b, -S) renames each existing destination file to
	// a backup name before it is replaced. Directories are merged into, not
	// backed up, as in GNU cp.
	Backup core.Backup
}

// Cp is the copy engine. Construct it with New.
type Cp struct {
	Options

	// PromptFn, when non-nil, is consulted instead of reading from standard
	// input. It is the injection point tests use to answer overwrite prompts.
	PromptFn func(question string) bool

	diag   *core.Diag
	prompt *core.Prompter

	// links maps a multiply-linked source (device, inode) to the first
	// destination written for it, so later names become hard links to it.
	links map[linkKey]string
	// created records the destinations written for earlier operands, so a
	// later operand with the same base name cannot silently replace them.
	created map[string]struct{}
}

type linkKey struct {
	dev, ino uint64
}

// New builds a copy engine writing to s.
func New(opts Options, s core.Streams) *Cp {
	return NewNamed(Name, opts, s)
}

// NewNamed builds a copy engine whose diagnostics and prompts are prefixed
// with prog instead of Name. mvx uses it so a cross-device move reports
// failures under its own name.
func NewNamed(prog string, opts Options, s core.Streams) *Cp {
	return &Cp{
		Options: opts,
		diag:    core.NewDiag(prog, s),
		prompt:  core.NewPrompter(prog, s),
		links:   make(map[linkKey]string),
		created: make(map[string]struct{}),
	}
}

// Run copies according to args, which take one of cp's forms: SOURCE DEST,
// SOURCE... DIRECTORY, or (with -t) SOURCE.... Diagnostics are written as they
// occur; the returned error is core.ErrFailed if any copy failed, a
// *core.UsageError for a malformed command line, otherwise nil.
func (c *Cp) Run(args []string) error {
	switch {
	case c.Link && c.SymbolicLink:
		return core.Usagef("cannot make both hard and symbolic links")
	case c.TargetDirectory != "" && c.NoTargetDirectory:
		return core.Usagef("cannot combine --target-directory (-t) and --no-target-directory (-T)")
	case c.backingUp() && c.NoClobber:
		return core.Usagef("--backup is mutually exclusive with -n or --update=none-fail")
	case len(args) == 0:
		return core.Usagef("missing file operand")
	}

	var (
		sources []string
		dest    string
		into    bool // dest is a directory to copy the sources into
	)
	switch {
	case c.TargetDirectory != "":
		sources, dest, into = args, c.TargetDirectory, true
		info, err := fsys.Stat(dest)
		if err == nil && !info.IsDir() {
			c.diag.Errorf("target directory '%s': Not a directory", dest)
			return c.diag.Result()
		}
		if err != nil {
			c.diag.Errorf("target directory '%s': %s", dest, core.ErrText(err))
			return c.diag.Result()
		}
	case len(args) == 1:
		return core.Usagef("missing destination file operand after '%s'", args[0])
	case c.NoTargetDirectory:
		if len(args) > 2 {
			return core.Usagef("extra operand '%s'", args[2])
		}
		sources, dest = args[:1], args[1]
	default:
		sources, dest = args[:len(args)-1], args[len(args)-1]
		if info, err := fsys.Stat(dest); err == nil && info.IsDir() {
			into = true
		} else if len(sources) > 1 {
			return core.Usagef("target '%s' is not a directory", dest)
		}
	}
	if c.Parents && !into {
		return core.Usagef("with --parents, the destination must be a directory")
	}

	for _, src := range sources {
		if c.StripTrailingSlashes {
			src = StripTrailingSlashes(src)
		}
		dst := dest
		if !into && c.Force && c.backingUp() && src == dest {
			// GNU's special case: "cp -fb f f" copies f to its backup name.
			if info, err := fsys.Stat(dest); err == nil && info.Mode().IsRegular() {
				backup := c.Backup
				c.Backup = core.Backup{}
				c.copyOperand(src, backup.BackupName(dest))
				c.Backup = backup
				continue
			}
		}
		if into {
			if c.Parents {
				var ok bool
				if dst, ok = c.makeParents(src, dest); !ok {
					continue
				}
			} else {
				dst = JoinInto(dest, src)
			}
		}
		c.copyOperand(src, dst)
	}
	return c.diag.Result()
}

// Copy copies the single entry src to exactly dst (dst is never treated as a
// directory to copy into), honoring the engine's options. It reports whether
// the copy fully succeeded; on failure the diagnostics have been written.
func (c *Cp) Copy(src, dst string) bool {
	return c.copyEntry(src, dst, true, nil, nil)
}

// copyOperand copies one command-line operand, refusing to replace a file an
// earlier operand of the same invocation produced (cp a/f b/f dir).
func (c *Cp) copyOperand(src, dst string) {
	key := filepath.Clean(dst)
	// A numbered backup keeps the earlier copy, so nothing is lost.
	if _, dup := c.created[key]; dup && c.Backup.Control != core.NumberedBackup {
		// Directories are exempt: same-named source directories merge.
		if info, err := fsys.Lstat(dst); err == nil && !info.IsDir() {
			c.diag.Errorf("will not overwrite just-created '%s' with '%s'", dst, src)
			return
		}
	}
	if c.Copy(src, dst) {
		c.created[key] = struct{}{}
	}
}

// copyEntry copies src to dst. top marks a command-line operand (which -H
// dereferences), parent is the source directory being walked (nil at the top)
// and ancestors the chain of source directories above src, used to detect
// symlink cycles under -L.
func (c *Cp) copyEntry(src, dst string, top bool, parent os.FileInfo, ancestors []os.FileInfo) bool {
	follow := c.follows(top)
	si, err := stat(src, follow)
	if err != nil {
		c.diag.Errorf("cannot stat '%s': %s", src, core.ErrText(err))
		return false
	}
	if si.IsDir() && !c.Recursive {
		c.diag.Errorf("-r not specified; omitting directory '%s'", src)
		return false
	}

	di, exists, ok := c.statDest(dst, si)
	if !ok {
		return false
	}

	if exists {
		if os.SameFile(si, di) {
			if c.Link {
				// Already the link that was asked for.
				return true
			}
			// With a backup the destination is moved aside first, so only
			// the very same name is refused.
			if !c.backingUp() || sameEntry(src, dst) {
				c.diag.Errorf("'%s' and '%s' are the same file", src, dst)
				return false
			}
		}
		switch {
		case si.IsDir():
			if !di.IsDir() && !c.backingUp() {
				c.diag.Errorf("cannot overwrite non-directory '%s' with directory '%s'", dst, src)
				return false
			}
		case c.Update && !di.ModTime().Before(si.ModTime()):
			return true
		case di.IsDir():
			c.diag.Errorf("cannot overwrite directory '%s' with non-directory", dst)
			return false
		case c.NoClobber:
			return true
		case c.Interactive && !c.ask(fmt.Sprintf("overwrite '%s'?", dst)):
			return true
		}
	}

	backup := ""
	if exists && c.backingUp() && !di.IsDir() && !dotOrDotDot(src) {
		if c.Backup.BackupWouldDestroy(src, si, dst) {
			c.diag.Errorf("backing up '%s' might destroy source;  '%s' not copied", dst, src)
			return false
		}
		name, err := c.Backup.MakeBackup(dst)
		if err != nil {
			c.diag.Errorf("cannot backup '%s': %s", dst, core.ErrText(err))
			return false
		}
		backup, exists = name, false
	}
	done := c.write(src, dst, si, exists, top, follow, parent, ancestors, backup)
	if !done && backup != "" {
		c.unBackup(backup, dst)
	}
	return done
}

// write creates dst from src once any backup has been made.
func (c *Cp) write(src, dst string, si os.FileInfo, exists, top, follow bool, parent os.FileInfo, ancestors []os.FileInfo, backup string) bool {
	if si.IsDir() {
		return c.copyDir(src, dst, si, exists, top, parent, ancestors)
	}

	if exists && c.RemoveDestination {
		if err := fsys.Remove(dst); err != nil && !os.IsNotExist(err) {
			c.diag.Errorf("cannot remove '%s': %s", dst, core.ErrText(err))
			return false
		}
		exists = false
		if c.Verbose {
			c.diag.Printf("removed '%s'", dst)
		}
	}

	if c.Verbose {
		if backup != "" {
			c.diag.Printf("'%s' -> '%s' (backup: '%s')", src, dst, backup)
		} else {
			c.diag.Printf("'%s' -> '%s'", src, dst)
		}
	}

	switch {
	case c.SymbolicLink:
		return c.makeSymlink(src, dst, exists)
	case c.Link:
		return c.makeHardLink(src, dst, exists, follow)
	}

	// --preserve=links: a source with several names is copied once and the
	// other names become hard links to that first copy.
	var key linkKey
	tracked := false
	if c.Preserve.Links {
		if ino, ok := inode(si); ok {
			dev, _ := core.DeviceID(si)
			key, tracked = linkKey{dev: dev, ino: ino}, true
			if first, seen := c.links[key]; seen {
				return c.linkTo(first, dst, exists)
			}
		}
	}

	var done bool
	switch {
	case si.Mode()&os.ModeSymlink != 0:
		done = c.copySymlink(src, dst, si, exists)
	case si.Mode().IsRegular() || !c.Recursive:
		// Without -R, cp reads a special file's contents like a regular file.
		done = c.copyFile(src, dst, si, exists)
	default:
		c.diag.Errorf("cannot create special file '%s': Operation not supported", dst)
	}
	if done && tracked {
		c.links[key] = dst
	}
	return done
}

// backingUp reports whether existing destinations are backed up.
func (c *Cp) backingUp() bool {
	return c.Backup.Control != core.NoBackup
}

// unBackup puts a backup back in place of a destination that could not be
// created, as GNU does.
func (c *Cp) unBackup(backup, dst string) {
	if _, err := fsys.Lstat(dst); err == nil {
		return
	}
	if err := fsys.Rename(backup, dst); err != nil {
		c.diag.Errorf("cannot un-backup '%s': %s", dst, core.ErrText(err))
		return
	}
	if c.Verbose {
		c.diag.Printf("'%s' -> '%s' (unbackup)", backup, dst)
	}
}

// dotOrDotDot reports whether the last component of path is "." or "..",
// whose destination is never backed up.
func dotOrDotDot(path string) bool {
	base := filepath.Base(path)
	return base == "." || base == ".."
}

// sameEntry reports whether a and b are the same name in the same directory,
// as opposed to two hard links to one file.
func sameEntry(a, b string) bool {
	if filepath.Base(a) != filepath.Base(b) {
		return false
	}
	da, err := fsys.Stat(filepath.Dir(a))
	if err != nil {
		return false
	}
	db, err := fsys.Stat(filepath.Dir(b))
	return err == nil && os.SameFile(da, db)
}

// follows reports whether a symbolic link at the current position is
// dereferenced.
func (c *Cp) follows(top bool) bool {
	switch c.Dereference {
	case DerefAlways:
		return true
	case DerefNever:
		return false
	case DerefCommandLine:
		return top
	default:
		// GNU: a recursive copy keeps links as links unless it is making
		// hard links; everything else follows them.
		return !c.Recursive || c.Link
	}
}

// statDest examines the destination. A regular source is written through a
// destination symlink, so that case follows links; everything else replaces
// the link itself, as does --remove-destination. ok is false when a
// diagnostic has been issued.
func (c *Cp) statDest(dst string, si os.FileInfo) (di os.FileInfo, exists, ok bool) {
	through := si.Mode().IsRegular() && !c.Link && !c.SymbolicLink && !c.RemoveDestination && !c.backingUp()
	di, err := stat(dst, through)
	switch {
	case err == nil:
		return di, true, true
	case !os.IsNotExist(err):
		c.diag.Errorf("cannot stat '%s': %s", dst, core.ErrText(err))
		return nil, false, false
	}
	if through {
		if _, lerr := fsys.Lstat(dst); lerr == nil {
			c.diag.Errorf("not writing through dangling symlink '%s'", dst)
			return nil, false, false
		}
	}
	return nil, false, true
}

// copyDir copies the directory src to dst, creating dst if needed and merging
// into it otherwise.
func (c *Cp) copyDir(src, dst string, si os.FileInfo, exists, top bool, parent os.FileInfo, ancestors []os.FileInfo) bool {
	for _, a := range ancestors {
		if os.SameFile(a, si) {
			c.diag.Errorf("cannot copy cyclic symbolic link '%s'", src)
			return false
		}
	}
	// Below the top level only a followed symlink can lead back into the
	// destination, so the (syscall-heavy) check is skipped otherwise.
	if (top || c.Dereference == DerefAlways) && IsWithin(src, dst) {
		c.diag.Errorf("cannot copy a directory, '%s', into itself, '%s'", src, dst)
		return false
	}

	// restore is the mode to put back once the contents are in place, when
	// the new directory had to be opened up to be populated.
	var restore os.FileMode
	needRestore := false
	if !exists {
		if err := fsys.Mkdir(dst, si.Mode().Perm()); err != nil {
			c.diag.Errorf("cannot create directory '%s': %s", dst, core.ErrText(err))
			return false
		}
		if c.Verbose {
			c.diag.Printf("'%s' -> '%s'", src, dst)
		}
		// A read-only source directory must not stop its own contents from
		// being copied: grant the owner access now and restore the mode
		// (as masked by the umask at creation) afterwards.
		if mode := createdDirMode(dst, si.Mode().Perm()); mode&0o700 != 0o700 {
			restore, needRestore = mode, true
			if err := fsys.Chmod(dst, restore|0o700); err != nil {
				needRestore = false
			}
		}
	}

	// -x: a mount point is itself created, but its contents are skipped.
	ok := true
	if !c.crossesDevice(si, parent) {
		entries, err := fsys.ReadDir(src)
		if err != nil {
			c.diag.Errorf("cannot access '%s': %s", src, core.ErrText(err))
			ok = false
		}
		// Each level writes only its own slot, so sharing the backing array
		// between siblings is safe in this sequential walk.
		ancestors = append(ancestors, si)
		for _, e := range entries {
			if !c.copyEntry(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()), false, si, ancestors) {
				ok = false
			}
		}
	}

	if needRestore && !c.Preserve.Mode {
		if err := fsys.Chmod(dst, restore); err != nil {
			c.diag.Errorf("setting permissions for '%s': %s", dst, core.ErrText(err))
			ok = false
		}
	}
	// Attributes go on last so copying the contents does not disturb the
	// directory's timestamps.
	if !c.preserveAttrs(dst, si) {
		ok = false
	}
	return ok
}

// crossesDevice reports whether -x forbids descending into the directory
// described by si, because it lives on a different device than its parent.
func (c *Cp) crossesDevice(si, parent os.FileInfo) bool {
	if !c.OneFileSystem || parent == nil {
		return false
	}
	dev, ok := core.DeviceID(si)
	pdev, pok := core.DeviceID(parent)
	return ok && pok && dev != pdev
}

// copyFile copies the contents of src to dst.
func (c *Cp) copyFile(src, dst string, si os.FileInfo, exists bool) bool {
	in, err := fsys.Open(src)
	if err != nil {
		c.diag.Errorf("cannot open '%s' for reading: %s", src, core.ErrText(err))
		return false
	}
	defer in.Close()

	var out *os.File
	if exists {
		out, err = fsys.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			if !c.Force {
				c.diag.Errorf("cannot create regular file '%s': %s", dst, core.ErrText(err))
				return false
			}
			// -f: the destination cannot be opened, so replace it outright.
			if rerr := fsys.Remove(dst); rerr != nil {
				c.diag.Errorf("cannot remove '%s': %s", dst, core.ErrText(rerr))
				return false
			}
			exists = false
		}
	}
	if !exists {
		// A new file takes the source's permission bits; the kernel applies
		// the umask, which is exactly cp's behavior without -p.
		out, err = fsys.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, si.Mode().Perm())
		if err != nil {
			c.diag.Errorf("cannot create regular file '%s': %s", dst, core.ErrText(err))
			return false
		}
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		c.diag.Errorf("error writing '%s': %s", dst, core.ErrText(err))
		return false
	}
	if err := out.Close(); err != nil {
		c.diag.Errorf("failed to close '%s': %s", dst, core.ErrText(err))
		return false
	}
	return c.preserveAttrs(dst, si)
}

// copySymlink recreates the symbolic link src at dst.
func (c *Cp) copySymlink(src, dst string, si os.FileInfo, exists bool) bool {
	target, err := fsys.Readlink(src)
	if err != nil {
		c.diag.Errorf("cannot read symbolic link '%s': %s", src, core.ErrText(err))
		return false
	}
	if exists {
		if err := fsys.Remove(dst); err != nil {
			c.diag.Errorf("cannot remove '%s': %s", dst, core.ErrText(err))
			return false
		}
	}
	if err := fsys.Symlink(target, dst); err != nil {
		c.diag.Errorf("cannot create symbolic link '%s' to '%s': %s", dst, target, core.ErrText(err))
		return false
	}
	return c.preserveAttrs(dst, si)
}

// makeSymlink implements -s: dst becomes a symbolic link to src.
func (c *Cp) makeSymlink(src, dst string, exists bool) bool {
	// A relative link is resolved against the link's own directory, so it
	// only means the same thing when that directory is the current one.
	if !filepath.IsAbs(src) && filepath.Dir(dst) != "." {
		c.diag.Errorf("'%s': can make relative symbolic links only in current directory", dst)
		return false
	}
	if exists && !c.replaceForLink(dst, "symbolic", src) {
		return false
	}
	if err := fsys.Symlink(src, dst); err != nil {
		c.diag.Errorf("cannot create symbolic link '%s' to '%s': %s", dst, src, core.ErrText(err))
		return false
	}
	return true
}

// makeHardLink implements -l: dst becomes a hard link to src.
func (c *Cp) makeHardLink(src, dst string, exists, follow bool) bool {
	from := src
	if follow {
		// link(2) does not follow a symlink everywhere; resolve it so -L
		// links to the referent on every platform.
		if resolved, err := fsys.EvalSymlinks(src); err == nil {
			from = resolved
		}
	}
	if exists && !c.replaceForLink(dst, "hard", src) {
		return false
	}
	if err := fsys.Link(from, dst); err != nil {
		c.diag.Errorf("cannot create hard link '%s' to '%s': %s", dst, src, core.ErrText(err))
		return false
	}
	return true
}

// replaceForLink clears an existing dst so a link can take its place, which
// cp only does under -f.
func (c *Cp) replaceForLink(dst, kind, src string) bool {
	if !c.Force {
		c.diag.Errorf("cannot create %s link '%s' to '%s': File exists", kind, dst, src)
		return false
	}
	if err := fsys.Remove(dst); err != nil {
		c.diag.Errorf("cannot remove '%s': %s", dst, core.ErrText(err))
		return false
	}
	return true
}

// linkTo makes dst a hard link to first, an earlier copy of the same source
// file (--preserve=links).
func (c *Cp) linkTo(first, dst string, exists bool) bool {
	if exists {
		if err := fsys.Remove(dst); err != nil {
			c.diag.Errorf("cannot remove '%s': %s", dst, core.ErrText(err))
			return false
		}
	}
	if err := fsys.Link(first, dst); err != nil {
		c.diag.Errorf("cannot create hard link '%s' to '%s': %s", dst, first, core.ErrText(err))
		return false
	}
	return true
}

// preserveAttrs applies the attributes selected by --preserve from si to dst.
// Ownership goes first because chown clears the setuid/setgid bits.
func (c *Cp) preserveAttrs(dst string, si os.FileInfo) bool {
	ok := true
	if c.Preserve.Ownership {
		if err := chown(dst, si); err != nil {
			c.diag.Errorf("failed to preserve ownership for '%s': %s", dst, core.ErrText(err))
			ok = false
		}
	}
	if si.Mode()&os.ModeSymlink != 0 {
		// The standard library cannot set a link's own times or mode; both
		// calls would act on the referent instead.
		return ok
	}
	if c.Preserve.Timestamps {
		// The zero access time leaves atime as it is: reading it back
		// portably is not possible through os.FileInfo.
		if err := fsys.Chtimes(dst, time.Time{}, si.ModTime()); err != nil {
			c.diag.Errorf("preserving times for '%s': %s", dst, core.ErrText(err))
			ok = false
		}
	}
	if c.Preserve.Mode {
		const bits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
		if err := fsys.Chmod(dst, si.Mode()&bits); err != nil {
			c.diag.Errorf("preserving permissions for '%s': %s", dst, core.ErrText(err))
			ok = false
		}
	}
	return ok
}

// makeParents implements --parents: it returns the destination for src under
// dir (dir/src) after creating the intermediate directories, each with the
// mode of the source directory it mirrors.
func (c *Cp) makeParents(src, dir string) (string, bool) {
	prefix := len(filepath.VolumeName(src))
	for prefix < len(src) && os.IsPathSeparator(src[prefix]) {
		prefix++
	}
	rel := filepath.Clean(src[prefix:])
	dst := filepath.Join(dir, rel)

	parent := filepath.Dir(rel)
	if parent == "." {
		return dst, true
	}
	parts := strings.Split(parent, string(filepath.Separator))
	for i := range parts {
		sub := filepath.Join(parts[:i+1]...)
		srcDir, dstDir := src[:prefix]+sub, filepath.Join(dir, sub)

		if info, err := fsys.Stat(dstDir); err == nil {
			if !info.IsDir() {
				c.diag.Errorf("'%s' exists but is not a directory", dstDir)
				return "", false
			}
			continue
		}
		info, err := fsys.Stat(srcDir)
		if err != nil {
			c.diag.Errorf("failed to get attributes of '%s': %s", srcDir, core.ErrText(err))
			return "", false
		}
		// Owner access is forced so the copy below can populate it.
		if err := fsys.Mkdir(dstDir, info.Mode().Perm()|0o700); err != nil {
			c.diag.Errorf("cannot make directory '%s': %s", dstDir, core.ErrText(err))
			return "", false
		}
		if c.Verbose {
			c.diag.Printf("'%s' -> '%s'", srcDir, dstDir)
		}
	}
	return dst, true
}

// ask poses a yes/no question, returning true on an affirmative answer.
func (c *Cp) ask(question string) bool {
	if c.PromptFn != nil {
		return c.PromptFn(question)
	}
	return c.prompt.Ask(question)
}

// stat returns Stat or Lstat information for path, depending on follow.
func stat(path string, follow bool) (os.FileInfo, error) {
	if follow {
		return fsys.Stat(path)
	}
	return fsys.Lstat(path)
}

// JoinInto returns the path a source operand takes inside the directory dir:
// dir joined with the source's last component. A source ending in "." or ".."
// (cp -r src/. dir) names dir itself, so its contents are merged into dir.
func JoinInto(dir, src string) string {
	base := StripTrailingSlashes(src[len(filepath.VolumeName(src)):])
	for i := len(base) - 1; i >= 0; i-- {
		if os.IsPathSeparator(base[i]) {
			base = base[i+1:]
			break
		}
	}
	if base == "" || base == "." || base == ".." {
		return dir
	}
	return filepath.Join(dir, base)
}

// StripTrailingSlashes removes trailing path separators from path, keeping a
// lone root intact.
func StripTrailingSlashes(path string) string {
	end := len(path)
	for end > 1 && os.IsPathSeparator(path[end-1]) {
		end--
	}
	return path[:end]
}

// IsWithin reports whether path is dir itself or lies beneath it, after
// resolving symbolic links in both. path need not exist yet: its longest
// existing prefix is resolved. It is how cp and mv refuse to put a directory
// inside itself.
func IsWithin(dir, path string) bool {
	rel, err := filepath.Rel(realPath(dir), realPath(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// realPath returns the absolute, symlink-free form of path, resolving as much
// of it as exists.
func realPath(path string) string {
	abs, err := fsys.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := fsys.EvalSymlinks(abs); err == nil {
		return resolved
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return abs
	}
	return filepath.Join(realPath(parent), filepath.Base(abs))
}
