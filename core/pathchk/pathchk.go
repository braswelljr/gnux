// Package pathchk implements the engine behind pathchkx, a GNU-compatible
// implementation of the GNU/UNIX pathchk command: check whether file names
// are valid on this system, or portable to any POSIX system.
package pathchk

import (
	"errors"
	"io/fs"
	"strings"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "pathchkx"

// The POSIX minimum limits every system allows: _POSIX_PATH_MAX (counting
// the terminating null byte) and _POSIX_NAME_MAX.
const (
	PosixPathMax = 256
	PosixNameMax = 14
)

// Options mirrors the GNU pathchk flag surface. The zero value checks names
// against the limits of the file systems they would be on.
type Options struct {
	// Portable (-p) checks against the POSIX minimum limits and the portable
	// file name character set instead of this system.
	Portable bool
	// Extra (-P) also rejects empty names and components that start with '-'.
	Extra bool
}

// Pathchk is the engine. Construct it with New.
type Pathchk struct {
	Options

	// NameMax and PathMax return pathconf's _PC_NAME_MAX and _PC_PATH_MAX for
	// a directory; a negative limit with a nil error means there is none.
	// They are the injection points tests use, defaulting to the system's.
	NameMax func(dir string) (int, error)
	PathMax func(dir string) (int, error)

	diag *core.Diag
}

// New builds a pathchk engine writing its diagnostics to s.
func New(opts Options, s core.Streams) *Pathchk {
	return &Pathchk{Options: opts, NameMax: nameMax, PathMax: pathMax, diag: core.NewDiag(Name, s)}
}

// Run checks every operand; one that fails is reported and the rest are
// still checked.
func (p *Pathchk) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	for _, name := range args {
		p.check(name)
	}
	return p.diag.Result()
}

// quoteaf and quotef are gnulib's quoting of a file name in a message.
func quoteaf(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscapeAlways})
}

func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

func quote(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale})
}

// check is GNU's validate_file_name.
func (p *Pathchk) check(file string) bool {
	if p.Extra {
		for _, c := range components(file) {
			if strings.HasPrefix(c, "-") {
				p.diag.Errorf("leading '-' in a component of file name %s", quoteaf(file))
				return false
			}
		}
	}
	if (p.Portable || p.Extra) && file == "" {
		p.diag.Errorf("empty file name")
		return false
	}

	exists := false
	if p.Portable {
		for i, r := range file {
			if !portable(r) {
				ch := file[i : i+1]
				if r != utf8.RuneError {
					ch = string(r)
				}
				p.diag.Errorf("non-portable character %s in file name %s", quote(ch), quoteaf(file))
				return false
			}
		}
	} else {
		_, err := fsys.Lstat(file)
		switch {
		case err == nil:
			exists = true
		case !errors.Is(err, fs.ErrNotExist) || file == "":
			p.diag.Errorf("%s: %s", quotef(file), core.ErrText(err))
			return false
		}
	}

	if p.Portable || (!exists && len(file) >= PosixPathMax) {
		limit := PosixPathMax
		if !p.Portable {
			dir := "."
			if strings.HasPrefix(file, "/") {
				dir = "/"
			}
			n, err := p.PathMax(dir)
			if err != nil {
				p.diag.Errorf("%s: unable to determine maximum file name length: %s", dir, core.ErrText(err))
				return false
			}
			limit = n
		}
		if limit >= 0 && len(file) >= limit {
			p.diag.Errorf("limit %d exceeded by length %d of file name %s", limit-1, len(file), quoteaf(file))
			return false
		}
	}

	// Components no longer than the POSIX minimum fit on any file system, so
	// pathconf is asked only when one is longer.
	checkLengths := p.Portable
	if !checkLengths && !exists {
		for _, c := range components(file) {
			if len(c) > PosixNameMax {
				checkLengths = true
				break
			}
		}
	}
	if !checkLengths {
		return true
	}

	limit := PosixNameMax // for file systems where pathconf fails
	known := 0
	if p.Portable {
		known = PosixNameMax
	}
	for _, c := range componentSpans(file) {
		if known != 0 {
			limit = known
		} else {
			dir := "."
			if c.start > 0 {
				dir = file[:c.start]
			}
			n, err := p.NameMax(dir)
			switch {
			case err == nil:
				limit = n
			case errors.Is(err, fs.ErrNotExist):
				// The directory does not exist: its parent's limit holds.
				known = limit
			default:
				p.diag.Errorf("%s: %s", quotef(dir), core.ErrText(err))
				return false
			}
		}
		name := file[c.start:c.end]
		if limit >= 0 && len(name) > limit {
			p.diag.Errorf("limit %d exceeded by length %d of file name component %s", limit, len(name), quote(name))
			return false
		}
	}
	return true
}

// portable reports whether r is in the POSIX portable file name character
// set, or is the separator.
func portable(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return r == '/' || r == '.' || r == '_' || r == '-'
}

type span struct{ start, end int }

// componentSpans returns where each component of file lies, skipping the
// slashes between them.
func componentSpans(file string) []span {
	var spans []span
	for i := 0; i < len(file); {
		for i < len(file) && file[i] == '/' {
			i++
		}
		if i == len(file) {
			break
		}
		j := i
		for j < len(file) && file[j] != '/' {
			j++
		}
		spans = append(spans, span{i, j})
		i = j
	}
	return spans
}

func components(file string) []string {
	var cs []string
	for _, s := range componentSpans(file) {
		cs = append(cs, file[s.start:s.end])
	}
	return cs
}
