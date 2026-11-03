package docs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestClearKeepsVersions checks that regenerating the docs keeps the docs of
// older versions (v0/, and v1.1/ as builds before 1.4.0 left them), which the
// website's build downloads and the generator cannot make again.
func TestClearKeepsVersions(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{"v0/index.mdx", "v12/tools/rmx.mdx", "v1.1/index.mdx", "tools/rmx.mdx", "index.mdx", "vendor/x.mdx", "v1.x/y.mdx"} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := clearKeepingVersions(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	want := []string{"v0", "v1.1", "v12"}
	if len(left) != len(want) {
		t.Fatalf("left %v, want %v", left, want)
	}
	for i := range want {
		if left[i] != want[i] {
			t.Fatalf("left %v, want %v", left, want)
		}
	}
}
