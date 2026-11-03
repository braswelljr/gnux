package cmd

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/braswelljr/gnux/internal/tool"
)

func TestLogoOnTerminalOnly(t *testing.T) {
	var piped bytes.Buffer
	writeOverview(&piped, tool.Style{})
	if !strings.HasPrefix(piped.String(), "Usage: gnux ") || strings.Contains(piped.String(), "█") {
		t.Fatalf("help that is not on a terminal must start with the usage line and draw nothing:\n%.200s", piped.String())
	}

	var term bytes.Buffer
	writeOverview(&term, tool.Style{Terminal: true})
	out := term.String()
	if !strings.HasPrefix(out, logo[0]) || !strings.Contains(out, "\n\nUsage: gnux ") {
		t.Fatalf("on a terminal the logo comes first, then the usage line:\n%.400s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatal("a style without color must not write escape codes")
	}
	for _, want := range []string{"gnux ", "108 tools: gnux --list", "gnux help rmx"} {
		if want == "108 tools: gnux --list" {
			want = strings.TrimPrefix(want, "108")
		}
		if !strings.Contains(out, want) {
			t.Errorf("the logo's side text lacks %q", want)
		}
	}

	var colored bytes.Buffer
	writeOverview(&colored, tool.Style{Terminal: true, Color: true})
	if !strings.Contains(colored.String(), "\x1b[36m"+logo[0]+"\x1b[0m") {
		t.Error("with color the mark is painted cyan")
	}
}

func TestLogoShape(t *testing.T) {
	if len(logo) != 12 {
		t.Fatalf("the logo has %d lines, want 12 (24 pixels, two to a cell)", len(logo))
	}
	for i, line := range logo {
		if n := utf8.RuneCountInString(line); n > logoWidth {
			t.Errorf("line %d is %d cells wide, more than %d", i, n, logoWidth)
		}
		if strings.Trim(line, " ▀▄█") != "" {
			t.Errorf("line %d holds something other than half blocks: %q", i, line)
		}
	}
}
