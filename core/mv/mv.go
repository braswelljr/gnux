// Package mv implements the engine behind mvx, a GNU-compatible implementation of the
// GNU/UNIX mv command. A move is a rename whenever the filesystem allows it;
// across filesystems it falls back to an archive copy (via core/cp) followed by
// removal of the source.
package mv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cp"
)

// Name is the command name, used as the prefix of every diagnostic and prompt.
const Name = "mvx"

// Options mirrors the GNU mv flag surface.
type Options struct {
	Force                bool   // -f: never prompt before overwriting
	Interactive          bool   // -i: prompt before overwriting
	NoClobber            bool   // -n: never overwrite an existing file
	Update               bool   // -u: move only when the source is newer or the destination is missing
	Verbose              bool   // -v: explain what is being done
	TargetDirectory      string // -t: move every source into this directory
	NoTargetDirectory    bool   // -T: treat the destination as a normal file
	StripTrailingSlashes bool   // --strip-trailing-slashes: trim '/' from each source
	NoCopy               bool   // --no-copy: fail rather than copy when the rename crosses file systems
	// Backup (--backup, -b, -S) renames an existing destination, directory
	// or not, to a backup name before the move replaces it.
	Backup core.Backup
}

// Mv is the move engine. Construct it with New.
type Mv struct {
	Options

	// PromptFn, when non-nil, is consulted instead of reading from standard
	// input. It is the injection point tests use to answer overwrite prompts.
	PromptFn func(question string) bool

	// RenameFn performs the rename and defaults to os.Rename. Tests replace
	// it to simulate a cross-device move without a second filesystem.
	RenameFn func(oldpath, newpath string) error

	streams core.Streams
	diag    *core.Diag
	prompt  *core.Prompter
	isTTY   bool

	// copier is the archive-copy engine for cross-device moves, built on
	// first use and kept so hard links survive across operands.
	copier *cp.Cp
	// copyFailed notes a failure the copier reported on its own Diag.
	copyFailed bool
	// moved records the destinations written for earlier operands, so a
	// later operand with the same base name cannot silently replace them.
	moved map[string]struct{}
}

// New builds a move engine. If s.In is an *os.File attached to a terminal,
// overwriting a write-protected destination prompts unless -f is given.
func New(opts Options, s core.Streams) *Mv {
	return &Mv{
		Options:  opts,
		RenameFn: os.Rename,
		streams:  s,
		diag:     core.NewDiag(Name, s),
		prompt:   core.NewPrompter(Name, s),
		isTTY:    core.IsTerminal(s.In),
		moved:    make(map[string]struct{}),
	}
}

// Run moves according to args, which take one of mv's forms: SOURCE DEST,
// SOURCE... DIRECTORY, or (with -t) SOURCE.... Diagnostics are written as they
// occur; the returned error is core.ErrFailed if any move failed, a
// *core.UsageError for a malformed command line, otherwise nil.
func (m *Mv) Run(args []string) error {
	switch {
	case m.TargetDirectory != "" && m.NoTargetDirectory:
		return core.Usagef("cannot combine --target-directory (-t) and --no-target-directory (-T)")
	case m.backingUp() && m.NoClobber:
		return core.Usagef("cannot combine --backup with --exchange, -n, or --update=none-fail")
	case len(args) == 0:
		return core.Usagef("missing file operand")
	}

	var (
		sources []string
		dest    string
		into    bool // dest is a directory to move the sources into
	)
	switch {
	case m.TargetDirectory != "":
		sources, dest, into = args, m.TargetDirectory, true
		info, err := os.Stat(dest)
		if err == nil && !info.IsDir() {
			m.diag.Errorf("target directory '%s': Not a directory", dest)
			return m.diag.Result()
		}
		if err != nil {
			m.diag.Errorf("target directory '%s': %s", dest, core.ErrText(err))
			return m.diag.Result()
		}
	case len(args) == 1:
		return core.Usagef("missing destination file operand after '%s'", args[0])
	case m.NoTargetDirectory:
		if len(args) > 2 {
			return core.Usagef("extra operand '%s'", args[2])
		}
		sources, dest = args[:1], args[1]
	default:
		sources, dest = args[:len(args)-1], args[len(args)-1]
		if info, err := os.Stat(dest); err == nil && info.IsDir() {
			into = true
		} else if len(sources) > 1 {
			return core.Usagef("target '%s' is not a directory", dest)
		}
	}

	for _, src := range sources {
		if m.StripTrailingSlashes {
			src = cp.StripTrailingSlashes(src)
		}
		dst := dest
		if into {
			dst = cp.JoinInto(dest, src)
		}
		m.moveOperand(src, dst)
	}
	if m.copyFailed {
		return core.ErrFailed
	}
	return m.diag.Result()
}

// moveOperand moves one command-line operand, refusing to replace a file an
// earlier operand of the same invocation produced (mv a/f b/f dir).
func (m *Mv) moveOperand(src, dst string) {
	key := filepath.Clean(dst)
	// A numbered backup keeps the earlier file, so nothing is lost.
	if _, dup := m.moved[key]; dup && m.Backup.Control != core.NumberedBackup {
		m.diag.Errorf("will not overwrite just-created '%s' with '%s'", dst, src)
		return
	}
	if m.move(src, dst) {
		m.moved[key] = struct{}{}
	}
}

// move moves src to exactly dst, reporting whether a move actually happened.
// A skipped move (-n, -u, a declined prompt) is not a failure but returns
// false, since nothing was written at dst.
func (m *Mv) move(src, dst string) bool {
	si, err := os.Lstat(src)
	if err != nil {
		m.diag.Errorf("cannot stat '%s': %s", src, core.ErrText(err))
		return false
	}

	di, err := os.Lstat(dst)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		m.diag.Errorf("cannot stat '%s': %s", dst, core.ErrText(err))
		return false
	}

	// Renaming a file to a different spelling of its own name (foo -> Foo on
	// a case-insensitive filesystem) finds "itself" at the destination; that
	// is a plain rename, not an overwrite.
	caseRename := exists && os.SameFile(si, di) && sameNameIgnoringCase(src, dst)

	// With a backup the destination is moved aside first, so only the very
	// same name is refused.
	if exists && !caseRename && os.SameFile(si, di) && (!m.backingUp() || sameEntry(src, dst)) {
		m.diag.Errorf("'%s' and '%s' are the same file", src, dst)
		return false
	}
	if si.IsDir() && !caseRename && cp.IsWithin(src, dst) {
		m.diag.Errorf("cannot move '%s' to a subdirectory of itself, '%s'", src, dst)
		return false
	}
	if exists && !caseRename {
		switch {
		case m.Update && !di.ModTime().Before(si.ModTime()):
			return false
		case m.NoClobber:
			return false
		case si.IsDir() && !di.IsDir() && !m.backingUp():
			m.diag.Errorf("cannot overwrite non-directory '%s' with directory '%s'", dst, src)
			return false
		case !si.IsDir() && di.IsDir() && !m.backingUp():
			m.diag.Errorf("cannot overwrite directory '%s' with non-directory", dst)
			return false
		case !m.confirmOverwrite(dst, di):
			return false
		}
	}

	backup := ""
	if exists && !caseRename && m.backingUp() && !dotOrDotDot(src) {
		if m.Backup.BackupWouldDestroy(src, si, dst) {
			m.diag.Errorf("backing up '%s' might destroy source;  '%s' not moved", dst, src)
			return false
		}
		name, err := m.Backup.MakeBackup(dst)
		if err != nil {
			m.diag.Errorf("cannot backup '%s': %s", dst, errText(err))
			return false
		}
		backup, exists = name, false
	}
	moved := m.rename(src, dst, di, exists, caseRename, backup)
	if !moved && backup != "" {
		m.unBackup(backup, dst)
	}
	return moved
}

// rename moves src to dst once any backup has been made: a rename, or across
// file systems a copy and a removal.
func (m *Mv) rename(src, dst string, di os.FileInfo, exists, caseRename bool, backup string) bool {
	if exists && !caseRename && di.IsDir() {
		// rename(2) lets a directory replace an empty directory, but
		// os.Rename refuses every existing directory, so the empty one is
		// removed first. A non-empty directory makes this fail, as it should.
		if err := os.Remove(dst); err != nil {
			m.diag.Errorf("cannot move '%s' to '%s': %s", src, dst, errText(err))
			return false
		}
		exists = false
	}

	err := m.RenameFn(src, dst)
	switch {
	case err == nil:
		if m.Verbose {
			m.diag.Printf("renamed '%s' -> '%s'%s", src, dst, backupNote(backup))
		}
		return true
	case isCrossDevice(err) && m.NoCopy:
		// Spelled out so it reads the same on every platform.
		m.diag.Errorf("cannot move '%s' to '%s': Invalid cross-device link", src, dst)
		return false
	case isCrossDevice(err):
		return m.moveAcross(src, dst, exists, backup)
	default:
		m.diag.Errorf("cannot move '%s' to '%s': %s", src, dst, errText(err))
		return false
	}
}

// confirmOverwrite asks before replacing dst when -i is in effect, or when dst
// is write-protected and standard input is a terminal (and -f was not given).
func (m *Mv) confirmOverwrite(dst string, di os.FileInfo) bool {
	switch {
	case m.Interactive:
		return m.ask(fmt.Sprintf("overwrite '%s'?", dst))
	case !m.Force && m.isTTY && di.Mode()&os.ModeSymlink == 0 && !core.Writable(dst):
		perm := di.Mode().Perm()
		return m.ask(fmt.Sprintf("replace '%s', overriding mode %04o (%s)?", dst, perm, perm.String()[1:]))
	default:
		return true
	}
}

// moveAcross moves src to dst on another filesystem: the destination is
// cleared, src is copied with every attribute preserved, and only once the
// copy is complete is src removed.
func (m *Mv) moveAcross(src, dst string, exists bool, backup string) bool {
	if exists {
		// rename(2) would have replaced the destination atomically; removing
		// it first keeps the same "replace, never merge" outcome.
		if err := os.Remove(dst); err != nil {
			m.diag.Errorf("inter-device move failed: '%s' to '%s'; unable to remove target: %s", src, dst, errText(err))
			return false
		}
	}

	if m.copier == nil {
		m.copier = cp.NewNamed(Name, cp.Options{
			Recursive:   true,
			Dereference: cp.DerefNever,
			Preserve:    cp.PreserveAll(),
		}, m.streams)
	}
	if !m.copier.Copy(src, dst) {
		// The copier has already said what went wrong, but on its own Diag,
		// so the failure is noted here. The source is left untouched.
		m.copyFailed = true
		return false
	}
	if m.Verbose {
		m.diag.Printf("copied '%s' -> '%s'%s", src, dst, backupNote(backup))
	}

	if err := os.RemoveAll(src); err != nil {
		m.diag.Errorf("cannot remove '%s': %s", src, errText(err))
		return false
	}
	if m.Verbose {
		m.diag.Printf("removed '%s'", src)
	}
	return true
}

// backingUp reports whether existing destinations are backed up.
func (m *Mv) backingUp() bool {
	return m.Backup.Control != core.NoBackup
}

// backupNote is what -v adds to a line about a replaced destination.
func backupNote(backup string) string {
	if backup == "" {
		return ""
	}
	return " (backup: '" + backup + "')"
}

// unBackup puts a backup back in place of a destination that could not be
// created, as GNU does.
func (m *Mv) unBackup(backup, dst string) {
	if _, err := os.Lstat(dst); err == nil {
		return
	}
	if err := os.Rename(backup, dst); err != nil {
		m.diag.Errorf("cannot un-backup '%s': %s", dst, errText(err))
		return
	}
	if m.Verbose {
		m.diag.Printf("'%s' -> '%s' (unbackup)", backup, dst)
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
	da, err := os.Stat(filepath.Dir(a))
	if err != nil {
		return false
	}
	db, err := os.Stat(filepath.Dir(b))
	return err == nil && os.SameFile(da, db)
}

// ask poses a yes/no question, returning true on an affirmative answer.
func (m *Mv) ask(question string) bool {
	if m.PromptFn != nil {
		return m.PromptFn(question)
	}
	return m.prompt.Ask(question)
}

// sameNameIgnoringCase reports whether a and b name the same directory entry
// spelled with different letter case.
func sameNameIgnoringCase(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b || !strings.EqualFold(filepath.Base(a), filepath.Base(b)) {
		return false
	}
	da, err := os.Stat(filepath.Dir(a))
	if err != nil {
		return false
	}
	db, err := os.Stat(filepath.Dir(b))
	if err != nil {
		return false
	}
	return os.SameFile(da, db)
}

// errText is core.ErrText, except that a non-empty directory is reported as
// such: the standard library folds ENOTEMPTY into "file exists", which would
// print the wrong reason for a refused directory replacement.
func errText(err error) string {
	if isNotEmpty(err) {
		return "Directory not empty"
	}
	return core.ErrText(err)
}
