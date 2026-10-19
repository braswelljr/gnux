package tr_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tr"
)

// FuzzSets checks that no SET1 and SET2 make trx panic: ranges, classes,
// equivalence classes, repeats ([x*], [x*n]), escapes and octal, with every
// combination of -c, -d, -s and -t.
func FuzzSets(f *testing.F) {
	for _, s := range [][2]string{
		{"a-z", "A-Z"}, {"[:lower:]", "[:upper:]"}, {"[:digit:]", ""}, {"a-c", "[x*]"}, {"abc", "[x*2]y"},
		{"\\n\\t", "  "}, {"\\101-\\132", "a-z"}, {"[=a=]", "b"}, {"z-a", "b"}, {"[:alpha:]", "[:digit:]"},
		{"a", ""}, {"", "a"}, {"[", "]"}, {"\\", "x"}, {"[:bogus:]", "x"}, {"[a*]", "b"}, {"a-", "-z"},
	} {
		f.Add(s[0], s[1], "hello, world\n123\n", uint8(0))
		f.Add(s[0], s[1], "aabbcc", uint8(15))
	}
	f.Fuzz(func(t *testing.T, set1, set2, input string, flags uint8) {
		if len(set1) > 60 || len(set2) > 60 || len(input) > 200 || tooLarge(set1, set2) {
			return
		}
		opts := tr.Options{Complement: flags&1 != 0, Delete: flags&2 != 0, Squeeze: flags&4 != 0, Truncate: flags&8 != 0}
		args := []string{set1}
		if flags&16 == 0 {
			args = append(args, set2)
		}
		s := core.Streams{In: strings.NewReader(input), Out: &budget{left: 1 << 16}, Err: io.Discard}
		_ = tr.New(opts, s).Run(args)
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
