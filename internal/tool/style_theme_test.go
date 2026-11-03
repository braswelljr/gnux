package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpFollowsTheTheme(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{"GNUX_CONFIG_DIR": dir}
	getenv := func(k string) string { return env[k] }
	base := Style{Color: true}

	// No theme anywhere: help keeps its fixed colors.
	if st := base.WithTheme(getenv); st.Heading("H") != "\x1b[1;36mH\x1b[0m" || st.Command("c") != "\x1b[32mc\x1b[0m" {
		t.Fatalf("default: %q %q", st.Heading("H"), st.Command("c"))
	}
	// A saved theme colors help.
	if err := os.WriteFile(filepath.Join(dir, "theme"), []byte("dracula\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	st := base.WithTheme(getenv)
	if !strings.Contains(st.Heading("H"), "38;2;") || !strings.Contains(st.Command("c"), "38;2;") {
		t.Fatalf("saved theme: %q %q", st.Heading("H"), st.Command("c"))
	}
	// GNUX_THEME wins over it.
	env["GNUX_THEME"] = "mono"
	if st := base.WithTheme(getenv); strings.Contains(st.Heading("H"), "38;2;") {
		t.Fatalf("GNUX_THEME=mono: %q", st.Heading("H"))
	}
	// Without color, nothing is painted whatever the theme.
	if got := (Style{}).WithTheme(getenv).Heading("H"); got != "H" {
		t.Fatalf("no color: %q", got)
	}
}
