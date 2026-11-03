package ls

import (
	"slices"
	"strconv"
	"strings"
)

// minColumnWidth is the narrowest a column can be: one character plus the
// two-space gutter.
const minColumnWidth = 3

// frills holds the widths of the optional -i and -s prefixes for one listing.
type frills struct {
	inode, blocks int
}

// newFrills measures the -i and -s columns. With pad false (the comma format)
// the numbers are not aligned.
func (l *Ls) newFrills(entries []*entry, pad bool) frills {
	var f frills
	if !pad {
		return f
	}
	for _, e := range slices.Concat(entries, l.measureAlso) {
		if l.Inode {
			f.inode = max(f.inode, len(inodeString(e)))
		}
		if l.ShowBlocks {
			f.blocks = max(f.blocks, len(l.blocksOf(e)))
		}
	}
	if l.Header && l.format == FormatLong {
		f.inode, f.blocks = max(f.inode, len("Inode")), max(f.blocks, len("Blocks"))
	}
	return f
}

func inodeString(e *entry) string {
	if e.info == nil {
		return "?"
	}
	return strconv.FormatUint(e.sys.ino, 10)
}

// blocksOf returns the -s column for an entry.
func (l *Ls) blocksOf(e *entry) string {
	if e.info == nil {
		return "?"
	}
	return l.blockString(e.sys.blocks)
}

// frillsOf returns the "inode blocks " prefix of an entry.
func (l *Ls) frillsOf(e *entry, f frills) string {
	var b strings.Builder
	if l.Inode {
		b.WriteString(padLeft(inodeString(e), f.inode))
		b.WriteString(" ")
	}
	if l.ShowBlocks {
		b.WriteString(padLeft(l.blocksOf(e), f.blocks))
		b.WriteString(" ")
	}
	return b.String()
}

// cellWidth is the number of columns writeCell will occupy for e.
func (l *Ls) cellWidth(e *entry, f frills) int {
	return len(l.frillsOf(e, f)) + l.leadWidth(e) + l.pad(e) + e.width + len(l.indicator(e.mode))
}

// writeCell writes an entry as it appears in every format but the long one:
// optional inode and block count, the (colored) name, and the type indicator.
func (l *Ls) writeCell(e *entry, f frills) {
	l.put(l.frillsOf(e, f))
	l.putName(e)
	l.put(l.indicator(e.mode))
}

// pad is 1 when e takes a leading space to line up with quoted names.
func (l *Ls) pad(e *entry) int {
	if l.someQuoted && !e.quoted {
		return 1
	}
	return 0
}

// putName writes the quoted name of e, padded to line up with quoted names,
// colored when color is on and linked with --hyperlink; after its tree
// branch and icon, with --tree and --icons.
func (l *Ls) putName(e *entry) {
	l.putLead(e)
	if l.pad(e) == 1 {
		l.put(" ")
	}
	code := ""
	if l.palette != nil {
		code = l.palette.entryColor(e)
	}
	l.putMarked(e.shown, e.quoted, code, e.path, &l.diredPos)
}

// putHeader writes the name of a directory before its listing.
func (l *Ls) putHeader(dir string) {
	shown := l.quoteDir(dir)
	l.putMarked(shown, false, "", dir, &l.subdired)
}

// putMarked writes shown, the quoted form of the file at path: in color
// code, linked to the file with --hyperlink, and with its start and end
// recorded in positions for --dired. A quoted name lined up with unquoted
// ones keeps its quotes outside the link, as in GNU.
func (l *Ls) putMarked(shown string, quoted bool, code, path string, positions *[]int64) {
	if code != "" {
		if !l.usedColor {
			l.usedColor = true
			l.putRaw("\x1b[0m")
		}
		l.putRaw("\x1b[" + code + "m")
	}
	inner, open, shut := shown, "", ""
	if l.hyperlink && l.someQuoted && quoted && len(shown) >= 2 {
		open, inner, shut = shown[:1], shown[1:len(shown)-1], shown[len(shown)-1:]
	}
	if l.dired {
		*positions = append(*positions, l.pos)
	}
	if l.hyperlink {
		l.put(open)
		l.putRaw("\x1b]8;;" + fileURL(l.host, path) + "\a")
		l.put(inner)
		l.putRaw("\x1b]8;;\a")
		l.put(shut)
	} else {
		l.put(shown)
	}
	if l.dired {
		*positions = append(*positions, l.pos)
	}
	if code != "" {
		l.putRaw("\x1b[0m")
	}
}

// columnLayout is the result of fitting entries into a line.
type columnLayout struct {
	cols   int
	widths []int // per column, gutter included (except the last)
}

// fitColumns finds the largest number of columns that fits the line width,
// using GNU ls's algorithm: every candidate column count is tracked at once,
// and a candidate is dropped as soon as its total width reaches the line width.
func (l *Ls) fitColumns(lengths []int, byColumns bool) columnLayout {
	n := len(lengths)
	maxCols := n
	if l.width > 0 {
		maxCols = min(n, max(1, l.width/minColumnWidth))
	}

	type candidate struct {
		valid  bool
		total  int
		widths []int
	}
	cands := make([]candidate, maxCols)
	for i := range cands {
		widths := make([]int, i+1)
		for j := range widths {
			widths[j] = minColumnWidth
		}
		cands[i] = candidate{valid: true, total: (i + 1) * minColumnWidth, widths: widths}
	}

	for idx, length := range lengths {
		for i := range cands {
			c := &cands[i]
			if !c.valid {
				continue
			}
			cols := i + 1
			var col int
			if byColumns {
				rows := (n + cols - 1) / cols
				col = idx / rows
			} else {
				col = idx % cols
			}
			need := length
			if col != cols-1 {
				need += 2
			}
			if c.widths[col] < need {
				c.total += need - c.widths[col]
				c.widths[col] = need
				c.valid = l.width == 0 || c.total < l.width
			}
		}
	}

	// One column always "fits", however long the names are.
	for i := maxCols - 1; i > 0; i-- {
		if cands[i].valid {
			return columnLayout{cols: i + 1, widths: cands[i].widths}
		}
	}
	return columnLayout{cols: 1, widths: cands[0].widths}
}

// printColumns writes entries in columns: sorted down each column (-C) when
// byColumns is set, across each row (-x) otherwise.
func (l *Ls) printColumns(entries []*entry, byColumns bool) {
	f := l.newFrills(entries, true)
	lengths := make([]int, len(entries))
	for i, e := range entries {
		lengths[i] = l.cellWidth(e, f)
	}
	layout := l.fitColumns(lengths, byColumns)
	n := len(entries)

	if !byColumns {
		pos := 0
		for i, e := range entries {
			col := i % layout.cols
			switch {
			case i == 0:
			case col == 0:
				l.put(l.eol)
				pos = 0
			default:
				l.indent(pos+lengths[i-1], pos+layout.widths[col-1])
				pos += layout.widths[col-1]
			}
			l.writeCell(e, f)
		}
		l.put(l.eol)
		return
	}

	rows := (n + layout.cols - 1) / layout.cols
	for row := range rows {
		pos := 0
		for col, i := 0, row; ; col, i = col+1, i+rows {
			l.writeCell(entries[i], f)
			if i+rows >= n {
				break
			}
			l.indent(pos+lengths[i], pos+layout.widths[col])
			pos += layout.widths[col]
		}
		l.put(l.eol)
	}
}

// indent advances the cursor from column from to column to, using tabs where a
// tab stop lies on the way (as GNU ls does) and spaces otherwise.
func (l *Ls) indent(from, to int) {
	for from < to {
		if l.TabSize > 0 && to/l.TabSize > (from+1)/l.TabSize {
			l.put("\t")
			from += l.TabSize - from%l.TabSize
		} else {
			l.put(" ")
			from++
		}
	}
}

// printCommas writes entries as a comma-separated stream (-m), breaking the
// line before an entry that would not fit.
func (l *Ls) printCommas(entries []*entry) {
	f := l.newFrills(entries, false)
	pos := 0
	for i, e := range entries {
		length := l.cellWidth(e, f)
		if i > 0 {
			if l.width == 0 || pos+length+2 < l.width {
				l.put(", ")
				pos += 2
			} else {
				l.put("," + l.eol)
				pos = 0
			}
		}
		l.writeCell(e, f)
		pos += length
	}
	l.put(l.eol)
}

func padLeft(s string, width int) string {
	if n := width - displayWidth(s); n > 0 {
		return strings.Repeat(" ", n) + s
	}
	return s
}

func padRight(s string, width int) string {
	if n := width - displayWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}
