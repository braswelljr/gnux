// Package fmt implements the engine behind fmtx, a GNU-compatible
// implementation of the GNU/UNIX fmt command: a simple optimal text
// formatter that refills paragraphs to a goal width.
//
// The formatting is a port of GNU fmt's: the same paragraph detection, the
// same cost function for choosing line breaks (short lines, ragged edges,
// widows, orphans and sentence ends all weigh in) and the same buffer limits,
// so output matches GNU's line for line.
package fmt

import (
	"bufio"
	"errors"
	"io"
	"math"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "fmtx"

// DefaultWidth is the maximum line width when none is given.
const DefaultWidth = 75

// MaxWidth is the largest width fmt accepts.
const MaxWidth = maxChars / 2

const (
	maxChars  = 5000 // bytes of paragraph text held before a forced flush
	maxWords  = 1000 // words held before a forced flush
	leeway    = 7    // the goal is this percentage short of the width
	tabWidth  = 8
	defIndent = 3 // the indent -t gives a one-line paragraph
)

// The cost function, as GNU defines it.
const (
	maxCost       = math.MaxInt64
	lineCost      = 70 * 70
	sentenceBonus = 50 * 50
	nobreakCost   = 600 * 600
	parenBonus    = 40 * 40
	punctBonus    = 40 * 40
)

func shortCost(n int64) int64  { return (n * 10) * (n * 10) }
func raggedCost(n int64) int64 { return shortCost(n) / 2 }
func widowCost(n int64) int64  { return 200 * 200 / (n + 2) }
func orphanCost(n int64) int64 { return 150 * 150 / (n + 2) }

// Options mirrors the GNU fmt flag surface. The zero value is plain fmt:
// paragraphs refilled to a goal of 70 within a width of 75.
type Options struct {
	// Crown (-c) keeps the indentation of the first two lines of each
	// paragraph and aligns the rest with the second.
	Crown bool
	// Tagged (-t) is like Crown, but the first line's indentation must differ
	// from the second's, or the first line is a paragraph of its own.
	Tagged bool
	// Split (-s) splits long lines but never joins short ones.
	Split bool
	// Uniform (-u) puts one space between words and two after sentences.
	Uniform bool
	// Prefix (-p) reformats only the lines that begin with it, keeping the
	// prefix on each output line.
	Prefix string
	// Width (-w) is the maximum line width; used when WidthSet is true.
	Width int
	// WidthSet records that -w was given.
	WidthSet bool
	// Goal (-g) is the goal width; used when GoalSet is true. Without it the
	// goal is 93% of the width.
	Goal int
	// GoalSet records that -g was given. Without -w, the width is then the
	// goal plus 10.
	GoalSet bool
}

// Widths returns the maximum and goal widths opts selects, as GNU computes
// them.
func (o Options) Widths() (width, goal int) {
	width = DefaultWidth
	if o.WidthSet {
		width = o.Width
	}
	if o.GoalSet {
		goal = o.Goal
		if !o.WidthSet {
			width = goal + 10
		}
	} else {
		goal = width * (2*(100-leeway) + 1) / 200
	}
	return width, goal
}

// word is one word of the paragraph being formatted.
type word struct {
	text   int // offset of the word in the paragraph buffer
	length int
	space  int // the spaces that follow it

	paren  bool // starts with an opening bracket or quote
	period bool // ends a sentence, ignoring closing brackets and quotes
	punct  bool // ends with punctuation
	final  bool // ends a sentence and is followed by enough space

	lineLength int   // the length of the line it starts, in the best layout
	bestCost   int64 // the cost of the best layout from this word on
	nextBreak  int   // where that layout breaks after this word
}

// Fmt is the engine. Construct it with New.
type Fmt struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	maxWidth, goalWidth int

	prefix                string // the prefix, without its leading and trailing spaces
	prefixFull, prefixLen int    // its length with and without trailing spaces
	prefixLead            int    // its leading spaces
	in                    *bufio.Reader
	readErr               error
	tabs                  bool
	inColumn, outColumn   int
	nextPrefixIndent      int
	prefixIndent          int
	firstIndent           int
	otherIndent           int
	lastLineLength        int
	nextChar              int
	parabuf               [maxChars]byte
	wptr                  int
	words                 [maxWords]word
	wordLimit             int
}

// New builds a fmt engine reading from and writing to s.
func New(opts Options, s core.Streams) *Fmt {
	f := &Fmt{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
	f.maxWidth, f.goalWidth = opts.Widths()
	f.setPrefix(opts.Prefix)
	return f
}

func (f *Fmt) setPrefix(p string) {
	f.prefixLead = 0
	for strings.HasPrefix(p, " ") {
		f.prefixLead++
		p = p[1:]
	}
	f.prefixFull = len(p)
	p = strings.TrimRight(p, " ")
	f.prefix = p
	f.prefixLen = len(p)
}

// Run formats each operand in turn; no operands, or "-", means standard
// input. A failing operand is reported and the rest still run. The returned
// error is core.ErrFailed if anything failed, otherwise nil.
func (f *Fmt) Run(args []string) error {
	f.out = core.NewWriter(f.streams.Out)
	if len(args) == 0 {
		args = []string{"-"}
	}
	for _, name := range args {
		var r io.Reader
		if name == "-" {
			r = f.streams.In
			if r == nil {
				r = strings.NewReader("")
			}
		} else {
			file, err := fsys.Open(name)
			if err != nil {
				f.diag.Errorf("cannot open '%s' for reading: %s", name, core.ErrText(err))
				continue
			}
			if info, err := file.Stat(); err == nil && info.IsDir() {
				f.diag.Errorf("%s: %s", name, core.ErrText(errIsDir))
				file.Close()
				continue
			}
			r = file
			defer file.Close()
		}
		f.format(r)
		if err := f.out.Flush(); err != nil {
			if !closedPipe(err) {
				f.diag.Errorf("write error: %s", core.ErrText(err))
			}
			return core.ErrFailed
		}
		if f.readErr != nil {
			f.diag.Errorf("%s: %s", name, core.ErrText(f.readErr))
		}
	}
	return f.diag.Result()
}

// format formats one input.
func (f *Fmt) format(r io.Reader) {
	f.in = core.NewReader(r)
	f.readErr = nil
	f.tabs = false
	f.otherIndent = 0
	f.nextChar = f.getPrefix()
	for f.getParagraph() {
		f.fmtParagraph()
		f.putParagraph(f.wordLimit)
	}
}

// getc reads one byte, or -1 at the end of the input.
func (f *Fmt) getc() int {
	c, err := f.in.ReadByte()
	if err != nil {
		if !errors.Is(err, io.EOF) && f.readErr == nil {
			f.readErr = err
		}
		return -1
	}
	return int(c)
}

func (f *Fmt) putc(c int) {
	_ = f.out.WriteByte(byte(c))
}

// getPrefix reads the indentation and prefix that start a line, leaving
// inColumn after them, and returns the first byte that follows.
func (f *Fmt) getPrefix() int {
	f.inColumn = 0
	c := f.getSpace(f.getc())
	if f.prefixLen == 0 {
		f.nextPrefixIndent = min(f.prefixLead, f.inColumn)
		return c
	}
	f.nextPrefixIndent = f.inColumn
	for i := 0; i < len(f.prefix); i++ {
		if c != int(f.prefix[i]) {
			return c
		}
		f.inColumn++
		c = f.getc()
	}
	return f.getSpace(c)
}

// getSpace skips spaces and tabs from c on, counting columns.
func (f *Fmt) getSpace(c int) int {
	for {
		switch c {
		case ' ':
			f.inColumn++
		case '\t':
			f.tabs = true
			f.inColumn = (f.inColumn/tabWidth + 1) * tabWidth
		default:
			return c
		}
		c = f.getc()
	}
}

// sameParagraph reports whether the line now being read continues the
// paragraph.
func (f *Fmt) sameParagraph(c int) bool {
	return f.nextPrefixIndent == f.prefixIndent &&
		f.inColumn >= f.nextPrefixIndent+f.prefixFull &&
		c != '\n' && c != -1
}

// getParagraph reads the next paragraph into the word list, copying blank
// lines and lines without the prefix through unchanged. It reports whether a
// paragraph was found.
func (f *Fmt) getParagraph() bool {
	f.lastLineLength = 0
	c := f.nextChar

	for c == '\n' || c == -1 ||
		f.nextPrefixIndent < f.prefixLead ||
		f.inColumn < f.nextPrefixIndent+f.prefixFull {
		c = f.copyRest(c)
		if c == -1 {
			f.nextChar = -1
			return false
		}
		f.putc('\n')
		c = f.getPrefix()
	}

	f.prefixIndent = f.nextPrefixIndent
	f.firstIndent = f.inColumn
	f.wptr = 0
	f.wordLimit = 0
	c = f.getLine(c)
	f.setOtherIndent(f.sameParagraph(c))

	switch {
	case f.Split:
		// One line per paragraph.
	case f.Crown:
		if f.sameParagraph(c) {
			c = f.getLine(c)
			for f.sameParagraph(c) && f.inColumn == f.otherIndent {
				c = f.getLine(c)
			}
		}
	case f.Tagged:
		if f.sameParagraph(c) && f.inColumn != f.firstIndent {
			c = f.getLine(c)
			for f.sameParagraph(c) && f.inColumn == f.otherIndent {
				c = f.getLine(c)
			}
		}
	default:
		for f.sameParagraph(c) && f.inColumn == f.otherIndent {
			c = f.getLine(c)
		}
	}

	last := &f.words[f.wordLimit-1]
	last.period, last.final = true, true
	f.nextChar = c
	return true
}

// copyRest copies the rest of a line that is not part of a paragraph,
// including the indentation and prefix already read.
func (f *Fmt) copyRest(c int) int {
	f.outColumn = 0
	if f.inColumn > f.nextPrefixIndent || (c != '\n' && c != -1) {
		f.putSpace(f.nextPrefixIndent)
		for i := 0; f.outColumn != f.inColumn && i < len(f.prefix); i++ {
			f.putc(int(f.prefix[i]))
			f.outColumn++
		}
		if c != -1 && c != '\n' {
			f.putSpace(f.inColumn - f.outColumn)
		}
		if c == -1 && f.inColumn >= f.nextPrefixIndent+f.prefixLen {
			f.putc('\n')
		}
	}
	for c != '\n' && c != -1 {
		f.putc(c)
		c = f.getc()
	}
	return c
}

// setOtherIndent fixes the indentation of a paragraph's lines after the
// first.
func (f *Fmt) setOtherIndent(sameParagraph bool) {
	switch {
	case f.Split:
		f.otherIndent = f.firstIndent
	case f.Crown:
		if sameParagraph {
			f.otherIndent = f.inColumn
		} else {
			f.otherIndent = f.firstIndent
		}
	case f.Tagged:
		if sameParagraph && f.inColumn != f.firstIndent {
			f.otherIndent = f.inColumn
		} else if f.otherIndent == f.firstIndent {
			// A one-line paragraph: keep the last indent unless it would be
			// the same as the first line's.
			if f.firstIndent == 0 {
				f.otherIndent = defIndent
			} else {
				f.otherIndent = 0
			}
		}
	default:
		f.otherIndent = f.firstIndent
	}
}

// getLine reads the words of one line, from c on, and returns the first
// byte after the next line's prefix.
func (f *Fmt) getLine(c int) int {
	for {
		w := &f.words[f.wordLimit]
		w.text = f.wptr
		for {
			if f.wptr == maxChars {
				f.setOtherIndent(true)
				f.flushParagraph()
				w = &f.words[f.wordLimit]
			}
			f.parabuf[f.wptr] = byte(c)
			f.wptr++
			c = f.getc()
			if c == -1 || isSpace(c) {
				break
			}
		}
		w.length = f.wptr - w.text
		f.inColumn += w.length
		checkPunctuation(w, f.parabuf[w.text:f.wptr])

		start := f.inColumn
		c = f.getSpace(c)
		w.space = f.inColumn - start
		w.final = c == -1 || (w.period && (c == '\n' || w.space > 1))
		if c == '\n' || c == -1 || f.Uniform {
			if w.final {
				w.space = 2
			} else {
				w.space = 1
			}
		}
		if f.wordLimit == maxWords-2 {
			f.setOtherIndent(true)
			f.flushParagraph()
		}
		f.wordLimit++
		if c == '\n' || c == -1 {
			break
		}
	}
	return f.getPrefix()
}

// checkPunctuation records how a word starts and ends.
func checkPunctuation(w *word, text []byte) {
	if len(text) == 0 {
		return
	}
	first, finish := text[0], len(text)-1
	w.paren = isOpen(first)
	w.punct = isPunct(text[finish])
	for finish > 0 && isClose(text[finish]) {
		finish--
	}
	w.period = isPeriod(text[finish])
}

// flushParagraph writes out the first part of a paragraph too long for the
// buffers, breaking at a good point near the end of what is held.
func (f *Fmt) flushParagraph() {
	if f.wordLimit == 0 {
		// It is all one word: write it as it is.
		_, _ = f.out.Write(f.parabuf[:f.wptr])
		f.wptr = 0
		f.words[0].text = 0
		return
	}

	f.fmtParagraph()

	split := f.wordLimit
	best := int64(maxCost)
	for w := f.words[0].nextBreak; w != f.wordLimit; w = f.words[w].nextBreak {
		gap := f.words[w].bestCost - f.words[f.words[w].nextBreak].bestCost
		if gap < best {
			split, best = w, gap
		}
		if best <= maxCost-lineCost {
			best += lineCost
		}
	}
	f.putParagraph(split)

	shift := f.words[split].text
	copy(f.parabuf[:], f.parabuf[shift:f.wptr])
	f.wptr -= shift
	for w := split; w <= f.wordLimit; w++ {
		f.words[w].text -= shift
	}
	n := copy(f.words[:], f.words[split:f.wordLimit+1])
	_ = n
	f.wordLimit -= split
}

// fmtParagraph finds the best line breaks for the words held, working back
// from the end: each word records the cheapest way to lay out the rest of the
// paragraph starting a line with it.
func (f *Fmt) fmtParagraph() {
	limit := f.wordLimit
	words := &f.words
	words[limit].bestCost = 0
	saved := words[limit].length
	words[limit].length = f.maxWidth // a sentinel

	for start := limit - 1; start >= 0; start-- {
		best := int64(maxCost)
		length := f.otherIndent
		if start == 0 {
			length = f.firstIndent
		}
		w := start
		length += words[w].length
		for {
			w++
			cost := f.lineCost(w, length) + words[w].bestCost
			if start == 0 && f.lastLineLength > 0 {
				cost += raggedCost(int64(length - f.lastLineLength))
			}
			if cost < best {
				best = cost
				words[start].nextBreak = w
				words[start].lineLength = length
			}
			if w == limit {
				break
			}
			length += words[w-1].space + words[w].length
			if length >= f.maxWidth {
				break
			}
		}
		words[start].bestCost = best + f.baseCost(start)
	}
	words[limit].length = saved
}

// lineCost is the cost of a line of length length that breaks before next.
func (f *Fmt) lineCost(next, length int) int64 {
	if next == f.wordLimit {
		return 0
	}
	cost := shortCost(int64(f.goalWidth - length))
	if f.words[next].nextBreak != f.wordLimit {
		cost += raggedCost(int64(length - f.words[next].lineLength))
	}
	return cost
}

// baseCost is the cost of starting a line with word i.
func (f *Fmt) baseCost(i int) int64 {
	cost := int64(lineCost)
	w := &f.words[i]
	if i > 0 {
		prev := &f.words[i-1]
		switch {
		case prev.period:
			if prev.final {
				cost -= sentenceBonus
			} else {
				cost += nobreakCost
			}
		case prev.punct:
			cost -= punctBonus
		case i > 1 && f.words[i-2].final:
			cost += widowCost(int64(prev.length))
		}
	}
	if w.paren {
		cost -= parenBonus
	} else if w.final {
		cost += orphanCost(int64(w.length))
	}
	return cost
}

// putParagraph writes the lines of the best layout, up to the word finish.
func (f *Fmt) putParagraph(finish int) {
	f.putLine(0, f.firstIndent)
	for w := f.words[0].nextBreak; w != finish; w = f.words[w].nextBreak {
		f.putLine(w, f.otherIndent)
	}
}

// putLine writes the line that starts with word w.
func (f *Fmt) putLine(w, indent int) {
	f.outColumn = 0
	f.putSpace(f.prefixIndent)
	_, _ = f.out.WriteString(f.prefix)
	f.outColumn += f.prefixLen
	f.putSpace(indent - f.outColumn)

	end := f.words[w].nextBreak - 1
	for ; w != end; w++ {
		f.putWord(w)
		f.putSpace(f.words[w].space)
	}
	f.putWord(w)
	f.lastLineLength = f.outColumn
	f.putc('\n')
}

func (f *Fmt) putWord(i int) {
	w := &f.words[i]
	_, _ = f.out.Write(f.parabuf[w.text : w.text+w.length])
	f.outColumn += w.length
}

// putSpace moves the output column on by space, with tabs where the input
// used them.
func (f *Fmt) putSpace(space int) {
	target := f.outColumn + space
	if f.tabs {
		tabTarget := target / tabWidth * tabWidth
		if f.outColumn+1 < tabTarget {
			for f.outColumn < tabTarget {
				f.putc('\t')
				f.outColumn = (f.outColumn/tabWidth + 1) * tabWidth
			}
		}
	}
	for f.outColumn < target {
		f.putc(' ')
		f.outColumn++
	}
}

// The C locale's character classes, and GNU fmt's own. GNU tests these with
// strchr, which also matches the terminating NUL, so a NUL byte counts too.
func isSpace(c int) bool {
	return c == ' ' || (c >= '\t' && c <= '\r')
}

func isOpen(c byte) bool   { return strings.IndexByte("(['`\"\x00", c) >= 0 }
func isClose(c byte) bool  { return strings.IndexByte(")]'\"\x00", c) >= 0 }
func isPeriod(c byte) bool { return strings.IndexByte(".?!\x00", c) >= 0 }

func isPunct(c byte) bool {
	return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
		(c >= '[' && c <= '`') || (c >= '{' && c <= '~')
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

func closedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
