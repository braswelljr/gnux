package shell

import (
	"io"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// read is the read builtin: read [-r] [NAME...]. It reads one line from
// standard input and splits it into fields at the characters of $IFS (blank,
// tab and newline when IFS is unset): each NAME gets one field, the last one
// the rest of the line, and with no NAME the whole line goes to REPLY. Without
// -r a backslash takes the next character literally and joins a line ending
// in one to the next. The status is 1 at the end of the input, after setting
// the names from whatever was read, so `while read line` ends there.
//
// The line is read one byte at a time, so that nothing past it is taken from
// a stream the next command reads. In a stage of a pipeline the line is read
// but nothing is set, as in a subshell.
func (sh *Shell) read(args []string, s core.Streams, piped bool) int {
	raw := false
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" {
		a := args[0]
		args = args[1:]
		if a == "--" {
			break
		}
		for _, c := range a[1:] {
			if c != 'r' {
				sh.errorf(s, "read: -%c: invalid option", c)
				return 2
			}
			raw = true
		}
	}
	names := args
	if len(names) == 0 {
		names = []string{"REPLY"}
	}
	for _, name := range names {
		if !isName(name) {
			sh.errorf(s, "read: '%s': not a valid identifier", name)
			return 2
		}
	}

	line, escaped, eof := readLine(s.In, raw)
	if eof && len(line) == 0 {
		if !piped {
			for _, name := range names {
				sh.vars[name] = ""
			}
		}
		return 1
	}

	ifs, set := sh.lookup("IFS")
	if !set {
		ifs = " \t\n"
	}
	fields := splitFields(line, escaped, ifs, len(names))
	if !piped {
		for i, name := range names {
			sh.vars[name] = fields[i]
		}
	}
	if eof {
		return 1
	}
	return 0
}

// readLine reads up to a newline, which it drops. Unless raw is set, a
// backslash escapes the character after it (marked in escaped) and a
// backslash-newline joins the next line on. eof is true when the input ended
// before a newline.
func readLine(in io.Reader, raw bool) (line []rune, escaped []bool, eof bool) {
	var pending []byte // the bytes of a character not yet complete
	var one [1]byte
	escape := false
	for {
		n, err := in.Read(one[:])
		if n == 0 {
			if err != nil {
				return line, escaped, true
			}
			continue
		}
		c := one[0]
		if len(pending) == 0 && c < 0x80 {
			switch {
			case escape:
				escape = false
				if c == '\n' {
					continue
				}
				line, escaped = append(line, rune(c)), append(escaped, true)
				continue
			case c == '\\' && !raw:
				escape = true
				continue
			case c == '\n':
				return line, escaped, false
			}
			line, escaped = append(line, rune(c)), append(escaped, false)
			continue
		}
		// A multi-byte character, gathered whole before it is added.
		pending = append(pending, c)
		if r, size := decodeRune(pending); size > 0 {
			line, escaped = append(line, r), append(escaped, escape)
			escape = false
			pending = pending[:0]
		}
	}
}

// decodeRune returns the character in b once b holds all of it: size is 0
// while more bytes are needed. A byte that cannot start or continue one is
// taken as it is.
func decodeRune(b []byte) (rune, int) {
	want := 1
	switch c := b[0]; {
	case c >= 0xF0:
		want = 4
	case c >= 0xE0:
		want = 3
	case c >= 0xC0:
		want = 2
	}
	if len(b) < want {
		return 0, 0
	}
	if r := []rune(string(b)); len(r) == 1 {
		return r[0], len(b)
	}
	return rune(b[0]), len(b)
}

// splitFields splits line into exactly n fields at the characters of ifs, as
// read does: blanks in ifs (space, tab, newline) are trimmed from both ends
// and run together; any other ifs character ends a field on its own. The last
// field takes the rest of the line. Escaped characters never split.
func splitFields(line []rune, escaped []bool, ifs string, n int) []string {
	fields := make([]string, n)
	if ifs == "" {
		fields[0] = string(line)
		return fields
	}
	isIFS := func(i int) bool { return !escaped[i] && strings.ContainsRune(ifs, line[i]) }
	isBlank := func(i int) bool { return isIFS(i) && strings.ContainsRune(" \t\n", line[i]) }

	i, end := 0, len(line)
	for i < end && isBlank(i) {
		i++
	}
	for end > i && isBlank(end-1) {
		end--
	}
	for f := 0; f < n; f++ {
		if f == n-1 {
			fields[f] = string(line[i:end])
			break
		}
		j := i
		for j < end && !isIFS(j) {
			j++
		}
		fields[f] = string(line[i:j])
		// One delimiter ends the field, with any blanks around it.
		for j < end && isBlank(j) {
			j++
		}
		if j < end && isIFS(j) && !isBlank(j) {
			j++
			for j < end && isBlank(j) {
				j++
			}
		}
		i = j
	}
	return fields
}
