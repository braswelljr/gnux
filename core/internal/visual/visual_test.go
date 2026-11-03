package visual_test

import (
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/internal/visual"
)

func TestThemeOverlay(t *testing.T) {
	th := visual.NewTheme("ur=33:*.md=1;35:reset:da=0", "uw=4")
	if got := th.Code(visual.UserRead); got != "33" {
		t.Errorf("ur = %q, want 33", got)
	}
	if got := th.Code(visual.Date); got != "" {
		t.Errorf("da = %q, want off", got)
	}
	if got := th.Code(visual.UserWrite); got != "4" {
		t.Errorf("uw = %q, want 4 from the second spec", got)
	}
	if got := th.Paint(visual.UserRead, "r"); got != "\x1b[33mr\x1b[0m" {
		t.Errorf("Paint = %q", got)
	}
	var none *visual.Theme
	if got := none.Paint(visual.UserRead, "r"); got != "r" {
		t.Errorf("nil theme painted %q", got)
	}
}

func TestGradients(t *testing.T) {
	sizes := map[int64]string{
		0: visual.SizeByte, 1023: visual.SizeByte, 1024: visual.SizeKilo,
		5 << 20: visual.SizeMega, 3 << 30: visual.SizeGiga, 2 << 40: visual.SizeTera,
	}
	for n, want := range sizes {
		if got := visual.SizeKey(n); got != want {
			t.Errorf("SizeKey(%d) = %s, want %s", n, got, want)
		}
	}
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	ages := map[time.Duration]string{
		-time.Hour: visual.AgeHour, 10 * time.Minute: visual.AgeHour, 5 * time.Hour: visual.AgeDay,
		3 * 24 * time.Hour: visual.AgeWeek, 20 * 24 * time.Hour: visual.AgeMonth,
		100 * 24 * time.Hour: visual.AgeYear, 800 * 24 * time.Hour: visual.AgeOlder,
	}
	for age, want := range ages {
		if got := visual.AgeKey(now.Add(-age), now); got != want {
			t.Errorf("AgeKey(%v) = %s, want %s", age, got, want)
		}
	}
}

func TestBranches(t *testing.T) {
	b := visual.Unicode
	cases := []struct {
		last []bool
		want string
	}{
		{nil, ""},
		{[]bool{false}, "├── "},
		{[]bool{true}, "└── "},
		{[]bool{false, true}, "│   └── "},
		{[]bool{true, false}, "    ├── "},
	}
	for _, c := range cases {
		if got := b.Prefix(c.last); got != c.want {
			t.Errorf("Prefix(%v) = %q, want %q", c.last, got, c.want)
		}
	}
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if visual.BranchesFor(env(map[string]string{"LANG": "C"})) != visual.ASCII {
		t.Error("the C locale should draw in ASCII")
	}
	if visual.BranchesFor(env(map[string]string{"LANG": "C", "LC_ALL": "en_US.UTF-8"})) != visual.Unicode {
		t.Error("LC_ALL should win over LANG")
	}
	if visual.BranchesFor(env(nil)) != visual.Unicode {
		t.Error("no locale at all should draw in Unicode")
	}
}

func TestBar(t *testing.T) {
	cases := []struct {
		fraction float64
		width    int
		unicode  bool
		want     string
	}{
		{0.5, 4, false, "##.."},
		{0, 3, false, "..."},
		{2, 3, false, "###"},
		{0.5, 4, true, "██░░"},
		{0.5625, 4, true, "██▎░"},
		{1, 2, true, "██"},
	}
	for _, c := range cases {
		var none *visual.Theme
		if got := none.Bar(c.fraction, c.width, c.unicode); got != c.want {
			t.Errorf("Bar(%v, %d, %v) = %q, want %q", c.fraction, c.width, c.unicode, got, c.want)
		}
	}
	th := visual.NewTheme()
	if got := th.Bar(0.95, 2, false); !strings.HasPrefix(got, "\x1b[1;31m") {
		t.Errorf("a nearly full bar should be in the alert color: %q", got)
	}
}

func TestIcon(t *testing.T) {
	cases := []struct {
		name string
		kind visual.Kind
		want string
	}{
		{"main.go", visual.KindFile, ""},
		{"src/App.TSX", visual.KindFile, ""},
		{"Makefile", visual.KindFile, ""},
		{"run", visual.KindExec, ""},
		{"build.sh", visual.KindExec, ""},
		{".git", visual.KindDir, ""},
		{"src", visual.KindDir, ""},
		{"current", visual.KindLinkDir, ""},
		{"README", visual.KindFile, ""},
		{"unknown.zzz", visual.KindFile, ""},
	}
	for _, c := range cases {
		if got := visual.Icon(c.name, c.kind); got != c.want {
			t.Errorf("Icon(%q) = %U, want %U", c.name, []rune(got), []rune(c.want))
		}
	}
}

func TestLoadTheme(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

	th, err := visual.LoadTheme("", env(map[string]string{"GNUX_THEME": "mono:da=35"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := th.Code(visual.UserRead); got != "1" {
		t.Errorf("mono ur = %q, want 1", got)
	}
	if got := th.Code(visual.Date); got != "35" {
		t.Errorf("override da = %q, want 35", got)
	}
	if got := th.Code(visual.GroupRead); got != "" {
		t.Errorf("mono gr = %q, want off", got)
	}

	// The flag's name wins over the variable's; overrides still apply.
	th, _ = visual.LoadTheme("dracula", env(map[string]string{"GNUX_THEME": "nord:hd=1"}))
	if got := th.Code(visual.Header); got != "1" {
		t.Errorf("GNUX_THEME overrides should apply: hd = %q", got)
	}
	if got := th.Code(visual.TypeDir); !strings.Contains(got, "38;2;189;147;249") {
		t.Errorf("dracula di = %q", got)
	}

	th, _ = visual.LoadTheme("", env(map[string]string{"GNUX_THEME": "ur=36"}))
	if got := th.Code(visual.UserRead); got != "36" {
		t.Errorf("a bare override list: ur = %q", got)
	}

	if _, err := visual.LoadTheme("neon", env(nil)); err == nil {
		t.Error("an unknown theme should be reported")
	}
	for _, name := range visual.ThemeNames() {
		if _, err := visual.LoadTheme(name, env(nil)); err != nil {
			t.Errorf("built-in theme %s: %v", name, err)
		}
	}
}

func TestDark(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	tests := []struct {
		env  map[string]string
		want bool
	}{
		{map[string]string{"GNUX_BACKGROUND": "light", "COLORFGBG": "15;0"}, false},
		{map[string]string{"GNUX_BACKGROUND": "DARK"}, true},
		{map[string]string{"COLORFGBG": "0;15"}, false},
		{map[string]string{"COLORFGBG": "15;default;0"}, true},
		{map[string]string{"COLORFGBG": "0;7"}, false},
	}
	for _, tt := range tests {
		if got := visual.Dark(env(tt.env)); got != tt.want {
			t.Errorf("Dark(%v) = %v, want %v", tt.env, got, tt.want)
		}
	}
}

func TestFamilies(t *testing.T) {
	light := func(k string) string { return map[string]string{"GNUX_BACKGROUND": "light"}[k] }
	dark := func(k string) string { return map[string]string{"GNUX_BACKGROUND": "dark"}[k] }
	if got := visual.Resolve("GitHub", light); got != "github-light" {
		t.Errorf("github on light = %q", got)
	}
	if got := visual.Resolve("github", dark); got != "github-dark" {
		t.Errorf("github on dark = %q", got)
	}
	if got := visual.Resolve("github-light", dark); got != "github-light" {
		t.Errorf("a variant is kept: %q", got)
	}
	if got := visual.Resolve("dracula", light); got != "dracula" {
		t.Errorf("a theme with one variant is kept: %q", got)
	}

	// Every family name resolves to a theme, on either background, and the
	// two variants differ.
	names := visual.ThemeNames()
	for _, name := range names {
		d, l := visual.Resolve(name, dark), visual.Resolve(name, light)
		if d == l {
			continue
		}
		dt, err := visual.LoadTheme(d, dark)
		if err != nil {
			t.Errorf("%s: %v", d, err)
			continue
		}
		lt, err := visual.LoadTheme(l, light)
		if err != nil {
			t.Errorf("%s: %v", l, err)
			continue
		}
		if dt.Code(visual.TypeDir) == lt.Code(visual.TypeDir) && dt.Code(visual.Punctuation) == lt.Code(visual.Punctuation) {
			t.Errorf("%s: the dark and light variants look the same", name)
		}
	}
	if len(names) < 40 {
		t.Errorf("only %d themes", len(names))
	}
}
