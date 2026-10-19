package ls

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// fileURL is the file:// URL --hyperlink gives path: its absolute form with
// every symbolic link resolved (what GNU's canonicalize_filename_mode gives),
// each byte outside RFC 3986's unreserved set percent-encoded and every
// separator written as "/".
func fileURL(host, path string) string {
	abs := canonical(path)
	escaped := uriEscape(abs, true)
	if !strings.HasPrefix(escaped, "/") {
		escaped = "/" + escaped
	}
	return "file://" + uriEscape(host, false) + escaped
}

// canonical resolves as much of path as exists, keeping the rest as written.
func canonical(path string) string {
	abs, err := fsys.Abs(path)
	if err != nil {
		return path
	}
	if resolved, err := fsys.EvalSymlinks(abs); err == nil {
		return resolved
	}
	parent := filepath.Dir(abs)
	if parent == abs {
		return abs
	}
	return filepath.Join(canonical(parent), filepath.Base(abs))
}

// uriEscape is GNU ls's file_escape.
func uriEscape(s string, path bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case path && (c == '/' || c == filepath.Separator):
			b.WriteByte('/')
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '~', c == '-', c == '.', c == '_':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02x", c)
		}
	}
	return b.String()
}

// diredSummary ends --dired output with the positions of every name and
// directory header, and the quoting style that names them.
func (l *Ls) diredSummary() {
	dump := func(prefix string, positions []int64) {
		if len(positions) == 0 {
			return
		}
		var b strings.Builder
		b.WriteString(prefix)
		for _, p := range positions {
			b.WriteByte(' ')
			b.WriteString(strconv.FormatInt(p, 10))
		}
		b.WriteByte('\n')
		l.put(b.String())
	}
	dump("//DIRED//", l.diredPos)
	dump("//SUBDIRED//", l.subdired)
	l.put("//DIRED-OPTIONS// --quoting-style=" + l.quoting.String() + "\n")
}
