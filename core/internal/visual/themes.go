package visual

import (
	"fmt"
	"slices"
	"strings"
)

// rgb is the SGR code for a 24-bit foreground color, which every current
// terminal (Windows Terminal included) shows.
func rgb(hex string, attrs ...string) string {
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return strings.Join(append(attrs, fmt.Sprintf("38;2;%d;%d;%d", r, g, b)), ";")
}

// palette fills the theme keys from a handful of named colors, so each
// built-in theme is described by its colors and not by forty codes.
type palette struct {
	read, write, exec, special, dim, dir, link, size, date, you, accent, warn, alert string
}

func (p palette) codes() map[string]string {
	bold := func(c string) string { return "1;" + c }
	return map[string]string{
		UserRead: bold(p.read), UserWrite: bold(p.write), UserExec: "1;4;" + p.exec, UserExecDir: bold(p.exec),
		GroupRead: p.read, GroupWrite: p.write, GroupExec: p.exec,
		OtherRead: p.read, OtherWrite: p.write, OtherExec: p.exec,
		Special: p.special, Punctuation: p.dim,
		TypeDir: bold(p.dir), TypeLink: p.link, TypePipe: p.warn, TypeSocket: p.special, TypeDevice: p.warn,
		Size: bold(p.size), SizeByte: p.size, SizeKilo: bold(p.size), SizeMega: bold(p.warn),
		SizeGiga: bold(p.alert), SizeTera: bold(p.special),
		Date: p.date, AgeHour: bold(p.accent), AgeDay: p.accent, AgeWeek: bold(p.date), AgeMonth: p.date,
		AgeYear: "2;" + p.date, AgeOlder: p.dim,
		You: bold(p.you), Links: bold(p.alert), Inode: p.special, Blocks: p.link, Octal: p.special,
		Branch: p.dim, BarFull: p.exec, BarWarn: p.warn, BarAlert: bold(p.alert), BarEmpty: p.dim,
		GitNew: p.exec, GitMod: p.dir, GitDel: p.alert, GitRename: p.warn, GitType: p.special,
		GitIgnore: p.dim, GitClash: bold(p.alert), GitClean: p.dim,
	}
}

// themes are the built-in themes, each an overlay on the default one. The
// default is eza's look in the 16 basic colors, which follow the terminal's
// own color scheme; the others are fixed 24-bit palettes.
var themes = map[string]map[string]string{
	"default": {},
	// mono keeps emphasis (bold, underline, dim) and drops every hue, for
	// colorblind readers and monochrome terminals.
	"mono": {
		UserRead: "1", UserWrite: "1", UserExec: "1;4", UserExecDir: "1",
		GroupRead: "", GroupWrite: "", GroupExec: "", OtherRead: "", OtherWrite: "", OtherExec: "",
		Special: "1", Punctuation: "2", TypeDir: "1", TypeLink: "4", TypePipe: "", TypeSocket: "", TypeDevice: "",
		Size: "1", SizeByte: "2", SizeKilo: "", SizeMega: "1", SizeGiga: "1;4", SizeTera: "1;4",
		Date: "", AgeHour: "1", AgeDay: "1", AgeWeek: "", AgeMonth: "", AgeYear: "2", AgeOlder: "2",
		You: "1", Links: "1", Inode: "2", Blocks: "2", Octal: "2", Branch: "2",
		BarFull: "", BarWarn: "1", BarAlert: "1;4", BarEmpty: "2",
		GitNew: "1", GitMod: "1", GitDel: "1;4", GitRename: "1", GitType: "1", GitIgnore: "2", GitClash: "1;4", GitClean: "2",
	},
	"dracula": palette{
		read: rgb("f1fa8c"), write: rgb("ff5555"), exec: rgb("50fa7b"), special: rgb("ff79c6"),
		dim: rgb("6272a4"), dir: rgb("bd93f9"), link: rgb("8be9fd"), size: rgb("50fa7b"),
		date: rgb("bd93f9"), you: rgb("f1fa8c"), accent: rgb("8be9fd"), warn: rgb("ffb86c"), alert: rgb("ff5555"),
	}.codes(),
	"nord": palette{
		read: rgb("ebcb8b"), write: rgb("bf616a"), exec: rgb("a3be8c"), special: rgb("b48ead"),
		dim: rgb("4c566a"), dir: rgb("81a1c1"), link: rgb("88c0d0"), size: rgb("a3be8c"),
		date: rgb("5e81ac"), you: rgb("ebcb8b"), accent: rgb("8fbcbb"), warn: rgb("d08770"), alert: rgb("bf616a"),
	}.codes(),
	"gruvbox": palette{
		read: rgb("fabd2f"), write: rgb("fb4934"), exec: rgb("b8bb26"), special: rgb("d3869b"),
		dim: rgb("928374"), dir: rgb("83a598"), link: rgb("8ec07c"), size: rgb("b8bb26"),
		date: rgb("83a598"), you: rgb("fabd2f"), accent: rgb("8ec07c"), warn: rgb("fe8019"), alert: rgb("fb4934"),
	}.codes(),
	"solarized": palette{
		read: rgb("b58900"), write: rgb("dc322f"), exec: rgb("859900"), special: rgb("d33682"),
		dim: rgb("586e75"), dir: rgb("268bd2"), link: rgb("2aa198"), size: rgb("859900"),
		date: rgb("6c71c4"), you: rgb("b58900"), accent: rgb("2aa198"), warn: rgb("cb4b16"), alert: rgb("dc322f"),
	}.codes(),
	"catppuccin": palette{
		read: rgb("f9e2af"), write: rgb("f38ba8"), exec: rgb("a6e3a1"), special: rgb("f5c2e7"),
		dim: rgb("6c7086"), dir: rgb("89b4fa"), link: rgb("94e2d5"), size: rgb("a6e3a1"),
		date: rgb("b4befe"), you: rgb("f9e2af"), accent: rgb("89dceb"), warn: rgb("fab387"), alert: rgb("f38ba8"),
	}.codes(),
}

// ThemeNames lists the built-in themes, sorted.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ErrUnknownTheme is wrapped by LoadTheme's error for a name that is not a
// built-in theme.
var ErrUnknownTheme = fmt.Errorf("unknown theme")

// LoadTheme builds the theme a tool shows, in increasing precedence: the
// default, the built-in theme named by name (when not empty) or else by
// GNUX_THEME, then EZA_COLORS, then the KEY=CODE overrides in GNUX_THEME.
// GNUX_THEME is written "NAME", "NAME:KEY=CODE:..." or just "KEY=CODE:...".
// An unknown theme name is reported, with the theme built without it, so a
// tool can warn and carry on.
func LoadTheme(name string, getenv func(string) string) (*Theme, error) {
	envName, overrides := splitThemeSpec(getenv("GNUX_THEME"))
	if name == "" {
		name = envName
	}
	var err error
	base := ""
	if name != "" {
		overlay, ok := themes[strings.ToLower(name)]
		if !ok {
			err = fmt.Errorf("%w %q (built-in themes: %s)", ErrUnknownTheme, name, strings.Join(ThemeNames(), ", "))
		}
		base = specOf(overlay)
	}
	return NewTheme(base, getenv("EZA_COLORS"), overrides), err
}

// splitThemeSpec separates a GNUX_THEME value into its theme name and its
// overrides.
func splitThemeSpec(spec string) (name, overrides string) {
	first, rest, _ := strings.Cut(spec, ":")
	if strings.Contains(first, "=") {
		return "", spec
	}
	return strings.TrimSpace(first), rest
}

// specOf writes an overlay as a KEY=CODE spec, with an empty code written
// as "0" so that NewTheme switches the key off.
func specOf(overlay map[string]string) string {
	items := make([]string, 0, len(overlay))
	for k, v := range overlay {
		if v == "" {
			v = "0"
		}
		items = append(items, k+"="+v)
	}
	return strings.Join(items, ":")
}
