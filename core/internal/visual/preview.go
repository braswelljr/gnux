package visual

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// Preview writes the built-in themes as gnux themes shows them: with color,
// a panel saying which theme is in use and how the background was read,
// then each family with the variant it picks here and each theme, every
// one drawn on a sample row in its own colors. Without color it writes the
// names alone, one a line, for a script.
func Preview(w io.Writer, getenv func(string) string, color, unicode bool) error {
	names := ThemeNames()
	if !color {
		_, err := io.WriteString(w, strings.Join(names, "\n")+"\n")
		return err
	}
	inUse, _ := splitThemeSpec(getenv("GNUX_THEME"))
	if inUse == "" {
		inUse = "default"
	}
	frame := NewTheme()
	background := "light"
	if Dark(getenv) {
		background = "dark"
	}
	var b strings.Builder
	b.WriteString(frame.Box("gnux themes", []string{
		frame.Paint(Punctuation, "In use      ") + frame.Paint(TypeDir, inUse) + described(inUse, getenv),
		frame.Paint(Punctuation, "Background  ") + background + frame.Paint(Punctuation, "  (GNUX_BACKGROUND, else COLORFGBG, else the system's appearance)"),
		frame.Paint(Punctuation, "Try one     ") + "lsx -l --theme=NAME",
		frame.Paint(Punctuation, "Keep one    ") + "export GNUX_THEME=NAME",
	}, unicode))

	width := 0
	for _, n := range names {
		width = max(width, len(n))
	}
	mark := "●"
	if !unicode {
		mark = "*"
	}
	row := func(name string) {
		resolved := Resolve(name, getenv)
		t, _ := LoadTheme(resolved, func(string) string { return "" })
		lead := "  "
		if strings.EqualFold(name, inUse) {
			lead = frame.Paint(Good, mark) + " "
		}
		label := name + strings.Repeat(" ", width-len(name))
		via := strings.Repeat(" ", width+3)
		if resolved != strings.ToLower(name) {
			arrow := "→"
			if !unicode {
				arrow = ">"
			}
			via = frame.Paint(Punctuation, arrow+" ") + resolved + strings.Repeat(" ", width+1-len(resolved))
		}
		fmt.Fprintf(&b, "%s%s %s  %s\n", lead, label, via, sample(t, unicode))
	}

	fmt.Fprintf(&b, "\n%s %s\n", frame.Paint(Header, "Families"), frame.Paint(Punctuation, "(the variant follows the background)"))
	for _, n := range names {
		if slices.Contains(families, n) {
			row(n)
		}
	}
	fmt.Fprintf(&b, "\n%s\n", frame.Paint(Header, "Themes"))
	for _, n := range names {
		if !slices.Contains(families, n) {
			row(n)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// described says what a theme name in use stands for, when it is a family.
func described(name string, getenv func(string) string) string {
	if r := Resolve(name, getenv); r != strings.ToLower(name) {
		return " (" + r + ")"
	}
	return ""
}

// sample is a row in t's colors: a long listing's columns, a check's
// verdicts and a usage bar.
func sample(t *Theme, unicode bool) string {
	perm := ""
	for i, key := range []string{TypeDir, UserRead, UserWrite, UserExecDir, GroupRead, Punctuation, GroupExec, OtherRead, Punctuation, OtherExec} {
		perm += t.Paint(key, string("drwxr-xr-x"[i]))
	}
	return strings.Join([]string{
		perm,
		t.Paint(SizeMega, "4.2M"),
		t.Paint(AgeHour, "2h"),
		t.Paint(GitMod, "M") + t.Paint(GitNew, "N"),
		t.Paint(TypeDir, "src/"),
		t.Paint(TypeLink, "docs"),
		t.Paint(Good, "OK"),
		t.Paint(Bad, "FAILED"),
		t.Bar(0.8, 8, unicode),
	}, " ")
}
