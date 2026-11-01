package ls

import (
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/internal/human"
	"github.com/braswelljr/gnux/core/internal/visual"
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
	git                        string // --git
}

// printLong writes entries in the long (-l) format.
func (l *Ls) printLong(entries []*entry) {
	f := l.newFrills(entries, true)
	now := l.Now()
	unknownTime := padLeft("?", len(l.style.format(now, now, l.Location)))

	// The columns are as wide as the widest of entries and of measureAlso,
	// the directory operands GNU measures before it sets them aside.
	measured := slices.Concat(entries, l.measureAlso)
	rows := make([]longRow, len(measured))
	var linksW, ownerW, groupW, sizeW, majorW, minorW, timeW int
	for i, e := range measured {
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
				r.size = l.sizeString(sizeOf(e))
			}
		}
		if l.Git {
			r.git = l.gitStatus(e)
		}

		linksW = max(linksW, len(r.links))
		ownerW = max(ownerW, displayWidth(r.owner))
		groupW = max(groupW, displayWidth(r.group))
		timeW = max(timeW, displayWidth(r.time))
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

	var h longHeader
	if l.Header {
		h = l.longHeader()
		linksW, ownerW, groupW = max(linksW, len(h.links)), max(ownerW, len(h.owner)), max(groupW, len(h.group))
		sizeW, timeW = max(sizeW, len(h.size)), max(timeW, len(h.time))
		l.putLongHeader(h, f, linksW, ownerW, groupW, sizeW, timeW)
	}

	for i, e := range entries {
		r := rows[i]
		l.diredIndent()
		l.put(r.frills)
		if l.OctalPermissions {
			l.paint(visual.Octal, octalOf(e))
			l.put(strings.Repeat(" ", 1+h.octalPad))
		}
		l.putMode(e, r.mode)
		l.put(strings.Repeat(" ", 1+h.modePad))
		l.paint(visual.Links, padLeft(r.links, linksW))
		l.put(" ")
		if !l.NoOwner {
			l.paint(l.idKey(e, false), alignID(r.owner, ownerW, r.ownerNumeric))
			l.put(" ")
		}
		if !l.NoGroup {
			l.paint(l.idKey(e, true), alignID(r.group, groupW, r.groupNumeric))
			l.put(" ")
		}
		if l.Author {
			// The author is the owner everywhere but the Hurd.
			l.paint(l.idKey(e, false), alignID(r.owner, ownerW, r.ownerNumeric))
			l.put(" ")
		}
		if r.device {
			device := padLeft(r.major, majorW) + ", " + padLeft(r.minor, minorW)
			l.paint(visual.TypeDevice, padLeft(device, sizeW))
		} else {
			l.paint(l.sizeKey(e), padLeft(r.size, sizeW))
		}
		l.put(" ")
		time := r.time
		if l.Header {
			time = padRight(time, timeW)
		}
		l.paint(l.timeKey(e), time)
		l.put(" ")
		if l.Git {
			l.putGit(r.git)
			l.put(strings.Repeat(" ", 1+h.gitPad))
		}

		l.putName(e)
		switch {
		case e.mode&os.ModeSymlink == 0:
			l.put(l.indicator(e.mode))
		case e.link != "":
			// The indicator of a link describes what it points to and follows
			// the target; the link's own name gets none.
			l.paint(visual.Punctuation, " -> ")
			target, quoted := l.quoteName(e.link)
			code := ""
			if l.palette != nil {
				code = l.palette.targetColor(e)
			}
			// A link target is never padded, and --dired does not mark it.
			var unmarked []int64
			l.putMarked(target, quoted, code, e.path, &unmarked)
			if e.target != nil {
				l.put(l.indicator(e.target.Mode()))
			}
		}
		l.put(l.eol)
	}
}

// longHeader holds the --header labels, and how much wider than their
// values the fixed-width columns become to hold them.
type longHeader struct {
	inode, blocks, octal, mode, links, owner, group, author, size, time, git, name string

	octalPad, modePad, gitPad int
}

func (l *Ls) longHeader() longHeader {
	time := "Modified"
	switch l.Time {
	case TimeAccessed:
		time = "Accessed"
	case TimeChanged:
		time = "Changed"
	case TimeBirth:
		time = "Created"
	}
	return longHeader{
		inode: "Inode", blocks: "Blocks", octal: "Octal", mode: "Permissions", links: "Links",
		owner: "User", group: "Group", author: "Author", size: "Size", time: time, git: "Git", name: "Name",
		octalPad: len("Octal") - 4, modePad: len("Permissions") - 10, gitPad: len("Git") - 2,
	}
}

// putLongHeader writes the --header row, each label in the header color
// and aligned as its column is: numbers to the right, the rest to the left.
func (l *Ls) putLongHeader(h longHeader, f frills, linksW, ownerW, groupW, sizeW, timeW int) {
	label := func(s string, width int, right bool) {
		if n := width - displayWidth(s); n > 0 && right {
			l.put(strings.Repeat(" ", n))
		}
		l.paint(visual.Header, s)
		if n := width - displayWidth(s); n > 0 && !right {
			l.put(strings.Repeat(" ", n))
		}
		l.put(" ")
	}
	l.diredIndent()
	if l.Inode {
		label(h.inode, f.inode, true)
	}
	if l.ShowBlocks {
		label(h.blocks, f.blocks, true)
	}
	if l.OctalPermissions {
		label(h.octal, len(h.octal), false)
	}
	label(h.mode, len(h.mode), false)
	label(h.links, linksW, true)
	if !l.NoOwner {
		label(h.owner, ownerW, false)
	}
	if !l.NoGroup {
		label(h.group, groupW, false)
	}
	if l.Author {
		label(h.author, ownerW, false)
	}
	label(h.size, sizeW, true)
	label(h.time, timeW, false)
	if l.Git {
		label(h.git, len(h.git), false)
	}
	l.paint(visual.Header, h.name)
	l.put(l.eol)
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
	return human.Readable(uint64(max(size, 0)), l.fileOpts, 1, l.fileBlock)
}

// blockString formats an allocation given in 512-byte units, in the -s and
// total unit.
func (l *Ls) blockString(blocks int64) string {
	return human.Readable(uint64(max(blocks, 0)), l.blockOpts, 512, l.blockSize)
}
