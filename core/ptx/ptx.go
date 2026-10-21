// Package ptx implements the engine behind ptxx, a GNU-compatible
// implementation of the GNU/UNIX ptx command: produce a permuted index of
// the words of a text, each keyword shown in its context.
package ptx

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "ptxx"

// Format is the kind of output.
type Format int

const (
	// Default is Dumb, or Roff with Traditional.
	Default Format = iota
	// Dumb lines up the index for a plain terminal.
	Dumb
	// Roff writes a .xx macro call per line (-O, --format=roff).
	Roff
	// TeX writes a \xx macro call per line (-T, --format=tex).
	TeX
)

// Options mirrors the GNU ptx flag surface. The zero value is plain ptx.
// Strings for -F, -S and -W take the backslash escapes GNU ptx takes
// (\n, \t, \x41, \0101, ...).
type Options struct {
	// AutoReference (-A) prefixes each line with FILE:LINE.
	AutoReference bool
	// Traditional (-G) behaves like System V ptx: words are whatever is
	// between blanks, contexts are lines, roff is the default format, and a
	// second operand names the output file.
	Traditional bool
	// FlagTruncation (-F), when FlagTruncationSet, marks truncated contexts
	// instead of "/"; empty turns the marks off.
	FlagTruncation    string
	FlagTruncationSet bool
	// MacroName (-M) is the roff or TeX macro; empty means "xx".
	MacroName string
	// Format (-O, -T, --format) is the output format.
	Format Format
	// RightSideRefs (-R) puts references after the right context.
	RightSideRefs bool
	// SentenceRegexp (-S), when SentenceRegexpSet, is what ends a context;
	// empty means the whole input is one context.
	SentenceRegexp    string
	SentenceRegexpSet bool
	// WordRegexp (-W) is what a keyword is; empty means a run of letters.
	WordRegexp string
	// BreakFile (-b) lists the characters that end a word.
	BreakFile string
	// IgnoreCase (-f) folds lower case to upper case for sorting.
	IgnoreCase bool
	// GapSize (-g) is the gap between fields; 0 means 3.
	GapSize int
	// IgnoreFile (-i) and OnlyFile (-o) list words, one per line, to leave
	// out of the index and to keep only.
	IgnoreFile, OnlyFile string
	// References (-r) takes the first field of each line as its reference.
	References bool
	// Width (-w) is the output width, references excluded; 0 means 72.
	Width int
}

// Ptx is the engine. Construct it with New.
type Ptx struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	contextRe, wordRe *regexp.Regexp
	wordAnchored      *regexp.Regexp
	contextSource     string
	wordChars         [256]bool
	ignoreTable       [][]byte
	onlyTable         [][]byte
	useIgnore         bool
	useOnly           bool

	names      []string // "" for standard input
	texts      [][]byte
	lineCounts []int64
	occurs     []occurrence
	totalLines int64
	maxWord    int
	refMax     int

	lineWidth, halfWidth, gap, beforeMax, keyafterMax int
	trunc                                             string
	edited                                            [256]bool
	format                                            Format
	macro                                             string
}

// occurrence is one keyword in its context, as offsets into its file's text.
type occurrence struct {
	file      int
	key, size int
	left      int   // where the left context starts
	right     int   // where the right context ends
	reference int64 // line number (-A), or where the reference starts (-r)
}

// New builds a ptx engine writing to s.Out.
func New(opts Options, s core.Streams) *Ptx {
	return &Ptx{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// abort is what fail panics with: ptx exits on every error.
type abort struct{}

func (p *Ptx) fail(format string, a ...any) {
	p.diag.Errorf(format, a...)
	panic(abort{})
}

func quote(s string) string { return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale}) }
func quotef(s string) string {
	return quotearg.Quote(s, quotearg.Options{Style: quotearg.ShellEscape, Also: ":"})
}

// Run indexes each INPUT (standard input when there are none, or for "-"
// and ""). With Traditional there is at most one INPUT, and a second operand
// is the output file.
func (p *Ptx) Run(args []string) (err error) {
	out := p.streams.Out
	if p.Traditional && len(args) > 2 {
		return core.Usagef("extra operand %s", quote(args[2]))
	}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(abort); !ok {
				panic(r)
			}
			if p.out != nil {
				_ = p.out.Flush()
			}
			err = core.ErrFailed
		}
	}()
	if p.Traditional && len(args) == 2 {
		f, ferr := fsys.Create(args[1])
		if ferr != nil {
			p.fail("%s: %s", quotef(args[1]), core.ErrText(ferr))
		}
		defer f.Close()
		out = f
		args = args[:1]
	}
	p.out = core.NewWriter(out)
	defer core.FlushOutput(p.out, p.diag, &err)

	p.format = p.Format
	if p.format == Default {
		p.format = Dumb
		if p.Traditional {
			p.format = Roff
		}
	}
	p.macro = p.MacroName
	if p.macro == "" {
		p.macro = "xx"
	}
	p.initRegex()
	if p.BreakFile != "" {
		data := p.read(p.BreakFile)
		for i := range p.wordChars {
			p.wordChars[i] = true
		}
		for _, b := range data {
			p.wordChars[b] = false
		}
		if p.Traditional {
			p.wordChars[' '], p.wordChars['\t'], p.wordChars['\n'] = false, false, false
		}
	}
	if p.IgnoreFile != "" {
		p.ignoreTable = p.wordFile(p.IgnoreFile)
		p.useIgnore = len(p.ignoreTable) > 0
	}
	if p.OnlyFile != "" {
		p.onlyTable = p.wordFile(p.OnlyFile)
		p.useOnly = len(p.onlyTable) > 0
	}

	if len(args) == 0 {
		args = []string{""}
	}
	for i, name := range args {
		if name == "-" {
			name = ""
		}
		p.names = append(p.names, name)
		p.texts = append(p.texts, p.read(name))
		p.findOccurs(i)
		p.totalLines++
		p.lineCounts = append(p.lineCounts, p.totalLines)
	}
	slices.SortStableFunc(p.occurs, func(a, b occurrence) int {
		if c := p.compareWords(p.word(a), p.word(b)); c != 0 {
			return c
		}
		if a.file != b.file {
			return a.file - b.file
		}
		return a.key - b.key
	})
	p.fixOutputParameters()
	for i := range p.occurs {
		p.outputLine(&p.occurs[i])
	}
	return nil
}

// read returns the whole of a file, or standard input for "".
func (p *Ptx) read(name string) []byte {
	var data []byte
	var err error
	if name == "" || name == "-" {
		data, err = io.ReadAll(p.streams.In)
		name = "-"
	} else {
		data, err = fsys.ReadFile(name)
	}
	if err != nil {
		p.fail("%s: %s", quotef(name), core.ErrText(err))
	}
	return data
}

func (p *Ptx) word(o occurrence) []byte { return p.texts[o.file][o.key : o.key+o.size] }

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 'a' + 'A'
	}
	return b
}

func (p *Ptx) compareWords(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		x, y := a[i], b[i]
		if p.IgnoreCase {
			x, y = upper(x), upper(y)
		}
		if x != y {
			return int(x) - int(y)
		}
	}
	return len(a) - len(b)
}

// wordFile reads a file of words, one per line, sorted for searching.
func (p *Ptx) wordFile(name string) [][]byte {
	var words [][]byte
	for line := range bytes.SplitSeq(p.read(name), []byte("\n")) {
		if len(line) > 0 {
			words = append(words, line)
		}
	}
	slices.SortFunc(words, p.compareWords)
	return words
}

func (p *Ptx) inTable(w []byte, table [][]byte) bool {
	_, found := slices.BinarySearchFunc(table, w, p.compareWords)
	return found
}

func (p *Ptx) compile(source string) *regexp.Regexp {
	translated, err := translateEmacs(source)
	if err == nil {
		prefix := "(?m)"
		if p.IgnoreCase {
			prefix = "(?mi)"
		}
		var re *regexp.Regexp
		if re, err = regexp.Compile(prefix + translated); err == nil {
			re.Longest()
			return re
		}
	}
	p.fail("%s (for regexp %s)", regexMessage(err), quote(source))
	return nil
}

func (p *Ptx) initRegex() {
	switch {
	case p.SentenceRegexpSet:
		if s := Unescape(p.SentenceRegexp); s != "" {
			p.contextSource = s
		}
	case !p.Traditional && !p.References:
		p.contextSource = "[.?!][]\"')}]*\\($\\|\t\\|  \\)[ \t\n]*"
	default:
		p.contextSource = "\n"
	}
	if p.contextSource != "" {
		p.contextRe = p.compile(p.contextSource)
	}
	if w := Unescape(p.WordRegexp); w != "" {
		p.wordRe = p.compile(w)
		translated, _ := translateEmacs(w)
		prefix := "\\A(?:"
		if p.IgnoreCase {
			prefix = "(?i)\\A(?:"
		}
		p.wordAnchored = regexp.MustCompile(prefix + translated + ")")
		p.wordAnchored.Longest()
	} else if p.BreakFile == "" {
		for c := range p.wordChars {
			if p.Traditional {
				p.wordChars[c] = c != ' ' && c != '\t' && c != '\n'
			} else {
				p.wordChars[c] = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
			}
		}
	}
}

func space(c byte) bool { return c == ' ' || (c >= '\t' && c <= '\r') }

// skipSomething advances past one word, or one character that is not part
// of a word: GNU's SKIP_SOMETHING.
func (p *Ptx) skipSomething(text []byte, cursor, limit int) int {
	if p.wordAnchored != nil {
		if loc := p.wordAnchored.FindIndex(text[cursor:limit]); loc != nil && loc[1] > 0 {
			return cursor + loc[1]
		}
		return cursor + 1
	}
	if p.wordChars[text[cursor]] {
		for cursor < limit && p.wordChars[text[cursor]] {
			cursor++
		}
		return cursor
	}
	return cursor + 1
}

func (p *Ptx) findOccurs(file int) {
	text := p.texts[file]
	end := len(text)
	lineStart, lineScan := 0, 0
	refLen := 0
	skipNonWhite := func(i, limit int) int {
		for i < limit && !space(text[i]) {
			i++
		}
		return i
	}
	skipWhite := func(i, limit int) int {
		for i < limit && space(text[i]) {
			i++
		}
		return i
	}
	if p.References {
		lineScan = skipNonWhite(lineScan, end)
		refLen = lineScan - lineStart
		lineScan = skipWhite(lineScan, end)
	}
	for cursor := 0; cursor < end; {
		contextStart := cursor
		next := end
		if p.contextRe != nil {
			if loc := p.contextRe.FindIndex(text[cursor:end]); loc != nil {
				if loc[0] == 0 {
					p.fail("error: regular expression has a match of length zero: %s", quote(p.contextSource))
				}
				next = cursor + loc[1]
			}
		}
		contextEnd := next
		for contextEnd > contextStart && space(text[contextEnd-1]) {
			contextEnd--
		}
		for {
			var wordStart, wordEnd int
			if p.wordRe != nil {
				loc := p.wordRe.FindIndex(text[cursor:contextEnd])
				if loc == nil {
					break
				}
				wordStart, wordEnd = cursor+loc[0], cursor+loc[1]
			} else {
				scan := cursor
				for scan < contextEnd && !p.wordChars[text[scan]] {
					scan++
				}
				if scan == contextEnd {
					break
				}
				wordStart = scan
				for scan < contextEnd && p.wordChars[text[scan]] {
					scan++
				}
				wordEnd = scan
			}
			cursor = wordStart
			if wordEnd == wordStart {
				cursor++
				continue
			}
			key, size := cursor, wordEnd-wordStart
			cursor += size
			p.maxWord = max(p.maxWord, size)

			if p.References {
				for lineScan < key {
					if text[lineScan] == '\n' {
						p.totalLines++
						lineScan++
						lineStart = lineScan
						lineScan = skipNonWhite(lineScan, end)
						refLen = lineScan - lineStart
					} else {
						lineScan++
					}
				}
				if lineScan > key {
					continue
				}
			}
			w := text[key : key+size]
			if p.useIgnore && p.inTable(w, p.ignoreTable) {
				continue
			}
			if p.useOnly && !p.inTable(w, p.onlyTable) {
				continue
			}
			o := occurrence{file: file, key: key, size: size}
			switch {
			case p.AutoReference:
				for lineScan < key {
					if text[lineScan] == '\n' {
						p.totalLines++
						lineScan++
						lineStart = lineScan
						lineScan = skipNonWhite(lineScan, end)
					} else {
						lineScan++
					}
				}
				o.reference = p.totalLines
			case p.References:
				o.reference = int64(lineStart - key)
				p.refMax = max(p.refMax, refLen)
			}
			if p.References && lineStart == contextStart {
				contextStart = skipNonWhite(contextStart, contextEnd)
				contextStart = skipWhite(contextStart, contextEnd)
			}
			o.left, o.right = contextStart, contextEnd
			p.occurs = append(p.occurs, o)
		}
		cursor = next
	}
}

func (p *Ptx) fixOutputParameters() {
	p.lineWidth = p.Width
	if p.lineWidth == 0 {
		p.lineWidth = 72
	}
	p.gap = p.GapSize
	if p.gap == 0 {
		p.gap = 3
	}
	if p.AutoReference {
		p.refMax = 0
		for i, name := range p.names {
			ordinal := p.lineCounts[i] + 1
			if i > 0 {
				ordinal -= p.lineCounts[i-1]
			}
			p.refMax = max(p.refMax, len(strconv.FormatInt(ordinal, 10))+len(name))
		}
		p.refMax++
	}
	if (p.AutoReference || p.References) && !p.RightSideRefs {
		p.lineWidth = max(0, p.lineWidth-(p.refMax+p.gap))
	}
	p.halfWidth = p.lineWidth >> 1
	p.beforeMax = p.halfWidth - p.gap
	p.keyafterMax = p.halfWidth
	p.trunc = "/"
	if p.FlagTruncationSet {
		p.trunc = Unescape(p.FlagTruncation)
	}
	if p.Traditional {
		p.keyafterMax -= 2*len(p.trunc) + 1
	} else {
		p.beforeMax = max(0, p.beforeMax-2*len(p.trunc))
		p.keyafterMax -= 2 * len(p.trunc)
	}
	for c := range p.edited {
		p.edited[c] = space(byte(c))
	}
	p.edited['\f'] = true
	switch p.format {
	case Roff:
		p.edited['"'] = true
	case TeX:
		for _, c := range []byte("$%&#_{}\\") {
			p.edited[c] = true
		}
	}
}

// field is a span of a file's text, or of other text (a reference).
type field struct {
	text       []byte
	start, end int
}

func (f field) width() int { return f.end - f.start }

func (p *Ptx) printField(f field) {
	if f.start >= f.end {
		return // a field that ends before it starts holds nothing
	}
	for _, c := range f.text[f.start:f.end] {
		if !p.edited[c] {
			_ = p.out.WriteByte(c)
			continue
		}
		switch c {
		case '"':
			_, _ = p.out.WriteString(`""`)
		case '$', '%', '&', '#', '_':
			_ = p.out.WriteByte('\\')
			_ = p.out.WriteByte(c)
		case '{', '}':
			fmt.Fprintf(p.out, "$\\%c$", c)
		case '\\':
			_, _ = p.out.WriteString("\\backslash{}")
		default:
			_ = p.out.WriteByte(' ')
		}
	}
}

func (p *Ptx) spaces(n int) {
	for ; n > 0; n-- {
		_ = p.out.WriteByte(' ')
	}
}

// outputLine works out the fields of one occurrence and prints them: GNU's
// define_all_fields and output_one_*_line.
func (p *Ptx) outputLine(o *occurrence) {
	text := p.texts[o.file]
	end := len(text)
	trunc := p.trunc != ""
	skipWhite := func(i, limit int) int {
		for i < limit && space(text[i]) {
			i++
		}
		return i
	}
	skipWhiteBack := func(i, start int) int {
		for i > start && space(text[i-1]) {
			i--
		}
		return i
	}

	keyafter := field{text: text, start: o.key, end: o.key + o.size}
	leftStart, rightEnd := o.left, o.right
	cursor := keyafter.end
	for cursor < rightEnd && cursor <= keyafter.start+p.keyafterMax {
		keyafter.end = cursor
		cursor = p.skipSomething(text, cursor, rightEnd)
	}
	if cursor <= keyafter.start+p.keyafterMax {
		keyafter.end = cursor
	}
	keyafterTrunc := trunc && keyafter.end < rightEnd
	keyafter.end = skipWhiteBack(keyafter.end, keyafter.start)

	var leftField int
	if o.key-o.left > p.halfWidth+p.maxWord {
		leftField = o.key - (p.halfWidth + p.maxWord)
		leftField = p.skipSomething(text, leftField, o.key)
	} else {
		leftField = o.left
	}

	before := field{text: text, start: leftField, end: skipWhiteBack(o.key, leftField)}
	for before.start+p.beforeMax < before.end {
		before.start = p.skipSomething(text, before.start, before.end)
	}
	beforeTrunc := false
	if trunc {
		c := skipWhiteBack(before.start, 0)
		beforeTrunc = c > leftStart
	}
	before.start = skipWhite(before.start, end)

	tail := field{text: text}
	tailTrunc := false
	if tailMax := p.beforeMax - before.width() - p.gap; tailMax > 0 {
		tail.start = skipWhite(keyafter.end, end)
		tail.end = tail.start
		c := tail.end
		for c < rightEnd && c < tail.start+tailMax {
			tail.end = c
			c = p.skipSomething(text, c, rightEnd)
		}
		if c < tail.start+tailMax {
			tail.end = c
		}
		if tail.end > tail.start {
			keyafterTrunc = false
			tailTrunc = trunc && tail.end < rightEnd
		}
		tail.end = skipWhiteBack(tail.end, tail.start)
	}

	head := field{text: text}
	headTrunc := false
	if headMax := p.keyafterMax - keyafter.width() - p.gap; headMax > 0 {
		head.end = skipWhiteBack(before.start, 0)
		head.start = leftField
		for head.start+headMax < head.end {
			head.start = p.skipSomething(text, head.start, head.end)
		}
		if head.end > head.start {
			beforeTrunc = false
			headTrunc = trunc && head.start > leftStart
		}
		head.start = skipWhite(head.start, head.end)
	}

	var reference field
	switch {
	case p.AutoReference:
		ordinal := o.reference + 1
		if o.file > 0 {
			ordinal -= p.lineCounts[o.file-1]
		}
		r := []byte(p.names[o.file] + ":" + strconv.FormatInt(ordinal, 10))
		reference = field{text: r, end: len(r)}
	case p.References:
		reference = field{text: text, start: o.key + int(o.reference)}
		reference.end = reference.start
		for reference.end < rightEnd && !space(text[reference.end]) {
			reference.end++
		}
	}
	refs := p.AutoReference || p.References
	tl := func(b bool) int {
		if b {
			return len(p.trunc)
		}
		return 0
	}

	switch p.format {
	case Roff:
		fmt.Fprintf(p.out, ".%s \"", p.macro)
		p.printField(tail)
		if tailTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		_, _ = p.out.WriteString("\" \"")
		if beforeTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		p.printField(before)
		_, _ = p.out.WriteString("\" \"")
		p.printField(keyafter)
		if keyafterTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		_, _ = p.out.WriteString("\" \"")
		if headTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		p.printField(head)
		_ = p.out.WriteByte('"')
		if refs {
			_, _ = p.out.WriteString(" \"")
			p.printField(reference)
			_ = p.out.WriteByte('"')
		}
		_ = p.out.WriteByte('\n')
	case TeX:
		fmt.Fprintf(p.out, "\\%s {", p.macro)
		p.printField(tail)
		_, _ = p.out.WriteString("}{")
		p.printField(before)
		_, _ = p.out.WriteString("}{")
		split := p.skipSomething(text, keyafter.start, keyafter.end)
		p.printField(field{text: text, start: keyafter.start, end: split})
		_, _ = p.out.WriteString("}{")
		p.printField(field{text: text, start: split, end: keyafter.end})
		_, _ = p.out.WriteString("}{")
		p.printField(head)
		_ = p.out.WriteByte('}')
		if refs {
			_ = p.out.WriteByte('{')
			p.printField(reference)
			_ = p.out.WriteByte('}')
		}
		_ = p.out.WriteByte('\n')
	default:
		if !p.RightSideRefs {
			switch {
			case p.AutoReference:
				p.printField(reference)
				_ = p.out.WriteByte(':')
				p.spaces(p.refMax + p.gap - reference.width() - 1)
			case refs:
				p.printField(reference)
				p.spaces(p.refMax + p.gap - reference.width())
			default:
				p.printField(reference)
				p.spaces(p.refMax + p.gap - reference.width())
			}
		}
		if tail.start < tail.end {
			p.printField(tail)
			if tailTrunc {
				_, _ = p.out.WriteString(p.trunc)
			}
			p.spaces(p.halfWidth - p.gap - before.width() - tl(beforeTrunc) - tail.width() - tl(tailTrunc))
		} else {
			p.spaces(p.halfWidth - p.gap - before.width() - tl(beforeTrunc))
		}
		if beforeTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		p.printField(before)
		p.spaces(p.gap)
		p.printField(keyafter)
		if keyafterTrunc {
			_, _ = p.out.WriteString(p.trunc)
		}
		if head.start < head.end {
			p.spaces(p.halfWidth - keyafter.width() - tl(keyafterTrunc) - head.width() - tl(headTrunc))
			if headTrunc {
				_, _ = p.out.WriteString(p.trunc)
			}
			p.printField(head)
		} else if refs && p.RightSideRefs {
			p.spaces(p.halfWidth - keyafter.width() - tl(keyafterTrunc))
		}
		if refs && p.RightSideRefs {
			p.spaces(p.gap)
			p.printField(reference)
		}
		_ = p.out.WriteByte('\n')
	}
}

// errBackReference reports a pattern Go's regexp cannot run.
var errBackReference = errors.New("Back references are not supported") //nolint:revive,staticcheck // shown as a regex message
