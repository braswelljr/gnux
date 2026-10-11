package ls

import (
	"os"
	"strconv"
	"strings"
)

// longRow holds the formatted columns of one long-format line, so that widths
// can be measured before anything is written.
type longRow struct {
	frills       string
	mode         string
	links        string
	owner, group string
	// Names are left-aligned but numeric IDs right-aligned, as GNU ls does.
	ownerNumeric, groupNumeric bool
	size                       string // non-devices
	major, minor               string // devices
	device                     bool
	time                       string
}

// printLong writes entries in the long (-l) format.
func (l *Ls) printLong(entries []*entry) {
	f := l.newFrills(entries, true)
	now := l.Now()
	unknownTime := padLeft("?", len(l.style.format(now, now, l.Location)))

	rows := make([]longRow, len(entries))
	var linksW, ownerW, groupW, sizeW, majorW, minorW int
	for i, e := range entries {
		r := longRow{frills: l.frillsOf(e, f), mode: modeString(e), links: "?", size: "?", time: unknownTime}
		r.owner, r.ownerNumeric = l.ownerOf(e)
		r.group, r.groupNumeric = l.groupOf(e)
		if e.info != nil {
			r.links = strconv.FormatUint(e.sys.nlink, 10)
			r.time = l.style.format(l.timeOf(e), now, l.Location)
			if e.mode&os.ModeDevice != 0 {
				major, minor := deviceNumbers(e.sys.rdev)
				r.device = true
				r.major, r.minor = strconv.FormatUint(major, 10), strconv.FormatUint(minor, 10)
			} else {
				r.size = l.sizeString(e.info.Size())
			}
		}

		linksW = max(linksW, len(r.links))
		ownerW = max(ownerW, displayWidth(r.owner))
		groupW = max(groupW, displayWidth(r.group))
		if r.device {
			majorW, minorW = max(majorW, len(r.major)), max(minorW, len(r.minor))
		} else {
			sizeW = max(sizeW, len(r.size))
		}
		rows[i] = r
	}
	// Device numbers share the size column, which grows to hold "major, minor".
	if majorW > 0 {
		sizeW = max(sizeW, majorW+2+minorW)
	}

	for i, e := range entries {
		r := rows[i]
		var b strings.Builder
		b.WriteString(r.frills)
		b.WriteString(r.mode + " " + padLeft(r.links, linksW) + " ")
		if !l.NoOwner {
			b.WriteString(alignID(r.owner, ownerW, r.ownerNumeric) + " ")
		}
		if !l.NoGroup {
			b.WriteString(alignID(r.group, groupW, r.groupNumeric) + " ")
		}
		if r.device {
			device := padLeft(r.major, majorW) + ", " + padLeft(r.minor, minorW)
			b.WriteString(padLeft(device, sizeW))
		} else {
			b.WriteString(padLeft(r.size, sizeW))
		}
		b.WriteString(" " + r.time + " ")
		l.put(b.String())

		l.putName(e)
		switch {
		case e.mode&os.ModeSymlink == 0:
			l.put(l.indicator(e.mode))
		case e.link != "":
			// The indicator of a link describes what it points to and follows
			// the target; the link's own name gets none.
			l.put(" -> ")
			target := l.quoteName(e.link)
			if l.palette != nil {
				l.putColored(target, l.palette.targetColor(e))
			} else {
				l.put(target)
			}
			if e.target != nil {
				l.put(l.indicator(e.target.Mode()))
			}
		}
		l.put("\n")
	}
}

func alignID(s string, width int, numeric bool) string {
	if numeric {
		return padLeft(s, width)
	}
	return padRight(s, width)
}

// ownerOf returns the owner column and whether it is a numeric ID.
func (l *Ls) ownerOf(e *entry) (string, bool) {
	if e.info == nil {
		return "?", false
	}
	if !l.NumericIDs {
		if name, ok := l.ids.user(e.sys.uid); ok {
			return name, false
		}
	}
	return strconv.FormatUint(uint64(e.sys.uid), 10), true
}

// groupOf returns the group column and whether it is a numeric ID.
func (l *Ls) groupOf(e *entry) (string, bool) {
	if e.info == nil {
		return "?", false
	}
	if !l.NumericIDs {
		if name, ok := l.ids.group(e.sys.gid); ok {
			return name, false
		}
	}
	return strconv.FormatUint(uint64(e.sys.gid), 10), true
}

// typeChar returns the first character of the mode string.
func typeChar(mode os.FileMode) byte {
	switch {
	case mode.IsDir():
		return 'd'
	case mode&os.ModeSymlink != 0:
		return 'l'
	case mode&os.ModeNamedPipe != 0:
		return 'p'
	case mode&os.ModeSocket != 0:
		return 's'
	case mode&os.ModeCharDevice != 0:
		return 'c'
	case mode&os.ModeDevice != 0:
		return 'b'
	case mode.IsRegular():
		return '-'
	}
	return '?'
}

// modeString renders the ten-character mode column ("drwxr-xr-x"), with s/S
// and t/T for setuid, setgid and sticky. An entry that could not be examined
// keeps its type, when the directory supplied one, and shows "?" for the rest.
func modeString(e *entry) string {
	if e.info == nil {
		c := typeChar(e.mode)
		if e.mode.Type() == 0 {
			// A zero type is "unknown" here, not "regular file".
			c = '?'
		}
		return string(c) + "?????????"
	}

	mode := e.mode
	buf := []byte{typeChar(mode), '-', '-', '-', '-', '-', '-', '-', '-', '-'}
	const rwx = "rwxrwxrwx"
	for i := range 9 {
		if mode&(1<<uint(8-i)) != 0 {
			buf[i+1] = rwx[i]
		}
	}
	special := func(set bool, pos int, lower, upper byte) {
		if !set {
			return
		}
		if buf[pos] == 'x' {
			buf[pos] = lower
		} else {
			buf[pos] = upper
		}
	}
	special(mode&os.ModeSetuid != 0, 3, 's', 'S')
	special(mode&os.ModeSetgid != 0, 6, 's', 'S')
	special(mode&os.ModeSticky != 0, 9, 't', 'T')
	return string(buf)
}

// indicator returns the -F / -p / --file-type suffix for a file mode.
func (l *Ls) indicator(mode os.FileMode) string {
	switch {
	case l.Indicator == IndicatorNone:
		return ""
	case mode.IsDir():
		return "/"
	case l.Indicator == IndicatorSlash:
		return ""
	case mode&os.ModeSymlink != 0:
		return "@"
	case mode&os.ModeNamedPipe != 0:
		return "|"
	case mode&os.ModeSocket != 0:
		return "="
	case l.Indicator == IndicatorClassify && mode.IsRegular() && mode&0o111 != 0:
		return "*"
	}
	return ""
}

// sizeString formats a byte count for the size column.
func (l *Ls) sizeString(size int64) string {
	if l.HumanReadable || l.SI {
		return humanSize(uint64(max(size, 0)), l.SI)
	}
	return strconv.FormatInt(size, 10)
}

// blockString formats an allocation given in 512-byte units: as a count of 1K
// blocks, or scaled with -h / --si.
func (l *Ls) blockString(blocks int64) string {
	blocks = max(blocks, 0)
	if l.HumanReadable || l.SI {
		return humanSize(uint64(blocks)*512, l.SI)
	}
	return strconv.FormatInt((blocks+1)/2, 10)
}

// humanSize scales n to at most three significant characters plus a unit
// suffix, rounding up as GNU ls does: 1.0K, 9.9K, 10K, 1023K, 1.0M. Powers of
// 1024 are used unless si is set, which selects powers of 1000 and a lowercase
// "k".
func humanSize(n uint64, si bool) string {
	base, suffixes := uint64(1024), "KMGTPE"
	if si {
		base, suffixes = 1000, "kMGTPE"
	}
	if n < base {
		return strconv.FormatUint(n, 10)
	}

	unit, exp := base, 0
	for exp < len(suffixes)-1 && n/unit >= base {
		unit *= base
		exp++
	}

	q, r := n/unit, n%unit
	if q < 10 {
		// One decimal place, rounded up to the next tenth.
		if tenths := q*10 + (r*10+unit-1)/unit; tenths < 100 {
			return strconv.FormatUint(tenths/10, 10) + "." + strconv.FormatUint(tenths%10, 10) + suffixes[exp:exp+1]
		}
		q, r = 10, 0
	}
	if r > 0 {
		q++
	}
	// Rounding up can spill into the next unit: 1023.5K is 1.0M.
	if q >= base && exp < len(suffixes)-1 {
		return "1.0" + suffixes[exp+1:exp+2]
	}
	return strconv.FormatUint(q, 10) + suffixes[exp:exp+1]
}
