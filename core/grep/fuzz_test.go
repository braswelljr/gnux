package grep_test

import (
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/grep"
)

// FuzzPattern checks that no pattern makes the translator or the matcher
// panic, in any syntax and under the options that change how a pattern is
// compiled or a line is printed.
func FuzzPattern(f *testing.F) {
	for _, p := range []string{
		"", "a", "a*", `\(a\)*b`, `a\{2,3\}`, "a{2,3}", "(a|b)+", "[[:alpha:]]", "[^]a-z]", `\<w\>`, `\bw\B`,
		"^a$", "a\nb", "[", "(", `\`, "a**", "{1", `\w\W\s\S`, "[a-", "[[:bogus:]]", `a\|b`, "*a", "+a", "?", "a{,3}",
		`\(`, `\)`, ")", "[[.a.]]", "[[=a=]]", "\x00", "é+", `\(a\)\1`, "a{99999}", "^*", "x\\{1",
	} {
		for mode := byte(0); mode < 12; mode++ {
			f.Add(p, "alpha beta\nAlpha\n\nword w\nlast", mode)
		}
	}
	f.Fuzz(func(t *testing.T, pattern, input string, mode byte) {
		if len(pattern) > 200 || len(input) > 400 {
			return
		}
		opts := grep.Options{
			Patterns:     []string{pattern},
			Syntax:       []grep.Syntax{grep.SyntaxBasic, grep.SyntaxExtended, grep.SyntaxFixed}[int(mode)%3],
			IgnoreCase:   mode&4 != 0,
			WordRegexp:   mode&8 != 0,
			LineRegexp:   mode&16 != 0,
			OnlyMatching: mode&32 != 0,
			Invert:       mode&64 != 0,
			LineNumber:   true,
			ByteOffset:   mode&128 != 0,
			After:        int(mode) % 3,
			Before:       int(mode) % 2,
		}
		s := core.Streams{In: strings.NewReader(input), Out: io.Discard, Err: io.Discard}
		_ = grep.New(opts, s).Run(nil)
	})
}
