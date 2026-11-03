package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/internal/config"
)

func env(dir string) func(string) string {
	return func(k string) string {
		if k == "GNUX_CONFIG_DIR" {
			return dir
		}
		return ""
	}
}

func TestSaveLoadReset(t *testing.T) {
	getenv := env(filepath.Join(t.TempDir(), "gnux"))
	if name, _, err := config.Theme(getenv); name != "" || err != nil {
		t.Fatalf("nothing saved: %q %v", name, err)
	}
	path, err := config.SaveTheme(getenv, "Dracula")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || (info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/') {
		t.Fatalf("saved file: %v %v", info, err)
	}
	if name, got, err := config.Theme(getenv); name != "dracula" || got != path || err != nil {
		t.Fatalf("loaded %q %q %v", name, got, err)
	}
	if _, err := config.SaveTheme(getenv, "no-such-theme"); err == nil {
		t.Fatal("saved an unknown theme")
	}
	if _, removed, err := config.ResetTheme(getenv); !removed || err != nil {
		t.Fatalf("reset: %v %v", removed, err)
	}
	if _, removed, err := config.ResetTheme(getenv); removed || err != nil {
		t.Fatalf("second reset: %v %v", removed, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestRefusesWhatGnuxDidNotWrite(t *testing.T) {
	for name, content := range map[string]string{
		"unknown":  "not-a-theme\n",
		"escapes":  "\x1b[31m\n",
		"too long": strings.Repeat("a", 65),
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "theme"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, _, err := config.Theme(env(dir)); got != "" || err == nil {
			t.Errorf("%s: %q %v", name, got, err)
		}
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "theme"), 0o700); err != nil {
		t.Fatal(err)
	}
	if got, _, err := config.Theme(env(dir)); got != "" || err == nil {
		t.Errorf("a directory: %q %v", got, err)
	}
}

func TestNoDirectoryIsNothingSaved(t *testing.T) {
	if name, path, err := config.Theme(func(string) string { return "" }); name != "" || path != "" || err != nil {
		t.Fatalf("%q %q %v", name, path, err)
	}
	if _, err := config.SaveTheme(func(string) string { return "" }, "nord"); err == nil {
		t.Fatal("saved with nowhere to save")
	}
}
