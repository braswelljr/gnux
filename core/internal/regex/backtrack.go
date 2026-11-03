package regex

import (
	"bytes"
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The step budget of one search: a fixed allowance plus some per byte of
// input. A back-reference pattern can take exponential time on a crafted
// line (GNU's engine too); past the budget the search is abandoned and Err
// says so, rather than the tool appearing to hang.
const (
	stepBase    = 1 << 22
	stepPerByte = 256
)

type opcode uint8

const (
	opEmpty opcode = iota
	opFail
	opLiteral
	opClass
	opAnyNotNL
	opAny
	opAssert
	opCapture
	opBackRef
	opConcat
	opAlternate
	opRepeat
	opLook
)

// node is one piece of a compiled pattern.
type node struct {
	op     opcode
	runes  []rune // opLiteral: the text; opClass: lo-hi pairs
	fold   bool   // opLiteral, opBackRef: compare ignoring case
	assert syntax.EmptyOp
	look   Lookaround
	cap    int // opCapture: its index; opBackRef: the group it repeats
	subs   []*node
	min    int
	max    int // -1: unbounded
	greedy bool
}

// program is a pattern compiled for the backtracking matcher.
type program struct {
	root     *node
	ncap     int   // Go's capture count, back-reference groups included
	slot     []int // Go capture index -> reported group number; 0 for none
	groups   int   // reported groups
	names    map[string]int
	backRefs int
	looks    int
	anchored bool // every match starts at the start of the text
	longest  bool
}

// newProgram compiles a parsed pattern. Groups are numbered by their opening
// parenthesis, which is Go's capture order; the back-reference groups are
// left out of the numbering.
func newProgram(tree *syntax.Regexp) (*program, error) {
	p := &program{ncap: tree.MaxCap(), names: map[string]int{}}
	p.slot = make([]int, p.ncap+1)
	var unnamed []int // Go capture index of each unnamed group, in order
	var refs []*node
	var refNums []int
	var refCaps []int

	var conv func(re *syntax.Regexp) *node
	conv = func(re *syntax.Regexp) *node {
		n := &node{}
		switch re.Op {
		case syntax.OpNoMatch:
			n.op = opFail
		case syntax.OpEmptyMatch:
			n.op = opEmpty
		case syntax.OpLiteral:
			n.op, n.runes, n.fold = opLiteral, re.Rune, re.Flags&syntax.FoldCase != 0
		case syntax.OpCharClass:
			n.op, n.runes = opClass, re.Rune
		case syntax.OpAnyCharNotNL:
			n.op = opAnyNotNL
		case syntax.OpAnyChar:
			n.op = opAny
		case syntax.OpBeginLine:
			n.op, n.assert = opAssert, syntax.EmptyBeginLine
		case syntax.OpEndLine:
			n.op, n.assert = opAssert, syntax.EmptyEndLine
		case syntax.OpBeginText:
			n.op, n.assert = opAssert, syntax.EmptyBeginText
		case syntax.OpEndText:
			n.op, n.assert = opAssert, syntax.EmptyEndText
		case syntax.OpWordBoundary:
			n.op, n.assert = opAssert, syntax.EmptyWordBoundary
		case syntax.OpNoWordBoundary:
			n.op, n.assert = opAssert, syntax.EmptyNoWordBoundary
		case syntax.OpCapture:
			if num, ok := strings.CutPrefix(re.Name, backRefName); ok {
				digit, _, _ := strings.Cut(num, "_")
				k, _ := strconv.Atoi(digit)
				n.op, n.fold = opBackRef, re.Flags&syntax.FoldCase != 0
				refs, refNums, refCaps = append(refs, n), append(refNums, k), append(refCaps, re.Cap)
				p.backRefs++
				return n
			}
			for _, kind := range []Lookaround{NotAhead, NotBehind, Ahead, Behind} {
				if strings.HasPrefix(re.Name, string(kind)+"_") {
					p.looks++
					n.op, n.look = opLook, kind
					n.subs = []*node{conv(re.Sub[0])}
					return n
				}
			}
			p.groups++
			p.slot[re.Cap] = p.groups
			if re.Name == "" {
				unnamed = append(unnamed, re.Cap)
			} else {
				p.names[re.Name] = p.groups
			}
			n.op, n.cap = opCapture, re.Cap
			n.subs = []*node{conv(re.Sub[0])}
		case syntax.OpConcat, syntax.OpAlternate:
			n.op = opConcat
			if re.Op == syntax.OpAlternate {
				n.op = opAlternate
			}
			for _, s := range re.Sub {
				n.subs = append(n.subs, conv(s))
			}
		case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
			n.op = opRepeat
			n.greedy = re.Flags&syntax.NonGreedy == 0
			switch re.Op {
			case syntax.OpStar:
				n.min, n.max = 0, -1
			case syntax.OpPlus:
				n.min, n.max = 1, -1
			case syntax.OpQuest:
				n.min, n.max = 0, 1
			default:
				n.min, n.max = re.Min, re.Max
			}
			n.subs = []*node{conv(re.Sub[0])}
		default:
			n.op = opFail
		}
		return n
	}
	p.root = conv(tree)

	// A reference must name a group that opens before it.
	for i, ref := range refs {
		k := refNums[i]
		if k < 1 || k > len(unnamed) || unnamed[k-1] > refCaps[i] {
			return nil, ErrBackReference
		}
		ref.cap = unnamed[k-1]
	}

	first := p.root
	for first.op == opConcat && len(first.subs) > 0 {
		first = first.subs[0]
	}
	p.anchored = first.op == opAssert && first.assert == syntax.EmptyBeginText
	return p, nil
}

func (p *program) named(name string) int {
	if i, ok := p.names[name]; ok {
		return i
	}
	return -1
}

// machine is the state of one search.
type machine struct {
	in        []byte
	caps      []int
	steps     int
	limit     int
	abandoned bool
}

// search finds the first match starting at or after pos: the longest one
// there when the program is set to Longest, otherwise the first found in
// Perl's order of preference. ok is false when the search was abandoned.
func (p *program) search(in []byte, pos int, groups bool) (loc []int, ok bool) {
	m := &machine{in: in, caps: make([]int, 2*(p.ncap+1)), limit: stepBase + stepPerByte*len(in)}
	best := make([]int, len(m.caps))
	for start := pos; start <= len(in); start += runeWidth(in, start) {
		if p.anchored && start > 0 {
			break
		}
		for i := range m.caps {
			m.caps[i] = -1
		}
		found := false
		m.run(p.root, start, func(end int) bool {
			if !found || end > best[1] {
				found = true
				copy(best, m.caps)
				best[0], best[1] = start, end
			}
			// Only a match that reaches the end cannot be beaten.
			return !p.longest || end == len(in)
		})
		if m.abandoned {
			return nil, false
		}
		if found {
			return p.report(best, groups), true
		}
	}
	return nil, true
}

// report lays a match out as regexp does: the whole match, then each group,
// -1 for one that took no part.
func (p *program) report(caps []int, groups bool) []int {
	if !groups {
		return []int{caps[0], caps[1]}
	}
	out := make([]int, 2*(p.groups+1))
	for i := range out {
		out[i] = -1
	}
	out[0], out[1] = caps[0], caps[1]
	for c := 1; c <= p.ncap; c++ {
		if s := p.slot[c]; s > 0 {
			out[2*s], out[2*s+1] = caps[2*c], caps[2*c+1]
		}
	}
	return out
}

// run matches n at i, then calls k with the position after it, and goes on
// to the next way of matching n each time k returns false. It returns true
// once k has (or the search was abandoned, which ends it too).
func (m *machine) run(n *node, i int, k func(int) bool) bool {
	m.steps++
	if m.steps > m.limit {
		m.abandoned = true
	}
	if m.abandoned {
		return true
	}
	in := m.in
	switch n.op {
	case opEmpty:
		return k(i)
	case opFail:
		return false
	case opLiteral:
		for _, r := range n.runes {
			if i >= len(in) {
				return false
			}
			c, w := decode(in, i)
			if c != r && (!n.fold || !foldEqual(c, r)) {
				return false
			}
			i += w
		}
		return k(i)
	case opClass:
		if i >= len(in) {
			return false
		}
		c, w := decode(in, i)
		if !inClass(n.runes, c) {
			return false
		}
		return k(i + w)
	case opAnyNotNL:
		if i >= len(in) || in[i] == '\n' {
			return false
		}
		return k(i + runeWidth(in, i))
	case opAny:
		if i >= len(in) {
			return false
		}
		return k(i + runeWidth(in, i))
	case opAssert:
		if !m.assert(n.assert, i) {
			return false
		}
		return k(i)
	case opCapture:
		c := n.cap
		oldStart, oldEnd := m.caps[2*c], m.caps[2*c+1]
		m.caps[2*c] = i
		if m.run(n.subs[0], i, func(j int) bool {
			saved := m.caps[2*c+1]
			m.caps[2*c+1] = j
			if k(j) {
				return true
			}
			m.caps[2*c+1] = saved
			return false
		}) {
			return true
		}
		m.caps[2*c], m.caps[2*c+1] = oldStart, oldEnd
		return false
	case opBackRef:
		s, e := m.caps[2*n.cap], m.caps[2*n.cap+1]
		if s < 0 || e < s {
			return false
		}
		want := in[s:e]
		if !n.fold {
			if !bytes.HasPrefix(in[i:], want) {
				return false
			}
			return k(i + len(want))
		}
		width, ok := foldPrefix(in[i:], want)
		if !ok {
			return false
		}
		return k(i + width)
	case opConcat:
		return m.seq(n.subs, i, k)
	case opAlternate:
		for _, s := range n.subs {
			if m.run(s, i, k) {
				return true
			}
		}
		return false
	case opRepeat:
		return m.repeat(n, i, 0, k)
	case opLook:
		return m.look(n, i, k)
	}
	return false
}

// look checks a lookaround assertion at i, then goes on with k. The groups a
// positive one sets stay set for what follows, as in Perl, and are put back
// if that fails.
func (m *machine) look(n *node, i int, k func(int) bool) bool {
	saved := append([]int(nil), m.caps...)
	found := false
	switch n.look {
	case Ahead, NotAhead:
		found = m.run(n.subs[0], i, func(int) bool { return true })
	default:
		// The assertion holds if the body matches some text that ends
		// exactly at i.
		for j := i; j >= 0 && !found && !m.abandoned; j-- {
			if j < i && !utf8.RuneStart(m.in[j]) {
				continue // the middle of a character
			}
			found = m.run(n.subs[0], j, func(e int) bool { return e == i })
		}
	}
	if m.abandoned {
		return true
	}
	if n.look == NotAhead || n.look == NotBehind {
		copy(m.caps, saved)
		if found {
			return false
		}
		return k(i)
	}
	if !found {
		copy(m.caps, saved)
		return false
	}
	if k(i) {
		return true
	}
	copy(m.caps, saved)
	return false
}

func (m *machine) seq(subs []*node, i int, k func(int) bool) bool {
	if len(subs) == 0 {
		return k(i)
	}
	return m.run(subs[0], i, func(j int) bool { return m.seq(subs[1:], j, k) })
}

// repeat matches the remaining iterations of a repetition that has matched
// count times so far, ending at i.
func (m *machine) repeat(n *node, i, count int, k func(int) bool) bool {
	more := func() bool {
		if n.max >= 0 && count >= n.max {
			return false
		}
		return m.run(n.subs[0], i, func(j int) bool {
			// Once the minimum is met, an iteration that matched nothing
			// adds nothing, and repeating it would never end. As in Go and
			// POSIX, it is still taken as the only iteration, so (a*)* sets
			// its group to the empty string, but never after another one.
			if j == i && count >= n.min {
				if count == 0 {
					return k(j)
				}
				return false
			}
			return m.repeat(n, j, count+1, k)
		})
	}
	if count < n.min {
		return more()
	}
	if n.greedy {
		return more() || k(i)
	}
	return k(i) || more()
}

func (m *machine) assert(op syntax.EmptyOp, i int) bool {
	in := m.in
	switch op {
	case syntax.EmptyBeginLine:
		return i == 0 || in[i-1] == '\n'
	case syntax.EmptyEndLine:
		return i == len(in) || in[i] == '\n'
	case syntax.EmptyBeginText:
		return i == 0
	case syntax.EmptyEndText:
		return i == len(in)
	case syntax.EmptyWordBoundary, syntax.EmptyNoWordBoundary:
		before := i > 0 && isWordByte(in[i-1])
		after := i < len(in) && isWordByte(in[i])
		return (before != after) == (op == syntax.EmptyWordBoundary)
	}
	return false
}

// isWordByte is RE2's \w, which \b is defined by: ASCII only.
func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// decode reads the character at b[i]; an invalid byte is U+FFFD, one wide,
// as in regexp.
func decode(b []byte, i int) (rune, int) {
	if c := b[i]; c < utf8.RuneSelf {
		return rune(c), 1
	}
	return utf8.DecodeRune(b[i:])
}

func inClass(ranges []rune, c rune) bool {
	for j := 0; j+1 < len(ranges); j += 2 {
		if c < ranges[j] {
			return false
		}
		if c <= ranges[j+1] {
			return true
		}
	}
	return false
}

// foldEqual reports whether a and b are the same letter in another case.
func foldEqual(a, b rune) bool {
	if a == b {
		return true
	}
	for f := unicode.SimpleFold(a); f != a; f = unicode.SimpleFold(f) {
		if f == b {
			return true
		}
	}
	return false
}

// foldPrefix reports whether in starts with want, ignoring case, and how
// many bytes of in that takes.
func foldPrefix(in, want []byte) (int, bool) {
	i := 0
	for j := 0; j < len(want); {
		if i >= len(in) {
			return 0, false
		}
		a, wa := decode(in, i)
		b, wb := decode(want, j)
		if !foldEqual(a, b) {
			return 0, false
		}
		i, j = i+wa, j+wb
	}
	return i, true
}
