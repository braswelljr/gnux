package xargs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// errTooLong reports an input item that cannot fit the command-line size
// limit even on its own.
var errTooLong = errors.New("argument line too long")

// quoteError reports a quote that was not closed on its line; its value is
// the quote character.
type quoteError byte

func (q quoteError) Error() string {
	kind := "single"
	if q == '"' {
		kind = "double"
	}
	return "unmatched " + kind + " quote; by default quotes are special to xargs unless you use the -0 option"
}

// ParseDelimiter turns the argument of -d into the delimiter byte, as GNU
// xargs reads it: a single character stands for itself, and a string starting
// with a backslash is a C escape (\a \b \f \n \r \t \v \\), an octal value
// (\NNN, so \0 is NUL) or a hexadecimal one (\xHH). The error is worded as
// GNU's diagnostic.
func ParseDelimiter(spec string) (byte, error) {
	if len(spec) == 1 {
		return spec[0], nil
	}
	if len(spec) < 2 || spec[0] != '\\' {
		return 0, delimiterErrorf("Invalid input delimiter specification %s: the delimiter must be "+
			"either a single character or an escape sequence starting with \\.", spec)
	}

	simple := map[byte]byte{'a': '\a', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v', '\\': '\\'}
	if c, ok := simple[spec[1]]; ok {
		if len(spec) > 2 {
			return 0, trailing(spec, spec[2:])
		}
		return c, nil
	}

	digits, base, limit := spec[1:], 8, "377"
	if spec[1] == 'x' {
		digits, base, limit = spec[2:], 16, "ff"
	}
	n := 0
	for n < len(digits) && isDigit(digits[n], base) {
		n++
	}
	if n == 0 {
		return 0, delimiterErrorf("Invalid escape sequence %s in input delimiter specification.", spec)
	}
	if v, err := strconv.ParseUint(digits[:n], base, 8); err == nil {
		if n < len(digits) {
			return 0, trailing(spec, digits[n:])
		}
		return byte(v), nil
	}
	return 0, delimiterErrorf("Invalid escape sequence %s in input delimiter specification; "+
		"character values must not exceed %s.", spec, limit)
}

// delimiterError is a -d argument that cannot be read. Its text is GNU's
// diagnostic, a full sentence, which is why it is not built with fmt.Errorf.
type delimiterError string

func (e delimiterError) Error() string { return string(e) }

func delimiterErrorf(format string, a ...any) error {
	return delimiterError(fmt.Sprintf(format, a...))
}

// trailing is the error for an escape sequence followed by more characters.
func trailing(spec, rest string) error {
	return delimiterErrorf("Invalid escape sequence %s in input delimiter specification; "+
		"trailing characters %s not recognized.", spec, rest)
}

// isDigit reports whether c is a digit in base 8 or 16.
func isDigit(c byte, base int) bool {
	switch {
	case c >= '0' && c <= '7':
		return true
	case base == 16:
		return c == '8' || c == '9' || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
	}
	return false
}

// input splits the item stream into items, one per call of next.
type input struct {
	r   *bufio.Reader
	buf []byte // the item being assembled, reused between items

	// max is the longest item, in bytes, that can ever be passed: one more
	// could not fit the size limit even without a command. It also bounds
	// the memory a stream with no separators can take.
	max int

	delimited bool   // -0 / -d: items end at delim, nothing else is special
	delim     byte   // the delimiter when delimited
	wholeLine bool   // -I: blanks do not separate, a line is one item
	eofStr    string // -E: the item that ends the input ("" for none)

	prev    byte   // the byte before the current one, for line continuation
	onNUL   func() // called once when a NUL byte is dropped from the input
	nulSeen bool
	done    bool
}

// next returns the next item. endsLine reports that the item completed an
// input line as -L counts them: a line whose last character is a blank
// continues on the next one. The error is io.EOF at the end of the input
// (physical, or the logical one of -E), a quoteError, errTooLong or a read
// error.
func (in *input) next() (item string, endsLine bool, err error) {
	if in.done {
		return "", false, io.EOF
	}
	if in.delimited {
		item, err = in.delimitedItem()
		endsLine = true
	} else {
		item, endsLine, err = in.quotedItem()
	}
	if err != nil {
		in.done = true
		return "", false, err
	}
	if in.eofStr != "" && item == in.eofStr {
		in.done = true
		return "", false, io.EOF
	}
	return item, endsLine, nil
}

// delimitedItem reads up to the next delimiter. An empty item between two
// delimiters is an item; the end of the input right after a delimiter is not.
func (in *input) delimitedItem() (string, error) {
	in.buf = in.buf[:0]
	for {
		chunk, err := in.r.ReadSlice(in.delim)
		switch {
		case err == nil:
			chunk = chunk[:len(chunk)-1]
			if len(in.buf)+len(chunk) > in.max {
				return "", errTooLong
			}
			if len(in.buf) == 0 {
				// The common case: the item is whole in the reader's buffer
				// and is copied exactly once, into the string.
				return string(chunk), nil
			}
			in.buf = append(in.buf, chunk...)
			return string(in.buf), nil
		case errors.Is(err, bufio.ErrBufferFull):
			in.buf = append(in.buf, chunk...)
			if len(in.buf) > in.max {
				return "", errTooLong
			}
		case errors.Is(err, io.EOF):
			in.buf = append(in.buf, chunk...)
			switch {
			case len(in.buf) == 0:
				return "", io.EOF
			case len(in.buf) > in.max:
				return "", errTooLong
			}
			return string(in.buf), nil
		default:
			return "", err
		}
	}
}

// The states of the default-mode parser.
const (
	stateSpace     = iota // between items
	stateNormal           // inside an item
	stateQuote            // inside a quoted part of an item
	stateBackslash        // right after a backslash
)

// quotedItem reads one item in the default mode: items are separated by
// blanks and newlines, a quoted part is taken literally up to the same quote
// on the same line, and a backslash protects the next character, a newline
// included. With wholeLine set (-I) only a newline ends an item.
func (in *input) quotedItem() (item string, endsLine bool, err error) {
	in.buf = in.buf[:0]
	state := stateSpace
	// quoted records that the item had a quoted part, which makes '' and ""
	// an item although they contribute no characters.
	var quoted bool
	var quote byte

	for {
		c, err := in.r.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return "", false, err
			}
			switch {
			case state == stateQuote:
				return "", false, quoteError(quote)
			case len(in.buf) == 0 && !quoted:
				return "", false, io.EOF
			}
			return string(in.buf), true, nil
		}
		prev := in.prev
		in.prev = c

		switch state {
		case stateSpace:
			if c == ' ' || (c >= '\t' && c <= '\r') {
				continue
			}
			state = stateNormal
			fallthrough
		case stateNormal:
			switch {
			case c == '\n':
				return string(in.buf), prev != ' ' && prev != '\t', nil
			case (c == ' ' || c == '\t') && !in.wholeLine:
				return string(in.buf), false, nil
			case c == '\\':
				state = stateBackslash
				continue
			case c == '\'' || c == '"':
				state, quote, quoted = stateQuote, c, true
				continue
			}
		case stateQuote:
			if c == '\n' {
				return "", false, quoteError(quote)
			}
			if c == quote {
				state = stateNormal
				continue
			}
		case stateBackslash:
			state = stateNormal
		}

		if c == 0 {
			// An argument cannot hold a NUL; say so once and leave it out.
			if !in.nulSeen {
				in.nulSeen = true
				in.onNUL()
			}
			continue
		}
		if len(in.buf) >= in.max {
			return "", false, errTooLong
		}
		in.buf = append(in.buf, c)
	}
}
