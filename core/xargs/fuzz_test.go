package xargs_test

import (
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/xargs"
)

// FuzzInput checks that no input makes the item parser or the batching
// panic, in every way of splitting the input, and that no command line ever
// goes over the size limit it was given. Nothing is run: Exec stands in for
// every command.
func FuzzInput(f *testing.F) {
	for _, in := range []string{
		"", "a b c\n", "one\ntwo\n", `'a b' "c d" e\ f`, "'unmatched", `"unmatched`, `trailing\`, "a\x00b\x00",
		"  lead  \n\n trail  \n", "x _ y\n", strings.Repeat("word ", 60), "a,b,,c", "line one \ncontinued\nnext\n",
	} {
		for mode := byte(0); mode < 8; mode++ {
			f.Add(in, mode, 3)
		}
	}
	f.Fuzz(func(t *testing.T, input string, mode byte, n int) {
		if len(input) > 600 {
			return
		}
		n = n%8 + 1
		if n < 1 {
			n = 1
		}
		opts := xargs.Options{NoRunIfEmpty: mode&64 != 0, Exit: mode&128 != 0}
		switch mode % 8 {
		case 1:
			opts.Null = true
		case 2:
			opts.Delimiter = ","
		case 3:
			opts.MaxArgs = n
		case 4:
			opts.MaxLines = n
		case 5:
			opts.Replace = "{}"
		case 6:
			opts.EOF = "_"
		case 7:
			opts.MaxChars = 40 + n
		}
		x := xargs.New(opts, core.Streams{In: strings.NewReader(input), Out: io.Discard, Err: io.Discard})
		x.Exec = func(name string, args []string, _ core.Streams) (int, bool) {
			if opts.MaxChars > 0 {
				size := len(name) + 1
				for _, a := range args {
					size += len(a) + 1
				}
				if size > opts.MaxChars {
					t.Errorf("command line of %d bytes is over -s %d: %q %q", size, opts.MaxChars, name, args)
				}
			}
			return 0, true
		}
		_ = x.Run([]string{"cmd", "{}"})
	})
}
