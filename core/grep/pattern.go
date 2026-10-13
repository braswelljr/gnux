package grep

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The pieces of Go syntax the translation is built from. A word is made of
// letters, digits and the underscore, as in GNU grep in a UTF-8 locale.
const (
	wordClass    = `[\p{L}\p{N}_]`
	nonWordClass = `[^\p{L}\p{N}_]`
	spaceClass   = `[\t\n\v\f\r ]`
	nonSpace     = `[^\t\n\v\f\r ]`
)

// dupMax is the largest repetition count GNU accepts (RE_DUP_MAX).
const dupMax = 0x7fff

// patternError is a pattern GNU grep would reject, carrying GNU's message.
type patternError struct{ msg string }

func (e *patternError) Error() string { return e.msg }

var (
	errTrailingBackslash = &patternError{"Trailing backslash"}
	errUnmatchedBracket  = &patternError{"Unmatched [, [^, [:, [., or [="}
	errUnmatchedOpen     = &patternError{`Unmatched ( or \(`}
	errUnmatchedClose    = &patternError{`Unmatched ) or \)`}
	errUnmatchedBrace    = &patternError{`Unmatched \{`}
	errBraceContent      = &patternError{`Invalid content of \{\}`}
	errPreceding         = &patternError{"Invalid preceding regular expression"}
	errTooBig            = &patternError{"Regular expression too big"}
	errRangeEnd          = &patternError{"Invalid range end"}
	errClassName         = &patternError{"Invalid character class name"}
	errCollation         = &patternError{"Invalid collation character"}
	errInvalid           = &patternError{"Invalid regular expression"}
	errBackReference     = &patternError{"back references are not supported"}
	errClassSyntax       = &patternError{"character class syntax is [[:space:]], not [:space:]"}
)

// translator rewrites one POSIX regular expression, basic or extended, with
// the GNU extensions, into Go (RE2) syntax.
//
// core/internal/bre does this for the basic syntax of expr, nl and csplit,
// but grep needs more than it offers: an anchor after \( and \| ('^a\|^b'),
// the word operators \< and \>, intervals checked the GNU way, and the
// warnings GNU grep 3.8 added. Both syntaxes share everything except which
// characters are operators, so one scanner serves both.
type translator struct {
	ere  bool
	warn func(string)

	out []byte
	// atom is the offset in out of the last complete atom, the thing a
	// repetition operator applies to; -1 when there is none.
	atom int
	// repeated records that the atom already carries a repetition, so that a
	// second one ("a**", "a+?") is wrapped: Go rejects the first and reads the
	// second as a lazy operator, which POSIX does not have.
	repeated bool
	// branchStart is true where a basic expression takes ^ as an anchor: at
	// the start of the pattern, after \( and after \|.
	branchStart bool
	groups      []int
}

// translate returns the Go syntax for pattern. warn receives the text of each
// warning GNU grep would print for it.
func translate(pattern string, ere bool, warn func(string)) (string, error) {
	t := &translator{ere: ere, warn: warn, atom: -1, branchStart: true}
	p := pattern
	for i := 0; i < len(p); {
		c := p[i]
		switch {
		case c == '\\':
			if i+1 >= len(p) {
				return "", errTrailingBackslash
			}
			next, err := t.escape(p, i+1)
			if err != nil {
				return "", err
			}
			i = next
		case c == '[':
			class, next, err := bracket(p, i)
			if err != nil {
				return "", err
			}
			t.emit(class)
			i = next
		case c == '.':
			t.emit(".")
			i++
		case c == '*':
			i++
			switch {
			case t.atom >= 0:
				t.repeat("*")
			case t.ere:
				// GNU reads it as a repetition of nothing.
				t.warn("* at start of expression")
			default:
				t.emit(`\*`)
			}
		case c == '^':
			i++
			if t.ere || t.branchStart {
				t.anchor("^")
			} else {
				t.emit(`\^`)
			}
		case c == '$':
			i++
			if t.ere || i == len(p) || strings.HasPrefix(p[i:], `\)`) || strings.HasPrefix(p[i:], `\|`) {
				t.anchor("$")
			} else {
				t.emit(`\$`)
			}
		case t.ere && c == '(':
			t.open()
			i++
		case t.ere && c == ')':
			if err := t.close(); err != nil {
				return "", err
			}
			i++
		case t.ere && c == '|':
			t.alternate()
			i++
		case t.ere && (c == '+' || c == '?'):
			i++
			if t.atom >= 0 {
				t.repeat(string(c))
			} else {
				t.warn(string(c) + " at start of expression")
			}
		case t.ere && c == '{':
			rep, next, err := interval(p, i+1, "}")
			switch {
			case errors.Is(err, errTooBig):
				return "", err
			case err != nil:
				// Not an interval: GNU takes the brace literally.
				t.emit(`\{`)
				i++
			case t.atom < 0:
				t.warn("{...} at start of expression")
				i = next
			default:
				t.repeat(rep)
				i = next
			}
		default:
			r, size := utf8.DecodeRuneInString(p[i:])
			if r == utf8.RuneError && size == 1 {
				return "", errInvalid
			}
			t.emit(regexp.QuoteMeta(p[i : i+size]))
			i += size
		}
	}
	if len(t.groups) > 0 {
		return "", errUnmatchedOpen
	}
	return string(t.out), nil
}

// emit appends a complete atom.
func (t *translator) emit(s string) {
	t.atom = len(t.out)
	t.out = append(t.out, s...)
	t.repeated = false
	t.branchStart = false
}

// anchor appends ^ or $, which no repetition applies to.
func (t *translator) anchor(s string) {
	t.out = append(t.out, s...)
	t.atom = -1
	t.repeated = false
	t.branchStart = false
}

// repeat appends a repetition operator to the last atom.
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

func (t *translator) open() {
	t.groups = append(t.groups, len(t.out))
	t.out = append(t.out, "(?:"...)
	t.atom = -1
	t.repeated = false
	t.branchStart = true
}

func (t *translator) close() error {
	if len(t.groups) == 0 {
		return errUnmatchedClose
	}
	start := t.groups[len(t.groups)-1]
	t.groups = t.groups[:len(t.groups)-1]
	t.out = append(t.out, ')')
	t.atom = start
	t.repeated = false
	t.branchStart = false
	return nil
}

func (t *translator) alternate() {
	t.out = append(t.out, '|')
	t.atom = -1
	t.repeated = false
	t.branchStart = true
}

// escape handles the character at p[j], which follows a backslash, and
// returns the index after what it consumed.
func (t *translator) escape(p string, j int) (int, error) {
	d := p[j]
	if !t.ere {
		switch d {
		case '(':
			t.open()
			return j + 1, nil
		case ')':
			return j + 1, t.close()
		case '|':
			t.alternate()
			return j + 1, nil
		case '{':
			rep, next, err := interval(p, j+1, `\}`)
			if err != nil {
				return 0, err
			}
			if t.atom < 0 {
				return 0, errPreceding
			}
			t.repeat(rep)
			return next, nil
		case '+', '?':
			if t.atom < 0 {
				t.emit(regexp.QuoteMeta(string(d)))
			} else {
				t.repeat(string(d))
			}
			return j + 1, nil
		}
	}

	switch d {
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return 0, errBackReference
	case 'w':
		t.emit(wordClass)
	case 'W':
		t.emit(nonWordClass)
	case 's':
		t.emit(spaceClass)
	case 'S':
		t.emit(nonSpace)
	case 'b', '<', '>':
		// RE2 has no "start of word" or "end of word", only the boundary.
		t.emit(`\b`)
	case 'B':
		t.emit(`\B`)
	case '`':
		t.emit(`\A`)
	case '\'':
		t.emit(`\z`)
	default:
		r, size := utf8.DecodeRuneInString(p[j:])
		if r == utf8.RuneError && size == 1 {
			return 0, errInvalid
		}
		if !t.special(r) {
			t.warn(strayBackslash(r))
		}
		t.emit(regexp.QuoteMeta(p[j : j+size]))
		return j + size, nil
	}
	return j + 1, nil
}

// special reports whether a backslash before r is an ordinary way of taking
// r literally, which GNU does not warn about.
func (t *translator) special(r rune) bool {
	if strings.ContainsRune(`\^$.*[]}`, r) {
		return true
	}
	return t.ere && strings.ContainsRune("()|+?{", r)
}

// strayBackslash is GNU's warning for a backslash before a character that
// needs none.
func strayBackslash(r rune) string {
	switch {
	case !unicode.IsPrint(r) && !unicode.IsSpace(r):
		return `stray \ before unprintable character`
	case unicode.IsSpace(r):
		return `stray \ before white space`
	}
	return `stray \ before '` + string(r) + `'`
}

// interval parses the body of an interval starting at p[j], just after the
// opening brace, up to and including closer. It returns the Go repetition.
func interval(p string, j int, closer string) (rep string, next int, err error) {
	end := strings.Index(p[j:], closer)
	if end < 0 {
		return "", 0, errUnmatchedBrace
	}
	body := p[j : j+end]
	next = j + end + len(closer)

	low, high, comma := strings.Cut(body, ",")
	if body == "" || !digits(low) || !digits(high) {
		return "", 0, errBraceContent
	}
	lo, err := count(low)
	if err != nil {
		return "", 0, err
	}
	switch {
	case !comma:
		if low == "" {
			return "", 0, errBraceContent
		}
		return "{" + strconv.Itoa(lo) + "}", next, nil
	case high == "":
		return "{" + strconv.Itoa(lo) + ",}", next, nil
	}
	hi, err := count(high)
	if err != nil {
		return "", 0, err
	}
	if lo > hi {
		return "", 0, errBraceContent
	}
	return "{" + strconv.Itoa(lo) + "," + strconv.Itoa(hi) + "}", next, nil
}

func digits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// count reads a repetition count; an absent one is zero.
func count(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n > dupMax {
		return 0, errTooBig
	}
	return n, nil
}

// classes maps the POSIX character class names to the inside of a Go class.
// The letter classes take Unicode letters, as GNU grep does in a UTF-8
// locale; the rest are the ASCII sets Go already knows.
var classes = map[string]string{
	"alpha":  `\p{L}`,
	"alnum":  `\p{L}\p{N}`,
	"upper":  `\p{Lu}`,
	"lower":  `\p{Ll}`,
	"digit":  `0-9`,
	"xdigit": `0-9A-Fa-f`,
	"space":  `\t\n\v\f\r `,
	"blank":  `\t `,
	"punct":  `[:punct:]`,
	"cntrl":  `[:cntrl:]`,
	"graph":  `[:graph:]`,
	"print":  `[:print:]`,
}

// bracket translates the bracket expression starting at p[i] and returns
// the index after its closing bracket.
func bracket(p string, i int) (class string, next int, err error) {
	j := i + 1
	var b strings.Builder
	b.WriteByte('[')
	if j < len(p) && p[j] == '^' {
		b.WriteByte('^')
		j++
	}
	bodyStart := j
	for first := true; ; first = false {
		if j >= len(p) {
			return "", 0, errUnmatchedBracket
		}
		c := p[j]
		if c == ']' && !first {
			break
		}
		if c == '[' && j+1 < len(p) && strings.IndexByte(":.=", p[j+1]) >= 0 {
			kind := p[j+1]
			end := strings.Index(p[j+2:], string(kind)+"]")
			if end < 0 {
				return "", 0, errUnmatchedBracket
			}
			name := p[j+2 : j+2+end]
			j += end + 4
			if kind == ':' {
				set, ok := classes[name]
				if !ok {
					return "", 0, errClassName
				}
				b.WriteString(set)
				continue
			}
			// A collating symbol or equivalence class of one character is
			// that character; there are no multi-character ones here.
			if utf8.RuneCountInString(name) != 1 {
				return "", 0, errCollation
			}
			writeClassRune(&b, name)
			continue
		}
		r, size := utf8.DecodeRuneInString(p[j:])
		if r == utf8.RuneError && size == 1 {
			return "", 0, errInvalid
		}
		if c == '-' {
			// Left bare, it keeps its POSIX meaning: a range between two
			// characters, itself at either end.
			b.WriteByte('-')
		} else {
			writeClassRune(&b, p[j:j+size])
		}
		j += size
	}
	// "[:space:]" alone is a set of the characters ':', 's', 'p', ...: GNU
	// refuses what is almost always a mistake.
	if body := p[bodyStart:j]; len(body) >= 2 && body[0] == ':' && body[len(body)-1] == ':' {
		return "", 0, errClassSyntax
	}
	b.WriteByte(']')
	return b.String(), j + 1, nil
}

// writeClassRune writes one literal character into a Go class.
func writeClassRune(b *strings.Builder, s string) {
	if strings.ContainsAny(s, `\]^[-`) {
		b.WriteByte('\\')
	}
	b.WriteString(s)
}

// compileError maps a Go regexp compilation error to GNU's message.
func compileError(err error) error {
	var serr *syntax.Error
	if errors.As(err, &serr) {
		switch serr.Code {
		case syntax.ErrInvalidCharRange:
			return errRangeEnd
		case syntax.ErrInvalidRepeatSize, syntax.ErrLarge, syntax.ErrNestingDepth:
			return errTooBig
		case syntax.ErrMissingRepeatArgument, syntax.ErrInvalidRepeatOp:
			return errPreceding
		}
	}
	return errInvalid
}

// literalOf reports whether the Go expression matches exactly one string, and
// which. Such a pattern is searched for with bytes.Index, not the regexp
// engine.
func literalOf(expr string) (string, bool) {
	re, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return "", false
	}
	re = re.Simplify()
	switch re.Op {
	case syntax.OpEmptyMatch:
		return "", true
	case syntax.OpLiteral:
		if re.Flags&syntax.FoldCase == 0 {
			return string(re.Rune), true
		}
	}
	return "", false
}
