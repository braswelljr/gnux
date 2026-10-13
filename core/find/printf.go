package find

import (
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// segment is one piece of a compiled -printf format: literal text, or a
// directive with its flags.
type segment struct {
	lit []byte

	kind byte // the directive letter, 0 for a literal
	sub  byte // the time field of %A, %B, %C and %T
	stop bool // \c: stop printing here

	left, zero, plus, alt bool
	width                 int
	prec                  int // -1: none
}

// printfPred compiles format and returns the action that prints it.
func (f *Find) printfPred(o *output, format string) (pred, error) {
	segs, err := f.compileFormat(format)
	if err != nil {
		return nil, err
	}
	return func(e *entry) bool {
		buf := f.buf[:0]
		for i := range segs {
			s := &segs[i]
			if s.stop {
				break
			}
			if s.kind == 0 {
				buf = append(buf, s.lit...)
				continue
			}
			buf = f.directive(buf, s, e)
		}
		f.buf = buf
		f.write(o, buf)
		return true
	}, nil
}

// compileFormat splits a -printf format into segments, resolving the
// backslash escapes once.
func (f *Find) compileFormat(format string) ([]segment, error) {
	var segs []segment
	var lit []byte
	flush := func() {
		if len(lit) > 0 {
			segs = append(segs, segment{lit: lit})
			lit = nil
		}
	}

	for i := 0; i < len(format); i++ {
		c := format[i]
		switch c {
		case '\\':
			if i+1 == len(format) {
				f.diag.Warnf("warning: escape '\\' followed by nothing at all")
				lit = append(lit, '\\')
				continue
			}
			i++
			switch d := format[i]; d {
			case 'a':
				lit = append(lit, '\a')
			case 'b':
				lit = append(lit, '\b')
			case 'f':
				lit = append(lit, '\f')
			case 'n':
				lit = append(lit, '\n')
			case 'r':
				lit = append(lit, '\r')
			case 't':
				lit = append(lit, '\t')
			case 'v':
				lit = append(lit, '\v')
			case '\\':
				lit = append(lit, '\\')
			case 'c':
				flush()
				segs = append(segs, segment{stop: true})
				return segs, nil
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// Up to three octal digits.
				n, j := 0, i
				for ; j < len(format) && j < i+3 && format[j] >= '0' && format[j] <= '7'; j++ {
					n = n*8 + int(format[j]-'0')
				}
				lit = append(lit, byte(n))
				i = j - 1
			default:
				f.diag.Warnf("warning: unrecognized escape '\\%c'", d)
				lit = append(lit, '\\', d)
			}
		case '%':
			if i+1 == len(format) {
				lit = append(lit, '%')
				continue
			}
			if format[i+1] == '%' {
				lit = append(lit, '%')
				i++
				continue
			}
			seg := segment{prec: -1}
			j := i + 1
			for ; j < len(format) && strings.IndexByte("-0+ #", format[j]) >= 0; j++ {
				switch format[j] {
				case '-':
					seg.left = true
				case '0':
					seg.zero = true
				case '+':
					seg.plus = true
				case '#':
					seg.alt = true
				}
			}
			for ; j < len(format) && format[j] >= '0' && format[j] <= '9'; j++ {
				seg.width = min(seg.width*10+int(format[j]-'0'), 1<<16)
			}
			if j < len(format) && format[j] == '.' {
				seg.prec = 0
				for j++; j < len(format) && format[j] >= '0' && format[j] <= '9'; j++ {
					seg.prec = min(seg.prec*10+int(format[j]-'0'), 1<<16)
				}
			}
			if j == len(format) {
				lit = append(lit, format[i:]...)
				i = j
				continue
			}
			seg.kind = format[j]
			switch {
			case strings.IndexByte("ABCT", seg.kind) >= 0:
				if j+1 == len(format) || !knownTimeField(format[j+1]) {
					f.diag.Warnf("warning: unrecognized format directive '%s'", format[i:min(j+2, len(format))])
					lit = append(lit, format[i:min(j+2, len(format))]...)
					i = min(j+1, len(format)-1)
					continue
				}
				j++
				seg.sub = format[j]
			case seg.kind == 'F' || seg.kind == 'Z':
				return nil, core.Usagef("the %%%c directive of -printf is not supported", seg.kind)
			case strings.IndexByte("abcdDfgGhHiklmMnpPsStuUyY", seg.kind) < 0:
				f.diag.Warnf("warning: unrecognized format directive '%s'", format[i:j+1])
				lit = append(lit, format[i:j+1]...)
				i = j
				continue
			}
			flush()
			segs = append(segs, seg)
			i = j
		default:
			lit = append(lit, c)
		}
	}
	flush()
	return segs, nil
}

// directive appends the value of one directive for e, padded to its width.
func (f *Find) directive(buf []byte, s *segment, e *entry) []byte {
	start := len(buf)
	numeric := false

	switch s.kind {
	case 'p':
		buf = append(buf, e.path...)
	case 'f':
		buf = append(buf, baseName(e.path)...)
	case 'h':
		buf = append(buf, dirName(e.path)...)
	case 'H':
		buf = append(buf, e.start...)
	case 'P':
		rest := e.path[len(e.start):]
		for rest != "" && isSep(rest[0]) {
			rest = rest[1:]
		}
		buf = append(buf, rest...)
	case 'd':
		buf, numeric = strconv.AppendInt(buf, int64(e.depth), 10), true
	case 'y':
		buf = append(buf, typeLetter(f.typeOf(e)))
	case 'Y':
		buf = append(buf, f.followedType(e))
	case 'l':
		target, _ := f.linkTarget(e)
		buf = append(buf, target...)
	default:
		info, ok := f.stat(e)
		if !ok {
			break
		}
		sys := e.sysstat()
		switch s.kind {
		case 's':
			buf, numeric = strconv.AppendInt(buf, info.Size(), 10), true
		case 'm':
			perm := unixPerm(info.Mode())
			if s.alt && perm != 0 {
				buf = append(buf, '0')
			}
			buf, numeric = strconv.AppendUint(buf, uint64(perm), 8), true
		case 'M':
			buf = appendModeString(buf, info.Mode())
		case 'u':
			buf = append(buf, f.names.display(sys.UID, true)...)
		case 'g':
			buf = append(buf, f.names.display(sys.GID, false)...)
		case 'U':
			buf, numeric = strconv.AppendInt(buf, max(sys.UID, 0), 10), true
		case 'G':
			buf, numeric = strconv.AppendInt(buf, max(sys.GID, 0), 10), true
		case 'n':
			buf, numeric = strconv.AppendUint(buf, sys.Nlink, 10), true
		case 'i':
			buf, numeric = strconv.AppendUint(buf, sys.Ino, 10), true
		case 'D':
			buf, numeric = strconv.AppendUint(buf, sys.Dev, 10), true
		case 'b':
			buf, numeric = strconv.AppendInt(buf, sys.Blocks, 10), true
		case 'k':
			buf, numeric = strconv.AppendInt(buf, (sys.Blocks+1)/2, 10), true
		case 'S':
			sparse := 1.0
			if info.Size() > 0 {
				sparse = float64(sys.Blocks) * 512 / float64(info.Size())
			}
			buf, numeric = strconv.AppendFloat(buf, sparse, 'g', 6, 64), true
		case 'a', 'c', 't':
			which := map[byte]byte{'a': 'a', 'c': 'c', 't': 'm'}[s.kind]
			if t, ok := fileTime(info, sys.Atime, sys.Ctime, sys.Btime, which); ok {
				buf = appendCtime(buf, t.In(f.Location))
			}
		case 'A', 'B', 'C', 'T':
			which := map[byte]byte{'A': 'a', 'B': 'B', 'C': 'c', 'T': 'm'}[s.kind]
			if t, ok := fileTime(info, sys.Atime, sys.Ctime, sys.Btime, which); ok {
				buf = appendTimeField(buf, t.In(f.Location), s.sub)
			}
		}
	}

	if s.plus && numeric && buf[start] != '-' {
		buf = append(buf, 0)
		copy(buf[start+1:], buf[start:])
		buf[start] = '+'
	}
	if s.prec >= 0 && !numeric && len(buf)-start > s.prec {
		buf = buf[:start+s.prec]
	}
	if pad := s.width - (len(buf) - start); pad > 0 {
		switch {
		case s.left:
			for range pad {
				buf = append(buf, ' ')
			}
		default:
			fill := byte(' ')
			if s.zero && numeric {
				fill = '0'
			}
			buf = append(buf, make([]byte, pad)...)
			copy(buf[start+pad:], buf[start:])
			for i := range pad {
				buf[start+i] = fill
			}
		}
	}
	return buf
}

// followedType answers %Y: the type with a symbolic link followed, 'L' for a
// link that loops, 'N' for one whose target is missing and '?' for any other
// failure.
func (f *Find) followedType(e *entry) byte {
	t := f.typeOf(e)
	if t&fs.ModeSymlink == 0 {
		return typeLetter(t)
	}
	info, err := os.Stat(e.path)
	switch {
	case err == nil:
		return typeLetter(info.Mode().Type())
	case errText(err) == "Too many levels of symbolic links":
		return 'L'
	case danglingLink(err):
		return 'N'
	}
	return '?'
}

// appendModeString appends the ten-character mode of ls -l ("drwxr-xr-x").
func appendModeString(buf []byte, mode fs.FileMode) []byte {
	t := typeLetter(mode.Type())
	switch t {
	case 'f':
		t = '-'
	case 'U':
		t = '?'
	}
	start := len(buf)
	buf = append(buf, t, '-', '-', '-', '-', '-', '-', '-', '-', '-')
	const rwx = "rwxrwxrwx"
	for i := range 9 {
		if mode&(1<<uint(8-i)) != 0 {
			buf[start+1+i] = rwx[i]
		}
	}
	special := func(set bool, pos int, lower, upper byte) {
		if !set {
			return
		}
		if buf[start+pos] == 'x' {
			buf[start+pos] = lower
		} else {
			buf[start+pos] = upper
		}
	}
	special(mode&fs.ModeSetuid != 0, 3, 's', 'S')
	special(mode&fs.ModeSetgid != 0, 6, 's', 'S')
	special(mode&fs.ModeSticky != 0, 9, 't', 'T')
	return buf
}

// appendFraction appends GNU's ten-digit fraction of a second: the
// nanoseconds and a trailing 0.
func appendFraction(buf []byte, t time.Time) []byte {
	buf = append(buf, '.')
	nanos := strconv.Itoa(t.Nanosecond())
	for range 9 - len(nanos) {
		buf = append(buf, '0')
	}
	buf = append(buf, nanos...)
	return append(buf, '0')
}

// appendCtime appends a time as %a, %c and %t print it: ctime(3) with the
// fraction of a second ("Mon Jan  2 15:04:05.0000000000 2006").
func appendCtime(buf []byte, t time.Time) []byte {
	buf = t.AppendFormat(buf, "Mon Jan _2 15:04:05")
	buf = appendFraction(buf, t)
	return t.AppendFormat(buf, " 2006")
}

// timeLayouts maps the time fields that are plain strftime conversions to
// Go layouts.
var timeLayouts = map[byte]string{
	'a': "Mon", 'A': "Monday", 'b': "Jan", 'h': "Jan", 'B': "January",
	'c': "Mon Jan _2 15:04:05 2006", 'd': "02", 'D': "01/02/06", 'e': "_2",
	'F': "2006-01-02", 'H': "15", 'I': "03", 'j': "002", 'm': "01", 'M': "04",
	'p': "PM", 'r': "03:04:05 PM", 'R': "15:04", 'x': "01/02/06", 'y': "06",
	'Y': "2006", 'z': "-0700", 'Z': "MST",
}

func knownTimeField(k byte) bool {
	if _, ok := timeLayouts[k]; ok {
		return true
	}
	return strings.IndexByte("@+STXklCsuwUWGgV", k) >= 0
}

// appendTimeField appends one field of a time, the k of %Ak. The fields
// that show seconds (@ + S T X) carry GNU's fraction.
func appendTimeField(buf []byte, t time.Time, k byte) []byte {
	two := func(buf []byte, n int, fill byte) []byte {
		if n < 10 {
			buf = append(buf, fill)
		}
		return strconv.AppendInt(buf, int64(n), 10)
	}
	switch k {
	case '@':
		buf = strconv.AppendInt(buf, t.Unix(), 10)
		return appendFraction(buf, t)
	case '+':
		buf = t.AppendFormat(buf, "2006-01-02+15:04:05")
		return appendFraction(buf, t)
	case 'S':
		buf = t.AppendFormat(buf, "05")
		return appendFraction(buf, t)
	case 'T', 'X':
		buf = t.AppendFormat(buf, "15:04:05")
		return appendFraction(buf, t)
	case 'k':
		return two(buf, t.Hour(), ' ')
	case 'l':
		return two(buf, (t.Hour()+11)%12+1, ' ')
	case 'C':
		return two(buf, t.Year()/100, '0')
	case 's':
		return strconv.AppendInt(buf, t.Unix(), 10)
	case 'u':
		return strconv.AppendInt(buf, int64((int(t.Weekday())+6)%7+1), 10)
	case 'w':
		return strconv.AppendInt(buf, int64(t.Weekday()), 10)
	case 'U':
		return two(buf, (t.YearDay()+6-int(t.Weekday()))/7, '0')
	case 'W':
		return two(buf, (t.YearDay()+6-(int(t.Weekday())+6)%7)/7, '0')
	case 'G':
		year, _ := t.ISOWeek()
		return strconv.AppendInt(buf, int64(year), 10)
	case 'g':
		year, _ := t.ISOWeek()
		return two(buf, year%100, '0')
	case 'V':
		_, week := t.ISOWeek()
		return two(buf, week, '0')
	}
	return t.AppendFormat(buf, timeLayouts[k])
}

// --- -ls ---

// lsWidths are the column widths of -ls. They start at GNU's and, as in GNU,
// only ever grow when a wider value turns up.
type lsWidths struct {
	inode, blocks, links, owner, group, size int
}

var defaultLsWidths = lsWidths{inode: 9, blocks: 6, links: 3, owner: 8, group: 8, size: 8}

// sixMonths is half an average Gregorian year, the age at which ls shows the
// year in place of the time of day.
const sixMonths = 15778476 * time.Second

// lsPred returns the action behind -ls and -fls: the file in the format of
// ls -dils, with sizes in 1K blocks.
func (f *Find) lsPred(o *output) pred {
	return func(e *entry) bool {
		info, ok := f.stat(e)
		if !ok {
			return true
		}
		sys := e.sysstat()
		w := &f.ls
		buf := f.buf[:0]

		num := func(buf []byte, text string, width *int) []byte {
			*width = max(*width, len(text))
			for range *width - len(text) {
				buf = append(buf, ' ')
			}
			return append(buf, text...)
		}
		name := func(buf []byte, text string, width *int) []byte {
			*width = max(*width, len(text))
			buf = append(buf, text...)
			for range *width - len(text) {
				buf = append(buf, ' ')
			}
			return buf
		}

		buf = num(buf, strconv.FormatUint(sys.Ino, 10), &w.inode)
		buf = append(buf, ' ')
		buf = num(buf, strconv.FormatInt((sys.Blocks+1)/2, 10), &w.blocks)
		buf = append(buf, ' ')
		buf = appendModeString(buf, info.Mode())
		buf = append(buf, ' ')
		buf = num(buf, strconv.FormatUint(sys.Nlink, 10), &w.links)
		buf = append(buf, ' ')
		buf = name(buf, f.names.display(sys.UID, true), &w.owner)
		buf = append(buf, ' ')
		buf = name(buf, f.names.display(sys.GID, false), &w.group)
		buf = append(buf, ' ')
		if info.Mode()&fs.ModeDevice != 0 {
			major, minor := sysstat.DeviceNumbers(sys.Rdev)
			text := strconv.FormatUint(major, 10)
			for range 3 - len(text) {
				buf = append(buf, ' ')
			}
			buf = append(append(buf, text...), ',', ' ')
			text = strconv.FormatUint(minor, 10)
			for range 3 - len(text) {
				buf = append(buf, ' ')
			}
			buf = append(buf, text...)
		} else {
			buf = num(buf, strconv.FormatInt(info.Size(), 10), &w.size)
		}
		buf = append(buf, ' ')

		mtime := info.ModTime().In(f.Location)
		if mtime.After(f.now.Add(-sixMonths)) && !mtime.After(f.now) {
			buf = mtime.AppendFormat(buf, "Jan _2 15:04")
		} else {
			buf = mtime.AppendFormat(buf, "Jan _2  2006")
		}
		buf = append(buf, ' ')
		buf = appendEscaped(buf, e.path)
		if target, ok := f.linkTarget(e); ok {
			buf = append(buf, " -> "...)
			buf = appendEscaped(buf, target)
		}
		buf = append(buf, '\n')

		f.buf = buf
		f.write(o, buf)
		return true
	}
}

// appendEscaped appends a file name as -ls prints it: white space, backslash
// and double quote with a backslash, other control bytes in octal.
func appendEscaped(buf []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case ' ', '\\', '"':
			buf = append(buf, '\\', c)
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\t':
			buf = append(buf, '\\', 't')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\f':
			buf = append(buf, '\\', 'f')
		case '\v':
			buf = append(buf, '\\', 'v')
		case '\a':
			buf = append(buf, '\\', 'a')
		case '\b':
			buf = append(buf, '\\', 'b')
		default:
			if c < 0x20 || c == 0x7f {
				buf = append(buf, '\\', '0'+c>>6, '0'+c>>3&7, '0'+c&7)
			} else {
				buf = append(buf, c)
			}
		}
	}
	return buf
}
