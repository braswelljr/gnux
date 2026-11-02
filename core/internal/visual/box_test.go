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
	for _, want := range []string{"gnux themes", "github-light", "light", "●"} {
		if !strings.Contains(out, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
}
