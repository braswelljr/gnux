package shell

import (
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

// part is a run of a word that shares one kind of quoting.
type part struct {
	text string
	mode quoting
}

// word is one argument before expansion: `a"$b"'c'` is a single word of three
// parts.
type word []part

// redirect is one `<`, `>`, `>>`, `2>` or `2>>` with its target.
type redirect struct {
	op     string
	target word
}

// command is a single program invocation: its words and redirections.
type command struct {
	words     []word
	redirects []redirect
}

// pipeline is one or more commands joined by `|`.
type pipeline []command

// step is a pipeline plus the operator that connects it to the previous one:
// "" or ";" (always run), "&&" (run on success) or "||" (run on failure).
type step struct {
	connector string
	pipeline  pipeline
}

// token is a lexical element: a word, or an operator when op is non-empty.
type token struct {
	op   string
	word word
}

// operators are matched longest first.
var operators = []string{"&&", "||", ">>", "|", ";", "<", ">"}

// special lists the characters a backslash can escape. Any other backslash is
// kept as written, so Windows paths such as C:\tools\bin survive unquoted.
const special = " \t\"'$\\|&;<>*?[#~"

// lex splits a line into tokens. A '#' at the start of a token begins a
// comment that runs to the end of the line.
func lex(line string) ([]token, error) {
	var tokens []token
	var cur word
	var buf strings.Builder
	inWord := false

	flush := func(mode quoting) {
		if buf.Len() > 0 {
			cur = append(cur, part{text: buf.String(), mode: mode})
			buf.Reset()
		}
	}
	endWord := func() {
		flush(raw)
		if inWord {
			tokens = append(tokens, token{word: cur})
		}
		cur, inWord = nil, false
	}

	for i := 0; i < len(line); {
		c := line[i]

		if c == ' ' || c == '\t' {
			endWord()
			i++
			continue
		}
		if c == '#' && !inWord {
			break
		}
		// A 2 that starts a word and is followed by > redirects standard
		// error; anywhere else ("a2>b") it is an ordinary character.
		if !inWord && (strings.HasPrefix(line[i:], "2>")) {
			op := "2>"
			if strings.HasPrefix(line[i:], "2>>") {
				op = "2>>"
			}
			tokens = append(tokens, token{op: op})
			i += len(op)
			continue
		}
		if op := matchOperator(line[i:]); op != "" {
			endWord()
			tokens = append(tokens, token{op: op})
			i += len(op)
			continue
		}

		inWord = true
		switch c {
		case '\'':
			end := strings.IndexByte(line[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("unterminated single quote")
			}
			flush(raw)
			// An empty quoted string is still an argument, so always add a part.
			cur = append(cur, part{text: line[i+1 : i+1+end], mode: literal})
			i += end + 2

		case '"':
			flush(raw)
			j := i + 1
			for ; j < len(line) && line[j] != '"'; j++ {
				// Inside double quotes a backslash escapes only ", $ and \.
				if line[j] == '\\' && j+1 < len(line) && strings.IndexByte(`"$\`, line[j+1]) >= 0 {
					flush(double)
					cur = append(cur, part{text: string(line[j+1]), mode: literal})
					j++
					continue
				}
				buf.WriteByte(line[j])
			}
			if j >= len(line) {
				return nil, fmt.Errorf("unterminated double quote")
			}
			flush(double)
			cur = append(cur, part{mode: double})
			i = j + 1

		case '\\':
			if i+1 < len(line) && strings.IndexByte(special, line[i+1]) >= 0 {
				flush(raw)
				cur = append(cur, part{text: string(line[i+1]), mode: literal})
				i += 2
				continue
			}
			buf.WriteByte(c)
			i++

		default:
			buf.WriteByte(c)
			i++
		}
	}
	endWord()
	return tokens, nil
}

func matchOperator(s string) string {
	for _, op := range operators {
		if strings.HasPrefix(s, op) {
			return op
		}
	}
	return ""
}

// parse turns a line into the steps to run, in order.
func parse(line string) ([]step, error) {
	tokens, err := lex(line)
	if err != nil {
		return nil, err
	}

	var steps []step
	connector := ""
	var pl pipeline
	var cmd command

	endCommand := func(op string) error {
		if len(cmd.words) == 0 {
			if len(cmd.redirects) > 0 {
				return fmt.Errorf("redirection without a command")
			}
			return fmt.Errorf("unexpected '%s'", op)
		}
		pl = append(pl, cmd)
		cmd = command{}
		return nil
	}

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t.op {
		case "":
			cmd.words = append(cmd.words, t.word)

		case "<", ">", ">>", "2>", "2>>":
			if i+1 >= len(tokens) || tokens[i+1].op != "" {
				return nil, fmt.Errorf("expected a file name after '%s'", t.op)
			}
			i++
			cmd.redirects = append(cmd.redirects, redirect{op: t.op, target: tokens[i].word})

		case "|":
			if err := endCommand(t.op); err != nil {
				return nil, err
			}

		case ";", "&&", "||":
			// A stray ';' (as in "a;" or ";;") is harmless; a dangling && is not.
			if len(cmd.words) == 0 && len(cmd.redirects) == 0 && len(pl) == 0 && t.op == ";" {
				continue
			}
			if err := endCommand(t.op); err != nil {
				return nil, err
			}
			steps = append(steps, step{connector: connector, pipeline: pl})
			pl, connector = nil, t.op
		}
	}

	if len(cmd.words) > 0 || len(cmd.redirects) > 0 || len(pl) > 0 {
		if err := endCommand("newline"); err != nil {
			return nil, err
		}
		steps = append(steps, step{connector: connector, pipeline: pl})
	} else if connector == "&&" || connector == "||" {
		return nil, fmt.Errorf("expected a command after '%s'", connector)
	}
	return steps, nil
}
