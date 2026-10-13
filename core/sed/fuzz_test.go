package sed_test

import (
	"io"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sed"
)

// FuzzScript checks that no script makes the parser or the executor panic,
// in either regular expression syntax. Scripts are run in sandbox mode, so
// none can read or write a file, and those that branch or restart the cycle
// (b, t, T, D) are only parsed: such a script may legitimately never end.
func FuzzScript(f *testing.F) {
	for _, s := range []string{
		"", "p", "s/a/b/", "s/a/b/g;s/x*/-/2p", `s/\(a\)b*/[\1&]/`, "2,4d", "$!N;P", "/a/,/b/{s/./X/;p}",
		"1~2d", "0,/a/d", "y/abc/xyz/", "a\\\ntext", "i text", "c\\\nnew", "=;l;l 5", "n;x;G;H;h;g", "q5", "Q",
		`s/a/\U&\E-\l\u&/`, "s/[[:alpha:]]\\{2,\\}//", `/\<a\>/I!p`, "{", "}", "s/a", "s/a/b/w", "5!{p", `\%a%p`,
		"z;F", "//p", "s/(a|b)+/\\n/", "addr", ":a", "#n\np", "2q;3,+2p;4,~2p", "s/a/b/3g", "y/a/",
	} {
		f.Add(s, "alpha\nbeta gamma\n\nlast", false)
		f.Add(s, "a\n", true)
	}
	f.Fuzz(func(t *testing.T, script, input string, extended bool) {
		if len(script) > 200 || len(input) > 400 {
			return
		}
		opts := sed.Options{Scripts: []sed.Script{{Text: script}}, Extended: extended, Sandbox: true}
		in := input
		if strings.ContainsAny(script, "btTD") {
			// Parsed, with nothing to run on.
			in = ""
		}
		s := core.Streams{In: strings.NewReader(in), Out: io.Discard, Err: io.Discard}
		_ = sed.New(opts, s).Run(nil)
	})
}
