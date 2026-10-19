package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVersion(t *testing.T) {
	for tag, want := range map[string]string{"v1.2.3": "v1.2", "v10.0.12": "v10.0"} {
		if got, err := Version(tag); err != nil || got != want {
			t.Errorf("Version(%q) = %q, %v; want %q", tag, got, err, want)
		}
	}
	for _, tag := range []string{"v1.2.3-rc.1", "1.2.3", "v1.2", "latest"} {
		if _, err := Version(tag); err == nil {
			t.Errorf("Version(%q) was accepted", tag)
		}
	}
}

func TestSnapshot(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("meta.json", `{"title":"gnux","pages":["index","---Tools---","...tools"]}`)
	write("index.mdx", "See [rmx](/docs/tools/rmx), [the docs](/docs), [a part](/docs#x) and [elsewhere](/install) or /docs/guide in prose.\n")
	write("tools/rmx/index.mdx", "[force](/docs/tools/rmx/force)\n")
	write("v1.0/index.mdx", "an older snapshot\n")

	dst := filepath.Join(out, "v1.2")
	if err := Snapshot(src, dst, "v1.2"); err != nil {
		t.Fatal(err)
	}
	index, _ := os.ReadFile(filepath.Join(dst, "index.mdx"))
	want := "See [rmx](/docs/v1.2/tools/rmx), [the docs](/docs/v1.2), [a part](/docs/v1.2#x) and [elsewhere](/install) or /docs/guide in prose.\n"
	if string(index) != want {
		t.Errorf("index.mdx = %q\nwant        %q", index, want)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "tools", "rmx", "index.mdx")); string(got) != "[force](/docs/v1.2/tools/rmx/force)\n" {
		t.Errorf("nested page = %q", got)
	}
	var meta map[string]any
	raw, _ := os.ReadFile(filepath.Join(dst, "meta.json"))
	if err := json.Unmarshal(raw, &meta); err != nil || meta["root"] != true || meta["title"] != "v1.2" {
		t.Errorf("meta.json = %s, %v", raw, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "v1.0")); err == nil {
		t.Error("a snapshot inside the source was copied into the new one")
	}
}
