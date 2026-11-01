package core

import "github.com/braswelljr/gnux/core/internal/visual"

// ThemeNames lists the built-in themes of gnux's terminal view, which a
// tool's --theme option and the GNUX_THEME variable take.
func ThemeNames() []string {
	return visual.ThemeNames()
}
