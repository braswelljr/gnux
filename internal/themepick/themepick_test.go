package themepick_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/braswelljr/gnux/internal/themepick"
)

var names = []string{"ayu", "default", "dracula", "nord", "solarized", "solarized-dark", "solarized-light"}

func newModel(start string) *themepick.Model {
	return themepick.New(names, start, "nord",
		func(n string) []string { return []string{"\x1b[31m" + n + " sample\x1b[0m", "second line"} },
		func(n string) string {
			if n == "solarized" {
				return "solarized-dark"
			}
			return n
		}, true)
}

func codes(keys []themepick.Key) []themepick.Code {
	var c []themepick.Code
	for _, k := range keys {
		c = append(c, k.Code)
	}
	return c
}

func TestParseKeys(t *testing.T) {
	tests := []struct {
		in   string
		want []themepick.Code
	}{
		{"\x1b[A\x1b[B\x1bOA", []themepick.Code{themepick.Up, themepick.Down, themepick.Up}},
		{"\x1b[5~\x1b[6~\x1b[H\x1b[F\x1b[1~\x1b[4~", []themepick.Code{themepick.PageUp, themepick.PageDown, themepick.Home, themepick.End, themepick.Home, themepick.End}},
		{"\x1b", []themepick.Code{themepick.Escape}},
		{"\r\x7f\x03\x15\t\x10\x0e", []themepick.Code{themepick.Enter, themepick.Backspace, themepick.Quit, themepick.ClearFilter, themepick.NextTab, themepick.Up, themepick.Down}},
		{"\x1b[Z\x1b[C\x1b[D", []themepick.Code{themepick.PrevTab, themepick.NextTab, themepick.PrevTab}},
		{"\x1b[1;5C", nil}, // Ctrl-Right: known shape, unused
		{"\x1b[", nil},     // cut short
		{"\x01\x02", nil},  // other control keys
		{"é", []themepick.Code{themepick.Char}},
	}
	for _, tt := range tests {
		if got := codes(themepick.ParseKeys([]byte(tt.in))); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: %v, want %v", tt.in, got, tt.want)
		}
	}
	if k := themepick.ParseKeys([]byte("é")); k[0].Rune != 'é' {
		t.Errorf("rune %q", k[0].Rune)
	}
}

func press(m *themepick.Model, s string) (done, chosen bool) {
	for _, k := range themepick.ParseKeys([]byte(s)) {
		if done, chosen = m.Handle(k, 3); done {
			return
		}
	}
	return
}

func TestMoveFilterAndChoose(t *testing.T) {
	m := newModel("dracula")
	if m.Selected() != "dracula" {
		t.Fatalf("starts on %q", m.Selected())
	}
	press(m, "\x1b[B")
	if m.Selected() != "nord" {
		t.Fatalf("down: %q", m.Selected())
	}
	press(m, "\x1b[A\x1b[A\x1b[A\x1b[A")
	if m.Selected() != "ayu" {
		t.Fatalf("up stops at the top: %q", m.Selected())
	}
	press(m, "\x1b[6~")
	if m.Selected() != "nord" {
		t.Fatalf("page down: %q", m.Selected())
	}
	press(m, "\x1b[F")
	if m.Selected() != "solarized-light" {
		t.Fatalf("end: %q", m.Selected())
	}
	// Typing filters; the cursor stays on its theme while it matches.
	press(m, "sol")
	if m.Selected() != "solarized-light" {
		t.Fatalf("filter kept the theme: %q", m.Selected())
	}
	press(m, "\x7f\x7f\x7fdr")
	if m.Selected() != "dracula" {
		t.Fatalf("filter dr: %q", m.Selected())
	}
	if done, chosen := press(m, "\r"); !done || !chosen || m.Selected() != "dracula" {
		t.Fatalf("enter: %v %v %q", done, chosen, m.Selected())
	}
}

func TestEscapeClearsTheFilterThenLeaves(t *testing.T) {
	m := newModel("ayu")
	press(m, "zzz")
	if m.Selected() != "" {
		t.Fatalf("no match: %q", m.Selected())
	}
	if done, _ := press(m, "\r"); done {
		t.Fatal("enter with nothing selected ended the picker")
	}
	if done, _ := press(m, "\x1b"); done {
		t.Fatal("the first Esc should clear the filter")
	}
	if done, chosen := press(m, "\x1b"); !done || chosen {
		t.Fatalf("second Esc: %v %v", done, chosen)
	}
	if done, chosen := press(newModel("ayu"), "\x03"); !done || chosen {
		t.Fatalf("Ctrl-C: %v %v", done, chosen)
	}
	m = newModel("ayu")
	press(m, "\u009b31m\u200e")
	if frame := m.Frame(100, 30); strings.Contains(frame, "\u009b") || strings.Contains(frame, "\u200e") {
		t.Fatal("a control character reached the frame")
	}
	m = newModel("ayu")
	press(m, strings.Repeat("a", 100))
	if done, _ := press(m, "\x15"); done || m.Selected() != "ayu" {
		t.Fatalf("Ctrl-U and a capped filter: %q", m.Selected())
	}
}

var escape = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestTabsGroupTheThemes(t *testing.T) {
	m := newModel("solarized")
	want := map[string][]string{
		"adaptive": {"default", "solarized"},
		"dark":     {"ayu", "dracula", "nord", "solarized-dark"},
		"light":    {"solarized-light"},
	}
	for _, tab := range []string{"adaptive", "dark", "light"} {
		press(m, "\t")
		var got []string
		press(m, "\x1b[H")
		for range 10 {
			if len(got) > 0 && got[len(got)-1] == m.Selected() {
				break
			}
			got = append(got, m.Selected())
			press(m, "\x1b[B")
		}
		if !reflect.DeepEqual(got, want[tab]) {
			t.Errorf("%s: %v, want %v", tab, got, want[tab])
		}
	}
	press(m, "\t") // back to All
	if press(m, "\x1b[Z"); !strings.Contains(escape.ReplaceAllString(m.Frame(100, 30), ""), "\x1b") && m.Selected() == "" {
		t.Fatal("shift-tab lost the selection")
	}
}

func TestFrameFitsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {60, 12}, {30, 8}, {5, 2}} {
		m := newModel("solarized")
		frame := m.Frame(size[0], size[1])
		if !strings.HasPrefix(frame, "\x1b[H") || !strings.HasSuffix(frame, "\x1b[J") {
			t.Fatalf("%v: frame is not drawn from the corner", size)
		}
		lines := strings.Split(escape.ReplaceAllString(frame, ""), "\r\n")
		if h := size[1]; len(lines) > h {
			t.Errorf("%v: %d lines", size, len(lines))
		}
		for _, l := range lines {
			if w := size[0]; utf8.RuneCountInString(l) > w {
				t.Errorf("%v: line wider than the terminal: %q", size, l)
			}
		}
		plain := escape.ReplaceAllString(frame, "")
		if size[0] >= 24 && size[1] >= 8 && !strings.Contains(plain, "solarized") {
			t.Errorf("%v: the cursor's theme is not shown", size)
		}
	}
	plain := escape.ReplaceAllString(newModel("solarized").Frame(100, 30), "")
	for _, want := range []string{"solarized → solarized-dark", "● nord", "solarized sample", "type to filter"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in\n%s", want, plain)
		}
	}
}
