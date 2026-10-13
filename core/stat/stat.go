// Package stat implements the engine behind statx, a GNU-compatible
// implementation of the GNU/UNIX stat command: display file or file system
// status.
package stat

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/fsstat"
	"github.com/braswelljr/gnux/core/internal/quote"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "statx"

// The formats GNU stat uses when none is given.
const (
	fileFormat = "  File: %N\n" +
		"  Size: %-10s\tBlocks: %-10b IO Block: %-6o %F\n"
	fileDevice    = "Device: %Hd,%Ld\tInode: %-10i  Links: %h\n"
	fileDeviceDev = "Device: %Hd,%Ld\tInode: %-10i  Links: %-5h Device type: %Hr,%Lr\n"
	fileRest      = "Access: (%04a/%10.10A)  Uid: (%5u/%8U)   Gid: (%5g/%8G)\n" +
		"Access: %x\n" +
		"Modify: %y\n" +
		"Change: %z\n" +
		" Birth: %w\n"
	fsFormat = "  File: %Qn\n" +
		"    ID: %-8i Namelen: %-7l Type: %T\n" +
		"Block size: %-10s Fundamental block size: %S\n" +
		"Blocks: Total: %-10b Free: %-10f Available: %a\n" +
		"Inodes: Total: %-10c Free: %d\n"

	// TerseFormat and TerseFSFormat are what -t and -t -f print.
	TerseFormat   = "%Qn %s %b %f %u %g %D %i %h %t %T %X %Y %Z %W %o\n"
	TerseFSFormat = "%Qn %i %l %t %s %S %b %f %a %c %d\n"
)

// Options mirrors the GNU stat flag surface. The zero value prints the
// status of each file in the default, multi-line format.
type Options struct {
	// Dereference (-L) follows symbolic links.
	Dereference bool
	// FileSystem (-f) shows the status of the file system holding each file.
	FileSystem bool
	// Format (-c) replaces the default format, a newline following each use;
	// it applies when FormatSet is true.
	Format    string
	FormatSet bool
	// Printf (--printf) is like Format but interprets backslash escapes and
	// adds no newline; it applies when PrintfSet is true.
	Printf    string
	PrintfSet bool
	// Terse (-t) prints everything on one line.
	Terse bool
}

// Stat is the engine. Construct it with New.
type Stat struct {
	Options

	// Location is the zone times are shown in; nil means the local zone.
	Location *time.Location
	// UserName and GroupName look up the names of owners; they return false
	// when there is none.
	UserName, GroupName func(id int64) (string, bool)

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
}

// New builds a stat engine writing to s.
func New(opts Options, s core.Streams) *Stat {
	return &Stat{
		Options:   opts,
		UserName:  lookupUser,
		GroupName: lookupGroup,
		streams:   s,
		diag:      core.NewDiag(Name, s),
	}
}

func lookupUser(id int64) (string, bool) {
	u, err := user.LookupId(strconv.FormatInt(id, 10))
	if err != nil {
		return "", false
	}
	return u.Username, true
}

func lookupGroup(id int64) (string, bool) {
	g, err := user.LookupGroupId(strconv.FormatInt(id, 10))
	if err != nil {
		return "", false
	}
	return g.Name, true
}

// errInvalid is a format directive GNU rejects, which ends the run.
type errInvalid struct{ directive string }

func (e *errInvalid) Error() string { return "'" + e.directive + "': invalid directive" }

// Run prints the status of each FILE ("-" is standard input). A file that
// cannot be read is reported and the rest are still shown. It returns a
// *core.UsageError when there is no operand, core.ErrFailed when anything
// failed, and nil otherwise.
func (st *Stat) Run(args []string) (err error) {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	st.out = core.NewWriter(st.streams.Out)
	defer core.FlushOutput(st.out, st.diag, &err)

	format, format2, escapes, trailing := st.formats()
	for _, name := range args {
		var err error
		if st.FileSystem {
			err = st.statfs(name, format, escapes, trailing)
		} else {
			err = st.stat(name, format, format2, escapes, trailing)
		}
		var invalid *errInvalid
		if errors.As(err, &invalid) {
			_ = st.out.Flush()
			st.diag.Errorf("%s", invalid)
			return core.ErrFailed
		}
	}
	return st.diag.Result()
}

// formats returns the format for ordinary files, the one for device files,
// whether backslash escapes apply, and what ends each use.
func (st *Stat) formats() (format, device string, escapes bool, trailing string) {
	switch {
	case st.PrintfSet:
		return st.Printf, st.Printf, true, ""
	case st.FormatSet:
		return st.Format, st.Format, false, "\n"
	case st.FileSystem && st.Terse:
		return TerseFSFormat, TerseFSFormat, false, ""
	case st.FileSystem:
		return fsFormat, fsFormat, false, ""
	case st.Terse:
		return TerseFormat, TerseFormat, false, ""
	}
	return fileFormat + fileDevice + fileRest, fileFormat + fileDeviceDev + fileRest, false, ""
}

// file is what a file directive is printed from.
type file struct {
	name string
	info os.FileInfo
	sys  sysstat.Stat
}

func (st *Stat) stat(name, format, format2 string, escapes bool, trailing string) error {
	var info os.FileInfo
	var err error
	switch {
	case name == "-":
		f, ok := st.streams.In.(*os.File)
		if !ok {
			st.diag.Errorf("cannot stat standard input: %s", core.ErrText(errors.New("bad file descriptor")))
			return nil
		}
		info, err = f.Stat()
		if err != nil {
			st.diag.Errorf("cannot stat standard input: %s", core.ErrText(err))
			return nil
		}
	case st.Dereference:
		info, err = os.Stat(name)
	default:
		info, err = os.Lstat(name)
	}
	if err != nil {
		st.diag.Errorf("cannot stat '%s': %s", name, core.ErrText(err))
		return nil
	}
	f := &file{name: name, info: info, sys: sysstat.Of(info)}
	if info.Mode()&os.ModeDevice != 0 {
		format = format2
	}
	return st.print(format, escapes, trailing, func(d directive) { st.fileDirective(d, f) })
}

func (st *Stat) statfs(name, format string, escapes bool, trailing string) error {
	if name == "-" {
		st.diag.Errorf("using '-' to denote standard input does not work in file system mode")
		return nil
	}
	fs, err := fsstat.Statfs(name)
	if err != nil {
		st.diag.Errorf("cannot read file system information for '%s': %s", name, core.ErrText(err))
		return nil
	}
	return st.print(format, escapes, trailing, func(d directive) { st.fsDirective(d, name, fs) })
}

// directive is one %-directive: its flags, width and precision as written,
// its modifier (H, L or Q) and its conversion character.
type directive struct {
	spec string // flags, width and precision, without the '%'
	mod  byte
	conv byte
}

// printfFlags are the flags a directive may carry.
const printfFlags = "'-+ #0I"

// print interprets a format, calling each for every directive. Only an
// invalid directive is returned; other failures are diagnosed as they occur.
func (st *Stat) print(format string, escapes bool, trailing string, each func(directive)) error {
	for i := 0; i < len(format); i++ {
		c := format[i]
		switch {
		case c == '%':
			j := i + 1
			for j < len(format) && strings.IndexByte(printfFlags, format[j]) >= 0 {
				j++
			}
			for j < len(format) && format[j] >= '0' && format[j] <= '9' {
				j++
			}
			if j < len(format) && format[j] == '.' {
				j++
				for j < len(format) && format[j] >= '0' && format[j] <= '9' {
					j++
				}
			}
			spec := format[i+1 : j]
			if j >= len(format) || format[j] == '%' {
				if spec != "" {
					text := "%" + spec
					if j < len(format) {
						text += "%"
					}
					return &errInvalid{text}
				}
				_ = st.out.WriteByte('%')
				if j >= len(format) {
					j--
				}
				i = j
				continue
			}
			d := directive{spec: spec, conv: format[j]}
			if (d.conv == 'H' || d.conv == 'L' || d.conv == 'Q') && j+1 < len(format) {
				next := format[j+1]
				if (d.conv == 'Q' && next == 'n') ||
					(!st.FileSystem && (d.conv == 'H' || d.conv == 'L') && (next == 'd' || next == 'r')) {
					d.mod, d.conv = d.conv, next
					j++
				}
			}
			each(d)
			i = j
		case c == '\\' && escapes:
			i = st.escape(format, i)
		default:
			_ = st.out.WriteByte(c)
		}
	}
	_, _ = st.out.WriteString(trailing)
	return nil
}

// escape writes the backslash escape at format[i] and returns the index of
// its last character.
func (st *Stat) escape(format string, i int) int {
	if i+1 >= len(format) {
		_ = st.out.Flush()
		st.diag.Warnf("warning: backslash at end of format")
		_ = st.out.WriteByte('\\')
		return i
	}
	i++
	c := format[i]
	switch {
	case c >= '0' && c <= '7':
		v, n := 0, 0
		for ; n < 3 && i < len(format) && format[i] >= '0' && format[i] <= '7'; n++ {
			v = v*8 + int(format[i]-'0')
			i++
		}
		_ = st.out.WriteByte(byte(v))
		return i - 1
	case c == 'x' && i+1 < len(format) && isHex(format[i+1]):
		v := hexVal(format[i+1])
		i++
		if i+1 < len(format) && isHex(format[i+1]) {
			i++
			v = v*16 + hexVal(format[i])
		}
		_ = st.out.WriteByte(byte(v))
		return i
	}
	switch c {
	case 'a':
		c = '\a'
	case 'b':
		c = '\b'
	case 'e':
		c = 0x1b
	case 'f':
		c = '\f'
	case 'n':
		c = '\n'
	case 'r':
		c = '\r'
	case 't':
		c = '\t'
	case 'v':
		c = '\v'
	case '"', '\\':
	default:
		_ = st.out.Flush()
		st.diag.Warnf("warning: unrecognized escape '\\%c'", c)
	}
	_ = st.out.WriteByte(c)
	return i
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func hexVal(c byte) int {
	switch {
	case c >= 'a':
		return int(c-'a') + 10
	case c >= 'A':
		return int(c-'A') + 10
	}
	return int(c - '0')
}

// goFormat builds a Go format from a directive's spec, keeping only the
// flags allowed for the kind of value.
func goFormat(spec, allowed, verb string) string {
	var b strings.Builder
	b.WriteByte('%')
	i := 0
	for ; i < len(spec) && strings.IndexByte(printfFlags, spec[i]) >= 0; i++ {
		if strings.IndexByte(allowed, spec[i]) >= 0 && spec[i] != '\'' {
			b.WriteByte(spec[i])
		}
	}
	b.WriteString(spec[i:])
	b.WriteString(verb)
	return b.String()
}

func (st *Stat) outString(d directive, s string) { fmt.Fprintf(st.out, goFormat(d.spec, "-", "s"), s) }
func (st *Stat) outInt(d directive, n int64)     { fmt.Fprintf(st.out, goFormat(d.spec, "'-+ 0", "d"), n) }
func (st *Stat) outUint(d directive, n uint64)   { fmt.Fprintf(st.out, goFormat(d.spec, "'-0", "d"), n) }
func (st *Stat) outOct(d directive, n uint64)    { fmt.Fprintf(st.out, goFormat(d.spec, "-#0", "o"), n) }
func (st *Stat) outHex(d directive, n uint64)    { fmt.Fprintf(st.out, goFormat(d.spec, "-#0", "x"), n) }

// outEpoch prints seconds since the epoch the way GNU stat does: as an
// integer, or with a precision as a decimal fraction, a width applying to
// the whole number.
func (st *Stat) outEpoch(d directive, t time.Time) {
	secs, nsec := t.Unix(), t.Nanosecond()
	if t.IsZero() {
		secs, nsec = 0, 0
	}
	spec := d.spec
	secSpec := spec
	width, precision := 0, 0
	if dot := strings.IndexByte(spec, '.'); dot >= 0 {
		secSpec = spec[:dot]
		if dot+1 < len(spec) {
			precision, _ = strconv.Atoi(spec[dot+1:])
		} else {
			precision = 9
		}
		if precision > 0 && dot > 0 && isDigit(spec[dot-1]) {
			p := dot
			for p > 0 && isDigit(spec[p-1]) {
				p--
			}
			width, _ = strconv.Atoi(spec[p:dot])
			if width > 1 {
				if spec[p] == '0' {
					p++
				}
				secSpec = spec[:p]
				if wd := width - 1; wd > 1 {
					if w := wd - precision; w > 1 {
						flags, left := "", false
						for _, c := range spec[:p] {
							if c == '-' {
								left = true
							} else {
								flags += string(c)
							}
						}
						secSpec = flags
						if !left {
							secSpec += strconv.Itoa(w)
						}
					}
				}
			}
		}
	}
	divisor := 1
	for i := precision; i < 9; i++ {
		divisor *= 10
	}
	frac := nsec / divisor
	var text string
	if secs < 0 && nsec != 0 {
		frac = 1000000000/divisor - frac - btoi(nsec%divisor != 0)
		if frac != 0 {
			secs++
		}
	}
	if secs == 0 && t.Unix() < 0 && nsec != 0 {
		text = fmt.Sprintf(goFormat(secSpec, "'-+ 0", "s"), "-0")
	} else {
		text = fmt.Sprintf(goFormat(secSpec, "'-+ 0", "d"), secs)
	}
	_, _ = st.out.WriteString(text)
	if precision > 0 {
		prec := min(precision, 9)
		trailingPrec := precision - prec
		trailingWidth := 0
		if n := len(text); n < width && 1 < width-n {
			trailingWidth = width - n - 1 - prec
		}
		fmt.Fprintf(st.out, ".%0*d", prec, frac)
		if trailingPrec > 0 {
			fmt.Fprintf(st.out, "%-*s", trailingWidth, strings.Repeat("0", trailingPrec))
		} else if trailingWidth > 0 {
			fmt.Fprintf(st.out, "%*s", trailingWidth, "")
		}
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// humanTime is GNU's "%Y-%m-%d %H:%M:%S.%N %z".
func (st *Stat) humanTime(t time.Time) string {
	loc := st.Location
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format("2006-01-02 15:04:05.000000000 -0700")
}

func (st *Stat) fileDirective(d directive, f *file) {
	s := f.sys
	switch d.conv {
	case 'n':
		name := f.name
		if d.mod == 'Q' {
			name = quote.ShellEscape(name)
		}
		st.outString(d, name)
	case 'N':
		st.outString(d, quote.ShellEscape(f.name))
		if f.info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(f.name)
			if err != nil {
				_ = st.out.Flush()
				st.diag.Errorf("cannot read symbolic link '%s': %s", f.name, core.ErrText(err))
				return
			}
			_, _ = st.out.WriteString(" -> ")
			st.outString(d, quote.ShellEscape(target))
		}
	case 'd':
		major, minor := sysstat.DeviceNumbers(s.Dev)
		switch d.mod {
		case 'H':
			st.outUint(d, major)
		case 'L':
			st.outUint(d, minor)
		default:
			st.outUint(d, s.Dev)
		}
	case 'D':
		st.outHex(d, s.Dev)
	case 'i':
		st.outUint(d, s.Ino)
	case 'a':
		st.outOct(d, uint64(s.RawMode&0o7777))
	case 'A':
		st.outString(d, modeString(s.RawMode))
	case 'f':
		st.outHex(d, uint64(s.RawMode))
	case 'F':
		st.outString(d, fileType(s.RawMode, f.info.Size()))
	case 'h':
		st.outUint(d, s.Nlink)
	case 'u':
		st.outID(d, s.UID)
	case 'U':
		st.outName(d, s.UID, st.UserName)
	case 'g':
		st.outID(d, s.GID)
	case 'G':
		st.outName(d, s.GID, st.GroupName)
	case 'm':
		st.outString(d, mountPoint(f))
	case 's':
		st.outUint(d, uint64(max(f.info.Size(), 0)))
	case 'r', 'R', 't', 'T':
		major, minor := sysstat.DeviceNumbers(s.Rdev)
		switch {
		case d.conv == 'r' && d.mod == 'H':
			st.outUint(d, major)
		case d.conv == 'r' && d.mod == 'L':
			st.outUint(d, minor)
		case d.conv == 'r':
			st.outUint(d, s.Rdev)
		case d.conv == 'R':
			st.outHex(d, s.Rdev)
		case d.conv == 't':
			st.outHex(d, major)
		default:
			st.outHex(d, minor)
		}
	case 'B':
		st.outUint(d, 512)
	case 'b':
		st.outUint(d, uint64(max(s.Blocks, 0)))
	case 'o':
		st.outUint(d, uint64(max(s.BlockSize, 0)))
	case 'w':
		if s.Btime.IsZero() {
			st.outString(d, "-")
		} else {
			st.outString(d, st.humanTime(s.Btime))
		}
	case 'W':
		st.outEpoch(d, s.Btime)
	case 'x':
		st.outString(d, st.humanTime(orMtime(s.Atime, f)))
	case 'X':
		st.outEpoch(d, orMtime(s.Atime, f))
	case 'y':
		st.outString(d, st.humanTime(f.info.ModTime()))
	case 'Y':
		st.outEpoch(d, f.info.ModTime())
	case 'z':
		st.outString(d, st.humanTime(orMtime(s.Ctime, f)))
	case 'Z':
		st.outEpoch(d, orMtime(s.Ctime, f))
	case 'C':
		_ = st.out.Flush()
		st.diag.Errorf("failed to get security context of '%s': Operation not supported", f.name)
	default:
		_ = st.out.WriteByte('?')
	}
}

// orMtime stands in the modification time for a time the platform does not
// keep (the change time on Windows).
func orMtime(t time.Time, f *file) time.Time {
	if t.IsZero() {
		return f.info.ModTime()
	}
	return t
}

// outID prints a numeric owner, or '?' where files have none (Windows).
func (st *Stat) outID(d directive, id int64) {
	if id < 0 {
		st.outString(d, "?")
		return
	}
	st.outUint(d, uint64(id))
}

func (st *Stat) outName(d directive, id int64, lookup func(int64) (string, bool)) {
	name, ok := "", false
	if id >= 0 {
		name, ok = lookup(id)
	}
	if !ok {
		name = "UNKNOWN"
	}
	st.outString(d, name)
}

func (st *Stat) fsDirective(d directive, name string, fs fsstat.FS) {
	switch d.conv {
	case 'n':
		if d.mod == 'Q' {
			name = quote.ShellEscape(name)
		}
		st.outString(d, name)
	case 'i':
		st.outHex(d, fs.ID)
	case 'l':
		if fs.NameMax == 0 {
			_ = st.out.WriteByte('?')
		} else {
			st.outUint(d, fs.NameMax)
		}
	case 't':
		st.outHex(d, fs.Type)
	case 'T':
		st.outString(d, fs.TypeName)
	case 'b':
		st.outInt(d, int64(fs.Blocks))
	case 'f':
		st.outInt(d, int64(fs.Bfree))
	case 'a':
		st.outInt(d, fs.Bavail)
	case 's':
		st.outUint(d, fs.IOSize)
	case 'S':
		st.outUint(d, fs.BlockSize)
	case 'c':
		if !fs.FilesKnown {
			_ = st.out.WriteByte('?')
		} else {
			st.outUint(d, fs.Files)
		}
	case 'd':
		if !fs.FilesKnown {
			_ = st.out.WriteByte('?')
		} else {
			st.outInt(d, fs.Ffree)
		}
	default:
		_ = st.out.WriteByte('?')
	}
}

// fileType names a file's type as GNU's file_type does.
func fileType(mode uint32, size int64) string {
	switch mode & 0o170000 {
	case 0o100000:
		if size == 0 {
			return "regular empty file"
		}
		return "regular file"
	case 0o040000:
		return "directory"
	case 0o120000:
		return "symbolic link"
	case 0o060000:
		return "block special file"
	case 0o020000:
		return "character special file"
	case 0o010000:
		return "fifo"
	case 0o140000:
		return "socket"
	}
	return "weird file"
}

// modeString renders a mode as ls -l does ("-rw-r--r--").
func modeString(mode uint32) string {
	b := []byte("?rwxrwxrwx")
	switch mode & 0o170000 {
	case 0o100000:
		b[0] = '-'
	case 0o040000:
		b[0] = 'd'
	case 0o120000:
		b[0] = 'l'
	case 0o060000:
		b[0] = 'b'
	case 0o020000:
		b[0] = 'c'
	case 0o010000:
		b[0] = 'p'
	case 0o140000:
		b[0] = 's'
	}
	for i := range 9 {
		if mode&(1<<(8-i)) == 0 {
			b[i+1] = '-'
		}
	}
	special := func(bit uint32, at int, set, unset byte) {
		if mode&bit != 0 {
			if b[at] == '-' {
				b[at] = unset
			} else {
				b[at] = set
			}
		}
	}
	special(0o4000, 3, 's', 'S')
	special(0o2000, 6, 's', 'S')
	special(0o1000, 9, 't', 'T')
	return string(b)
}

// mountPoint finds the mount point of a file: the highest directory above it
// on the same device, as gnulib's find_mount_point does.
func mountPoint(f *file) string {
	dir := f.name
	if !f.info.IsDir() {
		dir = filepath.Dir(dir)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "?"
	}
	dev := f.sys.Dev
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		info, err := os.Stat(parent)
		if err != nil || sysstat.Of(info).Dev != dev {
			return dir
		}
		dir = parent
	}
}
