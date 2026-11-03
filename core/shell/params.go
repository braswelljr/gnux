package shell

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// paramOps are the operators of a ${NAME<op>WORD} expansion, longest first so
// that ":-" is not read as ":" and "##" not as "#".
var paramOps = []string{":-", ":=", ":+", ":?", "##", "%%", "-", "=", "+", "?", "#", "%"}

// param expands the body of a ${...}, as POSIX sh does:
//
//	${NAME}                  the value
//	${#NAME}                 its length in characters
//	${NAME:-WORD} ${NAME-WORD}  WORD if NAME is unset (or, with the colon, empty)
//	${NAME:=WORD} ${NAME=WORD}  the same, and NAME is set to WORD
//	${NAME:+WORD} ${NAME+WORD}  WORD if NAME is set (and, with the colon, not empty)
//	${NAME:?WORD} ${NAME?WORD}  an error, WORD its message, if NAME is unset (or empty)
//	${NAME#PAT} ${NAME##PAT}    the value without the shortest / longest prefix PAT matches
//	${NAME%PAT} ${NAME%%PAT}    the value without the shortest / longest suffix PAT matches
//
// text is the whole expansion, for messages. ok is false when the expansion
// failed and was reported.
func (ex *expander) param(body, text string) (seg segment, ok bool) {
	if len(body) > 1 && body[0] == '#' && isParameter(body[1:]) {
		value, set := ex.lookup(body[1:])
		if !set && !ex.unbound(body[1:]) {
			return segment{}, false
		}
		return segment{text: strconv.Itoa(utf8.RuneCountInString(value)), fromVar: true}, true
	}

	name := paramName(body)
	if name == "" {
		ex.badSubstitution(text)
		return segment{}, false
	}
	rest := body[len(name):]
	if rest == "" {
		if name == "@" {
			return segment{list: true, items: ex.sh.args}, true
		}
		value, set := ex.lookup(name)
		if !set && !ex.unbound(name) {
			return segment{}, false
		}
		return segment{text: value, fromVar: true}, true
	}

	op := ""
	for _, o := range paramOps {
		if strings.HasPrefix(rest, o) {
			op = o
			break
		}
	}
	if op == "" {
		ex.badSubstitution(text)
		return segment{}, false
	}
	word := rest[len(op):]
	value, set := ex.lookup(name)
	// With a colon, an empty value counts as unset.
	missing := !set || (op[0] == ':' && value == "")

	switch strings.TrimPrefix(op, ":") {
	case "-":
		if missing {
			value = ex.word(word, false)
		}
	case "=":
		if missing {
			if !isName(name) {
				ex.sh.errorf(ex.s, "%s: cannot assign in this way", text)
				ex.failed = true
				return segment{}, false
			}
			value = ex.word(word, false)
			ex.sh.vars[name] = value
		}
	case "+":
		value = ""
		if !missing {
			value = ex.word(word, false)
		}
	case "?":
		if missing {
			msg := ex.word(word, false)
			if msg == "" {
				msg = "parameter not set"
				if op == ":?" {
					msg = "parameter null or not set"
				}
			}
			ex.sh.errorf(ex.s, "%s: %s", name, msg)
			ex.failed = true
			return segment{}, false
		}
	default: // # ## % %%
		if !set && !ex.unbound(name) {
			return segment{}, false
		}
		value = trim(value, ex.word(word, true), op)
	}
	return segment{text: value, fromVar: true}, true
}

// paramName returns the parameter at the start of the body of a ${...}: a
// variable name, an argument number or one of ? # @ *; "" if there is none.
func paramName(body string) string {
	if body == "" {
		return ""
	}
	switch c := body[0]; {
	case strings.IndexByte("?#@*", c) >= 0:
		return body[:1]
	case c >= '0' && c <= '9':
		j := 1
		for j < len(body) && body[j] >= '0' && body[j] <= '9' {
			j++
		}
		return body[:j]
	case isNameStart(c):
		j := 1
		for j < len(body) && isNameChar(body[j]) {
			j++
		}
		return body[:j]
	}
	return ""
}

// lookup is Shell.lookup, with $@ as its words joined, as "$*" has them.
func (ex *expander) lookup(name string) (string, bool) {
	if name == "@" {
		return strings.Join(ex.sh.args, " "), len(ex.sh.args) > 0
	}
	return ex.sh.lookup(name)
}

// unbound reports that name is not set when -u makes that an error, and
// returns false in that case.
func (ex *expander) unbound(name string) bool {
	if !ex.sh.NoUnset {
		return true
	}
	if !ex.failed {
		ex.sh.errorf(ex.s, "%s: unbound variable", name)
		ex.failed = true
	}
	return false
}

// trim removes from value the shortest (# %) or longest (## %%) prefix (#) or
// suffix (%) that pattern matches.
func trim(value, pattern, op string) string {
	r := []rune(value)
	n := len(r)
	switch op {
	case "#":
		for k := 0; k <= n; k++ {
			if match(pattern, string(r[:k])) {
				return string(r[k:])
			}
		}
	case "##":
		for k := n; k >= 0; k-- {
			if match(pattern, string(r[:k])) {
				return string(r[k:])
			}
		}
	case "%":
		for k := n; k >= 0; k-- {
			if match(pattern, string(r[k:])) {
				return string(r[:k])
			}
		}
	case "%%":
		for k := 0; k <= n; k++ {
			if match(pattern, string(r[k:])) {
				return string(r[:k])
			}
		}
	}
	return value
}

// word expands the WORD of a ${NAME<op>WORD}: its variables, and its quotes,
// which it may hold of its own (${x:-"a b"}). For a pattern (# % and the
// like), quoted text and variable values match only themselves, as
// everywhere else in this shell; the text written unquoted is the pattern.
func (ex *expander) word(text string, pattern bool) string {
	var b strings.Builder
	quoted := func(s string) {
		if pattern {
			s = globEscape(s)
		}
		b.WriteString(s)
	}
	expand := func(s string, inQuotes bool) {
		for _, seg := range ex.vars(s) {
			v := seg.text
			if seg.list {
				v = strings.Join(seg.items, " ")
			}
			if seg.fromVar || seg.list || inQuotes {
				quoted(v)
			} else {
				b.WriteString(v)
			}
		}
	}

	for i := 0; i < len(text); {
		switch c := text[i]; c {
		case '\'':
			end := strings.IndexByte(text[i+1:], '\'')
			if end < 0 {
				end = len(text) - i - 1
			}
			quoted(text[i+1 : i+1+end])
			i += end + 2
		case '"':
			var inner strings.Builder
			j := i + 1
			for ; j < len(text) && text[j] != '"'; j++ {
				if text[j] == '\\' && j+1 < len(text) && strings.IndexByte(`"$\`, text[j+1]) >= 0 {
					j++
				} else if strings.HasPrefix(text[j:], "${") {
					if end := braceEnd(text, j+2); end >= 0 {
						inner.WriteString(text[j : end+1])
						j = end
						continue
					}
				}
				inner.WriteByte(text[j])
			}
			expand(inner.String(), true)
			i = j + 1
		case '\\':
			if i+1 < len(text) {
				quoted(text[i+1 : i+2])
			}
			i += 2
		default:
			// Unquoted text runs to the next quote or backslash; a ${...}
			// inside it is taken whole.
			j := i
			for j < len(text) && strings.IndexByte(`'"\`, text[j]) < 0 {
				if strings.HasPrefix(text[j:], "${") {
					if end := braceEnd(text, j+2); end >= 0 {
						j = end + 1
						continue
					}
				}
				j++
			}
			expand(text[i:j], false)
			i = j
		}
	}
	return b.String()
}
