package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRelease writes a binary for every platform into dir: as bare files, or
// inside archives named as the release names them.
func fakeRelease(t *testing.T, dir string, archives bool) {
	t.Helper()
	for _, p := range platforms {
		content := []byte("binary for " + p.goos + "/" + p.goarch)
		if !archives {
			name := "gnux-" + p.goos + "-" + p.goarch
			if p.goos == "windows" {
				name += ".exe"
			}
			if err := os.WriteFile(filepath.Join(dir, name), content, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		base := "gnux-v1.2.3-" + p.goos + "-" + p.goarch
		if p.goos == "windows" {
			f, err := os.Create(filepath.Join(dir, base+".zip"))
			if err != nil {
				t.Fatal(err)
			}
			z := zip.NewWriter(f)
			w, _ := z.Create(base + "/README.md")
			_, _ = w.Write([]byte("readme"))
			w, _ = z.Create(base + "/gnux.exe")
			_, _ = w.Write(content)
			_ = z.Close()
			_ = f.Close()
			continue
		}
		f, err := os.Create(filepath.Join(dir, base+".tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		for name, data := range map[string][]byte{base + "/LICENSE": []byte("mit"), base + "/gnux": content} {
			_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg})
			_, _ = tw.Write(data)
		}
		_ = tw.Close()
		_ = gz.Close()
		_ = f.Close()
	}
}

func readManifest(t *testing.T, path string) manifest {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestBuild(t *testing.T) {
	for _, archives := range []bool{false, true} {
		from, out := t.TempDir(), t.TempDir()
		fakeRelease(t, from, archives)
		if err := Build("1.2.3", from, out, []string{"rmx", "cpx"}); err != nil {
			t.Fatalf("archives=%v: %v", archives, err)
		}

		main := readManifest(t, filepath.Join(out, "gnux", "package.json"))
		if main.Name != "gnux" || main.Version != "1.2.3" || len(main.OptionalDependencies) != len(platforms) {
			t.Fatalf("main manifest = %+v", main)
		}
		for _, name := range []string{"gnux", "rmx", "cpx"} {
			if main.Bin[name] != "bin/"+name+".js" {
				t.Errorf("bin %s = %q", name, main.Bin[name])
			}
		}
		if _, gnuName := main.Bin["rm"]; gnuName {
			t.Error("a GNU name would hide the system's rm in package scripts")
		}
		launcher, _ := os.ReadFile(filepath.Join(out, "gnux", "bin", "rmx.js"))
		if !strings.HasPrefix(string(launcher), "#!/usr/bin/env node\n") || !strings.Contains(string(launcher), "main('rmx')") {
			t.Errorf("rmx launcher = %q", launcher)
		}
		index, _ := os.ReadFile(filepath.Join(out, "gnux", "index.js"))
		if strings.Contains(string(index), "PLATFORMS") || !strings.Contains(string(index), `"win32-arm64": "gnux-win32-arm64"`) {
			t.Errorf("index.js has no platform table:\n%s", index)
		}

		for _, p := range platforms {
			dir := filepath.Join(out, p.pkgName())
			m := readManifest(t, filepath.Join(dir, "package.json"))
			if m.Version != "1.2.3" || len(m.OS) != 1 || m.OS[0] != p.os || len(m.CPU) != 1 || m.CPU[0] != p.cpu || m.Bin != nil {
				t.Errorf("%s manifest = %+v", p.pkgName(), m)
			}
			if main.OptionalDependencies[p.pkgName()] != "1.2.3" {
				t.Errorf("gnux does not depend on %s", p.pkgName())
			}
			data, err := os.ReadFile(filepath.Join(dir, "bin", p.exe()))
			if err != nil || string(data) != "binary for "+p.goos+"/"+p.goarch {
				t.Errorf("%s binary = %q, %v", p.pkgName(), data, err)
			}
		}
	}
}

func TestBuildNeedsEveryPlatform(t *testing.T) {
	from := t.TempDir()
	fakeRelease(t, from, false)
	if err := os.Remove(filepath.Join(from, "gnux-linux-riscv64")); err != nil {
		t.Fatal(err)
	}
	err := Build("1.2.3", from, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "gnux-linux-riscv64") {
		t.Fatalf("err = %v, want the missing platform named", err)
	}
}
