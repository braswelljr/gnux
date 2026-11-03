package regex

import (
	"reflect"
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"
)

// backtracker compiles expr for the backtracking matcher even when it has no
// back reference, so that it can be held to Go's engine.
func backtracker(t testing.TB, expr string, longest bool) *Regexp {
	t.Helper()
	re, err := regexp.Compile(expr)
	if err != nil {
		t.Skip()
	}
	tree, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		t.Fatal(err)
	}
	prog, err := newProgram(tree)
	if err != nil {
		t.Fatalf("%q: %v", expr, err)
	}
	r := &Regexp{re: re, bt: prog}
	if longest {
		r.Longest()
	}
	return r
}

var diffPatterns = []string{
	`a`, `a*`, `a+b`, `(a|ab)(c|bcd)(d*)`, `(?s)x.*y`, `^abc$`, `(?m)^b$`, `\bfoo\b`, `\Bo`,
	`[a-c]+`, `[^a-c]+`, `(?i)hello`, `(a*)*`, `(a*)+b`, `(a|b)*c`, `a{2,3}`, `(ab){2}`,
	`x*`, `(?i)[k]`, `.`, `é+`, `a??b`, `a*?b`, `(a+?)(a*)`, `(foo|foobar)`, `()`, `\A\z`,
	`(?:a|b|)+`, `[[:alpha:]]+`, `(a)|(b)`, `((a)|b)+`,
}

var diffInputs = []string{
	"", "a", "aaa", "ab", "abcd", "abcbcd", "xay\nzy", "abc", "a\nb\nc", "foo foobar", "food",
	"aabbc", "aaaa", "HeLLo", "K", "x\xffy", "éé", "aab", "foobar", "ababab", "bab",
}

// TestMatchesGoRegexp holds the backtracking matcher to Go's engine on
// patterns both can run: the same matches, in both preference orders, and the
// same groups in Perl's order (POSIX leaves the groups of a longest match to
// the engine, so only its bounds are compared).
func TestMatchesGoRegexp(t *testing.T) {
	for _, expr := range diffPatterns {
		for _, longest := range []bool{false, true} {
			want := regexp.MustCompile(expr)
			if longest {
				want.Longest()
			}
			got := backtracker(t, expr, longest)
			for _, in := range diffInputs {
				b := []byte(in)
				if g, w := got.Match(b), want.Match(b); g != w {
					t.Errorf("%q longest=%v Match(%q) = %v, want %v", expr, longest, in, g, w)
				}
				gAll, wAll := got.FindAllIndex(b, -1), want.FindAllIndex(b, -1)
				if !reflect.DeepEqual(gAll, wAll) {
					t.Errorf("%q longest=%v FindAllIndex(%q) = %v, want %v", expr, longest, in, gAll, wAll)
				}
				if longest {
					continue
				}
				if g, w := got.FindSubmatchIndex(b), want.FindSubmatchIndex(b); !reflect.DeepEqual(g, w) {
					t.Errorf("%q FindSubmatchIndex(%q) = %v, want %v", expr, in, g, w)
				}
			}
		}
	}
}

func TestBackReferences(t *testing.T) {
	tests := []struct {
		expr, in string
		want     []int // FindSubmatchIndex under Longest; nil for no match
	}{
		{`(a)` + BackRef(1), "xaa", []int{1, 3, 1, 2}},
		{`(.)` + BackRef(1), "abccd", []int{2, 4, 2, 3}},
		{`(.+)` + BackRef(1), "abcabc!", []int{0, 6, 0, 3}},
		{`^(.*)` + BackRef(1) + `$`, "abab", []int{0, 4, 0, 2}},
		{`^(.*)` + BackRef(1) + `$`, "aba", nil},
		{`(a)(b)` + BackRef(2) + BackRef(1), "abba", []int{0, 4, 0, 1, 1, 2}},
		{`(a)` + BackRef(1) + BackRef(1), "aaa", []int{0, 3, 0, 1}},
		{`(?i)(a)` + BackRef(1), "aA", []int{0, 2, 0, 1}},
		{`(a)` + BackRef(1), "aA", nil},
		{`(?i)(é)` + BackRef(1), "éÉ", []int{0, 4, 0, 2}},
		{`(a|b)*` + BackRef(1), "abb", []int{0, 3, 1, 2}},
		{`(x)?y` + BackRef(1), "yx", nil},
		// The longest match wins, not the first found.
		{`(a|ab)(c|bcd)` + BackRef(2), "abcdbcd", []int{0, 7, 0, 1, 1, 4}},
	}
	for _, tt := range tests {
		re, err := Compile(tt.expr)
		if err != nil {
			t.Fatalf("Compile(%q): %v", tt.expr, err)
		}
		if !re.HasBackRefs() {
			t.Fatalf("%q: not compiled for back references", tt.expr)
		}
		re.Longest()
		if got := re.FindSubmatchIndex([]byte(tt.in)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q on %q = %v, want %v", tt.expr, tt.in, got, tt.want)
		}
		if got := re.Match([]byte(tt.in)); got != (tt.want != nil) {
			t.Errorf("%q Match(%q) = %v", tt.expr, tt.in, got)
		}
	}
}

func TestBackReferenceNumbering(t *testing.T) {
	// A named group a caller wraps the pattern in is reported but not
	// counted by a reference, and the reference groups are neither.
	re := MustCompile(`(?P<word>(a)(b)` + BackRef(2) + `)`)
	if n := re.NumSubexp(); n != 3 {
		t.Fatalf("NumSubexp = %d, want 3", n)
	}
	if got := re.FindSubmatchIndex([]byte("abb")); !reflect.DeepEqual(got, []int{0, 3, 0, 3, 0, 1, 1, 2}) {
		t.Fatalf("groups = %v", got)
	}
	if i := re.SubexpIndex("word"); i != 1 {
		t.Fatalf("SubexpIndex = %d", i)
	}

	for _, expr := range []string{`(a)` + BackRef(2), BackRef(1) + `(a)`, `a` + BackRef(1)} {
		if _, err := Compile(expr); err != ErrBackReference {
			t.Errorf("Compile(%q) = %v, want ErrBackReference", expr, err)
		}
	}
}

func TestFindAllWithBackReferences(t *testing.T) {
	re := MustCompile(`(.)` + BackRef(1))
	re.Longest()
	got := re.FindAllSubmatchIndex([]byte("aabxxyzz"), -1)
	want := [][]int{{0, 2, 0, 1}, {3, 5, 3, 4}, {6, 8, 6, 7}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FindAllSubmatchIndex = %v, want %v", got, want)
	}
	if got := re.FindAllIndex([]byte("aabxxyzz"), 2); len(got) != 2 {
		t.Fatalf("n=2 gave %v", got)
	}
}

func TestAbandonedSearchIsReported(t *testing.T) {
	re := MustCompile(`^((a*)*)*` + BackRef(1) + `b$`)
	re.Longest()
	if re.Match([]byte(strings.Repeat("a", 64))) {
		t.Fatal("matched")
	}
	if re.Err() != ErrTooComplex {
		t.Fatalf("Err = %v, want ErrTooComplex", re.Err())
	}
}

func TestPlainPatternsStayOnGo(t *testing.T) {
	re := MustCompile(`(a)(?P<gnux>b)\(`)
	if re.HasBackRefs() {
		t.Fatal("a pattern without a reference must use Go's engine")
	}
}

// FuzzMatchesGoRegexp checks that the backtracking matcher agrees with Go's
// engine on whatever pattern both accept.
func FuzzMatchesGoRegexp(f *testing.F) {
	for _, p := range diffPatterns {
		f.Add(p, "abcabc aab\nfoo")
	}
	f.Fuzz(func(t *testing.T, expr, in string) {
		if len(expr) > 40 || len(in) > 64 {
			return
		}
		want, err := regexp.Compile(expr)
		if err != nil || strings.Contains(expr, "(?P<") || strings.Contains(expr, "(?<") {
			return
		}
		got := backtracker(t, expr, false)
		// In Perl's order, a loop whose body can match the empty string
		// is resolved by how RE2 compiles it; POSIX tools never use that
		// order, so such patterns are held to the longest match only.
		if tree, _ := syntax.Parse(expr, syntax.Perl); !emptyLoop(tree) {
			loc := got.FindSubmatchIndex([]byte(in))
			if w := want.FindSubmatchIndex([]byte(in)); got.Err() == nil && !reflect.DeepEqual(loc, w) {
				t.Fatalf("%q on %q = %v, want %v", expr, in, loc, w)
			}
		}
		got.Longest()
		want.Longest()
		g, w := got.FindIndex([]byte(in)), want.FindIndex([]byte(in))
		if got.Err() == nil && !reflect.DeepEqual(g, w) {
			t.Fatalf("longest %q on %q = %v, want %v", expr, in, g, w)
		}
	})
}

// emptyLoop reports whether re repeats something that can match the empty
// string.
func emptyLoop(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpStar, syntax.OpPlus, syntax.OpRepeat, syntax.OpQuest:
		if nullable(re.Sub[0]) && re.Op != syntax.OpQuest {
			return true
		}
	}
	for _, s := range re.Sub {
		if emptyLoop(s) {
			return true
		}
	}
	return false
}

func nullable(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpStar, syntax.OpQuest, syntax.OpBeginLine, syntax.OpEndLine,
		syntax.OpBeginText, syntax.OpEndText, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	case syntax.OpRepeat:
		return re.Min == 0 || nullable(re.Sub[0])
	case syntax.OpCapture, syntax.OpPlus:
		return nullable(re.Sub[0])
	case syntax.OpConcat:
		for _, s := range re.Sub {
			if !nullable(s) {
				return false
			}
		}
		return true
	case syntax.OpAlternate:
		for _, s := range re.Sub {
			if nullable(s) {
				return true
			}
		}
	}
	return false
}
