//go:build unix

package config_test

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/internal/config"
)

// A FIFO in the theme file's place must not block the tools that look.
func TestFIFOIsRefusedWithoutBlocking(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "theme"), 0o600); err != nil {
		t.Skip(err)
	}
	if got, _, err := config.Theme(env(dir)); got != "" || err == nil {
		t.Fatalf("%q %v", got, err)
	}
}

func TestXDGAndHome(t *testing.T) {
	getenv := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if d, _ := config.Dir(getenv(map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/h"})); d != "/x/gnux" {
		t.Errorf("XDG: %s", d)
	}
	if d, _ := config.Dir(getenv(map[string]string{"XDG_CONFIG_HOME": "rel", "HOME": "/h"})); d != "/h/.config/gnux" {
		t.Errorf("relative XDG is ignored: %s", d)
	}
}
