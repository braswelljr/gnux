package visual_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/internal/visual"
)

func TestBox(t *testing.T) {
	var none *visual.Theme
	got := none.Box("Disk", []string{"one", "three"}, true)
	want := "╭─ Disk ─╮\n│ one    │\n│ three  │\n╰────────╯\n"
	if got != want {
		t.Errorf("unicode:\n%s\nwant:\n%s", got, want)
	}
	got = none.Box("", []string{"ab"}, false)
	if want := "+----+\n| ab |\n+----+\n"; got != want {
		t.Errorf("ascii:\n%s\nwant:\n%s", got, want)
	}
	// Color does not change the frame's shape.
	plain := visual.NewTheme().Box("T", []string{visual.NewTheme().Paint(visual.Good, "OK")}, true)
	for _, line := range strings.Split(strings.TrimSuffix(plain, "\n"), "\n") {
		if visual.Width(line) != 7 {
			t.Errorf("width of %q = %d", line, visual.Width(line))
		}
	}
}

func TestPreview(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	var b bytes.Buffer
	if err := visual.Preview(&b, env(nil), false, true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(visual.ThemeNames(), "\n") + "\n"; b.String() != got {
		t.Errorf("uncolored preview is not the names: %q", b.String())
	}

	b.Reset()
	if err := visual.Preview(&b, env(map[string]string{"GNUX_THEME": "github", "GNUX_BACKGROUND": "light"}), true, true); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"gnux theme list", "github-light", "light", "●"} {
		if !strings.Contains(out, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
}

func TestTable(t *testing.T) {
	var none *visual.Theme
	got := none.Render(visual.Table{
		Header: []string{"Name", "Size"},
		Rows:   [][]string{{"a", "1"}, {"bb", "200"}},
		Footer: [][]string{{"total", "201"}},
		Right:  []bool{false, true},
	}, true)
	want := "" +
		"╭───────┬──────╮\n" +
		"│ Name  │ Size │\n" +
		"├───────┼──────┤\n" +
		"│ a     │    1 │\n" +
		"│ bb    │  200 │\n" +
		"├───────┼──────┤\n" +
		"│ total │  201 │\n" +
		"╰───────┴──────╯\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	got = none.Render(visual.Table{Header: []string{"x"}, Rows: [][]string{{"1"}}}, false)
	if want := "+---+\n| x |\n+---+\n| 1 |\n+---+\n"; got != want {
		t.Errorf("ascii:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableFits(t *testing.T) {
	var none *visual.Theme
	tb := visual.Table{
		Header:   []string{"Size", "Mounted on"},
		Rows:     [][]string{{"1G", "/Library/Developer/CoreSimulator/Volumes/iOS"}},
		Right:    []bool{true, false},
		MaxWidth: 30,
	}
	for _, line := range strings.Split(strings.TrimSuffix(none.Render(tb, true), "\n"), "\n") {
		if visual.Width(line) != 30 {
			t.Errorf("width %d: %q", visual.Width(line), line)
		}
	}
	if got := none.Render(tb, true); !strings.Contains(got, "/Library/Developer…") {
		t.Errorf("not cut with an ellipsis:\n%s", got)
	}
	if got := visual.Truncate("\x1b[1mabcdef\x1b[0m", 4, false); got != "\x1b[1mabc~\x1b[0m" {
		t.Errorf("Truncate colored = %q", got)
	}
	if got := visual.Truncate("abc", 4, true); got != "abc" {
		t.Errorf("Truncate fitting = %q", got)
	}
}
