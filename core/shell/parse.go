package shell

import (
	"errors"
	"fmt"
	"strings"
)

// quoting says how a piece of a word was written, which decides how it is
// expanded: raw text gets variables and globbing, double-quoted text variables
// only, and single-quoted (or backslash-escaped) text nothing at all.
type quoting int

const (
	raw quoting = iota
	double
	literal
)

// part is a run of a word that shares one kind of quoting, or one command
// substitution: for `$(...)`, subst is set, body holds the commands and mode
// says whether it stood inside double quotes.
type part struct {
	text  string
	mode  quoting
	subst bool
	body  []step
}

// word is one argument before expansion: `a"$b"'c'` is a single word of three
// parts.
type word []part

// redirect is one `<`, `>`, `>>`, `2>` or `2>>` with its target, or a
// `2>&1` or `>&2`, which has none.
type redirect struct {
	op     string
	target word
}

// command is one stage of a pipeline: a program invocation (its words), or a
// compound command (if, for, while, case), either with redirections.
type command struct {
	words     []word
	redirects []redirect
	compound  compound
}

// compound is one of *ifClause, *forClause, *whileClause and *caseClause.
type compound any

// branch is an `if` or `elif` condition with the commands it guards.
type branch struct {
	cond, body []step
}

type ifClause struct {
	branches []branch
	elseBody []step
}

// forClause runs body once per item; with no `in`, the items are the
// positional parameters.
type forClause struct {
	name     string
	items    []word
	overArgs bool
	body     []step
}

type whileClause struct {
	cond, body []step
}

type caseClause struct {
	subject word
	arms    []caseArm
}

type caseArm struct {
	patterns []word
	body     []step
}

// pipeline is one or more commands joined by `|`; a leading `!` inverts its
// exit status.
type pipeline struct {
	negate bool
	cmds   []command
}

// step is a pipeline plus the operator that connects it to the previous one:
// "" or ";" (always run), "&&" (run on success) or "||" (run on failure).
type step struct {
	connector string
	pipeline  pipeline
}

// token is a lexical element: a word, an operator when op is non-empty (a
// line break is the operator "\n"), or the end of the input.
type token struct {
	op   string
	word word
	eof  bool
}

// operators are matched longest first.
var operators = []string{"&&", "||", ">>", ";;", "|", ";", "<", ">", "(", ")"}

// special lists the characters a backslash can escape. Any other backslash is
// kept as written, so Windows paths such as C:\tools\bin survive unquoted.
const special = " \t\"'$\\|&;<>()*?[#~"

// keywords are the words that mean something at the start of a command. `in`
// and `!` are recognized only where the grammar expects them.
var keywords = map[string]bool{
	"if": true, "then": true, "elif": true, "else": true, "fi": true,
	"for": true, "while": true, "do": true, "done": true,
	"case": true, "esac": true,
}

// syntaxError is what the parser reports. incomplete is set when the input
// ended inside a construct that more lines could finish (an `if` without its
// `fi`, an unclosed `$(`), which is how a reader knows to ask for another
// line rather than give up.
type syntaxError struct {
	msg        string
	incomplete bool
}

func (e *syntaxError) Error() string { return e.msg }

// incomplete reports whether err is a syntax error that more input could fix.
func incomplete(err error) bool {
	var se *syntaxError
	return errors.As(err, &se) && se.incomplete
}

// parser is a recursive-descent parser reading src from pos, with one token
// of lookahead. Errors are raised with fail and recovered in parse.
type parser struct {
	src    string
	pos    int
	tok    token
	peeked bool
	// cases counts the case clauses being parsed: inside one, `;;` ends an
	// arm; outside, it is two harmless semicolons.
	cases int
}

// parse turns a script (one line or many) into the steps to run, in order.
func parse(src string) (steps []step, err error) {
	defer func() {
		if r := recover(); r != nil {
			se, ok := r.(*syntaxError)
			if !ok {
				panic(r)
			}
			steps, err = nil, se
		}
	}()

	p := &parser{src: src}
	steps = p.list()
	if t := p.peek(); !t.eof {
		p.fail("unexpected '%s'", describe(t))
	}
	return steps, nil
}

func (p *parser) fail(format string, a ...any) {
	panic(&syntaxError{msg: fmt.Sprintf(format, a...)})
}

// failAt reports that want was expected where t stands; at the end of the
// input the script is merely unfinished.
func (p *parser) failAt(t token, want string) {
	if t.eof {
		panic(&syntaxError{msg: "expected " + want, incomplete: true})
	}
	p.fail("expected %s, found '%s'", want, describe(t))
}

func (p *parser) peek() token {
	if !p.peeked {
		p.tok, p.peeked = p.lex(), true
	}
	return p.tok
}

func (p *parser) next() token {
	t := p.peek()
	p.peeked = false
	return t
}

// describe names a token for a diagnostic.
func describe(t token) string {
	switch {
	case t.eof, t.op == "\n":
		return "newline"
	case t.op != "":
		return t.op
	}
	var b strings.Builder
	for _, part := range t.word {
		if part.subst {
			b.WriteString("$(...)")
			continue
		}
		b.WriteString(part.text)
	}
	return b.String()
}

// keyword returns the reserved word t is, or "".
func keyword(t token) string {
	if s := plain(t); keywords[s] {
		return s
	}
	return ""
}

// plain returns the text of a token that is a single unquoted word, or "".
func plain(t token) string {
	if t.eof || t.op != "" || len(t.word) != 1 || t.word[0].mode != raw || t.word[0].subst {
		return ""
	}
	return t.word[0].text
}

// lex reads the next token. A '#' at the start of a token begins a comment
// that runs to the end of the line.
func (p *parser) lex() token {
	src := p.src
	i := p.pos
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if i < len(src) && src[i] == '#' {
		for i < len(src) && src[i] != '\n' {
			i++
		}
	}
	if i >= len(src) {
		p.pos = i
		return token{eof: true}
	}
	if src[i] == '\n' {
		p.pos = i + 1
		return token{op: "\n"}
	}
	// A 1 or 2 that starts a word and is followed by > names the stream it
	// redirects; anywhere else ("a2>b") it is an ordinary character.
	if op, width := redirectOperator(src[i:]); width > 0 {
		p.pos = i + width
		return token{op: op}
	}
	if strings.HasPrefix(src[i:], ">&") || strings.HasPrefix(src[i:], "1>&") || strings.HasPrefix(src[i:], "2>&") {
		p.fail("only 2>&1 and >&2 can duplicate a stream")
	}
	if src[i] == '&' && !strings.HasPrefix(src[i:], "&&") {
		p.fail("background jobs (&) are not supported")
	}
	if op := matchOperator(src[i:]); op != "" {
		p.pos = i + len(op)
		return token{op: op}
	}

	var cur word
	var buf strings.Builder
	flush := func(mode quoting) {
		if buf.Len() > 0 {
			cur = append(cur, part{text: buf.String(), mode: mode})
			buf.Reset()
		}
	}

word:
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '&' || matchOperator(src[i:]) != "" {
			break
		}

		switch {
		case c == '\'':
			end := strings.IndexAny(src[i+1:], "'\n")
			if end < 0 || src[i+1+end] == '\n' {
				p.fail("unterminated single quote")
			}
			flush(raw)
			// An empty quoted string is still an argument, so always add a part.
			cur = append(cur, part{text: src[i+1 : i+1+end], mode: literal})
			i += end + 2

		case c == '"':
			flush(raw)
			before := len(cur)
			j := i + 1
			for ; j < len(src) && src[j] != '"'; j++ {
				if src[j] == '\n' {
					break
				}
				// Inside double quotes a backslash escapes only ", $ and \.
				if src[j] == '\\' && j+1 < len(src) && strings.IndexByte(`"$\`, src[j+1]) >= 0 {
					flush(double)
					cur = append(cur, part{text: string(src[j+1]), mode: literal})
					j++
					continue
				}
				if strings.HasPrefix(src[j:], "${") {
					end := p.braceEnd(j + 2)
					buf.WriteString(src[j : end+1])
					j = end
					continue
				}
				if strings.HasPrefix(src[j:], "$(") {
					flush(double)
					var body []step
					body, j = p.substitution(j + 2)
					cur = append(cur, part{subst: true, body: body, mode: double})
					j-- // the loop steps past the closing parenthesis
					continue
				}
				buf.WriteByte(src[j])
			}
			if j >= len(src) || src[j] != '"' {
				p.fail("unterminated double quote")
			}
			flush(double)
			if len(cur) == before {
				// "" is an argument too; "$@" with no parameters is not.
				cur = append(cur, part{mode: double})
			}
			i = j + 1

		case c == '\\':
			if i+1 < len(src) && strings.IndexByte(special, src[i+1]) >= 0 {
				flush(raw)
				cur = append(cur, part{text: string(src[i+1]), mode: literal})
				i += 2
				continue word
			}
			buf.WriteByte(c)
			i++

		case strings.HasPrefix(src[i:], "$("):
			flush(raw)
			var body []step
			body, i = p.substitution(i + 2)
			cur = append(cur, part{subst: true, body: body, mode: raw})

		case strings.HasPrefix(src[i:], "${"):
			// A parameter expansion is one piece, quotes and blanks in its
			// word included: ${x:-"a b"}. The expander reads it.
			end := p.braceEnd(i + 2)
			buf.WriteString(src[i : end+1])
			i = end + 1

		default:
			buf.WriteByte(c)
			i++
		}
	}
	flush(raw)
	p.pos = i
	return token{word: cur}
}

// braceEnd returns the index of the } that closes the ${ whose body starts
// at start, skipping quoted text and nested ${...}. A ${ that never closes on
// its line is a syntax error.
func (p *parser) braceEnd(start int) int {
	if end := braceEnd(p.src, start); end >= 0 {
		return end
	}
	p.fail("missing '}' after '${'")
	return 0
}

// braceEnd is the index in s of the } that closes a ${ whose body starts at
// start, or -1 when there is none before the end of the line.
func braceEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '\n':
			return -1
		case '\\':
			i++
		case '\'':
			j := strings.IndexAny(s[i+1:], "'\n")
			if j < 0 || s[i+1+j] == '\n' {
				return -1
			}
			i += j + 1
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\n' {
					return -1
				}
				if s[i] == '\\' {
					i++
				}
			}
			if i >= len(s) {
				return -1
			}
		case '$':
			if i+1 < len(s) && s[i+1] == '{' {
				depth++
				i++
			}
		case '}':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// substitution parses the commands of a `$(` whose body starts at start, and
// returns them with the position just past its closing parenthesis.
func (p *parser) substitution(start int) ([]step, int) {
	sub := &parser{src: p.src, pos: start}
	body := sub.list()
	if t := sub.next(); t.op != ")" {
		sub.failAt(t, "')' to close '$('")
	}
	return body, sub.pos
}

// redirectOperator matches a redirection that names a stream at the start of
// s, returning the operator it means and how much of s it takes: 2> and 2>>
// redirect standard error, 1> and 1>> are > and >>, 2>&1 sends standard error
// where standard output goes, and >&2 (or 1>&2) the reverse.
func redirectOperator(s string) (op string, width int) {
	for _, r := range []struct{ text, op string }{
		{"2>&1", "2>&1"}, {"1>&2", ">&2"}, {">&2", ">&2"},
		{"2>>", "2>>"}, {"2>", "2>"}, {"1>>", ">>"}, {"1>", ">"},
	} {
		if strings.HasPrefix(s, r.text) {
			return r.op, len(r.text)
		}
	}
	return "", 0
}

// isDup reports whether op duplicates one stream onto the other, and so takes
// no file name.
func isDup(op string) bool {
	return op == "2>&1" || op == ">&2"
}

func matchOperator(s string) string {
	for _, op := range operators {
		if strings.HasPrefix(s, op) {
			return op
		}
	}
	return ""
}

// separator reports whether t ends a command without connecting it to the
// next: a newline, a `;`, or outside a case clause a `;;`.
func (p *parser) separator(t token) bool {
	return t.op == "\n" || t.op == ";" || (t.op == ";;" && p.cases == 0)
}

// endsList reports whether t closes the list being parsed. Which closer is
// acceptable is for the caller to check.
func endsList(t token) bool {
	if t.eof || t.op == ")" || t.op == ";;" {
		return true
	}
	switch keyword(t) {
	case "then", "elif", "else", "fi", "do", "done", "esac":
		return true
	}
	return false
}

// list parses commands up to whatever closes the list: the end of the input,
// a `)`, a `;;`, or a word such as `then`, `fi` or `done`.
func (p *parser) list() []step {
	var steps []step
	for {
		for p.separator(p.peek()) {
			p.next()
		}
		if endsList(p.peek()) {
			return steps
		}

		connector := ""
		if len(steps) > 0 {
			connector = ";"
		}
		for {
			steps = append(steps, step{connector: connector, pipeline: p.pipeline()})
			t := p.peek()
			if t.op != "&&" && t.op != "||" {
				break
			}
			p.next()
			connector = t.op
			// The command must follow on the same line: a line that ends in
			// && is far more often a mistake than a continuation.
			if nt := p.peek(); nt.eof || nt.op == "\n" || endsList(nt) {
				p.fail("expected a command after '%s'", connector)
			}
		}

		if t := p.peek(); !p.separator(t) && !endsList(t) {
			p.fail("unexpected '%s'", describe(t))
		}
	}
}

func (p *parser) pipeline() pipeline {
	var pl pipeline
	if plain(p.peek()) == "!" {
		p.next()
		pl.negate = true
	}
	compounds := 0
	for {
		cmd := p.command()
		if cmd.compound != nil {
			compounds++
		}
		pl.cmds = append(pl.cmds, cmd)
		if p.peek().op != "|" {
			break
		}
		p.next()
	}
	// The stages of a pipeline run at the same time, and a compound command
	// reads and writes the shell's variables as it goes, so only one fits.
	if compounds > 1 {
		p.fail("a pipeline can hold only one if, for, while or case")
	}
	return pl
}

func isRedirect(op string) bool {
	switch op {
	case "<", ">", ">>", "2>", "2>>", "2>&1", ">&2":
		return true
	}
	return false
}

func (p *parser) redirect() redirect {
	op := p.next().op
	if isDup(op) {
		return redirect{op: op}
	}
	t := p.next()
	if t.eof || t.op != "" {
		p.fail("expected a file name after '%s'", op)
	}
	return redirect{op: op, target: t.word}
}

func (p *parser) command() command {
	var cmd command
	switch kw := keyword(p.peek()); kw {
	case "":
	case "if":
		cmd.compound = p.ifClause()
	case "for":
		cmd.compound = p.forClause()
	case "while":
		cmd.compound = p.whileClause()
	case "case":
		cmd.compound = p.caseClause()
	default:
		p.fail("unexpected '%s'", kw)
	}
	if cmd.compound != nil {
		for isRedirect(p.peek().op) {
			cmd.redirects = append(cmd.redirects, p.redirect())
		}
		return cmd
	}

	for {
		t := p.peek()
		if t.eof {
			break
		}
		if t.op == "" {
			cmd.words = append(cmd.words, p.next().word)
			continue
		}
		if !isRedirect(t.op) {
			break
		}
		cmd.redirects = append(cmd.redirects, p.redirect())
	}
	if len(cmd.words) == 0 {
		if len(cmd.redirects) > 0 {
			p.fail("redirection without a command")
		}
		p.fail("unexpected '%s'", describe(p.peek()))
	}
	return cmd
}

// expect consumes the reserved word kw.
func (p *parser) expect(kw string) {
	if t := p.peek(); plain(t) != kw {
		p.failAt(t, "'"+kw+"'")
	}
	p.next()
}

// condition parses the commands after `if`, `elif` or `while`, which must
// hold at least one.
func (p *parser) condition(after string) []step {
	cond := p.list()
	if len(cond) == 0 {
		p.failAt(p.peek(), "a command after '"+after+"'")
	}
	return cond
}

func (p *parser) ifClause() *ifClause {
	p.next()
	c := &ifClause{}
	after := "if"
	for {
		cond := p.condition(after)
		p.expect("then")
		c.branches = append(c.branches, branch{cond: cond, body: p.list()})

		switch t := p.peek(); keyword(t) {
		case "elif":
			p.next()
			after = "elif"
			continue
		case "else":
			p.next()
			c.elseBody = p.list()
			p.expect("fi")
		case "fi":
			p.next()
		default:
			p.failAt(t, "'fi'")
		}
		return c
	}
}

func (p *parser) forClause() *forClause {
	p.next()
	t := p.next()
	c := &forClause{name: plain(t)}
	if !isName(c.name) {
		p.failAt(t, "a variable name after 'for'")
	}

	if plain(p.peek()) == "in" {
		p.next()
		for t := p.peek(); !t.eof && t.op == ""; t = p.peek() {
			c.items = append(c.items, p.next().word)
		}
	} else {
		c.overArgs = true
	}
	for t := p.peek(); t.op == ";" || t.op == "\n"; t = p.peek() {
		p.next()
	}
	p.expect("do")
	c.body = p.list()
	p.expect("done")
	return c
}

func (p *parser) whileClause() *whileClause {
	p.next()
	c := &whileClause{cond: p.condition("while")}
	p.expect("do")
	c.body = p.list()
	p.expect("done")
	return c
}

func (p *parser) caseClause() *caseClause {
	p.next()
	t := p.next()
	if t.eof || t.op != "" {
		p.failAt(t, "a word after 'case'")
	}
	c := &caseClause{subject: t.word}
	p.expect("in")

	p.cases++
	defer func() { p.cases-- }()
	for {
		for p.peek().op == "\n" {
			p.next()
		}
		if plain(p.peek()) == "esac" {
			p.next()
			return c
		}
		if p.peek().op == "(" {
			p.next()
		}

		var arm caseArm
		for {
			t := p.next()
			if t.eof || t.op != "" {
				p.failAt(t, "a pattern or 'esac'")
			}
			arm.patterns = append(arm.patterns, t.word)
			if p.peek().op != "|" {
				break
			}
			p.next()
		}
		if t := p.next(); t.op != ")" {
			p.failAt(t, "')' after a case pattern")
		}
		arm.body = p.list()
		c.arms = append(c.arms, arm)
		if p.peek().op == ";;" {
			p.next()
		}
	}
}
