//go:build !darwin && !windows

package visual

import "strings"

// systemDark reads a desktop's dark mode where it is in the environment: a
// GTK theme named ...-dark or NAME:dark. Elsewhere it does not know.
func systemDark(getenv func(string) string) (dark, ok bool) {
	gtk := strings.ToLower(getenv("GTK_THEME"))
	if gtk == "" {
		return false, false
	}
	return strings.Contains(gtk, "dark"), true
}
