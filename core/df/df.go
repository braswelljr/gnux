// Package df implements the engine behind dfx, a GNU-compatible
// implementation of the GNU/UNIX df command: report file system space usage.
package df

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/fsstat"
	"github.com/braswelljr/gnux/core/internal/human"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "dfx"

// Fields are the names --output accepts, in GNU's order.
var Fields = []string{"source", "fstype", "itotal", "iused", "iavail", "ipcent", "size", "used", "avail", "pcent", "file", "target"}

// Options mirrors the GNU df flag surface. The zero value is plain df: every
// real file system, in 1024-byte blocks.
type Options struct {
	// All (-a) includes pseudo, duplicate and inaccessible file systems.
	All bool
	// BlockSize (-B) is the output unit in GNU's syntax ("1K", "M",
	// "human-readable", "si"); empty means DF_BLOCK_SIZE, BLOCK_SIZE,
	// BLOCKSIZE or 1024 bytes (512 under POSIXLY_CORRECT).
	BlockSize string
	// Inodes (-i) reports inodes instead of blocks.
	Inodes bool
	// Local (-l) leaves out network file systems.
	Local bool
	// Portability (-P) uses the POSIX output format.
	Portability bool
	// PrintType (-T) adds the file system type.
	PrintType bool
	// Total (--total) adds a grand total.
	Total bool
	// Types (-t) and ExcludeTypes (-x) select file systems by type.
	Types, ExcludeTypes []string
	// Output (--output) lists the columns to show; with OutputSet and no
	// fields, all of them.
	Output    []string
	OutputSet bool
}

// Df is the engine. Construct it with New.
type Df struct {
	Options

	// Getenv reads the block size variables; it defaults to os.Getenv.
	Getenv func(string) string

	streams core.Streams
	diag    *core.Diag

	mounts func() ([]fsstat.Mount, error)
	statfs func(string) (fsstat.FS, error)
	devOf  func(string) (uint64, error)

	blockSize uint64
	humanOpts human.Opts
	mode      mode
	columns   []*column
	rows      [][]string
	total     totals
	processed bool
}

type mode int

const (
	defaultMode mode = iota
	humanMode
	inodesMode
	posixMode
	outputMode
)

// column is one output column.
type column struct {
	field   string
	caption string
	width   int
	right   bool
}

var fieldInfo = map[string]struct {
	caption string
	width   int
	right   bool
}{
	"source": {"Filesystem", 14, false}, "fstype": {"Type", 4, false},
	"size": {"blocks", 5, true}, "used": {"Used", 5, true}, "avail": {"Available", 5, true},
	"pcent": {"Use%", 4, true}, "itotal": {"Inodes", 5, true}, "iused": {"IUsed", 5, true},
	"iavail": {"IFree", 5, true}, "ipcent": {"IUse%", 4, true}, "target": {"Mounted on", 0, false},
	"file": {"File", 0, false},
}

// New builds a df engine writing to s.
func New(opts Options, s core.Streams) *Df {
	return &Df{
		Options: opts,
		Getenv:  os.Getenv,
		streams: s,
		diag:    core.NewDiag(Name, s),
		mounts:  fsstat.Mounts,
		statfs:  fsstat.Statfs,
		devOf: func(p string) (uint64, error) {
			info, err := os.Stat(p)
			if err != nil {
				return 0, err
			}
			return sysstat.Of(info).Dev, nil
		},
	}
}

// Run reports every mounted file system, or those holding each FILE. It
// returns a *core.UsageError for a command-line mistake, core.ErrFailed
// when anything failed, and nil otherwise.
func (d *Df) Run(args []string) error {
	if err := d.setup(); err != nil {
		return err
	}
	mounts, err := d.mounts()
	if err != nil {
		if len(args) == 0 || d.All || d.Local || len(d.Types) > 0 || len(d.ExcludeTypes) > 0 {
			d.diag.Errorf("cannot read table of mounted file systems: %s", core.ErrText(err))
			return core.ErrFailed
		}
		d.diag.Warnf("Warning: cannot read table of mounted file systems: %s", core.ErrText(err))
	}

	if len(args) > 0 {
		for _, name := range args {
			d.entry(name, mounts)
		}
	} else {
		if !d.All {
			mounts = d.filter(mounts)
		}
		for _, m := range mounts {
			d.row(m, "", "", true)
		}
	}

	if !d.processed {
		if !d.diag.Failed() {
			d.diag.Errorf("no file systems processed")
		}
		return core.ErrFailed
	}
	if d.Total {
		d.totalRow()
	}
	d.print()
	return d.diag.Result()
}

// setup resolves the unit, the columns and the option checks.
func (d *Df) setup() error {
	exclusive := func(a, b string) error {
		return core.Usagef("options %s and %s are mutually exclusive", a, b)
	}
	if d.OutputSet {
		switch {
		case d.Inodes:
			return exclusive("-i", "--output")
		case d.PrintType:
			return exclusive("-T", "--output")
		case d.Portability:
			return exclusive("-P", "--output")
		}
	}
	for _, t := range d.Types {
		if slices.Contains(d.ExcludeTypes, t) {
			d.diag.Errorf("file system type '%s' both selected and excluded", t)
			return core.ErrFailed
		}
	}

	switch {
	case d.BlockSize != "":
		size, opts, err := human.ParseBlockSize(d.BlockSize)
		if err != nil {
			return core.Usagef("invalid --block-size argument '%s'", d.BlockSize)
		}
		d.blockSize, d.humanOpts = size, opts
	case d.Portability:
		d.blockSize = 1024
		if d.Getenv("POSIXLY_CORRECT") != "" {
			d.blockSize = 512
		}
	default:
		d.blockSize, d.humanOpts = human.DefaultBlockSize(d.Getenv, "DF_BLOCK_SIZE")
	}

	switch {
	case d.OutputSet:
		d.mode = outputMode
	case d.Inodes:
		d.mode = inodesMode
	case d.humanOpts&human.Autoscale != 0:
		d.mode = humanMode
	case d.Portability:
		d.mode = posixMode
	}

	add := func(field, caption string) {
		info := fieldInfo[field]
		if caption == "" {
			caption = info.caption
		}
		d.columns = append(d.columns, &column{field: field, caption: caption, width: info.width, right: info.right})
	}
	switch d.mode {
	case outputMode:
		fields := d.Output
		if len(fields) == 0 {
			fields = Fields
		}
		seen := map[string]bool{}
		for _, f := range fields {
			if _, ok := fieldInfo[f]; !ok {
				return core.Usagef("option --output: field '%s' unknown", f)
			}
			if seen[f] {
				return core.Usagef("option --output: field '%s' used more than once", f)
			}
			seen[f] = true
			caption := ""
			switch {
			case f == "size" && d.humanOpts&human.Autoscale != 0:
				caption = "Size"
			case f == "avail" && d.humanOpts&human.Autoscale != 0:
				caption = "Avail"
			}
			add(f, caption)
		}
	default:
		add("source", "")
		if d.PrintType {
			add("fstype", "")
		}
		switch d.mode {
		case inodesMode:
			add("itotal", "")
			add("iused", "")
			add("iavail", "")
			add("ipcent", "")
		case humanMode:
			add("size", "Size")
			add("used", "")
			add("avail", "Avail")
			add("pcent", "")
		case posixMode:
			add("size", "")
			add("used", "")
			add("avail", "")
			add("pcent", "Capacity")
		default:
			add("size", "")
			add("used", "")
			add("avail", "")
			add("pcent", "")
		}
		add("target", "")
	}

	header := make([]string, len(d.columns))
	for i, c := range d.columns {
		header[i] = c.caption
		if c.field == "size" {
			switch {
			case d.mode == posixMode:
				header[i] = fmt.Sprintf("%d-blocks", d.blockSize)
			case d.mode == defaultMode || d.mode == outputMode && d.humanOpts&human.Autoscale == 0:
				header[i] = blocksHeader(d.blockSize, d.humanOpts)
			}
		}
	}
	d.addRow(header)
	return nil
}

// blocksHeader is the "1K-blocks" caption, in the base that renders the
// block size most exactly, as GNU df chooses it.
func blocksHeader(size uint64, outOpts human.Opts) string {
	opts := human.SuppressPointZero | human.Autoscale | human.SI | outOpts&(human.GroupDigits|human.Base1024|human.B)
	q1000, q1024 := size, size
	var by1000, by1024 bool
	for {
		by1000, by1024 = q1000%1000 == 0, q1024%1024 == 0
		q1000 /= 1000
		q1024 /= 1024
		if !by1000 || !by1024 {
			break
		}
	}
	if !by1000 && by1024 {
		opts |= human.Base1024
	}
	if !by1024 && by1000 {
		opts &^= human.Base1024
	}
	if opts&human.Base1024 == 0 {
		opts |= human.B
	}
	return human.Readable(size, opts, 1, 1) + "-blocks"
}

// selected reports whether a file system type passes -t and -x.
func (d *Df) selected(typ string) bool {
	if len(d.Types) > 0 && !slices.Contains(d.Types, typ) {
		return false
	}
	return !slices.Contains(d.ExcludeTypes, typ)
}

// filter drops the mounts df hides without -a, and keeps one entry per
// device, as GNU's filter_mount_list does.
func (d *Df) filter(mounts []fsstat.Mount) []fsstat.Mount {
	var kept []fsstat.Mount
	byDev := map[uint64]int{}
	for _, m := range mounts {
		if m.Dummy || (d.Local && m.Remote) || !d.selected(m.Type) {
			kept = append(kept, m) // left for row to drop, as GNU does
			continue
		}
		dev, err := d.devOf(m.Target)
		if err != nil {
			kept = append(kept, m)
			continue
		}
		i, seen := byDev[dev]
		if !seen {
			byDev[dev] = len(kept)
			kept = append(kept, m)
			continue
		}
		old := kept[i]
		switch {
		case !d.Total && m.Remote && old.Remote && old.Source != m.Source:
			kept = append(kept, m)
		case strings.Contains(m.Source, "/") && !strings.Contains(old.Source, "/"),
			len(old.Target) > len(m.Target),
			old.Source != m.Source && old.Target == m.Target:
			kept[i] = m
		}
	}
	return kept
}

// entry reports the file system holding a FILE operand.
func (d *Df) entry(name string, mounts []fsstat.Mount) {
	abs, err := filepath.Abs(name)
	if err == nil {
		abs, err = filepath.EvalSymlinks(abs)
	}
	if err == nil {
		_, err = os.Stat(abs)
	}
	if err != nil {
		d.diag.Errorf("%s: %s", name, core.ErrText(err))
		return
	}
	dev, _ := d.devOf(abs)
	best, bestLen := -1, -1
	for i, m := range mounts {
		if !within(abs, m.Target) || len(m.Target) <= bestLen {
			continue
		}
		if mdev, err := d.devOf(m.Target); err == nil && mdev != dev {
			continue
		}
		best, bestLen = i, len(m.Target)
	}
	m := fsstat.Mount{Source: "-", Target: abs}
	if best >= 0 {
		m = mounts[best]
	}
	d.row(m, name, abs, false)
}

// rooted reports whether a mount point is a path from a root, so that pseudo
// targets such as "none" are skipped. A path from a root without a volume
// ("/proc") counts too, which filepath.IsAbs denies on Windows.
func rooted(target string) bool {
	return filepath.IsAbs(target) || strings.HasPrefix(target, "/") || strings.HasPrefix(target, `\`)
}

// within reports whether path is target or below it.
func within(path, target string) bool {
	if strings.EqualFold(filepath.VolumeName(path), filepath.VolumeName(target)) && filepath.VolumeName(path) != "" {
		path, target = strings.ToLower(path), strings.ToLower(target)
	}
	if path == target || target == "/" || strings.HasSuffix(target, string(os.PathSeparator)) && strings.HasPrefix(path, target) {
		return true
	}
	return strings.HasPrefix(path, target+string(os.PathSeparator))
}

// totals accumulates --total, in bytes and inodes.
type totals struct {
	size, free, avail float64
	files, ffree      float64
	filesKnown        bool
}

// row adds a file system's line. file is the operand (for the File column)
// and stat the path to examine, the mount point when empty.
func (d *Df) row(m fsstat.Mount, file, stat string, all bool) {
	if d.Local && m.Remote {
		return
	}
	if m.Dummy && !d.All && all {
		return
	}
	if !d.selected(m.Type) {
		return
	}
	if all && !rooted(m.Target) {
		return
	}
	if stat == "" {
		stat = m.Target
	}
	fs, err := d.statfs(stat)
	unknown := false
	if err != nil {
		if all && fsstat.Unavailable(err) {
			if !d.All {
				return
			}
			unknown = true
			m.Type = "-"
		} else {
			d.diag.Errorf("%s: %s", stat, core.ErrText(err))
			return
		}
	}
	if !unknown && fs.Blocks == 0 && !d.All && all {
		return
	}
	d.processed = true
	if file == "" {
		file = "-"
	}
	typ := m.Type
	if typ == "" {
		typ = fs.TypeName
	}
	if typ == "" {
		typ = "-"
	}

	cells := make([]string, len(d.columns))
	for i, c := range d.columns {
		cells[i] = d.cell(c.field, m, typ, file, fs, unknown)
	}
	if !unknown {
		bs := float64(fs.BlockSize)
		d.total.size += float64(fs.Blocks) * bs
		d.total.free += float64(fs.Bfree) * bs
		d.total.avail += float64(fs.Bavail) * bs
		if fs.FilesKnown {
			d.total.files += float64(fs.Files)
			d.total.ffree += float64(fs.Ffree)
			d.total.filesKnown = true
		}
	}
	d.addRow(cells)
}

func (d *Df) cell(field string, m fsstat.Mount, typ, file string, fs fsstat.FS, unknown bool) string {
	blocks := func(n int64) string {
		if unknown {
			return "-"
		}
		return d.readable(n, fs.BlockSize)
	}
	inodes := func(n int64) string {
		if unknown || !fs.FilesKnown {
			return "-"
		}
		return d.readableInodes(n)
	}
	used := int64(fs.Blocks) - int64(fs.Bfree)
	iused := int64(fs.Files) - fs.Ffree
	switch field {
	case "source":
		return clean(m.Source)
	case "fstype":
		return clean(typ)
	case "size":
		return blocks(int64(fs.Blocks))
	case "used":
		return blocks(used)
	case "avail":
		return blocks(fs.Bavail)
	case "pcent":
		if unknown {
			return "-"
		}
		return percent(used, fs.Bavail)
	case "itotal":
		return inodes(int64(fs.Files))
	case "iused":
		return inodes(iused)
	case "iavail":
		return inodes(fs.Ffree)
	case "ipcent":
		if unknown || !fs.FilesKnown {
			return "-"
		}
		return percent(iused, fs.Ffree)
	case "file":
		return clean(file)
	case "target":
		return clean(m.Target)
	}
	return "-"
}

// readable renders n blocks of size bytes in the output unit; a negative
// count (available space below zero on file systems that reserve space)
// keeps its sign.
func (d *Df) readable(n int64, size uint64) string {
	if n < 0 {
		return "-" + human.Readable(uint64(-n), d.humanOpts, size, d.blockSize)
	}
	return human.Readable(uint64(n), d.humanOpts, size, d.blockSize)
}

func (d *Df) readableInodes(n int64) string {
	opts := d.humanOpts
	if n < 0 {
		return "-" + human.Readable(uint64(-n), opts, 1, 1)
	}
	return human.Readable(uint64(n), opts, 1, 1)
}

// percent is used/(used+available), rounded up, as GNU df computes Use%.
func percent(used, avail int64) string {
	total := used + avail
	if total == 0 {
		return "-"
	}
	if used >= 0 && avail >= 0 && used <= math.MaxInt64/100 {
		u100 := used * 100
		return fmt.Sprintf("%d%%", u100/total+btoi(u100%total != 0))
	}
	p := float64(used) * 100 / float64(total)
	return fmt.Sprintf("%.0f%%", math.Ceil(p))
}

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// totalRow adds the --total line.
func (d *Df) totalRow() {
	src := "total"
	if d.mode == outputMode && slices.ContainsFunc(d.columns, func(c *column) bool { return c.field == "source" }) {
		src = "-"
	}
	t := d.total
	fs := fsstat.FS{
		BlockSize: 1, Blocks: uint64(t.size), Bfree: uint64(t.free), Bavail: int64(t.avail),
		Files: uint64(t.files), Ffree: int64(t.ffree), FilesKnown: t.filesKnown,
	}
	cells := make([]string, len(d.columns))
	for i, c := range d.columns {
		switch c.field {
		case "source":
			cells[i] = "total"
		case "target":
			cells[i] = src
		case "fstype", "file":
			cells[i] = "-"
		default:
			cells[i] = d.cell(c.field, fsstat.Mount{}, "-", "-", fs, false)
		}
	}
	d.addRow(cells)
}

func (d *Df) addRow(cells []string) {
	for i, c := range cells {
		d.columns[i].width = max(d.columns[i].width, utf8.RuneCountInString(c))
	}
	d.rows = append(d.rows, cells)
}

// clean replaces control characters, which would break the table, with '?'.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) && r != ' ' {
			return '?'
		}
		return r
	}, s)
}

// print writes the table: text columns left-aligned, numbers right-aligned.
func (d *Df) print() {
	w := core.NewWriter(d.streams.Out)
	defer func() {
		if err := w.Flush(); err != nil && !core.IsClosedPipe(err) {
			d.diag.Errorf("write error: %s", core.ErrText(err))
		}
	}()
	for _, row := range d.rows {
		for i, cell := range row {
			c := d.columns[i]
			if i > 0 {
				_ = w.WriteByte(' ')
			}
			fill := c.width - utf8.RuneCountInString(cell)
			if c.right {
				_, _ = w.WriteString(strings.Repeat(" ", max(fill, 0)))
			}
			_, _ = w.WriteString(cell)
			if !c.right && i+1 < len(row) {
				_, _ = w.WriteString(strings.Repeat(" ", max(fill, 0)))
			}
		}
		_ = w.WriteByte('\n')
	}
}
