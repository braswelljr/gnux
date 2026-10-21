// Package fnmatch matches names against shell patterns as fnmatch(3) does
// with no flags: the matching GNU find's -name and dircolors' TERM lines use.
package fnmatch

import (
	"unicode"
	"unicode/utf8"
)

// Match reports whether name matches the shell pattern, with the semantics
// of fnmatch(3) with no flags, so '*' and '?' match a '/' and a
// leading '.', plus case folding when fold is set (find's -iname). filepath.Match is not used
// because it treats the separator specially, differs between platforms and
// rejects patterns fnmatch accepts (an unmatched '[' is a literal here).
//
// The pattern syntax is '*', '?', '[...]' (with ranges, '!' or '^' negation
// and [:class:] names) and a backslash that quotes the next character.
func Match(pattern, name string, fold bool) bool {
	// px and nx are where to resume after the most recent '*' when a later
	// part fails: one star of backtracking is enough without FNM_PATHNAME.
	px, nx := 0, 0
	starP, starN := -1, -1
	for px < len(pattern) || nx < len(name) {
		if px < len(pattern) {
			switch c := pattern[px]; c {
			case '*':
				starP, starN = px, nx
				px++
				continue
			case '?':
				if nx < len(name) {
					_, size := utf8.DecodeRuneInString(name[nx:])
					px++
					nx += size
					continue
				}
			case '[':
				if nx < len(name) {
					r, size := utf8.DecodeRuneInString(name[nx:])
					matched, next, ok := matchBracket(pattern, px, r, fold)
					if !ok {
						// No closing bracket: the '[' stands for itself.
						if r == '[' {
							px++
							nx += size
							continue
						}
					} else if matched {
						px = next
						nx += size
						continue
					}
				}
			default:
				if c == '\\' && px+1 < len(pattern) {
					px++
				}
				pr, psize := utf8.DecodeRuneInString(pattern[px:])
				if nx < len(name) {
					r, size := utf8.DecodeRuneInString(name[nx:])
					if pr == r || (fold && foldRune(pr) == foldRune(r)) {
						px += psize
						nx += size
						continue
					}
				}
			}
		}
		// Mismatch: let the last star swallow one more character.
		if starP >= 0 && starN < len(name) {
			_, size := utf8.DecodeRuneInString(name[starN:])
			starN += size
			px, nx = starP+1, starN
			continue
		}
		return false
	}
	return true
}

func foldRune(r rune) rune {
	return unicode.ToLower(unicode.ToUpper(r))
}

// matchBracket matches r against the bracket expression starting at
// pattern[start] ('['). It returns whether r is in the set, the index after
// the closing ']', and ok false when the expression is never closed.
func matchBracket(pattern string, start int, r rune, fold bool) (matched bool, next int, ok bool) {
	i := start + 1
	negate := false
	if i < len(pattern) && (pattern[i] == '!' || pattern[i] == '^') {
		negate = true
		i++
	}
	if fold {
		r = foldRune(r)
	}
	first := true
	for {
		if i >= len(pattern) {
			return false, 0, false
		}
		if pattern[i] == ']' && !first {
			return matched != negate, i + 1, true
		}
		first = false

		// A named class: [:alpha:].
		if pattern[i] == '[' && i+1 < len(pattern) && pattern[i+1] == ':' {
			if end := indexFrom(pattern, i+2, ":]"); end >= 0 {
				if class, known := classes[pattern[i+2:end]]; known {
					if class(r) || (fold && (class(unicode.ToUpper(r)) || class(unicode.ToLower(r)))) {
						matched = true
					}
					i = end + 2
					continue
				}
			}
		}

		lo, n := bracketRune(pattern, i)
		if n == 0 {
			return false, 0, false
		}
		i += n
		hi := lo
		if i+1 < len(pattern) && pattern[i] == '-' && pattern[i+1] != ']' {
			var m int
			hi, m = bracketRune(pattern, i+1)
			if m == 0 {
				return false, 0, false
			}
			i += 1 + m
		}
		if fold {
			lo, hi = foldRune(lo), foldRune(hi)
		}
		if lo <= r && r <= hi {
			matched = true
		}
	}
}

// bracketRune reads one character of a bracket expression, honoring a
// quoting backslash. It returns the bytes consumed, 0 at the end.
func bracketRune(pattern string, i int) (rune, int) {
	if i >= len(pattern) {
		return 0, 0
	}
	skip := 0
	if pattern[i] == '\\' && i+1 < len(pattern) {
		skip = 1
	}
	r, size := utf8.DecodeRuneInString(pattern[i+skip:])
	return r, size + skip
}

func indexFrom(s string, from int, sub string) int {
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// classes are the POSIX character classes of a bracket expression.
var classes = map[string]func(rune) bool{
	"alnum": func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) },
	"alpha": unicode.IsLetter,
	"blank": func(r rune) bool { return r == ' ' || r == '\t' },
	"cntrl": unicode.IsControl,
	"digit": func(r rune) bool { return r >= '0' && r <= '9' },
	"graph": func(r rune) bool { return unicode.IsGraphic(r) && !unicode.IsSpace(r) },
	"lower": unicode.IsLower,
	"print": unicode.IsPrint,
	"punct": func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) },
	"space": unicode.IsSpace,
	"upper": unicode.IsUpper,
	"xdigit": func(r rune) bool {
		return r < 128 && (r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F')
	},
}
