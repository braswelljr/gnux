package grep

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// span is one match inside a line: line[s:e].
type span struct{ s, e int }

// matcher decides whether a line matches the patterns, and where.
type matcher interface {
	// match reports whether line matches any pattern.
	match(line []byte) bool
	// spans appends the non-empty matches in line, left to right, to dst.
	// It is what -o prints and --color highlights.
	spans(line []byte, dst []span) []span
}

// compile builds the matcher for the pattern list. An empty list (an empty
// -f file) matches nothing.
func (g *Grep) compile(patterns []string) (matcher, error) {
	if len(patterns) == 0 {
		return noMatcher{}, nil
	}
	if g.Syntax == SyntaxFixed && !g.IgnoreCase {
		return newLiteral(patterns, g.WordRegexp, g.LineRegexp), nil
	}

	flags := "(?s)"
	if g.IgnoreCase {
		flags = "(?si)"
	}
	var (
		exprs    = make([]string, 0, len(patterns))
		literals = make([]string, 0, len(patterns))
		seen     = map[string]bool{}
		prefix   string
	)
	warn := func(msg string) {
		// One warning per distinct message, however many patterns earn it.
		if !seen[msg] {
			seen[msg] = true
			g.warnf("warning: %s", msg)
		}
	}
	for _, p := range patterns {
		var expr string
		if g.Syntax == SyntaxFixed {
			expr = regexp.QuoteMeta(p)
		} else {
			var err error
			if expr, err = translate(p, g.Syntax == SyntaxExtended, warn); err != nil {
				return nil, err
			}
		}
		// Compiled alone first, so that a mistake is reported for the pattern
		// that holds it, and to learn what the pattern is made of.
		re, err := regexp.Compile(flags + expr)
		if err != nil {
			return nil, compileError(err)
		}
		if lit, ok := literalOf(flags + expr); ok {
			literals = append(literals, lit)
		}
		prefix, _ = re.LiteralPrefix()
		exprs = append(exprs, expr)
	}
	if len(literals) == len(patterns) {
		return newLiteral(literals, g.WordRegexp, g.LineRegexp), nil
	}

	inner := exprs[0]
	if len(exprs) > 1 {
		inner = "(?:" + strings.Join(exprs, ")|(?:") + ")"
		// A literal every match starts with is only known for one pattern.
		prefix = ""
	}
	m := &regexpMatcher{prefix: []byte(prefix), word: g.WordRegexp, line: g.LineRegexp}
	var err error
	switch {
	case g.LineRegexp:
		m.re, err = regexp.Compile(flags + "^(?:" + inner + ")$")
	case g.WordRegexp:
		tail := "(" + inner + ")(?:" + nonWordClass + "|$)"
		if m.re, err = regexp.Compile(flags + "(?:^|" + nonWordClass + ")" + tail); err == nil {
			m.next, err = regexp.Compile(flags + nonWordClass + tail)
		}
	default:
		m.re, err = regexp.Compile(flags + inner)
	}
	if err != nil {
		return nil, compileError(err)
	}
	// POSIX takes the longest of the leftmost matches.
	m.re.Longest()
	if m.next != nil {
		m.next.Longest()
	}
	return m, nil
}

// noMatcher matches no line: the pattern list was empty.
type noMatcher struct{}

func (noMatcher) match([]byte) bool                 { return false }
func (noMatcher) spans(_ []byte, dst []span) []span { return dst }

// literalMatcher searches for fixed strings: the patterns of -F, and any
// regular expression that turns out to be plain text.
type literalMatcher struct {
	patterns [][]byte
	// whole indexes the patterns for -x, where a line is looked up, not
	// searched.
	whole map[string]struct{}
	word  bool
	line  bool
}

func newLiteral(patterns []string, word, line bool) *literalMatcher {
	m := &literalMatcher{word: word, line: line}
	if line {
		m.whole = make(map[string]struct{}, len(patterns))
	}
	for _, p := range patterns {
		if line {
			m.whole[p] = struct{}{}
			continue
		}
		m.patterns = append(m.patterns, []byte(p))
	}
	return m
}

func (m *literalMatcher) match(line []byte) bool {
	if m.line {
		_, ok := m.whole[string(line)] // no allocation: a map lookup by converted key
		return ok
	}
	if !m.word {
		for _, p := range m.patterns {
			if bytes.Contains(line, p) {
				return true
			}
		}
		return false
	}
	_, ok := m.find(line, 0)
	return ok
}

// find returns the leftmost match at or after from, the longest one where
// several patterns match there.
func (m *literalMatcher) find(line []byte, from int) (span, bool) {
	best := span{-1, -1}
	for _, p := range m.patterns {
		for pos := from; pos <= len(line); {
			i := bytes.Index(line[pos:], p)
			if i < 0 {
				break
			}
			s := pos + i
			e := s + len(p)
			if best.s >= 0 && s > best.s {
				break
			}
			if !m.word || wordBounded(line, s, e) {
				if best.s < 0 || s < best.s || e > best.e {
					best = span{s, e}
				}
				break
			}
			// Not a whole word here: the same pattern may be one further on.
			pos = s + 1
		}
	}
	return best, best.s >= 0
}

func (m *literalMatcher) spans(line []byte, dst []span) []span {
	if m.line {
		if len(line) > 0 && m.match(line) {
			dst = append(dst, span{0, len(line)})
		}
		return dst
	}
	for from := 0; from <= len(line); {
		sp, ok := m.find(line, from)
		if !ok {
			break
		}
		if sp.e > sp.s {
			dst = append(dst, sp)
			from = sp.e
		} else {
			from = sp.e + 1
		}
	}
	return dst
}

// wordBounded reports whether line[s:e] stands as a whole word: no word
// character touches it on either side.
func wordBounded(line []byte, s, e int) bool {
	if s > 0 {
		if r, _ := utf8.DecodeLastRune(line[:s]); isWord(r) {
			return false
		}
	}
	if e < len(line) {
		if r, _ := utf8.DecodeRune(line[e:]); isWord(r) {
			return false
		}
	}
	return true
}

// isWord matches wordClass.
func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// regexpMatcher searches with a compiled regular expression.
type regexpMatcher struct {
	// re selects lines. For -x it is anchored at both ends; for -w the
	// pattern is its first group, between what must surround a word.
	re *regexp.Regexp
	// next is the -w expression for a search that resumes inside a line. Go
	// cannot be told what precedes the text it searches, so the search
	// resumes one character early and next requires that character to be a
	// non-word one, where re would also accept the start of the text.
	next *regexp.Regexp
	// prefix is a literal every match contains. Looking for it first costs
	// far less than starting the regexp engine on a line that cannot match.
	prefix []byte
	word   bool
	line   bool
}

func (m *regexpMatcher) match(line []byte) bool {
	if len(m.prefix) > 0 && !bytes.Contains(line, m.prefix) {
		return false
	}
	return m.re.Match(line)
}

func (m *regexpMatcher) spans(line []byte, dst []span) []span {
	if len(m.prefix) > 0 && !bytes.Contains(line, m.prefix) {
		return dst
	}
	switch {
	case m.line:
		if len(line) > 0 && m.re.Match(line) {
			dst = append(dst, span{0, len(line)})
		}
	case m.word:
		loc := m.re.FindSubmatchIndex(line)
		base := 0
		for loc != nil {
			s, e := base+loc[2], base+loc[3]
			from := e
			if e > s {
				dst = append(dst, span{s, e})
			} else {
				from = e + 1
			}
			if from > len(line) {
				break
			}
			_, size := utf8.DecodeLastRune(line[:from])
			base = from - size
			loc = m.next.FindSubmatchIndex(line[base:])
		}
	default:
		for _, loc := range m.re.FindAllIndex(line, -1) {
			if loc[1] > loc[0] {
				dst = append(dst, span{loc[0], loc[1]})
			}
		}
	}
	return dst
}
