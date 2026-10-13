package sed

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"strings"
)

// This file turns the regular expressions of a sed script into Go's RE2
// syntax. core/internal/bre covers plain POSIX basic expressions; sed needs
// more than it offers (extended expressions with -E, the I and M modifiers,
// GNU's \< \> \` \' \cX \dNNN \oNNN \xHH escapes, anchors after \( and \|, a
// repetition of a repetition), so both dialects are translated here by one
// scanner.

// regexError is a pattern GNU sed would reject, carrying GNU's message.
type regexError string

func (e regexError) Error() string { return string(e) }

// The messages of the C library's regcomp, which GNU sed prints verbatim.
const (
	errTrailingBackslash = regexError("Trailing backslash")
	errUnmatchedParen    = regexError(`Unmatched ( or \(`)
	errUnmatchedClose    = regexError(`Unmatched ) or \)`)
	errUnmatchedBracket  = regexError("Unmatched [, [^, [:, [., or [=")
	errUnmatchedBrace    = regexError(`Unmatched \{`)
	errBadInterval       = regexError(`Invalid content of \{\}`)
	errBadPreceding      = regexError("Invalid preceding regular expression")
	errBadRange          = regexError("Invalid range end")
	errBadClass          = regexError("Invalid character class name")
	errBadCollation      = regexError("Invalid collation character")
	errTooBig            = regexError("Regular expression too big")
	errBadPattern        = regexError("Invalid regular expression")
	errBackReference     = regexError("back references in a regular expression are not supported")
)

// maxRepeat is the largest repetition count RE2 accepts.
const maxRepeat = 1000

// compileRegex compiles a sed regular expression: a basic one, or an
// extended one when extended is set. icase is the I modifier and multiline
// the M modifier. The result matches leftmost-longest, as POSIX requires.
func compileRegex(pattern string, extended, icase, multiline bool) (*regexp.Regexp, error) {
	t := translator{pat: pattern, ext: extended, ml: multiline, atom: -1}
	if err := t.run(); err != nil {
		return nil, err
	}
	// Without M, ^ and $ match at the ends of the pattern space only and a
	// dot matches a newline; with M they match at every line and it does not.
	flags := "(?s"
	if multiline {
		flags = "(?m"
	}
	if icase {
		flags += "i"
	}
	re, err := regexp.Compile(flags + ")" + string(t.out))
	if err != nil {
		return nil, regcompError(err)
	}
	re.Longest()
	return re, nil
}

// regcompError maps a Go syntax error to the message regcomp gives for the
// same mistake.
func regcompError(err error) error {
	var se *syntax.Error
	if !errors.As(err, &se) {
		return errBadPattern
	}
	switch se.Code {
	case syntax.ErrMissingParen:
		return errUnmatchedParen
	case syntax.ErrUnexpectedParen:
		return errUnmatchedClose
	case syntax.ErrMissingBracket:
		return errUnmatchedBracket
	case syntax.ErrMissingRepeatArgument, syntax.ErrInvalidRepeatOp:
		return errBadPreceding
	case syntax.ErrInvalidCharRange:
		return errBadRange
	case syntax.ErrInvalidRepeatSize, syntax.ErrLarge:
		return errTooBig
	case syntax.ErrTrailingBackslash:
		return errTrailingBackslash
	}
	return errBadPattern
}

// translator rewrites one pattern into RE2 syntax.
type translator struct {
	pat string
	ext bool // extended syntax
	ml  bool // the M modifier

	out []byte
	// atom is where in out the expression a repetition would apply to
	// starts, or -1 when there is none (the start of a branch), in which
	// case a repetition operator is an ordinary character.
	atom int
	// repeated says the atom already carries a repetition: POSIX allows
	// another one (a**, a+?) where RE2 either refuses it or reads a lazy
	// operator, so the atom is wrapped in a group first.
	repeated bool
	// branch is true at the start of the pattern, after an opening
	// parenthesis and after an alternation: where a basic ^ is an anchor.
	branch bool
	groups []int // where in out each open group starts
}

func (t *translator) run() error {
	p := t.pat
	t.branch = true
	for i := 0; i < len(p); i++ {
		c := p[i]
		wasBranch := t.branch
		t.branch = false
		switch {
		case c == '\\':
			if i+1 >= len(p) {
				return errTrailingBackslash
			}
			n, err := t.escape(i + 1)
			if err != nil {
				return err
			}
			i = n
		case c == '[':
			n, err := t.bracket(i)
			if err != nil {
				return err
			}
			i = n
		case c == '*':
			if t.atom < 0 {
				t.literal('*')
			} else {
				t.repeat("*")
			}
		case c == '.':
			t.begin()
			t.out = append(t.out, '.')
		case c == '^':
			if t.ext || wasBranch {
				// An anchor repeats nothing: a * after it is literal.
				t.out = append(t.out, '^')
				t.atom = -1
			} else {
				t.literal('^')
			}
		case c == '$':
			if t.ext || i == len(p)-1 || strings.HasPrefix(p[i+1:], `\)`) || strings.HasPrefix(p[i+1:], `\|`) {
				t.out = append(t.out, '$')
				t.atom = -1
			} else {
				t.literal('$')
			}
		case t.ext && c == '(':
			t.open()
		case t.ext && c == ')':
			if err := t.closeGroup(); err != nil {
				return err
			}
		case t.ext && c == '|':
			t.alternate()
		case t.ext && (c == '+' || c == '?'):
			if t.atom < 0 {
				t.literal(c)
			} else {
				t.repeat(string(c))
			}
		case t.ext && c == '{':
			// A brace that does not open a valid interval is literal.
			rep, n, ok := interval(p, i+1, "}")
			if !ok || t.atom < 0 {
				t.literal('{')
				break
			}
			if rep == "" {
				return errTooBig
			}
			t.repeat(rep)
			i = n
		default:
			t.literal(c)
		}
	}
	if len(t.groups) > 0 {
		return errUnmatchedParen
	}
	return nil
}

// escape translates the backslash sequence whose second character is at i
// and returns the index of its last character.
func (t *translator) escape(i int) (int, error) {
	p := t.pat
	d := p[i]
	if !t.ext {
		switch d {
		case '(':
			t.open()
			return i, nil
		case ')':
			return i, t.closeGroup()
		case '|':
			t.alternate()
			return i, nil
		case '+', '?':
			if t.atom < 0 {
				t.literal(d)
			} else {
				t.repeat(string(d))
			}
			return i, nil
		case '{':
			rep, n, ok := interval(p, i+1, `\}`)
			switch {
			case !ok && !strings.Contains(p[i+1:], `\}`):
				return 0, errUnmatchedBrace
			case !ok:
				return 0, errBadInterval
			case rep == "":
				return 0, errTooBig
			case t.atom < 0:
				return 0, errBadPreceding
			}
			t.repeat(rep)
			return n, nil
		}
	}
	switch d {
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return 0, errBackReference
	case 'w', 'W', 's', 'S', 'B':
		t.begin()
		t.out = append(t.out, '\\', d)
	case 'b', '<', '>':
		// RE2 has no start-of-word or end-of-word assertion; a word
		// boundary is the closest it offers.
		t.out = append(t.out, `\b`...)
		t.atom = -1
	case '`':
		t.out = append(t.out, `\A`...)
		t.atom = -1
	case '\'':
		t.out = append(t.out, `\z`...)
		t.atom = -1
	default:
		if b, n, ok := charEscape(p, i); ok {
			t.literal(b)
			return n, nil
		}
		t.literal(d)
	}
	return i, nil
}

// begin marks the start of a new atom.
func (t *translator) begin() {
	t.atom = len(t.out)
	t.repeated = false
}

// literal emits one byte of the pattern as an ordinary character.
func (t *translator) literal(b byte) {
	// The bytes after the first of a UTF-8 sequence belong to its atom.
	if b&0xC0 != 0x80 {
		t.begin()
	}
	t.out = appendLiteral(t.out, b)
}

func (t *translator) open() {
	t.groups = append(t.groups, len(t.out))
	t.out = append(t.out, '(')
	t.atom = -1
	t.branch = true
}

func (t *translator) closeGroup() error {
	if len(t.groups) == 0 {
		return errUnmatchedClose
	}
	t.out = append(t.out, ')')
	t.atom = t.groups[len(t.groups)-1]
	t.groups = t.groups[:len(t.groups)-1]
	t.repeated = false
	return nil
}

func (t *translator) alternate() {
	t.out = append(t.out, '|')
	t.atom = -1
	t.branch = true
}

// repeat applies the repetition operator op to the current atom.
func (t *translator) repeat(op string) {
	if t.repeated {
		wrapped := make([]byte, 0, len(t.out)+len(op)+4)
		wrapped = append(wrapped, t.out[:t.atom]...)
		wrapped = append(wrapped, "(?:"...)
		wrapped = append(wrapped, t.out[t.atom:]...)
		wrapped = append(wrapped, ')')
		t.out = wrapped
	}
	t.out = append(t.out, op...)
	t.repeated = true
}

// interval parses the body of an interval expression starting at i, up to
// and including closer. It returns the RE2 repetition and the index of the
// closer's last character; the repetition is empty when a count is more
// than RE2 allows.
func interval(p string, i int, closer string) (rep string, end int, ok bool) {
	number := func() (int, bool) {
		start, n := i, 0
		for i < len(p) && p[i] >= '0' && p[i] <= '9' {
			if n <= maxRepeat {
				n = n*10 + int(p[i]-'0')
			}
			i++
		}
		return n, i > start
	}
	lo, hasLo := number()
	hi, hasHi, comma := 0, false, false
	if i < len(p) && p[i] == ',' {
		comma = true
		i++
		hi, hasHi = number()
	}
	if !strings.HasPrefix(p[i:], closer) || (!hasLo && !hasHi) || (hasHi && hi < lo) {
		return "", 0, false
	}
	end = i + len(closer) - 1
	if lo > maxRepeat || hi > maxRepeat {
		return "", end, true
	}
	var b []byte
	b = append(b, '{')
	b = appendDecimal(b, lo)
	if comma {
		b = append(b, ',')
		if hasHi {
			b = appendDecimal(b, hi)
		}
	}
	b = append(b, '}')
	return string(b), end, true
}

func appendDecimal(b []byte, n int) []byte {
	if n >= 10 {
		b = appendDecimal(b, n/10)
	}
	return append(b, byte('0'+n%10))
}

// classNames are the POSIX character classes, all of which RE2 knows.
var classNames = map[string]bool{
	"alpha": true, "digit": true, "alnum": true, "upper": true, "lower": true, "space": true,
	"blank": true, "punct": true, "print": true, "graph": true, "cntrl": true, "xdigit": true,
}

// bracket translates the bracket expression opening at i and returns the
// index of its closing bracket.
func (t *translator) bracket(i int) (int, error) {
	p := t.pat
	t.begin()
	t.out = append(t.out, '[')
	j := i + 1
	negated := j < len(p) && p[j] == '^'
	if negated {
		t.out = append(t.out, '^')
		j++
	}
	for first := true; ; first = false {
		if j >= len(p) {
			return 0, errUnmatchedBracket
		}
		c := p[j]
		switch {
		case c == ']' && !first:
			if negated && t.ml {
				// With M, a non-matching list does not match a newline.
				t.out = append(t.out, `\n`...)
			}
			t.out = append(t.out, ']')
			return j, nil
		case c == '[' && j+1 < len(p) && strings.IndexByte(":.=", p[j+1]) >= 0:
			kind := p[j+1]
			n := strings.Index(p[j+2:], string(kind)+"]")
			if n < 0 {
				return 0, errUnmatchedBracket
			}
			name := p[j+2 : j+2+n]
			if kind == ':' {
				if !classNames[name] {
					return 0, errBadClass
				}
				t.out = append(t.out, "[:"+name+":]"...)
			} else {
				// A collating symbol or equivalence class of one
				// character is that character; nothing longer exists
				// outside a locale.
				if len([]rune(name)) != 1 {
					return 0, errBadCollation
				}
				t.out = appendClassChars(t.out, name)
			}
			j += n + 4
		case c == '\\':
			// A backslash is literal in a bracket expression, but sed
			// has already turned its own escapes into characters.
			if b, n, ok := charEscape(p, j+1); ok {
				t.out = appendClassChars(t.out, string(b))
				j = n + 1
			} else {
				t.out = append(t.out, `\\`...)
				j++
			}
		default:
			t.out = appendClassChars(t.out, string(c))
			j++
		}
	}
}

// appendClassChars appends s inside a RE2 character class, escaping what RE2
// would read as syntax there.
func appendClassChars(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '[' || c == ']' || c == '^' || c == '\\':
			out = append(out, '\\', c)
		case c < 0x20 || c == 0x7f:
			out = appendHex(out, c)
		default:
			out = append(out, c)
		}
	}
	return out
}

// appendLiteral appends b to a RE2 pattern as an ordinary character.
func appendLiteral(out []byte, b byte) []byte {
	switch {
	case b < 0x20 || b == 0x7f:
		return appendHex(out, b)
	case strings.IndexByte(`\.+*?()|[]{}^$`, b) >= 0:
		return append(out, '\\', b)
	}
	return append(out, b)
}

func appendHex(out []byte, b byte) []byte {
	const digits = "0123456789abcdef"
	return append(out, '\\', 'x', digits[b>>4], digits[b&15])
}

// charEscape decodes the escapes GNU sed turns into a single character
// wherever they appear (a regular expression, a replacement, a y string or
// the text of a, i and c): \a \f \n \r \t \v, \cX, \dNNN, \oNNN and \xHH.
// i is the index of the character after the backslash; it returns the
// character and the index of the sequence's last character.
func charEscape(s string, i int) (b byte, end int, ok bool) {
	if i >= len(s) {
		return 0, 0, false
	}
	digits := func(n, base int) (byte, bool) {
		if i+n >= len(s) {
			return 0, false
		}
		v := 0
		for _, c := range []byte(s[i+1 : i+1+n]) {
			d := hexValue(c)
			if d < 0 || d >= base {
				return 0, false
			}
			v = v*base + d
		}
		if v > 0xFF {
			return 0, false
		}
		return byte(v), true
	}
	switch s[i] {
	case 'a':
		return '\a', i, true
	case 'f':
		return '\f', i, true
	case 'n':
		return '\n', i, true
	case 'r':
		return '\r', i, true
	case 't':
		return '\t', i, true
	case 'v':
		return '\v', i, true
	case 'c':
		if i+1 < len(s) && s[i+1] != '\\' {
			c := s[i+1]
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
			return c ^ 0x40, i + 1, true
		}
	case 'd':
		if v, ok := digits(3, 10); ok {
			return v, i + 3, true
		}
	case 'o':
		if v, ok := digits(3, 8); ok {
			return v, i + 3, true
		}
	case 'x':
		if v, ok := digits(2, 16); ok {
			return v, i + 2, true
		}
	}
	return 0, 0, false
}

func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
