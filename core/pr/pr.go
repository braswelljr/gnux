// Package pr implements the engine behind prx, a GNU-compatible
// implementation of the GNU/UNIX pr command: paginate or columnate files for
// printing. The page layout is a port of GNU pr's, so that page breaks,
// column balancing and padding fall where GNU puts them.
package pr

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/date"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "prx"

// Options are the settings of a pr run, once the command line has been
// resolved (prx works out what -s and -w mean next to -S, -W and the column
// options, as GNU does). The zero value is plain pr: 66-line pages of 72
// columns with a five-line header and footer.
type Options struct {
	// Columns (-COLUMN, --columns) is the number of columns; 0 means 1.
	Columns int
	// Across (-a) fills the columns across rather than down.
	Across bool
	// ShowControlChars (-c) writes control characters as ^G and others as
	// octal; ShowNonprinting (-v) writes them all as octal escapes.
	ShowControlChars, ShowNonprinting bool
	// DoubleSpace (-d) puts a blank line after each line.
	DoubleSpace bool
	// DateFormat (-D) formats the header's date; empty means %Y-%m-%d %H:%M.
	DateFormat string
	// ExpandTabs (-e) expands InputTab characters (0 means tab) to tab stops
	// every InputTabWidth columns (0 means 8).
	ExpandTabs    bool
	InputTab      byte
	InputTabWidth int
	// FormFeed (-f, -F) ends each page with a form feed, not blank lines.
	FormFeed bool
	// Header (-h), when HeaderSet, replaces the file name in the header.
	Header    string
	HeaderSet bool
	// OutputTabs (-i) replaces spaces with OutputTab characters (0 means
	// tab) at tab stops every OutputTabWidth columns (0 means 8).
	OutputTabs     bool
	OutputTab      byte
	OutputTabWidth int
	// JoinLines (-J) prints whole lines, never truncating them.
	JoinLines bool
	// Length (-l) is the page length in lines; 0 means 66.
	Length int
	// Merge (-m) prints the files side by side, one per column.
	Merge bool
	// NumberLines (-n) numbers lines, NumberWidth digits wide (0 means 5)
	// followed by NumberSeparator (0 means tab).
	NumberLines     bool
	NumberSeparator byte
	NumberWidth     int
	// FirstLineNumber (-N), when FirstLineNumberSet, is the number of the
	// first line printed.
	FirstLineNumber    int
	FirstLineNumberSet bool
	// Indent (-o) is the left margin.
	Indent int
	// NoFileWarnings (-r) is silent about files that cannot be opened.
	NoFileWarnings bool
	// Separator, when UseSeparator, separates the columns (-S, or -s as
	// resolved); otherwise columns are aligned and separated by a space.
	Separator    string
	UseSeparator bool
	// OmitHeader (-t) leaves out headers and footers, keeping form feeds in
	// the input; OmitPagination (-T) also drops them.
	OmitHeader, OmitPagination bool
	// Width (-w, -W) is the page width; 0 means 72. Truncate cuts lines
	// longer than their column.
	Width    int
	Truncate bool
	// FirstPage and LastPage (+FIRST[:LAST], --pages) are the pages to
	// print; 0 means the first and the last.
	FirstPage, LastPage uint64
	// Balance (-b) balances the columns of the last page; it is always the
	// case when printing down columns.
	Balance bool
}

type status int

const (
	open status = iota
	ffFound
	onHold
	closed
)

// column is GNU's struct COLUMN.
type column struct {
	in              *bufio.Reader
	file            *os.File // nil for standard input
	name            string
	status          status
	store           bool // char_func is store_char, print_func print_stored
	currentLine     int
	linesStored     int
	linesToPrint    int
	startPosition   int
	numbered        bool
	fullPagePrinted bool
}

// Pr is the engine. Construct it with New.
type Pr struct {
	Options

	// Now is the date of headers for standard input and parallel files.
	// Location is the zone dates are shown in; nil means the local zone.
	Now      func() time.Time
	Location *time.Location

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	cols []column

	buff        []byte
	lineVector  []int
	endVector   []int
	storing     bool
	balance     bool
	extremities bool
	keepFF      bool
	printAFF    bool
	printHeader bool

	linesPerPage, linesPerBody int
	charsPerLine, charsPerCol  int
	truncate, join, useSep     bool
	sep                        string
	untabify, tabify           bool
	inTab                      byte
	inTabWidth                 int
	outTab                     byte
	outTabWidth                int
	spacesNotPrinted           int
	outputPosition             int
	inputPosition              int
	columns                    int
	explicitColumns            bool
	parallel                   bool
	firstPage, lastPage        uint64
	filesReady                 int
	pageNumber                 uint64
	lineNumber                 int
	lineCount                  int
	numSep                     byte
	numWidthChars              int
	numberWidth                int
	separatorsNotPrinted       int
	paddingNotPrinted          int
	padVertically              bool
	alignEmptyCols, emptyLine  bool
	ffOnly                     bool
	lastLine                   bool
	dateText, fileText         string
	headerWidth                int
	clump                      []byte
	failedOpens                bool
	now                        time.Time
}

// New builds a pr engine reading from and writing to s.
func New(opts Options, s core.Streams) *Pr {
	return &Pr{Options: opts, Now: time.Now, streams: s, diag: core.NewDiag(Name, s)}
}

// abort is what fail panics with: GNU pr exits on the spot for these
// errors, wherever it is in a page, and Run recovers it.
type abort struct{}

// fail reports an error GNU pr exits on, and stops the run.
func (p *Pr) fail(format string, a ...any) {
	p.diag.Errorf(format, a...)
	panic(abort{})
}

func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

// Run paginates each FILE in turn (side by side with Merge); none, or "-",
// is standard input.
func (p *Pr) Run(args []string) (err error) {
	p.out = core.NewWriter(p.streams.Out)
	defer core.FlushOutput(p.out, p.diag, &err)
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(abort); !ok {
				panic(r)
			}
			p.closeAll()
			err = core.ErrFailed
		}
	}()

	p.columns = max(p.Columns, 1)
	p.explicitColumns = p.Columns > 0
	p.parallel = p.Merge
	p.storing = !p.Across && !p.Merge
	p.balance = p.Balance
	p.extremities = !p.OmitHeader && !p.OmitPagination
	p.keepFF = p.OmitHeader
	p.linesPerPage = p.Length
	if p.linesPerPage == 0 {
		p.linesPerPage = 66
	}
	p.charsPerLine = p.Width
	if p.charsPerLine == 0 {
		p.charsPerLine = 72
	}
	p.truncate, p.join, p.useSep, p.sep = p.Truncate, p.JoinLines, p.UseSeparator, p.Separator
	p.untabify, p.tabify = p.ExpandTabs, p.OutputTabs
	p.inTab, p.inTabWidth = orByte(p.InputTab, '\t'), orInt(p.InputTabWidth, 8)
	p.outTab, p.outTabWidth = orByte(p.OutputTab, '\t'), orInt(p.OutputTabWidth, 8)
	p.numSep, p.numWidthChars = orByte(p.NumberSeparator, '\t'), orInt(p.NumberWidth, 5)
	p.lineCount = 1
	p.firstPage, p.lastPage = p.FirstPage, p.LastPage
	if p.firstPage == 0 {
		p.firstPage = 1
	}
	if p.lastPage == 0 {
		p.lastPage = math.MaxUint64
	}
	if p.DateFormat == "" {
		p.DateFormat = "%Y-%m-%d %H:%M"
	}
	if p.parallel && p.explicitColumns {
		p.fail("cannot specify number of columns when printing in parallel")
	}
	if p.parallel && p.Across {
		p.fail("cannot specify both printing across and printing in parallel")
	}
	p.now = p.Now()

	switch {
	case len(args) == 0:
		p.printFiles(nil)
	case p.parallel:
		p.printFiles(args)
	default:
		for i := range args {
			p.printFiles(args[i : i+1])
		}
	}
	if p.failedOpens {
		return core.ErrFailed
	}
	return nil
}

func orByte(b, def byte) byte {
	if b == 0 {
		return def
	}
	return b
}

func orInt(i, def int) int {
	if i == 0 {
		return def
	}
	return i
}

func tabWidth(c, h int) int    { return c - h%c }
func posAfterTab(c, h int) int { return h + tabWidth(c, h) }

func (p *Pr) initParameters(files int) {
	charsUsedByNumber := 0
	p.linesPerBody = p.linesPerPage - 5 - 5
	if p.linesPerBody <= 0 {
		p.extremities = false
		p.keepFF = true
	}
	if !p.extremities {
		p.linesPerBody = p.linesPerPage
	}
	if p.DoubleSpace {
		p.linesPerBody = max(1, p.linesPerBody/2)
	}
	if files == 0 {
		p.parallel = false
	}
	if p.parallel {
		p.columns = files
	}
	if p.storing {
		p.balance = true
	}
	if p.columns > 1 {
		if !p.useSep {
			if p.join {
				p.sep = "\t"
			} else {
				p.sep = " "
			}
			p.useSep = true
		} else if !p.join && p.sep == "\t" {
			p.sep = " "
		}
		p.truncate = true
		if p.sep != "\t" {
			p.untabify = true
		}
		p.tabify = true
	} else {
		p.storing = false
	}
	if p.join {
		p.truncate = false
	}
	if p.NumberLines {
		p.lineCount = 1
		if p.FirstLineNumberSet {
			p.lineCount = p.FirstLineNumber
		}
		if p.numSep == '\t' {
			p.numberWidth = p.numWidthChars + tabWidth(8, p.numWidthChars)
		} else {
			p.numberWidth = p.numWidthChars + 1
		}
		if p.parallel {
			charsUsedByNumber = p.numberWidth
		}
	}
	sepChars := (p.columns - 1) * len(p.sep)
	useful := p.charsPerLine - charsUsedByNumber - sepChars
	p.charsPerCol = max(useful, 0) / p.columns
	if p.charsPerCol < 1 {
		p.fail("page width too narrow")
	}
	p.clump = make([]byte, 0, max(8, p.inTabWidth))
}

func (p *Pr) openFile(name string, c *column) bool {
	if name == "-" {
		c.name = "standard input"
		c.in = core.NewReader(p.streams.In)
		c.file = nil
	} else {
		// A directory opens, as with fopen; reading it then fails.
		f, err := fsys.Open(name)
		if err != nil {
			p.failedOpens = true
			if !p.NoFileWarnings {
				p.diag.Errorf("%s: %s", quotef(name), core.ErrText(err))
			}
			return false
		}
		c.name, c.file, c.in = name, f, core.NewReader(f)
	}
	c.status = open
	c.fullPagePrinted = false
	return true
}

func (p *Pr) initFiles(names []string) bool {
	p.cols = make([]column, p.columns)
	total := 0
	if p.parallel {
		var cols []column
		for _, name := range names {
			var c column
			if p.openFile(name, &c) {
				cols = append(cols, c)
				total++
			}
		}
		p.cols = cols
		p.columns = len(cols)
		if p.columns == 0 {
			return false
		}
		p.initHeader("", nil)
	} else {
		c := &p.cols[0]
		if len(names) > 0 {
			if !p.openFile(names[0], c) {
				return false
			}
			p.initHeader(names[0], c.file)
		} else {
			c.name, c.in, c.status = "standard input", core.NewReader(p.streams.In), open
			p.initHeader("", nil)
		}
		total = 1
		for i := 1; i < p.columns; i++ {
			p.cols[i] = column{in: c.in, file: c.file, name: c.name, status: open}
		}
	}
	p.filesReady = total
	return true
}

func (p *Pr) initHeader(name string, f *os.File) {
	t := p.now
	if name != "-" && f != nil {
		if info, err := f.Stat(); err == nil {
			t = info.ModTime()
		}
	}
	loc := p.Location
	if loc == nil {
		loc = time.Local
	}
	p.dateText = date.Format(p.DateFormat, t.In(loc))
	switch {
	case p.HeaderSet:
		p.fileText = p.Header
	case f == nil:
		p.fileText = ""
	default:
		p.fileText = name
	}
	p.headerWidth = p.charsPerLine - utf8.RuneCountInString(p.dateText) - utf8.RuneCountInString(p.fileText)
}

func (p *Pr) printFiles(names []string) {
	p.initParameters(len(names))
	if !p.initFiles(names) {
		return
	}
	defer p.closeAll()
	if p.storing {
		p.buff = p.buff[:0]
		p.lineVector = make([]int, p.linesPerBody*p.columns+1)
		p.endVector = make([]int, p.linesPerBody*p.columns)
	}
	if p.firstPage > 1 {
		if !p.skipToPage(p.firstPage) {
			return
		}
		p.pageNumber = p.firstPage
	} else {
		p.pageNumber = 1
	}
	p.initFuncs()
	p.lineNumber = p.lineCount
	for p.printPage() {
	}
}

func (p *Pr) closeAll() {
	for i := range p.cols {
		if f := p.cols[i].file; f != nil {
			f.Close()
			p.cols[i].file = nil
		}
	}
}

func (p *Pr) initFuncs() {
	h := p.Indent
	hNext := 0
	if p.truncate {
		if p.parallel && p.NumberLines {
			hNext = h + p.charsPerCol + p.numberWidth
		} else {
			hNext = h + p.charsPerCol
		}
	}
	h += len(p.sep)
	i := 1
	for ; i < p.columns; i++ {
		c := &p.cols[i-1]
		c.store = p.storing
		c.numbered = p.NumberLines && (!p.parallel || i == 1)
		c.startPosition = h
		if !p.truncate {
			h, hNext = 0, 0
		} else {
			h = hNext + len(p.sep)
			hNext = h + p.charsPerCol
		}
	}
	c := &p.cols[i-1]
	c.store = p.storing && p.balance
	c.numbered = p.NumberLines && (!p.parallel || i == 1)
	c.startPosition = h
}

func (p *Pr) colsReady() int {
	n := 0
	for i := range p.cols {
		c := &p.cols[i]
		if c.status == open || c.status == ffFound || (p.storing && c.linesStored > 0 && c.linesToPrint > 0) {
			n++
		}
	}
	return n
}

func (p *Pr) initPage() {
	if p.storing {
		p.storeColumns()
		for i := 0; i < p.columns-1; i++ {
			p.cols[i].linesToPrint = p.cols[i].linesStored
		}
		last := &p.cols[p.columns-1]
		switch {
		case p.balance:
			last.linesToPrint = last.linesStored
		case last.status == open:
			last.linesToPrint = p.linesPerBody
		default:
			last.linesToPrint = 0
		}
		return
	}
	for i := range p.cols {
		if p.cols[i].status == open {
			p.cols[i].linesToPrint = p.linesPerBody
		} else {
			p.cols[i].linesToPrint = 0
		}
	}
}

func (p *Pr) printPage() bool {
	p.initPage()
	if p.colsReady() == 0 {
		return false
	}
	if p.extremities {
		p.printHeader = true
	}
	p.padVertically = false
	pv := false
	left := p.linesPerBody
	if p.DoubleSpace {
		left *= 2
	}
	for left > 0 && p.colsReady() > 0 {
		p.outputPosition, p.spacesNotPrinted, p.separatorsNotPrinted = 0, 0, 0
		p.padVertically, p.alignEmptyCols, p.emptyLine = false, false, true
		for j := range p.cols {
			c := &p.cols[j]
			p.inputPosition = 0
			if c.linesToPrint > 0 || c.status == ffFound {
				p.ffOnly = false
				p.paddingNotPrinted = c.startPosition
				var ok bool
				if c.store {
					ok = p.printStored(c)
				} else {
					ok = p.readLine(c)
				}
				if !ok {
					p.readRestOfLine(c)
				}
				pv = pv || p.padVertically
				c.linesToPrint--
				if c.linesToPrint <= 0 && p.colsReady() == 0 {
					break
				}
				if p.parallel && c.status != open {
					if p.emptyLine {
						p.alignEmptyCols = true
					} else if c.status == closed || (c.status == onHold && p.ffOnly) {
						p.alignColumn(c)
					}
				}
			} else if p.parallel {
				if p.emptyLine {
					p.alignEmptyCols = true
				} else {
					p.alignColumn(c)
				}
			}
			if p.useSep {
				p.separatorsNotPrinted++
			}
		}
		if p.padVertically {
			_ = p.out.WriteByte('\n')
			left--
		}
		if p.colsReady() == 0 && !p.extremities {
			break
		}
		if p.DoubleSpace && pv {
			_ = p.out.WriteByte('\n')
			left--
		}
	}
	if left == 0 {
		for i := range p.cols {
			if p.cols[i].status == open {
				p.cols[i].fullPagePrinted = true
			}
		}
	}
	p.padVertically = pv
	if p.padVertically && p.extremities {
		p.padDown(left + 5)
	} else if p.keepFF && p.printAFF {
		_ = p.out.WriteByte('\f')
		p.printAFF = false
	}
	p.pageNumber++
	if p.lastPage < p.pageNumber {
		return false
	}
	p.resetStatus()
	return true
}

func (p *Pr) storeColumns() {
	line := 0
	buffStart := 0
	p.buff = p.buff[:0]
	lastCol := p.columns - 1
	if p.balance {
		lastCol = p.columns
	}
	for i := 0; i < lastCol; i++ {
		p.cols[i].linesStored = 0
	}
	for i := 0; i < lastCol && p.filesReady > 0; i++ {
		c := &p.cols[i]
		c.currentLine = line
		for j := p.linesPerBody; j > 0 && p.filesReady > 0; j-- {
			if c.status != open {
				continue
			}
			p.inputPosition = 0
			if !p.readLine(c) {
				p.readRestOfLine(c)
			}
			if c.status == open || buffStart != len(p.buff) {
				c.linesStored++
				p.lineVector[line] = buffStart
				p.endVector[line] = p.inputPosition
				line++
				buffStart = len(p.buff)
			}
		}
	}
	p.lineVector[line] = buffStart
	if p.balance {
		first := 0
		for i := range p.cols {
			lines := line / p.columns
			if i+1 <= line%p.columns {
				lines++
			}
			p.cols[i].linesStored = lines
			p.cols[i].currentLine = first
			first += lines
		}
	}
}

// emit writes a byte for column c: into the store, or out.
func (p *Pr) emit(c *column, b byte) {
	if c.store {
		p.buff = append(p.buff, b)
		return
	}
	p.printChar(b)
}

func (p *Pr) printChar(b byte) {
	if p.tabify {
		if b == ' ' {
			p.spacesNotPrinted++
			return
		}
		if p.spacesNotPrinted > 0 {
			p.printWhiteSpace()
		}
		switch {
		case b >= 0x20 && b < 0x7f:
			p.outputPosition++
		case b == '\b':
			p.outputPosition--
		}
	}
	_ = p.out.WriteByte(b)
}

func (p *Pr) addLineNumber(c *column) {
	text := strconv.Itoa(p.lineNumber)
	if len(text) < p.numWidthChars {
		text = fmt.Sprintf("%*s", p.numWidthChars, text)
	}
	p.lineNumber++
	// The low digits are the informative ones: cut the high ones.
	text = text[len(text)-p.numWidthChars:]
	for i := 0; i < len(text); i++ {
		p.emit(c, text[i])
	}
	if p.columns > 1 {
		if p.numSep == '\t' {
			for i := p.numberWidth - p.numWidthChars; i > 0; i-- {
				p.emit(c, ' ')
			}
		} else {
			p.emit(c, p.numSep)
		}
	} else {
		p.emit(c, p.numSep)
		if p.numSep == '\t' {
			p.outputPosition = posAfterTab(p.outTabWidth, p.outputPosition)
		}
	}
	if p.truncate && !p.parallel {
		p.inputPosition += p.numberWidth
	}
}

func (p *Pr) padAcrossTo(position int) {
	if p.tabify {
		p.spacesNotPrinted = position - p.outputPosition
		return
	}
	for h := p.outputPosition + 1; h <= position; h++ {
		_ = p.out.WriteByte(' ')
	}
	p.outputPosition = position
}

func (p *Pr) padDown(lines int) {
	if p.FormFeed {
		_ = p.out.WriteByte('\f')
		return
	}
	for ; lines > 0; lines-- {
		_ = p.out.WriteByte('\n')
	}
}

func (p *Pr) printWhiteSpace() {
	h := p.outputPosition
	goal := h + p.spacesNotPrinted
	for goal-h > 1 {
		next := posAfterTab(p.outTabWidth, h)
		if next > goal {
			break
		}
		_ = p.out.WriteByte(p.outTab)
		h = next
	}
	for ; h < goal; h++ {
		_ = p.out.WriteByte(' ')
	}
	p.outputPosition = goal
	p.spacesNotPrinted = 0
}

func (p *Pr) printSepString() {
	if p.separatorsNotPrinted <= 0 {
		if p.spacesNotPrinted > 0 {
			p.printWhiteSpace()
		}
		return
	}
	// GNU counts the separator's length down once across all the pending
	// separators, so only the first is printed in full.
	l := len(p.sep)
	s := 0
	for ; p.separatorsNotPrinted > 0; p.separatorsNotPrinted-- {
		for ; l > 0; l-- {
			if p.sep[s] == ' ' {
				s++
				p.spacesNotPrinted++
			} else {
				if p.spacesNotPrinted > 0 {
					p.printWhiteSpace()
				}
				_ = p.out.WriteByte(p.sep[s])
				s++
				p.outputPosition++
			}
		}
		if p.spacesNotPrinted > 0 {
			p.printWhiteSpace()
		}
	}
}

func (p *Pr) alignColumn(c *column) {
	p.paddingNotPrinted = c.startPosition
	if len(p.sep) < p.paddingNotPrinted {
		p.padAcrossTo(p.paddingNotPrinted - len(p.sep))
		p.paddingNotPrinted = 0
	}
	if p.useSep {
		p.printSepString()
	}
	if c.numbered {
		p.addLineNumber(c)
	}
}

func (p *Pr) closeFile(c *column) {
	if c.status == closed {
		return
	}
	if c.file != nil {
		err := c.file.Close()
		for i := range p.cols {
			if p.cols[i].file == c.file {
				p.cols[i].file = nil
			}
		}
		if err != nil {
			p.fail("%s: %s", quotef(c.name), core.ErrText(err))
		}
	}
	if !p.parallel {
		for i := range p.cols {
			p.cols[i].status = closed
			if p.cols[i].linesStored == 0 {
				p.cols[i].linesToPrint = 0
			}
		}
	} else {
		c.status = closed
		c.linesToPrint = 0
	}
	p.filesReady--
}

func (p *Pr) holdFile(c *column) {
	if !p.parallel {
		for i := range p.cols {
			if p.storing {
				p.cols[i].status = ffFound
			} else {
				p.cols[i].status = onHold
			}
		}
	} else {
		c.status = onHold
	}
	c.linesToPrint = 0
	p.filesReady--
}

func (p *Pr) resetStatus() {
	for i := range p.cols {
		if p.cols[i].status == onHold {
			p.cols[i].status = open
			p.filesReady++
		}
	}
	if p.storing {
		if p.cols[0].status == closed {
			p.filesReady = 0
		} else {
			p.filesReady = 1
		}
	}
}

// getc reads a byte, or -1 at the end of the input. A read error ends the
// run, as it does in GNU's close_file.
func (p *Pr) getc(c *column) int {
	b, err := c.in.ReadByte()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			p.fail("%s: %s", quotef(c.name), core.ErrText(err))
		}
		return -1
	}
	return int(b)
}

func (p *Pr) ungetc(c *column, ch int) {
	if ch >= 0 {
		_ = c.in.UnreadByte()
	}
}

func (p *Pr) skipToPage(page uint64) bool {
	for n := uint64(1); n < page; n++ {
		for i := 1; i < p.linesPerBody; i++ {
			for j := range p.cols {
				if p.cols[j].status == open {
					p.skipRead(&p.cols[j], j+1)
				}
			}
		}
		p.lastLine = true
		for j := range p.cols {
			if p.cols[j].status == open {
				p.skipRead(&p.cols[j], j+1)
			}
		}
		if p.storing {
			for j := range p.cols {
				if p.cols[j].status != closed {
					p.cols[j].status = onHold
				}
			}
		}
		p.resetStatus()
		p.lastLine = false
		if p.filesReady < 1 {
			p.diag.Warnf("starting page number %d exceeds page count %d", page, n)
			break
		}
	}
	return p.filesReady > 0
}

func (p *Pr) skipRead(c *column, columnNumber int) {
	ch := p.getc(c)
	if ch == '\f' && c.fullPagePrinted {
		if ch = p.getc(c); ch == '\n' {
			ch = p.getc(c)
		}
	}
	c.fullPagePrinted = false
	singleFF := ch == '\f'
	if p.lastLine {
		c.fullPagePrinted = true
	}
	for ch != '\n' {
		if ch == '\f' {
			if p.lastLine {
				if !p.parallel {
					for i := range p.cols {
						p.cols[i].fullPagePrinted = false
					}
				} else {
					c.fullPagePrinted = false
				}
			}
			if next := p.getc(c); next != '\n' {
				p.ungetc(c, next)
			}
			p.holdFile(c)
			break
		}
		if ch < 0 {
			p.closeFile(c)
			break
		}
		ch = p.getc(c)
	}
	if !p.FirstLineNumberSet && (!p.parallel || columnNumber == 1) && !singleFF {
		p.lineCount++
	}
}

func (p *Pr) printHeaderLines() {
	p.outputPosition = 0
	p.padAcrossTo(p.Indent)
	p.printWhiteSpace()
	pageText := "Page " + strconv.FormatUint(p.pageNumber, 10)
	avail := max(0, p.headerWidth-len(pageText))
	lhs := avail >> 1
	rhs := avail - lhs
	pad := func(n int) string {
		if n < 1 {
			n = 1
		}
		return fmt.Sprintf("%*s", n, " ")
	}
	fmt.Fprintf(p.out, "\n\n%*s%s%s%s%s%s\n\n\n", p.Indent, "", p.dateText, pad(lhs), p.fileText, pad(rhs), pageText)
	p.printHeader = false
	p.outputPosition = 0
}

func (p *Pr) readLine(c *column) bool {
	ch := p.getc(c)
	lastInput := p.inputPosition
	if ch == '\f' && c.fullPagePrinted {
		if ch = p.getc(c); ch == '\n' {
			ch = p.getc(c)
		}
	}
	c.fullPagePrinted = false

	chars := 0
	switch ch {
	case '\f':
		if next := p.getc(c); next != '\n' {
			p.ungetc(c, next)
		}
		p.ffOnly = true
		if p.printHeader && !p.storing {
			p.padVertically = true
			p.printHeaderLines()
		} else if p.keepFF {
			p.printAFF = true
		}
		p.holdFile(c)
		return true
	case -1:
		p.closeFile(c)
		return true
	case '\n':
	default:
		chars = p.charToClump(byte(ch))
	}
	if p.truncate && p.inputPosition > p.charsPerCol {
		p.inputPosition = lastInput
		return false
	}
	if !c.store {
		p.padVertically = true
		if p.printHeader && !p.storing {
			p.printHeaderLines()
		}
		if p.parallel && p.alignEmptyCols {
			k := p.separatorsNotPrinted
			p.separatorsNotPrinted = 0
			for j := 0; j < k; j++ {
				p.alignColumn(&p.cols[j])
				p.separatorsNotPrinted++
			}
			p.paddingNotPrinted = c.startPosition
			if p.truncate {
				p.spacesNotPrinted = p.charsPerCol
			} else {
				p.spacesNotPrinted = 0
			}
			p.alignEmptyCols = false
		}
		if len(p.sep) < p.paddingNotPrinted {
			p.padAcrossTo(p.paddingNotPrinted - len(p.sep))
			p.paddingNotPrinted = 0
		}
		if p.useSep {
			p.printSepString()
		}
	}
	if c.numbered {
		p.addLineNumber(c)
	}
	p.emptyLine = false
	if ch == '\n' {
		return true
	}
	p.printClump(c, chars)
	for {
		ch = p.getc(c)
		switch ch {
		case '\n':
			return true
		case '\f':
			if next := p.getc(c); next != '\n' {
				p.ungetc(c, next)
			}
			if p.keepFF {
				p.printAFF = true
			}
			p.holdFile(c)
			return true
		case -1:
			p.closeFile(c)
			return true
		}
		lastInput = p.inputPosition
		chars = p.charToClump(byte(ch))
		if p.truncate && p.inputPosition > p.charsPerCol {
			p.inputPosition = lastInput
			return false
		}
		p.printClump(c, chars)
	}
}

func (p *Pr) printClump(c *column, n int) {
	for i := 0; i < n; i++ {
		p.emit(c, p.clump[i])
	}
}

func (p *Pr) printStored(c *column) bool {
	line := c.currentLine
	c.currentLine++
	first, last := p.lineVector[line], p.lineVector[line+1]
	p.padVertically = true
	if p.printHeader {
		p.printHeaderLines()
	}
	if c.status == ffFound {
		for i := range p.cols {
			p.cols[i].status = onHold
		}
		if p.cols[0].linesToPrint <= 0 {
			if !p.extremities {
				p.padVertically = false
			}
			return true
		}
	}
	if len(p.sep) < p.paddingNotPrinted {
		p.padAcrossTo(p.paddingNotPrinted - len(p.sep))
		p.paddingNotPrinted = 0
	}
	if p.useSep {
		p.printSepString()
	}
	for _, b := range p.buff[first:last] {
		p.printChar(b)
	}
	if p.spacesNotPrinted == 0 {
		p.outputPosition = c.startPosition + p.endVector[line]
		if c.startPosition-len(p.sep) == p.Indent {
			p.outputPosition -= len(p.sep)
		}
	}
	return true
}

// charToClump turns an input character into what is printed for it, in
// p.clump, and advances the input position by its width.
func (p *Pr) charToClump(ch byte) int {
	p.clump = p.clump[:0]
	width := 0
	perTab := 8
	if ch == p.inTab {
		perTab = p.inTabWidth
	}
	switch {
	case ch == p.inTab || ch == '\t':
		width = tabWidth(perTab, p.inputPosition)
		if p.untabify {
			for i := 0; i < width; i++ {
				p.clump = append(p.clump, ' ')
			}
		} else {
			p.clump = append(p.clump, ch)
		}
	case ch < 0x20 || ch >= 0x7f:
		switch {
		case p.ShowNonprinting:
			width = 4
			p.clump = fmt.Appendf(p.clump, "\\%03o", ch)
		case p.ShowControlChars && ch < 0o200:
			width = 2
			p.clump = append(p.clump, '^', ch^0o100)
		case p.ShowControlChars:
			width = 4
			p.clump = fmt.Appendf(p.clump, "\\%03o", ch)
		case ch == '\b':
			width = -1
			p.clump = append(p.clump, ch)
		default:
			p.clump = append(p.clump, ch)
		}
	default:
		width = 1
		p.clump = append(p.clump, ch)
	}
	chars := len(p.clump)
	switch {
	case width < 0 && p.inputPosition == 0:
		chars = 0
	case width < 0 && p.inputPosition <= -width:
		p.inputPosition = 0
	default:
		p.inputPosition += width
	}
	return chars
}

func (p *Pr) readRestOfLine(c *column) {
	for {
		ch := p.getc(c)
		switch ch {
		case '\n':
			return
		case '\f':
			if next := p.getc(c); next != '\n' {
				p.ungetc(c, next)
			}
			if p.keepFF {
				p.printAFF = true
			}
			p.holdFile(c)
			return
		case -1:
			p.closeFile(c)
			return
		}
	}
}
