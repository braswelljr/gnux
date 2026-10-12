package bre

import "testing"

func TestCompile(t *testing.T) {
	tests := []struct {
		pattern, text string
		match         bool
	}{
		{"abc", "xxabcxx", true},
		{"^abc", "xxabc", false},
		{"^abc", "abcxx", true},
		{"abc$", "xxabc", true},
		{"a*b", "aaab", true},
		{"*a", "*a", true},
		{"^*a", "*a", true},
		{`a\{2\}`, "xaax", true},
		{`a\{2\}`, "xax", false},
		{`\(ab\)\+`, "abab", true},
		{`a\|b`, "b", true},
		{"a+", "a+", true},
		{"a+", "aa", false},
		{"[[:digit:]]", "x7", true},
		{"[]a]", "]", true},
		{"a.c", "a\nc", true},
	}
	for _, tt := range tests {
		re, err := Compile(tt.pattern)
		if err != nil {
			t.Fatalf("Compile(%q): %v", tt.pattern, err)
		}
		if got := re.MatchString(tt.text); got != tt.match {
			t.Errorf("%q on %q = %v, want %v", tt.pattern, tt.text, got, tt.match)
		}
	}
}

func TestAnchored(t *testing.T) {
	re, err := CompileAnchored("b")
	if err != nil || re.MatchString("ab") || !re.MatchString("ba") {
		t.Fatalf("CompileAnchored: %v", err)
	}
}

func TestErrors(t *testing.T) {
	for pattern, msg := range map[string]string{
		`a\`:      "Trailing backslash",
		`\(a`:     "Unmatched ( or \\(",
		`a\)`:     "Unmatched ) or \\)",
		`[a`:      "Unmatched [, [^, [:, [., or [=",
		`\(a\)\1`: "back references are not supported",
	} {
		_, err := Compile(pattern)
		e, ok := err.(*Error)
		if !ok || e.Msg != msg {
			t.Errorf("Compile(%q) = %v, want %q", pattern, err, msg)
		}
	}
}
