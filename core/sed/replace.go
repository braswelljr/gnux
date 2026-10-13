package sed

import (
	"unicode"
	"unicode/utf8"
)

// partKind says what one piece of a replacement is.
type partKind uint8

const (
	partLiteral partKind = iota // text copied as written
	partGroup                   // & or \N: what a group matched
	partCase                    // \L \U \E \l \u: a case conversion switch
)

// part is one piece of the right-hand side of an s command.
type part struct {
	kind  partKind
	text  []byte // partLiteral
	group int    // partGroup: 0 is the whole match
	op    byte   // partCase: one of L U E l u
}

// replacement is the parsed right-hand side of an s command.
type replacement struct {
	parts []part
	cased bool // a case conversion is used: take the slower path
}

// parseReplacement parses the right-hand side of an s command. groups is the
// number of groups in the regular expression, or -1 when it is the empty
// expression, whose groups are only known when the command runs. It returns
// the number of a group the expression does not have, or -1.
func parseReplacement(s string, groups int) (replacement, int) {
	var r replacement
	var text []byte
	flush := func() {
		if len(text) > 0 {
			r.parts = append(r.parts, part{kind: partLiteral, text: text})
			text = nil
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '&':
			flush()
			r.parts = append(r.parts, part{kind: partGroup})
		case c != '\\' || i+1 >= len(s):
			text = append(text, c)
		default:
			i++
			d := s[i]
			switch {
			case d >= '0' && d <= '9':
				n := int(d - '0')
				if groups >= 0 && n > groups {
					return r, n
				}
				flush()
				r.parts = append(r.parts, part{kind: partGroup, group: n})
			case d == 'L' || d == 'U' || d == 'E' || d == 'l' || d == 'u':
				flush()
				r.parts = append(r.parts, part{kind: partCase, op: d})
				r.cased = true
			default:
				if b, end, ok := charEscape(s, i); ok {
					text = append(text, b)
					i = end
				} else {
					text = append(text, d)
				}
			}
		}
	}
	flush()
	return r, -1
}

// expand appends the replacement for one match to dst. src is the pattern
// space and loc the match's group offsets, as regexp returns them.
func (r *replacement) expand(dst, src []byte, loc []int) []byte {
	if !r.cased {
		for i := range r.parts {
			p := &r.parts[i]
			if p.kind == partLiteral {
				dst = append(dst, p.text...)
			} else if g := 2 * p.group; g+1 < len(loc) && loc[g] >= 0 {
				dst = append(dst, src[loc[g]:loc[g+1]]...)
			}
		}
		return dst
	}
	// mode is the conversion \L or \U left on; once is the \l or \u waiting
	// for the next character, which outlives a group that matched nothing.
	var mode, once byte
	for i := range r.parts {
		p := &r.parts[i]
		switch p.kind {
		case partLiteral:
			dst = appendCased(dst, p.text, mode, &once)
		case partGroup:
			if g := 2 * p.group; g+1 < len(loc) && loc[g] >= 0 {
				dst = appendCased(dst, src[loc[g]:loc[g+1]], mode, &once)
			}
		case partCase:
			switch p.op {
			case 'L', 'U':
				mode = p.op
			case 'E':
				mode, once = 0, 0
			default:
				once = p.op
			}
		}
	}
	return dst
}

// appendCased appends s to dst converted as mode ('L', 'U' or 0) says, with
// its first character converted as *once ('l', 'u' or 0) says instead.
func appendCased(dst, s []byte, mode byte, once *byte) []byte {
	for len(s) > 0 {
		r, size := utf8.DecodeRune(s)
		if r == utf8.RuneError && size <= 1 {
			// Not UTF-8: the byte passes through untouched.
			dst = append(dst, s[0])
			s = s[1:]
			*once = 0
			continue
		}
		op := mode
		if *once != 0 {
			op = *once
			*once = 0
		}
		switch op {
		case 'L', 'l':
			r = unicode.ToLower(r)
		case 'U', 'u':
			r = unicode.ToUpper(r)
		}
		dst = utf8.AppendRune(dst, r)
		s = s[size:]
	}
	return dst
}

// translit is the table of a y command.
type translit struct {
	// bytes maps every byte when both strings are ASCII, which lets the
	// pattern space be rewritten in place.
	bytes *[256]byte
	runes map[rune]rune
}

// unescapeY resolves the escapes of one string of a y command: \\ is a
// backslash, the usual character escapes are their character, and a
// backslash before anything else stays.
func unescapeY(s string) []rune {
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			if s[i+1] == '\\' {
				out = append(out, '\\')
				i++
				continue
			}
			if b, end, ok := charEscape(s, i+1); ok {
				out = append(out, b)
				i = end
				continue
			}
		}
		out = append(out, s[i])
	}
	return []rune(string(out))
}

// newTranslit builds the table mapping each character of src to the one at
// the same place in dst. It reports false when their lengths differ.
func newTranslit(src, dst string) (*translit, bool) {
	from, to := unescapeY(src), unescapeY(dst)
	if len(from) != len(to) {
		return nil, false
	}
	ascii := true
	for i := range from {
		if from[i] >= utf8.RuneSelf || to[i] >= utf8.RuneSelf {
			ascii = false
		}
	}
	t := &translit{}
	if ascii {
		t.bytes = new([256]byte)
		for i := range t.bytes {
			t.bytes[i] = byte(i)
		}
		// The first of two mappings for the same character wins.
		for i := len(from) - 1; i >= 0; i-- {
			t.bytes[from[i]] = byte(to[i])
		}
		return t, true
	}
	t.runes = make(map[rune]rune, len(from))
	for i := len(from) - 1; i >= 0; i-- {
		t.runes[from[i]] = to[i]
	}
	return t, true
}

// apply transliterates ps, using scratch as working space, and returns the
// new pattern space and the buffer to keep as scratch.
func (t *translit) apply(ps, scratch []byte) (out, spare []byte) {
	if t.bytes != nil {
		for i, b := range ps {
			ps[i] = t.bytes[b]
		}
		return ps, scratch
	}
	dst := scratch[:0]
	for i := 0; i < len(ps); {
		r, size := utf8.DecodeRune(ps[i:])
		if to, ok := t.runes[r]; ok && (r != utf8.RuneError || size > 1) {
			dst = utf8.AppendRune(dst, to)
		} else {
			dst = append(dst, ps[i:i+size]...)
		}
		i += size
	}
	return dst, ps
}
