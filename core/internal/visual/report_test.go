package visual_test

import (
	"testing"

	"github.com/braswelljr/gnux/core/internal/visual"
)

func TestReport(t *testing.T) {
	th := visual.NewTheme("pa=1:xx=2:ok=32:er=31:hd=4")
	tests := []struct{ in, want string }{
		{"removed 'a'", "\x1b[31mremoved \x1b[0m\x1b[1m'a'\x1b[0m"},
		{"'a' -> 'b'", "\x1b[1m'a'\x1b[0m\x1b[2m -> \x1b[0m\x1b[1m'b'\x1b[0m"},
		{"mkdirx: created directory 'd'", "\x1b[2mmkdirx: \x1b[0m\x1b[32mcreated directory \x1b[0m\x1b[1m'd'\x1b[0m"},
		{"removed 'it's'", "\x1b[31mremoved \x1b[0m\x1b[1m'it's'\x1b[0m"},
		{"mode of 'a' changed from 0644 (rw-r--r--) to 0600 (rw-------)",
			"\x1b[4mmode of \x1b[0m\x1b[1m'a'\x1b[0m changed from 0644 (rw-r--r--) to 0600 (rw-------)"},
		{"no quotes", "no quotes"},
	}
	for _, tt := range tests {
		if got := th.Report(tt.in); got != tt.want {
			t.Errorf("Report(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	var none *visual.Theme
	if got := none.Report("removed 'a'"); got != "removed 'a'" {
		t.Errorf("nil theme: %q", got)
	}
}
