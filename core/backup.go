package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// BackupControl says whether, and how, a destination is backed up before it
// is replaced: GNU's --backup=CONTROL, shared by cp, mv, ln and install.
type BackupControl int

const (
	// NoBackup (none, off) makes no backups. It is the zero value: a tool
	// without --backup.
	NoBackup BackupControl = iota
	// SimpleBackup (simple, never) renames FILE to FILE plus the suffix,
	// replacing any earlier simple backup.
	SimpleBackup
	// ExistingBackup (existing, nil) makes a numbered backup if FILE already
	// has one, and a simple backup otherwise. It is GNU's default CONTROL.
	ExistingBackup
	// NumberedBackup (numbered, t) renames FILE to FILE.~N~, N one more than
	// the highest number already used.
	NumberedBackup
)

// DefaultBackupSuffix is the simple backup suffix when none is given.
const DefaultBackupSuffix = "~"

// Backup describes the backups a tool makes of the destinations it replaces
// (--backup, -b and -S, --suffix). The zero value makes none.
type Backup struct {
	Control BackupControl
	// Suffix is the simple backup suffix. Empty, or a suffix that names a
	// directory (one containing a '/'), means DefaultBackupSuffix, as in GNU.
	Suffix string
}

// backupWords are GNU's CONTROL words, synonyms together, in the order its
// diagnostics list them.
var backupWords = []struct {
	word    string
	control BackupControl
}{
	{"none", NoBackup}, {"off", NoBackup},
	{"simple", SimpleBackup}, {"never", SimpleBackup},
	{"existing", ExistingBackup}, {"nil", ExistingBackup},
	{"numbered", NumberedBackup}, {"t", NumberedBackup},
}

// ParseBackupControl reads a CONTROL word as GNU does: an unambiguous
// abbreviation is accepted, and an empty word means ExistingBackup. context
// names what was read in the error ("backup type", or "$VERSION_CONTROL" for
// the environment variable); the error is a *UsageError that lists the valid
// words, as GNU's does.
func ParseBackupControl(context, word string) (BackupControl, error) {
	if word == "" {
		return ExistingBackup, nil
	}
	found, ambiguous := -1, false
	for i, w := range backupWords {
		if w.word == word {
			return w.control, nil
		}
		if strings.HasPrefix(w.word, word) {
			switch {
			case found < 0:
				found = i
			case backupWords[found].control != w.control:
				ambiguous = true
			}
		}
	}
	if found >= 0 && !ambiguous {
		return backupWords[found].control, nil
	}
	problem := "invalid"
	if ambiguous {
		problem = "ambiguous"
	}
	var b strings.Builder
	b.WriteString(problem)
	b.WriteString(" argument '")
	b.WriteString(word)
	b.WriteString("' for '")
	b.WriteString(context)
	b.WriteString("'\nValid arguments are:")
	for i, w := range backupWords {
		if i == 0 || backupWords[i-1].control != w.control {
			b.WriteString("\n  - '")
			b.WriteString(w.word)
			b.WriteString("'")
		} else {
			b.WriteString(", '")
			b.WriteString(w.word)
			b.WriteString("'")
		}
	}
	return NoBackup, &UsageError{Msg: b.String()}
}

// SuffixInUse is the simple backup suffix b actually uses.
func (b Backup) SuffixInUse() string {
	if s := b.Suffix; s != "" && lastComponent(s) == s {
		return s
	}
	return DefaultBackupSuffix
}

// lastComponent is gnulib's last_component: what follows the last run of
// slashes that is followed by something else.
func lastComponent(s string) string {
	base := strings.TrimLeft(s, "/")
	start, slash := 0, false
	for i := 0; i < len(base); i++ {
		switch {
		case base[i] == '/':
			slash = true
		case slash:
			start, slash = i, false
		}
	}
	return base[start:]
}

// maxNameLen is the longest file name the systems gnux runs on accept.
const maxNameLen = 255

// BackupName returns the name the backup of path would get, without making
// it. With NoBackup it is "".
func (b Backup) BackupName(path string) string {
	name, _ := b.backupName(path)
	return name
}

// backupName also says whether the name is a numbered one.
func (b Backup) backupName(path string) (string, bool) {
	if b.Control == NoBackup {
		return "", false
	}
	path = strings.TrimRight(path, "/")
	dir, base := filepath.Split(path)
	if b.Control != SimpleBackup {
		if version, ok := highestBackup(dir, base); ok || b.Control == NumberedBackup {
			return dir + fitName(base, base+".~"+increment(version)+"~"), true
		}
	}
	return dir + fitName(base, base+b.SuffixInUse()), false
}

// highestBackup returns the highest N of the numbered backups base.~N~ in
// dir, as a decimal string, or "0" when there is none.
func highestBackup(dir, base string) (version string, ok bool) {
	if dir == "" {
		dir = "."
	}
	entries, err := fsys.ReadDir(dir)
	version = "0"
	if err != nil {
		return version, false
	}
	prefix := base + ".~"
	for _, e := range entries {
		n, found := strings.CutPrefix(e.Name(), prefix)
		if !found {
			continue
		}
		n, found = strings.CutSuffix(n, "~")
		if !found || n == "" || n[0] < '1' || n[0] > '9' || strings.Trim(n, "0123456789") != "" {
			continue
		}
		// Compare as numbers of any length: longer is larger.
		if len(n) > len(version) || len(n) == len(version) && n > version {
			version = n
		}
		ok = true
	}
	return version, ok
}

// increment adds one to the decimal string n.
func increment(n string) string {
	digits := []byte(n)
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < '9' {
			digits[i]++
			return string(digits)
		}
		digits[i] = '0'
	}
	return "1" + string(digits)
}

// fitName shortens a backup name that is too long for the file system the
// way gnulib does: what follows the original name becomes a single '~', and
// if that is still too long, the character before it goes too.
func fitName(base, name string) string {
	if len(name) <= maxNameLen {
		return name
	}
	keep := min(len(base), maxNameLen-1)
	return base[:keep] + "~"
}

// MakeBackup renames path to its backup name and returns that name. A path
// that does not exist needs no backup: the name is "" and the error nil. A
// numbered backup never replaces another file; a simple one replaces the
// previous simple backup.
func (b Backup) MakeBackup(path string) (string, error) {
	if b.Control == NoBackup {
		return "", nil
	}
	for range 16 {
		name, numbered := b.backupName(path)
		if numbered {
			// Another process may have taken the number since the scan.
			if _, err := fsys.Lstat(name); err == nil {
				continue
			}
		}
		err := fsys.Rename(strings.TrimRight(path, "/"), name)
		switch {
		case err == nil:
			return name, nil
		case errors.Is(err, fs.ErrNotExist):
			if _, lerr := fsys.Lstat(path); errors.Is(lerr, fs.ErrNotExist) {
				return "", nil
			}
		}
		return "", err
	}
	return "", &os.PathError{Op: "rename", Path: path, Err: fs.ErrExist}
}

// BackupWouldDestroy reports whether backing up dst would replace src: src is
// dst's simple backup (dst plus the suffix, the same file). GNU refuses that
// copy or move, since the backup would be renamed over the source.
func (b Backup) BackupWouldDestroy(src string, si os.FileInfo, dst string) bool {
	if b.Control == NoBackup || b.Control == NumberedBackup {
		return false
	}
	suffix := b.SuffixInUse()
	if filepath.Base(src) != filepath.Base(dst)+suffix {
		return false
	}
	bi, err := fsys.Stat(strings.TrimRight(dst, "/") + suffix)
	return err == nil && os.SameFile(si, bi)
}
