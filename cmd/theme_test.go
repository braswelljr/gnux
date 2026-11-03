package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeSetShowReset(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gnux")
	t.Setenv("GNUX_CONFIG_DIR", dir)
	t.Setenv("GNUX_THEME", "")

	// Not a terminal: gnux theme alone shows the theme in use.
	if status, out, _ := run(t, "gnux", "theme"); status != 0 || out != "default\n" {
		t.Fatalf("show with nothing saved: %d %q", status, out)
	}
	status, out, errb := run(t, "gnux", "theme", "set", "Nord")
	if status != 0 || !strings.HasPrefix(out, "Theme set to nord (saved in ") || errb != "" {
		t.Fatalf("set: %d %q %q", status, out, errb)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "theme")); string(data) != "nord\n" {
		t.Fatalf("file: %q", data)
	}
	if _, out, _ := run(t, "gnux", "theme", "show"); out != "nord\n" {
		t.Fatalf("show: %q", out)
	}
	// GNUX_THEME wins over the saved theme, and saying so is a warning.
	t.Setenv("GNUX_THEME", "dracula:di=1")
	if _, out, _ := run(t, "gnux", "theme", "show"); out != "dracula\n" {
		t.Fatalf("GNUX_THEME wins: %q", out)
	}
	if status, _, errb := run(t, "gnux", "theme", "set", "github"); status != 0 || !strings.Contains(errb, "GNUX_THEME=dracula is set") {
		t.Fatalf("set under GNUX_THEME: %d %q", status, errb)
	}
	t.Setenv("GNUX_THEME", "")
	if status, out, _ := run(t, "gnux", "theme", "set", "default"); status != 0 || !strings.HasPrefix(out, "Theme reset") {
		t.Fatalf("set default resets: %d %q", status, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "theme")); !os.IsNotExist(err) {
		t.Fatalf("set default left the file: %v", err)
	}
	run(t, "gnux", "theme", "set", "nord")
	if status, out, _ := run(t, "gnux", "theme", "reset"); status != 0 || !strings.HasPrefix(out, "Theme reset") {
		t.Fatalf("reset: %d %q", status, out)
	}
	if status, out, _ := run(t, "gnux", "theme", "reset"); status != 0 || !strings.HasPrefix(out, "No theme was saved") {
		t.Fatalf("second reset: %d %q", status, out)
	}
}

func TestThemeMistakes(t *testing.T) {
	t.Setenv("GNUX_CONFIG_DIR", t.TempDir())
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"set"}, "gnux theme: missing theme name after 'set'\n"},
		{[]string{"set", "a", "b"}, "gnux theme: extra operand 'b'\n"},
		{[]string{"set", "nope"}, "gnux theme: invalid theme 'nope'\nValid themes are: "},
		{[]string{"show", "x"}, "gnux theme: extra operand 'x'\n"},
		{[]string{"bogus"}, "gnux theme: unknown command 'bogus' (list, show, set NAME, reset)\n"},
	} {
		status, out, errb := run(t, append([]string{"gnux", "theme"}, tt.args...)...)
		if status != 1 || out != "" || !strings.HasPrefix(errb, tt.want) || !strings.HasSuffix(errb, "Try 'gnux theme --help' for more information.\n") {
			t.Errorf("%v: %d %q %q", tt.args, status, out, errb)
		}
	}
}

func TestThemeListAndTheThemesAlias(t *testing.T) {
	t.Setenv("GNUX_CONFIG_DIR", t.TempDir())
	_, list, _ := run(t, "gnux", "theme", "list")
	_, alias, _ := run(t, "gnux", "themes")
	if list == "" || list != alias {
		t.Fatalf("list %q, themes %q", list, alias)
	}
	for _, name := range []string{"ios-light", "xcode-dark", "colorful", "monochrome", "synthwave", "flexoki"} {
		if !strings.Contains(list, "\n"+name+"\n") {
			t.Errorf("%s is not listed", name)
		}
	}
}

// A saved theme the tools would not accept is reported once per run and
// the default colors are used; piped output never reads it.
func TestBadSavedThemeIsReported(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GNUX_CONFIG_DIR", dir)
	t.Setenv("GNUX_THEME", "")
	t.Setenv("NO_COLOR", "")
	if err := os.WriteFile(filepath.Join(dir, "theme"), []byte("\x1b[31m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, errb := run(t, "wcx", "--pretty=always", "main.go"); !strings.Contains(errb, "ignoring the saved theme") {
		t.Fatalf("not reported: %q", errb)
	}
	if _, _, errb := run(t, "wcx", "main.go"); errb != "" {
		t.Fatalf("a pipe read the saved theme: %q", errb)
	}
}
