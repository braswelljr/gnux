//go:build unix

package mkdir_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core/mkdir"
)

// setUmask changes the process umask for the duration of the test. The mask
// is process-wide, so tests using it must not run in parallel.
func setUmask(t *testing.T, mask int) {
	t.Helper()
	old := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(old) })
}

func TestModeIgnoresUmask(t *testing.T) {
	dir := t.TempDir()
	setUmask(t, 0o077)

	tests := []struct {
		mode string
		want os.FileMode
	}{
		{"777", 0o777},
		{"750", 0o750},
		{"u=rwx,go=rx", 0o755},
		{"a=rwx", 0o777},
		{"go-w", 0o755},
		// No user class, so the umask still applies to what is granted.
		{"=rwx", 0o700},
		{"", 0o700},
	}
	for _, tt := range tests {
		t.Run("mode "+tt.mode, func(t *testing.T) {
			path := filepath.Join(dir, "d"+tt.mode)
			m, _, errb := newEngine(mkdir.Options{Mode: tt.mode})
			if err := m.Run([]string{path}); err != nil {
				t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
			}
			if got := mustDir(t, path).Mode().Perm(); got != tt.want {
				t.Fatalf("mode = %#o, want %#o", got, tt.want)
			}
		})
	}
}

func TestStickyMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sticky")
	m, _, errb := newEngine(mkdir.Options{Mode: "1777"})
	if err := m.Run([]string{path}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	info := mustDir(t, path)
	if info.Mode()&os.ModeSticky == 0 || info.Mode().Perm() != 0o777 {
		t.Fatalf("mode = %v, want sticky 0777", info.Mode())
	}
}

func TestParentsStayTraversable(t *testing.T) {
	// This umask would leave intermediate directories without owner write or
	// search permission, making the next component impossible to create.
	dir := t.TempDir()
	setUmask(t, 0o377)
	leaf := filepath.Join(dir, "a", "b", "c")

	m, _, errb := newEngine(mkdir.Options{Parents: true, Mode: "700"})
	if err := m.Run([]string{leaf}); err != nil {
		t.Fatalf("Run: %v (stderr: %s)", err, errb.String())
	}
	for _, p := range []string{filepath.Join(dir, "a"), filepath.Join(dir, "a", "b")} {
		if got := mustDir(t, p).Mode().Perm(); got != 0o700 {
			t.Fatalf("intermediate %q mode = %#o, want 0700 (0400 from the umask plus u+wx)", p, got)
		}
	}
	// -m applies to the operand only, never to the parents made for it.
	if got := mustDir(t, leaf).Mode().Perm(); got != 0o700 {
		t.Fatalf("leaf mode = %#o, want 0700", got)
	}
}
