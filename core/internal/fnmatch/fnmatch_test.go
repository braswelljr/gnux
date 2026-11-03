package fnmatch

import (
	"path"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern, name string
		fold, want    bool
	}{
		{"", "", false, true},
		{"", "a", false, false},
		{"*", "", false, true},
		{"*", "anything/at all", false, true},
		{"*.go", "main.go", false, true},
		{"*.go", "main.go.bak", false, false},
		{"*.go", ".hidden.go", false, true}, // no FNM_PERIOD: a leading dot is ordinary
		{"a*b*c", "aXbYbZc", false, true},
		{"a*b*c", "aXbYbZ", false, false},
		{"?", "é", false, true}, // one character, not one byte
		{"??", "é", false, false},
		{"a?c", "a/c", false, true}, // no FNM_PATHNAME: ? matches a slash
		{"[abc]", "b", false, true},
		{"[!abc]", "b", false, false},
		{"[^abc]", "d", false, true},
		{"[a-c]x", "bx", false, true},
		{"[]]", "]", false, true},
		{"[!]]", "]", false, false},
		{"[a-]", "-", false, true},
		{"[[:digit:]]*", "7up", false, true},
		{"[[:alpha:]]", "é", false, true},
		{"[[:upper:]]", "a", false, false},
		{"[[:upper:]]", "a", true, true},
		{"[", "[", false, true},     // never closed: a literal [
		{"a[b", "a[b", false, true}, // the same in the middle
		{`\*`, "*", false, true},
		{`\*`, "a", false, false},
		{`[\]]`, "]", false, true},
		{"README*", "readme.md", false, false},
		{"README*", "readme.md", true, true},
		{"[a-c]", "B", true, true},
		{"*é*", "café au lait", false, true},
	}
	for _, tt := range tests {
		if got := Match(tt.pattern, tt.name, tt.fold); got != tt.want {
			t.Errorf("Match(%q, %q, fold=%v) = %v, want %v", tt.pattern, tt.name, tt.fold, got, tt.want)
		}
	}
}

// FuzzMatchesPathMatch holds Match to path.Match where their syntaxes agree:
// names without a slash (which path.Match treats specially), patterns
// without escapes, classes, negation or a leading ] in a set (which path.Match
// reads differently; the table above pins those), and patterns path.Match
// accepts, all ASCII: after a * path.Match steps through a name a byte at a
// time, so its ? can match half of a multi-byte character, and on a stray byte
// it gives up where fnmatch, like glibc's, takes the byte as one character.
func FuzzMatchesPathMatch(f *testing.F) {
	for _, s := range []string{"*.go", "a?c", "[a-c]*", "[!x]y", "*a*b", "[]a]", "x[", "**"} {
		f.Add(s, "abc.go")
	}
	f.Fuzz(func(t *testing.T, pattern, name string) {
		if !ascii(name) || !ascii(pattern) || strings.ContainsAny(name, "/") || strings.ContainsAny(pattern, `\/^!`) || strings.Contains(pattern, "[:") ||
			strings.Contains(pattern, "[]") || strings.Contains(pattern, "[!]") {
			return
		}
		want, err := path.Match(pattern, name)
		if err != nil {
			return
		}
		if got := Match(pattern, name, false); got != want {
			t.Fatalf("Match(%q, %q) = %v, path.Match says %v", pattern, name, got, want)
		}
	})
}

func ascii(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
