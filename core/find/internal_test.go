package find

import (
	"testing"
	"time"
)

func TestFnmatch(t *testing.T) {
	tests := []struct {
		pattern, name string
		fold, want    bool
	}{
		{"", "", false, true},
		{"", "a", false, false},
		{"abc", "abc", false, true},
		{"abc", "abd", false, false},
		{"abc", "ABC", false, false},
		{"abc", "ABC", true, true},
		{"*", "", false, true},
		{"*", "anything/at/all", false, true},
		{"*", ".hidden", false, true},
		{"a*c", "a/b/c", false, true},
		{"a*c", "abcd", false, false},
		{"*.go", "x.go", false, true},
		{"*.go", "x.got", false, false},
		{"a**b", "ab", false, true},
		{"*a*b*c*", "xaxbxcx", false, true},
		{"*a*b*c*", "xcxbxax", false, false},
		{"?", "", false, false},
		{"?", "/", false, true},
		{"?", "é", false, true},
		{"??", "é", false, false},
		{"a?c", "abc", false, true},
		{"[abc]", "b", false, true},
		{"[abc]", "d", false, false},
		{"[!abc]", "d", false, true},
		{"[^abc]", "a", false, false},
		{"[a-c]x", "bx", false, true},
		{"[a-c]", "B", false, false},
		{"[a-c]", "B", true, true},
		{"[A-C]", "b", true, true},
		{"[]]", "]", false, true},
		{"[!]]", "]", false, false},
		{"[]-]", "-", false, true},
		{"[a-]", "-", false, true},
		{"[[:digit:]]", "7", false, true},
		{"[[:digit:]]", "x", false, false},
		{"[[:alpha:][:digit:]]", "x", false, true},
		{"[![:space:]]", " ", false, false},
		{"[[:upper:]]", "a", false, false},
		{"[[:upper:]]", "a", true, true},
		{"[[:nope:]]", ":", false, false},
		// An unclosed bracket is a literal.
		{"[abc", "[abc", false, true},
		{"a[", "a[", false, true},
		{"[", "x", false, false},
		{`\*`, "*", false, true},
		{`\*`, "a", false, false},
		{`a\?b`, "a?b", false, true},
		{`a\?b`, "axb", false, false},
		{`[\]]`, "]", false, true},
		{`\\`, `\`, false, true},
		{`a\`, `a\`, false, true},
		{"é*", "École", true, true},
		{"*/b/*", "a/b/c/d", false, true},
		{"x*", "", false, false},
	}
	for _, tt := range tests {
		if got := fnmatch(tt.pattern, tt.name, tt.fold); got != tt.want {
			t.Errorf("fnmatch(%q, %q, fold=%v) = %v, want %v", tt.pattern, tt.name, tt.fold, got, tt.want)
		}
	}
}

func TestCompileRegex(t *testing.T) {
	tests := []struct {
		kind, pattern, text string
		fold, want          bool
	}{
		{"emacs", `a+b?c*`, "aac", false, true},
		{"emacs", `a+`, "", false, false},
		{"emacs", `ab`, "xab", false, false},
		{"emacs", `ab`, "abx", false, false},
		{"emacs", `a{2}`, "a{2}", false, true},
		{"emacs", `(a|b)`, "(a|b)", false, true},
		{"emacs", `\(a\|b\)+`, "abba", false, true},
		{"emacs", `*a`, "*a", false, true},
		{"emacs", `\(*a\)`, "*a", false, true},
		{"emacs", `a^b`, "a^b", false, true},
		{"emacs", `a$b`, "a$b", false, true},
		{"emacs", `^ab$`, "ab", false, true},
		{"emacs", `\(^a\|b$\)`, "b", false, true},
		{"emacs", `a.b`, "a\nb", false, true},
		{"emacs", `[a\]+`, `a\a`, false, true},
		{"emacs", `[]a]+`, "a]a", false, true},
		{"emacs", `[^]a]`, "b", false, true},
		{"emacs", `[[:digit:]]+`, "123", false, true},
		{"emacs", `\w+\.txt`, "file.txt", false, true},
		{"emacs", `a\.b`, "axb", false, false},
		{"emacs", `ABC`, "abc", true, true},
		{"emacs", `a**`, "aaa", false, true},
		{"emacs", "\\`a\\'", "a", false, true},
		{"emacs", `\<a\>`, "a", false, true},
		{"findutils-default", `\(x\)+`, "xx", false, true},
		{"posix-extended", `(a|b)+c{2,3}`, "abcc", false, true},
		{"posix-extended", `(a|b)+c{2,3}`, "abc", false, false},
		{"posix-extended", `a{x`, "a{x", false, true},
		{"posix-extended", `a\(b\)`, "a(b)", false, true},
		{"posix-extended", `^a|b$`, "b", false, true},
		{"posix-extended", `a?`, "", false, true},
		{"posix-egrep", `x+`, "xxx", false, true},
		{"egrep", `x+`, "", false, false},
		{"awk", `[0-9]+`, "42", false, true},
		{"gnu-awk", `a|b`, "b", false, true},
		{"posix-awk", `a.`, "ab", false, true},
		{"posix-basic", `a\{2\}`, "aa", false, true},
		{"posix-basic", `a+`, "a+", false, true},
		{"posix-basic", `a\+`, "aaa", false, true},
		{"posix-basic", `\(a\|b\)*`, "abab", false, true},
		{"posix-minimal-basic", `a*`, "aaa", false, true},
		{"sed", `a.c`, "abc", false, true},
		{"grep", `a.c`, "ac", false, false},
		{"ed", `x*y`, "xxy", false, true},
	}
	for _, tt := range tests {
		re, err := compileRegex(tt.kind, tt.pattern, tt.fold)
		if err != nil {
			t.Errorf("compileRegex(%q, %q): %v", tt.kind, tt.pattern, err)
			continue
		}
		if got := re.MatchString(tt.text); got != tt.want {
			t.Errorf("%s %q on %q = %v, want %v (compiled to %s)", tt.kind, tt.pattern, tt.text, got, tt.want, re)
		}
	}

	for _, tt := range []struct{ kind, pattern, want string }{
		{"emacs", `a\{1`, ""},
		{"emacs", `[[.a.]]`, "the [. .] and [= =] bracket forms are not supported in a regular expression"},
		{"emacs", `[[=a=]]`, "the [. .] and [= =] bracket forms are not supported in a regular expression"},
		{"emacs", `[[:digit:`, "Unmatched [, [^, [:, [., or [="},
		{"emacs", `a*+`, "Invalid preceding regular expression"},
		{"posix-extended", `(a`, `Unmatched ( or \(`},
		{"posix-extended", `a)`, `Unmatched ) or \)`},
		{"posix-extended", `(a)\1`, "back references are not supported in a regular expression"},
		{"posix-extended", `[[:nope:]]`, "Invalid regular expression"},
	} {
		_, err := compileRegex(tt.kind, tt.pattern, false)
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("compileRegex(%q, %q): %v", tt.kind, tt.pattern, err)
		case tt.want != "" && (err == nil || err.Error() != tt.want):
			t.Errorf("compileRegex(%q, %q) error = %v, want %q", tt.kind, tt.pattern, err, tt.want)
		}
	}
}

func TestSymbolicMode(t *testing.T) {
	tests := []struct {
		spec string
		want uint32
		ok   bool
	}{
		{"u=rwx", 0o700, true},
		{"u=rwx,g=rx,o=", 0o750, true},
		{"a+r", 0o444, true},
		{"+x", 0o111, true},
		{"u+rw-w", 0o400, true},
		{"u=rw,g=u,o=g", 0o666, true},
		{"u+s,g+s,+t", 0o7000, true},
		{"o+t", 0o1000, true},
		{"u+t", 0, true},
		{"ug=rw,o=r", 0o664, true},
		{"u=x,a+X", 0o111, true},
		{"a+X", 0, true},
		{"u=rwx,u=", 0, true},
		{"", 0, false},
		{"u", 0, false},
		{"u=rwx,", 0, false},
		{"u=q", 0, false},
		{"x", 0, false},
		{"u=rwx;g=r", 0, false},
	}
	for _, tt := range tests {
		got, ok := symbolicMode(tt.spec)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("symbolicMode(%q) = %o, %v; want %o, %v", tt.spec, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	est := time.FixedZone("EST", -5*3600)
	tests := []struct {
		in   string
		want time.Time
	}{
		{"2024-06-14", time.Date(2024, 6, 14, 0, 0, 0, 0, est)},
		{"2024-06-14 10:30", time.Date(2024, 6, 14, 10, 30, 0, 0, est)},
		{"2024-06-14 10:30:15", time.Date(2024, 6, 14, 10, 30, 15, 0, est)},
		{"2024-06-14 10:30:15.25", time.Date(2024, 6, 14, 10, 30, 15, 250000000, est)},
		{"2024-06-14T10:30:15", time.Date(2024, 6, 14, 10, 30, 15, 0, est)},
		{"2024-06-14T10:30:15Z", time.Date(2024, 6, 14, 10, 30, 15, 0, time.UTC)},
		{"2024-06-14 10:30:15 +0200", time.Date(2024, 6, 14, 8, 30, 15, 0, time.UTC)},
		{" 20240614 ", time.Date(2024, 6, 14, 0, 0, 0, 0, est)},
		{"Jun 14 2024", time.Date(2024, 6, 14, 0, 0, 0, 0, est)},
		{"14 Jun 2024 09:00", time.Date(2024, 6, 14, 9, 0, 0, 0, est)},
		{"@0", time.Unix(0, 0)},
		{"@1718366400", time.Unix(1718366400, 0)},
		{"@-5", time.Unix(-5, 0)},
		{"@1.5", time.Unix(1, 500000000)},
		{"@1.0000000019", time.Unix(1, 1)},
	}
	for _, tt := range tests {
		got, ok := parseTimestamp(tt.in, est)
		if !ok || !got.Equal(tt.want) {
			t.Errorf("parseTimestamp(%q) = %v, %v; want %v", tt.in, got, ok, tt.want)
		}
	}
	for _, bad := range []string{"", "yesterday", "@", "@x", "@1.x", "2024-13-01", "10:30"} {
		if got, ok := parseTimestamp(bad, est); ok {
			t.Errorf("parseTimestamp(%q) = %v, want a failure", bad, got)
		}
	}
}

func TestPathParts(t *testing.T) {
	tests := []struct{ path, base, dir string }{
		{".", ".", "."},
		{"..", "..", "."},
		{"a", "a", "."},
		{"a/", "a", "."},
		{"a/b", "b", "a"},
		{"a/b/", "b", "a"},
		{"a//b", "b", "a"},
		{"/", "/", "/"},
		{"/a", "a", "/"},
		{"./a/b.txt", "b.txt", "./a"},
		{"a/.", ".", "a"},
	}
	for _, tt := range tests {
		if got := baseName(tt.path); got != tt.base {
			t.Errorf("baseName(%q) = %q, want %q", tt.path, got, tt.base)
		}
		if got := dirName(tt.path); got != tt.dir {
			t.Errorf("dirName(%q) = %q, want %q", tt.path, got, tt.dir)
		}
	}
	if got := joinPath("a/", "b"); got != "a/b" {
		t.Errorf("joinPath keeps one separator: %q", got)
	}
}

func TestAppendEscaped(t *testing.T) {
	if got, want := string(appendEscaped(nil, "a b\\c\"d\ne\tf\x01g")), `a\ b\\c\"d\ne\tf\001g`; got != want {
		t.Errorf("appendEscaped = %q, want %q", got, want)
	}
}
