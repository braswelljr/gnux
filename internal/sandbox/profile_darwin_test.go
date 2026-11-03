//go:build darwin

package sandbox

import (
	"strings"
	"testing"
)

// TestProfileQuotesPaths checks that a directory name cannot add rules to the
// Seatbelt profile: quotes and backslashes in it stay inside the string.
func TestProfileQuotesPaths(t *testing.T) {
	hostile := `/tmp/x") (allow default) (allow file-write* (subpath "/`
	profile := profileFor(Policy{Write: []string{hostile}, Read: []string{`/tmp/back\slash`}}, "/usr/bin/true")
	if strings.Contains(profile, "\n(allow default") {
		t.Fatalf("the path escaped its string:\n%s", profile)
	}
	if !strings.Contains(profile, `(literal "/tmp/x\") (allow default) (allow file-write* (subpath \"/")`) {
		t.Fatalf("the hostile path is not one escaped literal:\n%s", profile)
	}
	if !strings.Contains(profile, `(literal "/tmp/back\\slash")`) {
		t.Fatalf("a backslash is not escaped:\n%s", profile)
	}
	if !strings.HasPrefix(profile, "(version 1)\n(deny default)\n") {
		t.Fatalf("the profile does not start by denying everything:\n%s", profile)
	}
	// Every parenthesis outside a string balances: nothing was cut short.
	depth, inString := 0, false
	for i := 0; i < len(profile); i++ {
		switch c := profile[i]; {
		case inString && c == '\\':
			i++
		case c == '"':
			inString = !inString
		case !inString && c == '(':
			depth++
		case !inString && c == ')':
			depth--
			if depth < 0 {
				t.Fatalf("unbalanced profile at byte %d:\n%s", i, profile)
			}
		}
	}
	if depth != 0 || inString {
		t.Fatalf("unbalanced profile (depth %d, in string %v):\n%s", depth, inString, profile)
	}
}
