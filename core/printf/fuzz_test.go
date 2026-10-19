package printf_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printf"
)

// FuzzFormat checks that no format and arguments make printfx panic: the
// directives, flags, widths, escapes (\\c, \\u, octal) and the numeric
// conversions of each argument ('x, 0x1p3, 1e999, -0).
func FuzzFormat(f *testing.F) {
	for _, s := range [][2]string{
		{"%s\\n", "a"}, {"%d %i", "12 0x1F"}, {"%5.2f|%-8e|%g", "3.14159 1e10 -0"}, {"%x %X %o", "255 255 8"},
		{"%c%c", "abc"}, {"%b", "a\\tb\\c"}, {"%q", "it's"}, {"%%", ""}, {"%*d", "5 42"}, {"%.*s", "2 abcdef"},
		{"\\u00e9\\x41\\101", ""}, {"%d", "'a"}, {"%a", "1"}, {"%", ""}, {"%z", "1"}, {"%d", "x"},
		{"%-+ #010.3f", "-1.5"}, {"%s %s", "only"}, {"%1$s", "a"}, {"\\", ""}, {"%ld %hhd %Lf", "1 2 3"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, format, args string) {
		if len(format) > 100 || len(args) > 200 || tooLarge(format, args) {
			return
		}
		s := core.Streams{In: strings.NewReader(""), Out: &budget{left: 1 << 16}, Err: io.Discard}
		_ = printf.New(printf.Options{}, s).Run(append([]string{format}, strings.Fields(args)...))
	})
}

// budget is an output that fails once it has taken limit bytes, so that a
// fuzzed command line asking for endless output ends with a write error.
type budget struct{ left int }

func (b *budget) Write(p []byte) (int, error) {
	if len(p) > b.left {
		return 0, errors.New("output budget spent")
	}
	b.left -= len(p)
	return len(p), nil
}

// tooLarge reports a run of more than four digits, which a width,
// precision or repeat count would turn into a huge allocation: a size the
// user asked for, not a bug.
func tooLarge(ss ...string) bool {
	for _, s := range ss {
		run := 0
		for i := range len(s) {
			if s[i] >= '0' && s[i] <= '9' {
				if run++; run > 4 {
					return true
				}
			} else {
				run = 0
			}
		}
	}
	return false
}
