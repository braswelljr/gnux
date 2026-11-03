package testx

import "testing"

func TestCommand(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		status int
	}{
		{"no expression is false", nil, 1},
		{"a non-empty string", []string{"x"}, 0},
		{"the empty string", []string{""}, 1},
		{"-n", []string{"-n", "x"}, 0},
		{"-z", []string{"-z", "x"}, 1},
		{"string equality", []string{"a", "=", "a"}, 0},
		{"integer comparison", []string{"2", "-lt", "10"}, 0},
		{"negation", []string{"!", "2", "-lt", "10"}, 1},
		{"-a and -o", []string{"x", "-a", "", "-o", "y"}, 0},
		{"parentheses", []string{"(", "1", "-eq", "1", ")"}, 0},
		{"a missing file", []string{"-e", "/no/such/file"}, 1},
		{"a bad integer is status 2", []string{"a", "-lt", "1"}, 2},
		{"a missing operand is status 2", []string{"1", "-eq"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errText, status := execute(t, Spec(), "", tt.args...)
			if out != "" || status != tt.status || (status == 2) != (errText != "") {
				t.Fatalf("test %q: out=%q status=%d stderr=%q; want status %d", tt.args, out, status, errText, tt.status)
			}
		})
	}
}

func TestBracket(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		status int
	}{
		{"true", []string{"a", "=", "a", "]"}, 0},
		{"false", []string{"a", "=", "b", "]"}, 1},
		{"empty", []string{"]"}, 1},
		{"missing ]", []string{"a", "=", "a"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errText, status := execute(t, BracketSpec(), "", tt.args...)
			if status != tt.status || (status == 2) != (errText != "") {
				t.Fatalf("[ %q: status=%d stderr=%q; want %d", tt.args, status, errText, tt.status)
			}
		})
	}
}
