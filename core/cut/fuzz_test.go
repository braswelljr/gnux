package cut_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cut"
)

// FuzzList checks that no LIST, delimiter or input makes cutx panic, in each
// mode, with and without --complement, -s and an output delimiter.
func FuzzList(f *testing.F) {
	for _, l := range []string{"1", "1,3", "2-", "-2", "1-3,5-", "3-1", "0", "", ",", "1-2-3", "1 2", "-", "a",
		"18446744073709551616", "1,1,1", "5-5", "2-,1"} {
		f.Add(l, ":", "a:b:c\nno delim\n\n:x:\n", uint8(0))
		f.Add(l, "", "a\tb\nhé\n", uint8(0x0f))
	}
	f.Fuzz(func(t *testing.T, list, delim, input string, flags uint8) {
		if len(list) > 60 || len(delim) > 4 || len(input) > 300 {
			return
		}
		opts := cut.Options{
			Mode:               cut.Mode(1 + flags%3),
			List:               list,
			Delimiter:          delim,
			OnlyDelimited:      flags&4 != 0,
			Complement:         flags&8 != 0,
			OutputDelimiter:    "|",
			OutputDelimiterSet: flags&16 != 0,
			ZeroTerminated:     flags&32 != 0,
		}
		s := core.Streams{In: strings.NewReader(input), Out: &budget{left: 1 << 16}, Err: io.Discard}
		_ = cut.New(opts, s).Run(nil)
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
