package visual

import (
	"fmt"
	"slices"
	"strconv"
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
		Heading: bold(p.special), Good: bold(p.exec), Bad: bold(p.alert), Path: bold(p.link), Branch: p.dim, BarFull: p.exec, BarWarn: p.warn, BarAlert: bold(p.alert), BarEmpty: p.dim,
		GitNew: p.exec, GitMod: p.dir, GitDel: p.alert, GitRename: p.warn, GitType: p.special,
		GitIgnore: p.dim, GitClash: bold(p.alert), GitClean: p.dim,
	}
}

// scheme is a theme's colors as its editor theme names them, in 24-bit hex.
// Each built-in theme is one, mapped onto the keys the same way, so the
// themes differ only in their colors.
type scheme struct {
	yellow, red, green, magenta, comment, blue, cyan, purple, orange string
}

func (c scheme) codes() map[string]string {
	return palette{
		read: rgb(c.yellow), write: rgb(c.red), exec: rgb(c.green), special: rgb(c.magenta),
		dim: rgb(c.comment), dir: rgb(c.blue), link: rgb(c.cyan), size: rgb(c.green),
		date: rgb(c.purple), you: rgb(c.yellow), accent: rgb(c.cyan), warn: rgb(c.orange), alert: rgb(c.red),
	}.codes()
}

// schemes are the built-in themes with a fixed palette, after the editor
// themes of the same names. A name ending in -dark or -light is one half
// of a family (families).
var schemes = map[string]scheme{
	"vscode-dark":         {"dcdcaa", "f44747", "89d185", "c586c0", "808080", "569cd6", "4ec9b0", "9cdcfe", "ce9178"},
	"vscode-light":        {"795e26", "cd3131", "098658", "af00db", "6e7681", "0070c1", "267f99", "001080", "a31515"},
	"github-dark":         {"d29922", "ff7b72", "3fb950", "f778ba", "8b949e", "58a6ff", "39c5cf", "d2a8ff", "ffa657"},
	"github-light":        {"9a6700", "cf222e", "1a7f37", "bf3989", "6e7781", "0969da", "1b7c83", "8250df", "bc4c00"},
	"one-dark":            {"e5c07b", "e06c75", "98c379", "c678dd", "5c6370", "61afef", "56b6c2", "c678dd", "d19a66"},
	"one-light":           {"c18401", "e45649", "50a14f", "a626a4", "a0a1a7", "4078f2", "0184bc", "a626a4", "986801"},
	"solarized-dark":      {"b58900", "dc322f", "859900", "d33682", "586e75", "268bd2", "2aa198", "6c71c4", "cb4b16"},
	"solarized-light":     {"b58900", "dc322f", "859900", "d33682", "93a1a1", "268bd2", "2aa198", "6c71c4", "cb4b16"},
	"gruvbox-dark":        {"fabd2f", "fb4934", "b8bb26", "d3869b", "928374", "83a598", "8ec07c", "d3869b", "fe8019"},
	"gruvbox-light":       {"b57614", "9d0006", "79740e", "8f3f71", "928374", "076678", "427b58", "8f3f71", "af3a03"},
	"catppuccin-dark":     {"f9e2af", "f38ba8", "a6e3a1", "f5c2e7", "6c7086", "89b4fa", "94e2d5", "b4befe", "fab387"},
	"catppuccin-light":    {"df8e1d", "d20f39", "40a02b", "ea76cb", "9ca0b0", "1e66f5", "179299", "7287fd", "fe640b"},
	"tokyo-night-dark":    {"e0af68", "f7768e", "9ece6a", "bb9af7", "565f89", "7aa2f7", "7dcfff", "bb9af7", "ff9e64"},
	"tokyo-night-light":   {"8c6c3e", "f52a65", "587539", "9854f1", "848cb5", "2e7de9", "007197", "9854f1", "b15c00"},
	"ayu-dark":            {"e6b450", "f07178", "aad94c", "d2a6ff", "626a73", "59c2ff", "95e6cb", "d2a6ff", "ff8f40"},
	"ayu-light":           {"f2ae49", "f07171", "86b300", "a37acc", "787b80", "399ee6", "4cbf99", "a37acc", "fa8d3e"},
	"rose-pine-dark":      {"f6c177", "eb6f92", "9ccfd8", "c4a7e7", "6e6a86", "c4a7e7", "ebbcba", "c4a7e7", "ebbcba"},
	"rose-pine-light":     {"ea9d34", "b4637a", "56949f", "907aa9", "9893a5", "286983", "d7827e", "907aa9", "d7827e"},
	"everforest-dark":     {"dbbc7f", "e67e80", "a7c080", "d699b6", "859289", "7fbbb3", "83c092", "d699b6", "e69875"},
	"everforest-light":    {"dfa000", "f85552", "8da101", "df69ba", "939f91", "3a94c5", "35a77c", "df69ba", "f57d26"},
	"night-owl-dark":      {"ecc48d", "ef5350", "addb67", "c792ea", "637777", "82aaff", "7fdbca", "c792ea", "f78c6c"},
	"night-owl-light":     {"daaa01", "de3d3b", "08916a", "994cc3", "989fb1", "288ed7", "2aa298", "994cc3", "aa0982"},
	"high-contrast-dark":  {"ffd700", "ff6b6b", "00ff7f", "ff79ff", "c0c0c0", "6fc3ff", "00ffff", "d7a6ff", "ffa500"},
	"high-contrast-light": {"7a5c00", "b5200d", "116329", "9a1a9a", "3a3a3a", "0f4a85", "0b6168", "5c2a9e", "8f3d00"},
	// Themes that come in one, dark, variant.
	"dracula":    {"f1fa8c", "ff5555", "50fa7b", "ff79c6", "6272a4", "bd93f9", "8be9fd", "bd93f9", "ffb86c"},
	"nord":       {"ebcb8b", "bf616a", "a3be8c", "b48ead", "616e88", "81a1c1", "88c0d0", "5e81ac", "d08770"},
	"monokai":    {"e6db74", "f92672", "a6e22e", "ae81ff", "75715e", "66d9ef", "a1efe4", "ae81ff", "fd971f"},
	"kanagawa":   {"e6c384", "e46876", "98bb6c", "d27e99", "727169", "7e9cd8", "7fb4ca", "957fb8", "ffa066"},
	"palenight":  {"ffcb6b", "f07178", "c3e88d", "c792ea", "676e95", "82aaff", "89ddff", "c792ea", "f78c6c"},
	"ayu-mirage": {"ffcc66", "f28779", "d5ff80", "dfbfff", "6c7a8b", "73d0ff", "95e6cb", "dfbfff", "ffa659"},
}

// families are the themes with a dark and a light variant, named
// FAMILY-dark and FAMILY-light. The family's own name takes the variant
// that suits the background (Dark).
var families = []string{
	"catppuccin", "everforest", "github", "gruvbox", "high-contrast", "night-owl",
	"one", "rose-pine", "solarized", "tokyo-night", "vscode", "ayu",
}

// themes are the built-in themes, each an overlay on the default one. The
// default is in the 16 basic colors, which follow the terminal's own color
// scheme; the others are fixed 24-bit palettes.
var themes = func() map[string]map[string]string {
	m := map[string]map[string]string{
		"default": {},
		// mono keeps emphasis (bold, underline, dim) and drops every hue, for
		// colorblind readers and monochrome terminals.
		"mono": {
			UserRead: "1", UserWrite: "1", UserExec: "1;4", UserExecDir: "1",
			GroupRead: "", GroupWrite: "", GroupExec: "", OtherRead: "", OtherWrite: "", OtherExec: "",
			Special: "1", Punctuation: "2", TypeDir: "1", TypeLink: "4", TypePipe: "", TypeSocket: "", TypeDevice: "",
			Size: "1", SizeByte: "2", SizeKilo: "", SizeMega: "1", SizeGiga: "1;4", SizeTera: "1;4",
			Date: "", AgeHour: "1", AgeDay: "1", AgeWeek: "", AgeMonth: "", AgeYear: "2", AgeOlder: "2",
			You: "1", Links: "1", Inode: "2", Blocks: "2", Octal: "2", Heading: "1;4", Good: "1", Bad: "1;4", Path: "1", Branch: "2",
			BarFull: "", BarWarn: "1", BarAlert: "1;4", BarEmpty: "2",
			GitNew: "1", GitMod: "1", GitDel: "1;4", GitRename: "1", GitType: "1", GitIgnore: "2", GitClash: "1;4", GitClean: "2",
		},
	}
	for name, c := range schemes {
		m[name] = c.codes()
	}
	return m
}()

// ThemeNames lists the built-in themes, sorted: every theme and every
// family name.
func ThemeNames() []string {
	names := slices.Clone(families)
	for name := range themes {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Resolve gives the theme a built-in name stands for: a family's dark or
// light variant, as Dark says, and any other name as it is (lower-cased).
func Resolve(name string, getenv func(string) string) string {
	name = strings.ToLower(name)
	if slices.Contains(families, name) {
		if Dark(getenv) {
			return name + "-dark"
		}
		return name + "-light"
	}
	return name
}

// Dark reports whether the terminal's background is dark, which picks a
// family's variant: GNUX_BACKGROUND (dark or light) when set, else the
// background in COLORFGBG ("15;0"), which many terminals set, else the
// system's own appearance (macOS, Windows and GTK desktops have a dark
// mode), else dark, the common terminal default.
func Dark(getenv func(string) string) bool {
	switch strings.ToLower(getenv("GNUX_BACKGROUND")) {
	case "dark":
		return true
	case "light":
		return false
	}
	if v := getenv("COLORFGBG"); v != "" {
		bg := v[strings.LastIndexByte(v, ';')+1:]
		if n, err := strconv.Atoi(bg); err == nil {
			// 0-6 and 8 are the dark basic colors, 7 and 9-15 the light.
			return n < 7 || n == 8
		}
	}
	if dark, ok := systemDark(getenv); ok {
		return dark
	}
	return true
}

// ErrUnknownTheme is wrapped by LoadTheme's error for a name that is not a
// built-in theme.
var ErrUnknownTheme = fmt.Errorf("unknown theme")

// LoadTheme builds the theme a tool shows, in increasing precedence: the
// default, the built-in theme named by name (when not empty) or else by
// GNUX_THEME (a family taking its variant for the background, Resolve),
// then the KEY=CODE overrides in GNUX_THEME.
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
		overlay, ok := themes[Resolve(name, getenv)]
		if !ok {
			err = fmt.Errorf("%w %q (built-in themes: %s)", ErrUnknownTheme, name, strings.Join(ThemeNames(), ", "))
		}
		base = specOf(overlay)
	}
	return NewTheme(base, overrides), err
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

// Use returns the theme an engine paints with: nil when color is off, and
// otherwise LoadTheme's, a bad GNUX_THEME being reported through warn
// (an engine's Diag.Warnf) and the theme built without it.
func Use(color bool, name string, getenv func(string) string, warn func(format string, a ...any)) *Theme {
	if !color {
		return nil
	}
	theme, err := LoadTheme(name, getenv)
	if err != nil {
		warn("ignoring the theme: %v", err)
	}
	return theme
}
