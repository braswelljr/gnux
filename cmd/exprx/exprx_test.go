package exprx

import "testing"

func TestCommand(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		out    string
		status int
	}{
		{"arithmetic", []string{"2", "+", "3", "*", "4"}, "14\n", 0},
		{"a null result is status 1", []string{"3", "-", "3"}, "0\n", 1},
		{"match length", []string{"hello", ":", "hel"}, "3\n", 0},
		{"match group", []string{"v1.2.3", ":", `v\([0-9]*\)`}, "1\n", 0},
		{"back reference", []string{"abab", ":", `\(.*\)\1`}, "ab\n", 0},
		{"length is in bytes, as in the C locale", []string{"length", "héllo"}, "6\n", 0},
		{"options are operands", []string{"-1", "+", "1"}, "0\n", 1},
		{"syntax error is status 2", []string{"1", "+"}, "", 2},
		{"division by zero is status 2", []string{"1", "/", "0"}, "", 2},
		{"bad back reference is status 2", []string{"a", ":", `\(a\)\2`}, "", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errText, status := execute(t, Spec(), "", tt.args...)
			if out != tt.out || status != tt.status {
				t.Fatalf("expr %q: out=%q status=%d stderr=%q; want %q %d", tt.args, out, status, errText, tt.out, tt.status)
			}
			if (status == 2) != (errText != "") {
				t.Fatalf("expr %q: stderr=%q with status %d", tt.args, errText, status)
			}
		})
	}
}
