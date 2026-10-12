package quote

import "testing"

func TestShellEscape(t *testing.T) {
	for in, want := range map[string]string{
		"a.txt":     "a.txt",
		"dir/f-1_2": "dir/f-1_2",
		"":          "''",
		"a b":       "'a b'",
		"it's":      `'it'\''s'`,
		"a\nb":      `'a'$'\n''b'`,
		"$HOME":     "'$HOME'",
		"~x":        "'~x'",
		"café":      "café",
		"tab\there": `'tab'$'\t''here'`,
		"\x01":      `$'\001'`,
	} {
		if got := ShellEscape(in); got != want {
			t.Errorf("ShellEscape(%q) = %s, want %s", in, got, want)
		}
	}
}
