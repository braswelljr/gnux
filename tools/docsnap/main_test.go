package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	write("tools/rmx/force.mdx", "the force option\n")
	write("tools/truex/index.mdx", "true\n")
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
	if _, err := os.Stat(filepath.Join(dst, "tools", "truex.mdx")); err != nil {
		t.Errorf("a tool without option pages was not made a page of its own: %v", err)
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

func TestFlatten(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tools/truex/index.mdx", "true page")
	write("tools/truex/meta.json", `{"title":"truex","pages":[]}`)
	write("tools/echox/index.mdx", "echo page")
	write("tools/rmx/index.mdx", "rm page")
	write("tools/rmx/force.mdx", "force page")
	write("tools/rmx/meta.json", `{"title":"rmx","pages":["force"]}`)
	if err := Flatten(root); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{"tools/truex.mdx": "true page", "tools/echox.mdx": "echo page", "tools/rmx/index.mdx": "rm page"} {
		if got, err := os.ReadFile(filepath.Join(root, rel)); err != nil || string(got) != want {
			t.Errorf("%s = %q, %v", rel, got, err)
		}
	}
	for _, gone := range []string{"tools/truex", "tools/echox"} {
		if _, err := os.Stat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Errorf("%s still exists", gone)
		}
	}
	if err := Flatten(t.TempDir()); err != nil {
		t.Errorf("a tree without tools: %v", err)
	}
}

func TestPackIsReproducible(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "v1.2")
	for rel, content := range map[string]string{"meta.json": `{"title":"v1.2","root":true}`, "index.mdx": "home", "tools/rmx/index.mdx": "rm", "tools/rmx/force.mdx": "force"} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	a, b := filepath.Join(out, "a.tar.gz"), filepath.Join(out, "b.tar.gz")
	if err := Pack(dir, a); err != nil {
		t.Fatal(err)
	}
	if err := Pack(dir, b); err != nil {
		t.Fatal(err)
	}
	da, _ := os.ReadFile(a)
	db, _ := os.ReadFile(b)
	if len(da) == 0 || string(da) != string(db) {
		t.Fatal("packing the same pages twice gave different archives")
	}
	sum, err := os.ReadFile(a + ".sha256")
	if err != nil || !strings.HasSuffix(string(sum), "  a.tar.gz\n") || len(sum) != 64+2+len("a.tar.gz")+1 {
		t.Fatalf("a.tar.gz.sha256 = %q, %v", sum, err)
	}

	f, err := os.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, h.Name)
	}
	want := "v1.2/ v1.2/index.mdx v1.2/meta.json v1.2/tools/ v1.2/tools/rmx/ v1.2/tools/rmx/force.mdx v1.2/tools/rmx/index.mdx"
	if strings.Join(names, " ") != want {
		t.Fatalf("entries = %v\nwant      %s", names, want)
	}
}

func TestPrereleaseVersion(t *testing.T) {
	if v, err := prereleaseVersion("v1.3.0-rc.1"); err != nil || v != "v1.3" {
		t.Fatalf("v1.3.0-rc.1: %q %v", v, err)
	}
	if _, err := prereleaseVersion("v1.3.0"); err == nil {
		t.Fatal("a stable tag passed as a prerelease")
	}
}
