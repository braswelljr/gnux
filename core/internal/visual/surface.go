package visual

// surfaces are the background and foreground of each built-in theme with a
// fixed palette, after the editor theme it follows, in 24-bit hex. The
// tools never paint a background (a terminal keeps its own); a host that
// draws a whole terminal, such as the web playground, uses these to show a
// theme as its editor does.
var surfaces = map[string][2]string{
	"vscode-dark":         {"1e1e1e", "d4d4d4"},
	"vscode-light":        {"ffffff", "3b3b3b"},
	"github-dark":         {"0d1117", "e6edf3"},
	"github-light":        {"ffffff", "1f2328"},
	"one-dark":            {"282c34", "abb2bf"},
	"one-light":           {"fafafa", "383a42"},
	"solarized-dark":      {"002b36", "839496"},
	"solarized-light":     {"fdf6e3", "657b83"},
	"gruvbox-dark":        {"282828", "ebdbb2"},
	"gruvbox-light":       {"fbf1c7", "3c3836"},
	"catppuccin-dark":     {"1e1e2e", "cdd6f4"},
	"catppuccin-light":    {"eff1f5", "4c4f69"},
	"tokyo-night-dark":    {"1a1b26", "c0caf5"},
	"tokyo-night-light":   {"e1e2e7", "3760bf"},
	"ayu-dark":            {"0d1017", "bfbdb6"},
	"ayu-light":           {"fcfcfc", "5c6166"},
	"rose-pine-dark":      {"191724", "e0def4"},
	"rose-pine-light":     {"faf4ed", "575279"},
	"everforest-dark":     {"2d353b", "d3c6aa"},
	"everforest-light":    {"fdf6e3", "5c6a72"},
	"night-owl-dark":      {"011627", "d6deeb"},
	"night-owl-light":     {"fbfbfb", "403f53"},
	"high-contrast-dark":  {"000000", "ffffff"},
	"high-contrast-light": {"ffffff", "000000"},
	"dracula":             {"282a36", "f8f8f2"},
	"nord":                {"2e3440", "d8dee9"},
	"monokai":             {"272822", "f8f8f2"},
	"kanagawa":            {"1f1f28", "dcd7ba"},
	"palenight":           {"292d3e", "a6accd"},
	"ayu-mirage":          {"1f2430", "cccac2"},
}

// Surface returns the background and foreground, as #rrggbb, of the theme
// a built-in name stands for (a family takes its variant from getenv, as
// Resolve does). ok is false for default and mono, which follow the
// terminal's own colors, and for a name that is not built in.
func Surface(name string, getenv func(string) string) (background, foreground string, ok bool) {
	s, ok := surfaces[Resolve(name, getenv)]
	if !ok {
		return "", "", false
	}
	return "#" + s[0], "#" + s[1], true
}
