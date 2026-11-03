// Package du implements the engine behind dux, a GNU-compatible
// implementation of the GNU/UNIX du command: estimate file space usage.
//
// Directories are read in parallel, then sizes are added up in one pass in
// name order, so a large tree is measured quickly and the output, including
// which path a hard-linked file is counted under, is the same on every run.
package du

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/date"
	"github.com/braswelljr/gnux/core/internal/human"
	"github.com/braswelljr/gnux/core/internal/quote"
	"github.com/braswelljr/gnux/core/internal/sysstat"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "dux"

// Deref says which symbolic links are followed.
type Deref int

const (
	// DerefNone (-P, the default) follows no symbolic link.
	DerefNone Deref = iota
	// DerefArgs (-H, -D) follows symbolic links given as operands.
	DerefArgs
	// DerefAll (-L) follows every symbolic link.
	DerefAll
)

// Options mirrors the GNU du flag surface. The zero value is plain du: the
// disk usage of each directory under the operands, in 1024-byte blocks.
type Options struct {
	// All (-a) lists files as well as directories.
	All bool
	// Apparent (--apparent-size) counts file sizes rather than disk usage.
	Apparent bool
	// BlockSize (-B) is the output unit in GNU's syntax ("1K", "M",
	// "human-readable", "si"); empty means DU_BLOCK_SIZE, BLOCK_SIZE,
	// BLOCKSIZE or 1024 bytes (512 under POSIXLY_CORRECT).
	BlockSize string
	// Total (-c) adds a grand total.
	Total bool
	// Deref chooses the symbolic links followed.
	Deref Deref
	// MaxDepth (-d) limits the listing to that many levels below each
	// operand; it applies when MaxDepthSet is true.
	MaxDepth    int
	MaxDepthSet bool
	// Summarize (-s) lists each operand only.
	Summarize bool
	// SeparateDirs (-S) leaves the size of subdirectories out of a
	// directory's total.
	SeparateDirs bool
	// CountLinks (-l) counts a hard-linked file each time it is met.
	CountLinks bool
	// OneFileSystem (-x) skips directories on other file systems.
	OneFileSystem bool
	// Inodes (--inodes) counts inodes rather than space.
	Inodes bool
	// Threshold (-t) leaves out entries smaller than it, or when negative
	// entries larger than its magnitude.
	Threshold int64
	// Time (--time) shows the latest "mtime", "atime" or "ctime" in each
	// subtree; empty means no time column.
	Time string
	// TimeStyle (--time-style) is full-iso, long-iso, iso or +FORMAT; empty
	// means TIME_STYLE or long-iso.
	TimeStyle string
	// Exclude (--exclude) skips entries whose path, or any part of it after
	// a '/', matches a shell pattern; ExcludeFrom (-X) reads such patterns,
	// one per line, from files.
	Exclude     []string
	ExcludeFrom []string
	// Files0From (--files0-from) reads NUL-separated operands from a file
	// ("-" is standard input).
	Files0From string
	// Null (-0) ends each output line with NUL.
	Null bool

	// Pretty is gnux's terminal view, not a GNU option: each operand's
	// lines are aligned, with a bar and a percentage giving each line's
	// share of the operand, and with Color the sizes on the theme's
	// gradient and the names highlighted, in the colors of Theme (a
	// built-in theme; "" takes GNUX_THEME's). It has no effect with Null.
	Pretty bool
	Color  bool
	Theme  string
}

// Du is the engine. Construct it with New.
type Du struct {
	Options

	// Getenv reads the block size and time style variables; it defaults to
	// os.Getenv.
	Getenv func(string) string
	// IsTerminal decides whether names are quoted for the terminal.
	IsTerminal func(any) bool
	// Location is the zone times are shown in; nil means the local zone.
	Location *time.Location

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	blockSize  uint64
	humanOpts  human.Opts
	timeFormat string
	exclude    []*regexp.Regexp
	quoteNames bool
	hashAll    bool
	seen       map[devIno]bool
	seenPath   map[string]bool // where file IDs are unknown (Windows), by absolute path
	sem        chan struct{}
	pending    []line // the pretty view's lines, held until their operand is done
	theme      *visual.Theme
	trap       core.Trap // a panic on a scanning goroutine, raised again by the caller
}

type devIno struct{ dev, ino uint64 }

// New builds a du engine writing to s.
func New(opts Options, s core.Streams) *Du {
	return &Du{
		Options:    opts,
		Getenv:     os.Getenv,
		IsTerminal: core.IsTerminal,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// Run measures each operand (the current directory when there is none). A
// file that cannot be read is reported and the rest are still measured. It
// returns a *core.UsageError for a command-line mistake, core.ErrFailed
// when anything failed, and nil otherwise.
func (d *Du) Run(args []string) (err error) {
	if err := d.setup(); err != nil {
		return err
	}
	operands := args
	if d.Files0From != "" {
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'\nfile operands cannot be combined with --files0-from", args[0])
		}
		names, err := d.readFiles0()
		if err != nil {
			return err
		}
		operands = names
		d.hashAll = true
	} else if len(operands) == 0 {
		operands = []string{"."}
	}
	if len(operands) > 1 || d.Deref == DerefAll {
		d.hashAll = true
	}

	d.out = core.NewWriter(d.streams.Out)
	defer core.FlushOutput(d.out, d.diag, &err)
	d.theme = visual.Use(d.Pretty && d.Color, d.Theme, d.Getenv, d.diag.Warnf)
	var grand sums
	for _, name := range operands {
		root := d.scanRoot(name)
		if root == nil {
			continue
		}
		d.walk(root, 0, &grand)
		d.flushPretty()
	}
	if d.Total {
		d.print(grand, "total")
		d.flushPretty()
	}
	return d.diag.Result()
}

// setup checks the options and resolves units, time style and patterns.
func (d *Du) setup() error {
	if d.All && d.Summarize {
		return core.Usagef("cannot both summarize and show all entries")
	}
	if d.Summarize && d.MaxDepthSet {
		if d.MaxDepth != 0 {
			return core.Usagef("warning: summarizing conflicts with --max-depth=%d", d.MaxDepth)
		}
		d.diag.Warnf("warning: summarizing is the same as using --max-depth=0")
	}
	if d.Summarize {
		d.MaxDepth, d.MaxDepthSet = 0, true
	}
	if !d.MaxDepthSet {
		d.MaxDepth = int(^uint(0) >> 1)
	}

	if d.BlockSize != "" {
		size, opts, err := human.ParseBlockSize(d.BlockSize)
		if err != nil {
			return core.Usagef("invalid --block-size argument '%s'", d.BlockSize)
		}
		d.blockSize, d.humanOpts = size, opts
	} else {
		d.blockSize, d.humanOpts = human.DefaultBlockSize(d.Getenv, "DU_BLOCK_SIZE")
	}
	if d.Inodes {
		if d.Apparent {
			d.diag.Warnf("warning: options --apparent-size and -b are ineffective with --inodes")
		}
		d.blockSize = 1
	}

	switch d.Time {
	case "", "mtime", "atime", "ctime":
	default:
		return core.Usagef("invalid argument '%s' for '--time'", d.Time)
	}
	if d.Time != "" {
		f, err := timeFormat(d.TimeStyle, d.Getenv)
		if err != nil {
			return err
		}
		d.timeFormat = f
	}

	patterns := slices.Clone(d.Exclude)
	for _, name := range d.ExcludeFrom {
		data, err := fsys.ReadFile(name)
		if err != nil {
			d.diag.Errorf("%s: %s", name, core.ErrText(err))
			return core.ErrFailed
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				patterns = append(patterns, line)
			}
		}
	}
	for _, p := range patterns {
		re, err := fnmatch(p)
		if err != nil {
			return core.Usagef("invalid pattern '%s'", p)
		}
		d.exclude = append(d.exclude, re)
	}

	d.quoteNames = !d.Null && d.IsTerminal(d.streams.Out)
	d.seen = map[devIno]bool{}
	d.seenPath = map[string]bool{}
	d.sem = make(chan struct{}, max(4, 2*runtime.NumCPU()))
	return nil
}

// timeFormat resolves --time-style as GNU du does.
func timeFormat(style string, getenv func(string) string) (string, error) {
	if style == "" {
		style = getenv("TIME_STYLE")
		switch {
		case style == "" || style == "locale":
			style = "long-iso"
		case strings.HasPrefix(style, "+"):
			style, _, _ = strings.Cut(style, "\n")
		default:
			for strings.HasPrefix(style, "posix-") {
				style = style[len("posix-"):]
			}
		}
	}
	if strings.HasPrefix(style, "+") {
		return style[1:], nil
	}
	switch style {
	case "full-iso":
		return "%Y-%m-%d %H:%M:%S.%N %z", nil
	case "long-iso":
		return "%Y-%m-%d %H:%M", nil
	case "iso":
		return "%Y-%m-%d", nil
	}
	return "", core.Usagef("invalid argument '%s' for 'time style'\nValid arguments are:\n"+
		"  - 'full-iso'\n  - 'long-iso'\n  - 'iso'\n  - '+FORMAT' (e.g., +%%H:%%M) for a 'date'-style format", style)
}

// readFiles0 reads the NUL-separated operands of --files0-from.
func (d *Du) readFiles0() ([]string, error) {
	r := d.streams.In
	if d.Files0From != "-" {
		f, err := fsys.Open(d.Files0From)
		if err != nil {
			d.diag.Errorf("cannot open '%s' for reading: %s", d.Files0From, core.ErrText(err))
			return nil, core.ErrFailed
		}
		defer f.Close()
		r = f
	}
	if r == nil {
		return nil, nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		d.diag.Errorf("%s: read error: %s", d.Files0From, core.ErrText(err))
		return nil, core.ErrFailed
	}
	var names []string
	for i, name := range bytes.Split(bytes.TrimSuffix(data, []byte{0}), []byte{0}) {
		switch {
		case len(data) == 0:
		case len(name) == 0:
			d.diag.Errorf("%s:%d: invalid zero-length file name", d.Files0From, i+1)
		case d.Files0From == "-" && string(name) == "-":
			d.diag.Errorf("when reading file names from standard input, no file name of '-' allowed")
		default:
			names = append(names, string(name))
		}
	}
	return names, nil
}

// entry is one file of the scanned tree.
type entry struct {
	path     string
	dir      bool
	id       devIno
	nlink    uint64
	size     uint64 // bytes, disk usage or apparent size
	time     time.Time
	excluded bool
	// accessErr means the entry could not be examined; readErr that a
	// directory could not be read (its own size still counts).
	accessErr error
	readErr   error
	children  []*entry
}

// scanRoot examines an operand and, for a directory, its whole tree.
func (d *Du) scanRoot(name string) *entry {
	stat := fsys.Lstat
	if d.Deref != DerefNone {
		stat = fsys.Stat
	}
	info, err := stat(name)
	if err != nil {
		return &entry{path: name, accessErr: err}
	}
	root := d.newEntry(name, info)
	if root.excluded || !root.dir {
		return root
	}
	d.scanDir(root, root.id.dev, []devIno{root.id})
	d.trap.Release()
	return root
}

func (d *Du) newEntry(path string, info os.FileInfo) *entry {
	s := sysstat.Of(info)
	e := &entry{path: path, dir: info.IsDir(), id: devIno{s.Dev, s.Ino}, nlink: s.Nlink}
	switch {
	case d.Apparent:
		if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			e.size = uint64(max(info.Size(), 0))
		}
	default:
		e.size = uint64(max(s.Blocks, 0)) * 512
	}
	switch d.Time {
	case "atime":
		e.time = s.Atime
	case "ctime":
		e.time = s.Ctime
	}
	if e.time.IsZero() {
		e.time = info.ModTime()
	}
	e.excluded = d.excluded(path)
	return e
}

// excluded reports whether path, or any part of it after a '/', matches an
// --exclude pattern, as gnulib's exclude does without EXCLUDE_ANCHORED.
func (d *Du) excluded(path string) bool {
	// Patterns are written with '/', as on Unix; Windows paths use '\'.
	path = filepath.ToSlash(path)
	for _, re := range d.exclude {
		if re.MatchString(path) {
			return true
		}
		for i := 0; i < len(path); i++ {
			if isSep(path[i]) && i+1 < len(path) && !isSep(path[i+1]) && re.MatchString(path[i+1:]) {
				return true
			}
		}
	}
	return false
}

func isSep(c byte) bool { return c == '/' || c == os.PathSeparator }

// scanDir reads a directory and its subdirectories, in parallel. ancestors
// are the directories above it, to stop at a cycle under -L.
func (d *Du) scanDir(e *entry, rootDev uint64, ancestors []devIno) {
	// The slot is given back as soon as the directory has been read, and by
	// the defer if reading it panics: a slot never returned would leave the
	// other readers waiting for it.
	d.sem <- struct{}{}
	reading := true
	done := func() {
		if reading {
			reading = false
			<-d.sem
		}
	}
	defer done()
	dirents, err := fsys.ReadDir(e.path)
	children := make([]*entry, 0, len(dirents))
	for _, de := range dirents {
		path := join(e.path, de.Name())
		var info os.FileInfo
		var serr error
		if d.Deref == DerefAll && de.Type()&os.ModeSymlink != 0 {
			info, serr = fsys.Stat(path)
		} else {
			info, serr = de.Info()
		}
		if serr != nil {
			children = append(children, &entry{path: path, accessErr: serr})
			continue
		}
		c := d.newEntry(path, info)
		if d.OneFileSystem && c.id.dev != rootDev {
			c.excluded = true
		}
		children = append(children, c)
	}
	done()
	if err != nil {
		e.readErr = err
	}

	var wg sync.WaitGroup
	for _, c := range children {
		if !c.dir || c.excluded || c.accessErr != nil {
			continue
		}
		// A cycle through symbolic links: skip it. Where file IDs are unknown
		// (Windows reports none) every directory would match, so none is a cycle.
		if c.id != (devIno{}) && slices.Contains(ancestors, c.id) {
			c.excluded = true
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer d.trap.Catch()
			d.scanDir(c, rootDev, append(slices.Clip(ancestors), c.id))
		}()
	}
	wg.Wait()
	slices.SortFunc(children, func(a, b *entry) int { return strings.Compare(a.path, b.path) })
	e.children = children
}

// join appends a name to a directory path the way fts does.
func join(dir, name string) string {
	if dir != "" && isSep(dir[len(dir)-1]) {
		return dir + name
	}
	return dir + string(os.PathSeparator) + name
}

// sums is what is added up for an entry.
type sums struct {
	size, inodes uint64
	tmax         time.Time
}

func (s *sums) add(o sums) {
	s.size += o.size
	s.inodes += o.inodes
	if o.tmax.After(s.tmax) {
		s.tmax = o.tmax
	}
}

// walk adds up an entry in name order, printing as GNU du does after each
// directory's contents. It returns what the entry contributes to its
// parent's total, and whether it was counted at all.
func (d *Du) walk(e *entry, level int, grand *sums) (sums, bool) {
	if e.accessErr != nil {
		d.errorf("cannot access '%s': %s", e.path, core.ErrText(e.accessErr))
		return sums{}, false
	}
	if e.excluded {
		return sums{}, false
	}
	if !d.CountLinks && (d.hashAll || (!e.dir && e.nlink > 1)) {
		if e.id != (devIno{}) {
			if d.seen[e.id] {
				return sums{}, false
			}
			d.seen[e.id] = true
		} else if d.hashAll {
			// No file IDs (Windows): with several operands, a path already
			// counted under another operand is still counted only once.
			key := e.path
			if abs, err := fsys.Abs(e.path); err == nil {
				key = strings.ToLower(abs)
			}
			if d.seenPath[key] {
				return sums{}, false
			}
			d.seenPath[key] = true
		}
	}
	if e.readErr != nil {
		d.errorf("cannot read directory '%s': %s", e.path, core.ErrText(e.readErr))
	}

	own := sums{size: e.size, inodes: 1, tmax: e.time}
	grand.add(own)
	total := own
	for _, c := range e.children {
		sub, ok := d.walk(c, level+1, grand)
		if !ok {
			continue
		}
		if c.dir && d.SeparateDirs {
			continue
		}
		total.add(sub)
	}

	if (e.dir || d.All || level == 0) && level <= d.MaxDepth {
		v := total.size
		if d.Inodes {
			v = total.inodes
		}
		if d.Threshold < 0 && v <= uint64(-d.Threshold) || d.Threshold >= 0 && v >= uint64(d.Threshold) {
			d.printEntry(total, e.path, e.dir)
		}
	}
	return total, true
}

func (d *Du) errorf(format string, a ...any) {
	_ = d.out.Flush()
	d.diag.Errorf(format, a...)
}

// print writes one line: the size, the time when asked for, and the name.
func (d *Du) print(s sums, name string) {
	d.printEntry(s, name, false)
}

// printEntry writes one line, or holds it for the pretty view.
func (d *Du) printEntry(s sums, name string, dir bool) {
	if d.Pretty && !d.Null {
		d.pending = append(d.pending, line{s: s, name: name, dir: dir})
		return
	}
	v := s.size
	if d.Inodes {
		v = s.inodes
	}
	_, _ = d.out.WriteString(human.Readable(v, d.humanOpts, 1, d.blockSize))
	if d.timeFormat != "" {
		loc := d.Location
		if loc == nil {
			loc = time.Local
		}
		fmt.Fprintf(d.out, "\t%s", date.Format(d.timeFormat, s.tmax.In(loc)))
	}
	if d.quoteNames {
		name = quote.ShellEscape(name)
	}
	_ = d.out.WriteByte('\t')
	_, _ = d.out.WriteString(name)
	if d.Null {
		_ = d.out.WriteByte(0)
	} else {
		_ = d.out.WriteByte('\n')
	}
}

// fnmatch compiles a shell pattern in which '*' and '?' also match '/', as
// fnmatch does without FNM_PATHNAME.
func fnmatch(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString(`^(?s:`)
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			b.WriteString(`.*`)
		case '?':
			b.WriteString(`.`)
		case '\\':
			if i+1 < len(pattern) {
				i++
				b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
			} else {
				b.WriteString(`\\`)
			}
		case '[':
			end := strings.IndexByte(pattern[i+1:], ']')
			if end == 0 { // "[]...]": the ] is literal
				end = strings.IndexByte(pattern[i+2:], ']') + 1
				if end == 0 {
					end = -1
				}
			}
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			class := pattern[i+1 : i+1+end]
			i += end + 1
			b.WriteByte('[')
			if strings.HasPrefix(class, "!") || strings.HasPrefix(class, "^") {
				b.WriteByte('^')
				class = class[1:]
			}
			b.WriteString(strings.NewReplacer(`\`, `\\`, `]`, `\]`).Replace(class))
			b.WriteByte(']')
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString(`)$`)
	return regexp.Compile(b.String())
}

// line is one line of the pretty view.
type line struct {
	s    sums
	name string
	dir  bool
}

// prettyBar is the width of the pretty view's bar.
const prettyBar = 16

// flushPretty writes the held lines: the size right-aligned (on the size
// gradient, in bytes), a bar and a percentage of the largest line (the
// operand, unless -S), the time with --time, then the name with its last
// part highlighted.
func (d *Du) flushPretty() {
	if len(d.pending) == 0 {
		return
	}
	lines := d.pending
	d.pending = nil
	value := func(l line) uint64 {
		if d.Inodes {
			return l.s.inodes
		}
		return l.s.size
	}
	var most uint64
	sizes := make([]string, len(lines))
	width := 0
	for i, l := range lines {
		most = max(most, value(l))
		sizes[i] = human.Readable(value(l), d.humanOpts, 1, d.blockSize)
		width = max(width, len(sizes[i]))
	}
	unicode := visual.UTF8(d.Getenv)
	loc := d.Location
	if loc == nil {
		loc = time.Local
	}
	for i, l := range lines {
		fraction := 1.0
		if most > 0 {
			fraction = float64(value(l)) / float64(most)
		}
		key := visual.Size
		if !d.Inodes {
			key = visual.SizeKey(int64(min(l.s.size, 1<<62)))
		}
		_, _ = d.out.WriteString(strings.Repeat(" ", width-len(sizes[i])))
		_, _ = d.out.WriteString(d.theme.Paint(key, sizes[i]))
		_, _ = d.out.WriteString("  ")
		_, _ = d.out.WriteString(d.theme.Bar(fraction, prettyBar, unicode))
		fmt.Fprintf(d.out, " %3.0f%%  ", fraction*100)
		if d.timeFormat != "" {
			_, _ = d.out.WriteString(d.theme.Paint(visual.Date, date.Format(d.timeFormat, l.s.tmax.In(loc))))
			_, _ = d.out.WriteString("  ")
		}
		name := l.name
		if d.quoteNames {
			name = quote.ShellEscape(name)
		}
		cut := strings.LastIndexAny(name, "/"+string(os.PathSeparator)) + 1
		if cut == len(name) {
			cut = 0
		}
		_, _ = d.out.WriteString(d.theme.Paint(visual.Punctuation, name[:cut]))
		last := visual.Header
		if l.dir {
			last = visual.TypeDir
		}
		_, _ = d.out.WriteString(d.theme.Paint(last, name[cut:]))
		_ = d.out.WriteByte('\n')
	}
}
