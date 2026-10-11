// Package touch implements the engine behind touchx, a GNU-compatible implementation of
// the GNU/UNIX touch command: update the access and modification times of
// files, creating the ones that do not exist.
package touch

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "touchx"

// Options mirrors the GNU touch flag surface.
type Options struct {
	Access        bool   // -a: change only the access time
	Modify        bool   // -m: change only the modification time
	NoCreate      bool   // -c: do not create any files
	Date          string // -d: parse this string and use it instead of the current time
	Stamp         string // -t: use [[CC]YY]MMDDhhmm[.ss] instead of the current time
	Reference     string // -r: use this file's times instead of the current time
	NoDereference bool   // -h: affect a symbolic link itself instead of its target
	Time          string // --time=WORD: access/atime/use (-a) or modify/mtime (-m)
}

// Touch is the engine. Construct it with New.
type Touch struct {
	Options

	// Now is the clock. It defaults to time.Now; tests replace it to get
	// reproducible timestamps. Dates without a zone are read in its location.
	Now func() time.Time

	diag *core.Diag
}

// New builds a touch engine writing to s.
func New(opts Options, s core.Streams) *Touch {
	return &Touch{
		Options: opts,
		Now:     time.Now,
		diag:    core.NewDiag(Name, s),
	}
}

// Run updates the timestamps of each file in args. Diagnostics are written as
// they occur; the returned error is core.ErrFailed if any file could not be
// touched.
func (t *Touch) Run(args []string) error {
	changeAccess, changeModify := t.Access, t.Modify
	switch t.Time {
	case "":
	case "access", "atime", "use":
		changeAccess = true
	case "modify", "mtime":
		changeModify = true
	default:
		return core.Usagef("invalid argument '%s' for '--time'", t.Time)
	}
	if !changeAccess && !changeModify {
		changeAccess, changeModify = true, true
	}

	if t.Stamp != "" && (t.Date != "" || t.Reference != "") {
		return core.Usagef("cannot specify times from more than one source")
	}
	if len(args) == 0 {
		return core.Usagef("missing file operand")
	}

	atime, mtime, err := t.times()
	if err != nil {
		return err
	}
	// os.Chtimes and lchtimes leave a zero time untouched, which is how the
	// timestamp that was not asked for is preserved.
	if !changeAccess {
		atime = time.Time{}
	}
	if !changeModify {
		mtime = time.Time{}
	}

	for _, path := range args {
		t.touch(path, atime, mtime)
	}
	return t.diag.Result()
}

// times works out the access and modification times to apply from -t, -d and
// -r, falling back to the current time.
func (t *Touch) times() (atime, mtime time.Time, err error) {
	now := t.Now()

	if t.Stamp != "" {
		when, err := parseStamp(t.Stamp, now)
		if err != nil {
			return atime, mtime, core.Usagef("invalid date format '%s'", t.Stamp)
		}
		return when, when, nil
	}

	atime, mtime = now, now
	var date time.Time
	if t.Date != "" {
		if date, err = parseDate(t.Date, now); err != nil {
			return atime, mtime, core.Usagef("invalid date format '%s'", t.Date)
		}
	}

	if t.Reference != "" {
		info, err := t.stat(t.Reference)
		if err != nil {
			t.diag.Errorf("failed to get attributes of '%s': %s", t.Reference, core.ErrText(err))
			return atime, mtime, t.diag.Result()
		}
		atime, mtime = AccessTime(info), info.ModTime()
		// With -r a date is relative to the reference file's times, so "now"
		// means those times rather than the wall clock.
		if isNow(t.Date) {
			return atime, mtime, nil
		}
	}
	if t.Date != "" {
		atime, mtime = date, date
	}
	return atime, mtime, nil
}

// touch updates one file, creating it first if that is allowed.
func (t *Touch) touch(path string, atime, mtime time.Time) {
	if _, err := t.stat(path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			t.diag.Errorf("cannot touch '%s': %s", path, core.ErrText(err))
			return
		}
		// A missing file is not an error under -c. -h never creates either:
		// there is no link whose times could be changed.
		if t.NoCreate || t.NoDereference {
			return
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			t.diag.Errorf("cannot touch '%s': %s", path, core.ErrText(err))
			return
		}
		_ = f.Close()
	}

	set := os.Chtimes
	if t.NoDereference {
		set = lchtimes
	}
	if err := set(path, atime, mtime); err != nil {
		t.diag.Errorf("setting times of '%s': %s", path, core.ErrText(err))
	}
}

// stat looks up path, following symbolic links unless -h was given.
func (t *Touch) stat(path string) (os.FileInfo, error) {
	if t.NoDereference {
		return os.Lstat(path)
	}
	return os.Stat(path)
}

func isNow(date string) bool {
	return strings.EqualFold(strings.TrimSpace(date), "now")
}
