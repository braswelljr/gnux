package grep

import (
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/internal/regex"
)

// translatePerl rewrites a Perl-compatible pattern (-P) into the Go syntax of
// core/internal/regex. Go's syntax is Perl's own for nearly everything, so
// most of a pattern passes through; what is rewritten is what Go spells
// differently or lacks: back references (\1, \g{N}, \g{-N}, \k<name>),
// lookaround ((?=...), (?!...), (?<=...), (?<!...)), \h \H \v \V \R \N \Z \e
// \cX, and named groups, which are numbered as plain ones. offset is the
// number of groups in the patterns before this one, and groups is the number
// this one has, so that joined patterns keep one numbering.
//
// What Go cannot run is refused with a message naming it: possessive and
// atomic groups, recursion, conditionals, \K, \G, \X and the like.
func translatePerl(p string, offset int) (expr string, groups int, err error) {
	t := perlTranslator{p: p, offset: offset, names: map[string]int{}}
	expr, err = t.run()
	return expr, t.groups, err
}

type perlTranslator struct {
	p      string
	offset int
	groups int
	names  map[string]int
	// looks holds, for each open parenthesis, the kind of lookaround it
	// starts, or "" for any other group, so that its ) knows what to close.
	looks []string
	// quantified says the last thing written was a quantifier, after which
	// a + would make it possessive.
	quantified bool
}

func unsupportedPerl(construct string) error {
	return &patternError{"the Perl construct '" + construct + "' is not supported"}
}

func (t *perlTranslator) run() (string, error) {
	p := t.p
	var b strings.Builder
	for i := 0; i < len(p); {
		c := p[i]
		quantified := false
		switch {
		case c == '\\':
			n, err := t.escape(&b, i)
			if err != nil {
				return "", err
			}
			i = n
		case c == '[':
			n, err := perlClass(&b, p, i)
			if err != nil {
				return "", err
			}
			i = n
		case c == '(':
			n, err := t.open(&b, i)
			if err != nil {
				return "", err
			}
			i = n
		case c == ')':
			if len(t.looks) == 0 {
				return "", errUnmatchedClose
			}
			t.looks = t.looks[:len(t.looks)-1]
			b.WriteByte(')')
			i++
		case c == '*' || c == '+' || c == '?' || c == '{' && isQuantifier(p[i:]):
			if t.quantified {
				// Go refuses a** too; a quantifier's own + or ? is
				// consumed below, with the quantifier.
				return "", errPreceding
			}
			end := i + 1
			if c == '{' {
				end = i + strings.IndexByte(p[i:], '}') + 1
			}
			b.WriteString(p[i:end])
			i = end
			switch {
			case i < len(p) && p[i] == '+':
				return "", unsupportedPerl("possessive quantifier " + p[end-1:i+1])
			case i < len(p) && p[i] == '?':
				b.WriteByte('?') // lazy, which Go has
				i++
			}
			quantified = true
		default:
			b.WriteByte(c)
			i++
		}
		t.quantified = quantified
	}
	if len(t.looks) > 0 {
		return "", errUnmatchedOpen
	}
	return b.String(), nil
}

// isQuantifier reports whether s starts with a {n}, {n,} or {n,m}; any other
// brace is a literal one in Perl.
func isQuantifier(s string) bool {
	end := strings.IndexByte(s, '}')
	if end < 0 {
		return false
	}
	low, high, _ := strings.Cut(s[1:end], ",")
	return low != "" && digits(low) && digits(high)
}

// open handles the ( at p[i] and returns the index after what it consumed.
func (t *perlTranslator) open(b *strings.Builder, i int) (int, error) {
	p := t.p
	rest := p[i:]
	lookaround := func(kind regex.Lookaround, width int) (int, error) {
		// regex.Look wraps a body; the body is written as the pattern
		// goes, so the opening half is written here and ) closes it.
		open := regex.Look(kind, "")
		b.WriteString(open[:len(open)-1])
		t.looks = append(t.looks, string(kind))
		return i + width, nil
	}
	switch {
	case strings.HasPrefix(rest, "(?="):
		return lookaround(regex.Ahead, 3)
	case strings.HasPrefix(rest, "(?!"):
		return lookaround(regex.NotAhead, 3)
	case strings.HasPrefix(rest, "(?<="):
		return lookaround(regex.Behind, 4)
	case strings.HasPrefix(rest, "(?<!"):
		return lookaround(regex.NotBehind, 4)
	case strings.HasPrefix(rest, "(?#"):
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return 0, errUnmatchedOpen
		}
		return i + end + 1, nil
	case strings.HasPrefix(rest, "(?P<"), strings.HasPrefix(rest, "(?<"), strings.HasPrefix(rest, "(?'"):
		start := strings.IndexAny(rest, "<'") + 1
		closer := byte('>')
		if rest[start-1] == '\'' {
			closer = '\''
		}
		end := strings.IndexByte(rest[start:], closer)
		if end < 0 {
			return 0, errInvalid
		}
		name := rest[start : start+end]
		t.groups++
		t.names[name] = t.groups
		t.looks = append(t.looks, "")
		b.WriteByte('(')
		return i + start + end + 1, nil
	case strings.HasPrefix(rest, "(?>"):
		return 0, unsupportedPerl("(?>")
	case strings.HasPrefix(rest, "(?|"):
		return 0, unsupportedPerl("(?|")
	case strings.HasPrefix(rest, "(?("):
		return 0, unsupportedPerl("(?(")
	case strings.HasPrefix(rest, "(?R"), strings.HasPrefix(rest, "(?&"), strings.HasPrefix(rest, "(?P>"),
		len(rest) > 2 && rest[1] == '?' && (rest[2] >= '0' && rest[2] <= '9' || rest[2] == '+' || rest[2] == '-' && len(rest) > 3 && rest[3] >= '0' && rest[3] <= '9'):
		return 0, unsupportedPerl("recursion")
	case strings.HasPrefix(rest, "(*"):
		return 0, unsupportedPerl("(*")
	case strings.HasPrefix(rest, "(?"):
		// Flags, (?i) or (?i:...), and (?:...): Go's own syntax. A flag
		// group with no colon closes at once and opens nothing.
		end := strings.IndexAny(rest, ":)")
		if end < 0 {
			return 0, errUnmatchedOpen
		}
		b.WriteString(rest[:end+1])
		if rest[end] == ':' {
			t.looks = append(t.looks, "")
		}
		return i + end + 1, nil
	}
	t.groups++
	t.looks = append(t.looks, "")
	b.WriteByte('(')
	return i + 1, nil
}

// The Perl escapes Go spells differently.
const (
	perlHSpace = `\t \x{a0}\x{1680}\x{180e}\x{2000}-\x{200a}\x{202f}\x{205f}\x{3000}`
	perlVSpace = `\n\x0b\f\r\x{85}\x{2028}\x{2029}`
)

// escape handles the backslash at p[i] and returns the index after what it
// consumed.
func (t *perlTranslator) escape(b *strings.Builder, i int) (int, error) {
	p := t.p
	if i+1 >= len(p) {
		return 0, errTrailingBackslash
	}
	d := p[i+1]
	backRef := func(n, next int) (int, error) {
		if n < 1 || n > t.groups {
			return 0, errBackReference
		}
		b.WriteString(regex.BackRef(t.offset + n))
		return next, nil
	}
	switch d {
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return backRef(int(d-'0'), i+2)
	case 'g':
		body, next := p[i+2:], i+2
		if strings.HasPrefix(body, "{") {
			end := strings.IndexByte(body, '}')
			if end < 0 {
				return 0, errInvalid
			}
			body, next = body[1:end], next+end+1
		} else {
			j := 0
			for j < len(body) && (body[j] >= '0' && body[j] <= '9' || j == 0 && body[j] == '-') {
				j++
			}
			body, next = body[:j], next+j
		}
		if n, ok := t.names[body]; ok {
			return backRef(n, next)
		}
		n, err := strconv.Atoi(body)
		if err != nil {
			return 0, errInvalid
		}
		if n < 0 {
			n = t.groups + 1 + n
		}
		return backRef(n, next)
	case 'k':
		if i+2 >= len(p) {
			return 0, errInvalid
		}
		closer := map[byte]byte{'<': '>', '{': '}', '\'': '\''}[p[i+2]]
		end := strings.IndexByte(p[i+3:], closer)
		if closer == 0 || end < 0 {
			return 0, errInvalid
		}
		n, ok := t.names[p[i+3:i+3+end]]
		if !ok {
			return 0, errBackReference
		}
		return backRef(n, i+3+end+1)
	case 'Q':
		// Everything up to \E is literal; Go knows \Q...\E, but the text in
		// between must not be read as syntax here either.
		end := strings.Index(p[i+2:], `\E`)
		if end < 0 {
			b.WriteString(p[i:])
			return len(p), nil
		}
		b.WriteString(p[i : i+2+end+2])
		return i + 2 + end + 2, nil
	case 'h':
		b.WriteString("[" + perlHSpace + "]")
	case 'H':
		b.WriteString("[^" + perlHSpace + "]")
	case 'v':
		b.WriteString("[" + perlVSpace + "]")
	case 'V':
		b.WriteString("[^" + perlVSpace + "]")
	case 'R':
		b.WriteString(`(?:\r\n|[` + perlVSpace + `])`)
	case 'N':
		b.WriteString(`[^\n]`)
	case 'Z':
		b.WriteString(`(?:\n?\z)`)
	case 'e':
		b.WriteString(`\x1b`)
	case 'c':
		if i+2 >= len(p) {
			return 0, errTrailingBackslash
		}
		c := p[i+2]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b.WriteString(`\x{`)
		b.WriteString(strconv.FormatInt(int64(c^0x40), 16))
		b.WriteString(`}`)
		return i + 3, nil
	case 'K', 'G', 'X', 'C':
		return 0, unsupportedPerl(`\` + string(d))
	default:
		b.WriteString(p[i : i+2])
	}
	return i + 2, nil
}

// perlClass copies the bracket expression at p[i] and returns the index
// after it. Inside one, \h and \v are rewritten as outside; the rest is Go's
// syntax already.
func perlClass(b *strings.Builder, p string, i int) (int, error) {
	j := i + 1
	b.WriteByte('[')
	if j < len(p) && p[j] == '^' {
		b.WriteByte('^')
		j++
	}
	if j < len(p) && p[j] == ']' {
		// A ] first is a literal one.
		b.WriteString(`\]`)
		j++
	}
	for j < len(p) {
		switch c := p[j]; {
		case c == ']':
			b.WriteByte(']')
			return j + 1, nil
		case c == '[' && strings.HasPrefix(p[j:], "[:"):
			end := strings.Index(p[j:], ":]")
			if end < 0 {
				return 0, errUnmatchedBracket
			}
			b.WriteString(p[j : j+end+2])
			j += end + 2
		case c == '\\' && j+1 < len(p):
			switch p[j+1] {
			case 'h':
				b.WriteString(perlHSpace)
			case 'v':
				b.WriteString(perlVSpace)
			case 'e':
				b.WriteString(`\x1b`)
			default:
				b.WriteString(p[j : j+2])
			}
			j += 2
		default:
			b.WriteByte(c)
			j++
		}
	}
	return 0, errUnmatchedBracket
}
