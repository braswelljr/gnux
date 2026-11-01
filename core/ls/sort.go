package ls

import (
	"slices"
	"strings"
	"time"
)

// sortEntries orders entries by the selected key. Ties always fall back to the
// name, so the order is deterministic.
func (l *Ls) sortEntries(entries []*entry) {
	if l.sortKey == SortNone {
		return
	}
	cmp := l.comparator()
	slices.SortStableFunc(entries, func(a, b *entry) int {
		// Grouping is not subject to -r: directories stay first either way.
		if l.GroupDirectories {
			if da, db := a.isDirLike(), b.isDirLike(); da != db {
				if da {
					return -1
				}
				return 1
			}
		}
		c := cmp(a, b)
		if c == 0 {
			c = strings.Compare(a.name, b.name)
		}
		if l.Reverse {
			return -c
		}
		return c
	})
}

// comparator returns the primary comparison for the selected sort key.
func (l *Ls) comparator() func(a, b *entry) int {
	switch l.sortKey {
	case SortTime:
		// Newest first.
		return func(a, b *entry) int { return l.timeOf(b).Compare(l.timeOf(a)) }
	case SortSize:
		// Largest first.
		return func(a, b *entry) int {
			sa, sb := sizeOf(a), sizeOf(b)
			switch {
			case sa > sb:
				return -1
			case sa < sb:
				return 1
			}
			return 0
		}
	case SortExtension:
		return func(a, b *entry) int { return strings.Compare(extension(a.name), extension(b.name)) }
	case SortVersion:
		return func(a, b *entry) int { return versionCompare(a.name, b.name) }
	case SortWidth:
		return func(a, b *entry) int { return a.width - b.width }
	default:
		// Byte-wise comparison: the C locale's collation.
		return func(a, b *entry) int { return strings.Compare(a.name, b.name) }
	}
}

// timeOf returns the timestamp selected by -c, -u or --time. A timestamp the
// platform does not record falls back to the modification time.
func (l *Ls) timeOf(e *entry) time.Time {
	if e.info == nil {
		return time.Time{}
	}
	var t time.Time
	switch l.Time {
	case TimeAccessed:
		t = e.sys.atime
	case TimeChanged:
		t = e.sys.ctime
	case TimeBirth:
		t = e.sys.btime
	}
	if t.IsZero() {
		return e.info.ModTime()
	}
	return t
}

// sizeOf is what -S sorts on: the file's size, or with --total-size what a
// directory holds.
func sizeOf(e *entry) int64 {
	if e.info == nil {
		return 0
	}
	if e.hasTotal {
		return e.total
	}
	return e.info.Size()
}

// extension returns the text from the last dot on, or "" when there is none.
func extension(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// versionCompare orders names the way GNU's filevercmp does: runs of digits
// compare numerically, so "file9" sorts before "file10". Dot files come first
// and a trailing run of ".suffix" groups is ignored on the first pass, so
// "a.tar.gz" sorts before "a-1.tar.gz".
func versionCompare(a, b string) int {
	switch {
	case a == "" || b == "":
		return len(a) - len(b)
	case a[0] == '.' && b[0] != '.':
		return -1
	case a[0] != '.' && b[0] == '.':
		return 1
	case a[0] == '.':
		// "." first, then "..", then the other dot files.
		if ra, rb := dotRank(a), dotRank(b); ra != rb || ra < 2 {
			return ra - rb
		}
	}

	pa, pb := versionPrefixLen(a), versionPrefixLen(b)
	if c := versionRunCompare(a[:pa], b[:pb]); c != 0 || (pa == len(a) && pb == len(b)) {
		return c
	}
	return versionRunCompare(a, b)
}

// dotRank orders the two directory self-references ahead of other dot files.
func dotRank(name string) int {
	switch name {
	case ".":
		return 0
	case "..":
		return 1
	}
	return 2
}

// versionPrefixLen returns the length of s without its file suffixes: the
// longest tail matching (\.[A-Za-z~][A-Za-z0-9~]*)*.
func versionPrefixLen(s string) int {
	n := len(s)
	prefix := 0
	for i := 0; i < n; {
		i++
		prefix = i
		for i+1 < n && s[i] == '.' && (isAlpha(s[i+1]) || s[i+1] == '~') {
			for i += 2; i < n && (isAlpha(s[i]) || isDigit(s[i]) || s[i] == '~'); i++ {
			}
		}
	}
	return prefix
}

// versionRunCompare compares alternating runs of non-digits and digits.
func versionRunCompare(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		for (i < len(a) && !isDigit(a[i])) || (j < len(b) && !isDigit(b[j])) {
			if ca, cb := versionOrder(a, i), versionOrder(b, j); ca != cb {
				return ca - cb
			}
			i++
			j++
		}
		for i < len(a) && a[i] == '0' {
			i++
		}
		for j < len(b) && b[j] == '0' {
			j++
		}
		// With leading zeros gone, the longer run of digits is the larger
		// number; equal lengths are decided by the first differing digit.
		firstDiff := 0
		for i < len(a) && j < len(b) && isDigit(a[i]) && isDigit(b[j]) {
			if firstDiff == 0 {
				firstDiff = int(a[i]) - int(b[j])
			}
			i++
			j++
		}
		switch {
		case i < len(a) && isDigit(a[i]):
			return 1
		case j < len(b) && isDigit(b[j]):
			return -1
		case firstDiff != 0:
			return firstDiff
		}
	}
	return 0
}

// versionOrder ranks the byte at s[i] for version comparison: "~" sorts before
// the end of the string, which sorts before digits, letters and everything else.
func versionOrder(s string, i int) int {
	if i >= len(s) {
		return -1
	}
	c := s[i]
	switch {
	case isDigit(c):
		return 0
	case isAlpha(c):
		return int(c)
	case c == '~':
		return -2
	}
	return int(c) + 256
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
