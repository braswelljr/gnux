package find

import (
	"errors"
	"regexp"
	"strings"

	"github.com/braswelljr/gnux/core/internal/bre"
)

// The regular expression dialects -regextype names, grouped by the
// translation they get. GNU's regex engine has finer distinctions between
// the members of a family than RE2 can express; each family is translated as
// its most common member.
const (
	dialectEmacs = iota
	dialectBasic
	dialectExtended
)

// regexTypes lists GNU's names in the order GNU prints them.
var regexTypes = []struct {
	name    string
	dialect int
}{
	{"findutils-default", dialectEmacs},
	{"ed", dialectBasic},
	{"emacs", dialectEmacs},
	{"gnu-awk", dialectExtended},
	{"grep", dialectBasic},
	{"posix-awk", dialectExtended},
	{"awk", dialectExtended},
	{"posix-basic", dialectBasic},
	{"posix-egrep", dialectExtended},
	{"egrep", dialectExtended},
	{"posix-extended", dialectExtended},
	{"posix-minimal-basic", dialectBasic},
	{"sed", dialectBasic},
}

func knownRegexType(name string) bool {
	for _, t := range regexTypes {
		if t.name == name {
			return true
		}
	}
	return false
}

func regexTypeList() string {
	names := make([]string, len(regexTypes))
	for i, t := range regexTypes {
		names[i] = "'" + t.name + "'"
	}
	return strings.Join(names, ", ")
}

// compileRegex compiles a -regex pattern of the given type into an RE2
// expression that must match the whole path, as GNU find requires.
func compileRegex(kind, pattern string, fold bool) (*regexp.Regexp, error) {
	dialect := dialectEmacs
	for _, t := range regexTypes {
		if t.name == kind {
			dialect = t.dialect
		}
	}

	var expr string
	var err error
	if dialect == dialectBasic {
		expr, err = bre.Translate(pattern)
	} else {
		expr, err = translateRegex(pattern, dialect == dialectEmacs)
	}
	if err != nil {
		return nil, err
	}

	// A dot matches a newline in every syntax find offers.
	flags := "(?s)"
	if fold {
		flags = "(?si)"
	}
	re, err := regexp.Compile(flags + `\A(?:` + expr + `)\z`)
	if err != nil {
		return nil, errors.New("Invalid regular expression") //nolint:staticcheck,revive // GNU's wording
	}
	return re, nil
}

// translateRegex rewrites an Emacs-syntax expression (the default: groups and
// alternation are \( \) \|, while + and ? are operators) or a POSIX extended
// one into RE2 syntax. Back references cannot be expressed and are refused.
func translateRegex(pattern string, emacs bool) (string, error) {
	var b strings.Builder
	depth := 0
	// atStart is true where a repetition operator has nothing to repeat and
	// so stands for itself, and where Emacs reads ^ as an anchor.
	atStart := true

	open := func() {
		b.WriteByte('(')
		depth++
		atStart = true
	}
	closeGroup := func() error {
		if depth == 0 {
			return errors.New("Unmatched ) or \\)") //nolint:staticcheck,revive // GNU's wording
		}
		depth--
		b.WriteByte(')')
		atStart = false
		return nil
	}
	alternate := func() {
		b.WriteByte('|')
		atStart = true
	}
	// endsBranch reports whether position i ends the pattern or a branch,
	// which is where Emacs reads $ as an anchor.
	endsBranch := func(i int) bool {
		return i == len(pattern) || strings.HasPrefix(pattern[i:], `\)`) || strings.HasPrefix(pattern[i:], `\|`)
	}

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '\\':
			if i+1 >= len(pattern) {
				return "", errors.New("Trailing backslash") //nolint:staticcheck,revive // GNU's wording
			}
			i++
			d := pattern[i]
			switch {
			case emacs && d == '(':
				open()
				continue
			case emacs && d == ')':
				if err := closeGroup(); err != nil {
					return "", err
				}
				continue
			case emacs && d == '|':
				alternate()
				continue
			case d >= '1' && d <= '9':
				return "", errors.New("back references are not supported in a regular expression")
			case d == '<' || d == '>':
				b.WriteString(`\b`)
			case d == '`':
				b.WriteString(`\A`)
			case d == '\'':
				b.WriteString(`\z`)
			case strings.IndexByte("wWsSbB", d) >= 0:
				b.WriteByte('\\')
				b.WriteByte(d)
			default:
				b.WriteString(regexp.QuoteMeta(string(d)))
			}
		case '[':
			class, end, err := translateBracket(pattern, i)
			if err != nil {
				return "", err
			}
			b.WriteString(class)
			i = end
		case '*', '+', '?':
			if atStart {
				b.WriteString(regexp.QuoteMeta(string(c)))
				atStart = false
			} else {
				b.WriteByte(c)
				// RE2 refuses a repeated repetition (a**); GNU reads it
				// as one.
				for i+1 < len(pattern) && strings.IndexByte("*+?", pattern[i+1]) >= 0 {
					if pattern[i+1] != c {
						return "", errors.New("Invalid preceding regular expression") //nolint:staticcheck,revive // GNU's wording
					}
					i++
				}
			}
			continue
		case '(', ')', '|':
			if emacs {
				b.WriteString(regexp.QuoteMeta(string(c)))
				break
			}
			switch c {
			case '(':
				open()
			case ')':
				if err := closeGroup(); err != nil {
					return "", err
				}
			default:
				alternate()
			}
			continue
		case '{':
			if end := intervalEnd(pattern, i); !emacs && !atStart && end > 0 {
				b.WriteString(pattern[i : end+1])
				i = end
			} else {
				b.WriteString(`\{`)
			}
		case '^':
			if !emacs || atStart {
				b.WriteByte('^')
				continue
			}
			b.WriteString(`\^`)
		case '$':
			if !emacs || endsBranch(i+1) {
				b.WriteByte('$')
			} else {
				b.WriteString(`\$`)
			}
		case '.':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
		atStart = false
	}
	if depth > 0 {
		return "", errors.New("Unmatched ( or \\(") //nolint:staticcheck,revive // GNU's wording
	}
	return b.String(), nil
}

// intervalEnd returns the index of the '}' closing a valid interval
// ({n}, {n,} or {n,m}) that opens at pattern[i], or -1.
func intervalEnd(pattern string, i int) int {
	j := i + 1
	digits := 0
	for j < len(pattern) && pattern[j] >= '0' && pattern[j] <= '9' {
		j++
		digits++
	}
	if digits == 0 {
		return -1
	}
	if j < len(pattern) && pattern[j] == ',' {
		j++
		for j < len(pattern) && pattern[j] >= '0' && pattern[j] <= '9' {
			j++
		}
	}
	if j < len(pattern) && pattern[j] == '}' {
		return j
	}
	return -1
}

// translateBracket copies the bracket expression opening at pattern[i] into
// RE2 syntax, where (unlike POSIX) a backslash inside brackets is an escape.
// It returns the index of the closing ']'.
func translateBracket(pattern string, i int) (string, int, error) {
	unmatched := errors.New("Unmatched [, [^, [:, [., or [=") //nolint:staticcheck,revive // GNU's wording
	j := i + 1
	if j < len(pattern) && pattern[j] == '^' {
		j++
	}
	if j < len(pattern) && pattern[j] == ']' {
		j++
	}
	for j < len(pattern) && pattern[j] != ']' {
		if pattern[j] == '[' && j+1 < len(pattern) && strings.IndexByte(":.=", pattern[j+1]) >= 0 {
			if pattern[j+1] != ':' {
				return "", 0, errors.New("the [. .] and [= =] bracket forms are not supported in a regular expression")
			}
			end := strings.Index(pattern[j+2:], ":]")
			if end < 0 {
				return "", 0, unmatched
			}
			j += end + 4
			continue
		}
		j++
	}
	if j >= len(pattern) {
		return "", 0, unmatched
	}

	body := pattern[i+1 : j]
	neg := strings.HasPrefix(body, "^")
	body = strings.TrimPrefix(body, "^")
	body = strings.ReplaceAll(body, `\`, `\\`)
	if strings.HasPrefix(body, "]") {
		body = `\]` + body[1:]
	}
	// A '[' that does not open a class is a literal; RE2 wants it escaped
	// only when it could be read as one, which the scan above ruled out.
	out := "["
	if neg {
		out += "^"
	}
	return out + body + "]", j, nil
}
