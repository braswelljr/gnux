package ddx

import (
	"strings"
	"testing"
)

func TestCommandRuns(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		in     string
		out    string
		status int
	}{
		{"copies standard input", []string{"status=none"}, "hello\n", "hello\n", 0},
		{"block size and count", []string{"bs=2", "count=2", "status=none"}, "abcdef", "abcd", 0},
		{"skip", []string{"bs=1", "skip=3", "status=none"}, "abcdef", "def", 0},
		{"conversion", []string{"conv=ucase", "status=none"}, "MiXed\n", "MIXED\n", 0},
		{"an unknown operand is refused", []string{"bogus=1"}, "", "", 1},
		{"a bad number is refused", []string{"bs=x"}, "", "", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errText, status := execute(t, Spec(), tt.in, tt.args...)
			if out != tt.out || status != tt.status {
				t.Fatalf("dd %q: out=%q status=%d stderr=%q; want %q %d", tt.args, out, status, errText, tt.out, tt.status)
			}
			if status != 0 && !strings.HasPrefix(errText, "ddx: ") {
				t.Fatalf("dd %q: stderr=%q, want a ddx diagnostic", tt.args, errText)
			}
		})
	}

	t.Run("the transfer summary goes to standard error", func(t *testing.T) {
		out, errText, status := execute(t, Spec(), "abc", "bs=1")
		if out != "abc" || status != 0 || !strings.Contains(errText, "3+0 records in\n3+0 records out\n") {
			t.Fatalf("out=%q status=%d stderr=%q", out, status, errText)
		}
	})
}
