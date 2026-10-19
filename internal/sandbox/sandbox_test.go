package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/fsys"
)

// helperEnv names the directory a re-run of this test binary confines
// itself to: entering a sandbox cannot be undone, so it never happens in the
// test process itself.
const helperEnv = "GNUX_SANDBOX_TEST_DIR"

func TestMain(m *testing.M) {
	if dir := os.Getenv(helperEnv); dir != "" {
		os.Exit(helper(dir))
	}
	os.Exit(m.Run())
}

// helper enters a sandbox allowing dir, then reports what it can reach.
func helper(dir string) int {
	outside := os.Getenv(helperEnv + "_OUTSIDE")
	if err := Enter(Policy{Write: []string{dir}}); err != nil {
		if errors.Is(err, ErrUnsupported) {
			fmt.Println("unsupported:", err)
			return 0
		}
		fmt.Println("error:", err)
		return 1
	}
	report := func(what string, err error) {
		if err != nil {
			fmt.Println(what, "denied")
		} else {
			fmt.Println(what, "allowed")
		}
	}
	// Through core/fsys, as the engines go: under Capsicum that is the only
	// way to a file, and elsewhere it is the os package.
	report("write inside", fsys.WriteFile(filepath.Join(dir, "new"), []byte("x"), 0o644))
	_, err := fsys.ReadFile(filepath.Join(dir, "new"))
	report("read inside", err)
	_, err = fsys.ReadFile(filepath.Join(outside, "secret"))
	report("read outside", err)
	report("write outside", fsys.WriteFile(filepath.Join(outside, "new"), []byte("x"), 0o644))
	if _, err := fsys.Lstat(filepath.Join(dir, "escape")); err == nil {
		_, err = fsys.ReadFile(filepath.Join(dir, "escape"))
		report("read through a link out", err)
	}
	return 0
}

func TestConfinement(t *testing.T) {
	if fsys.Sandboxed() {
		// The gnux_rooted build roots the whole disk, and os.Root then opens
		// every directory above the test's, which a kernel sandbox refuses;
		// the sandbox itself roots only the directories it allows.
		t.Skip("the gnux_rooted build's file system is not one a sandbox uses")
	}
	root := t.TempDir()
	inside, outside := filepath.Join(root, "inside"), filepath.Join(root, "outside")
	for _, d := range []string{inside, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Windows may not let an ordinary user make a symbolic link; the rest of
	// the test stands without it.
	linked := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(inside, "escape")) == nil

	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), helperEnv+"="+inside, helperEnv+"_OUTSIDE="+outside)
	// Standard output only: a test binary built for coverage writes its data
	// as it exits, which the sandbox refuses, on standard error.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	got := string(out)
	if err != nil {
		t.Fatalf("helper: %v\n%s%s", err, got, stderr.String())
	}
	if strings.HasPrefix(got, "unsupported:") {
		// Linux may lack Landlock (or have cgo); macOS, Windows and FreeBSD
		// always have their sandbox.
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "freebsd" {
			t.Fatalf("%s must be supported: %s", runtime.GOOS, got)
		}
		t.Skip(strings.TrimSpace(got))
	}
	want := "write inside allowed\nread inside allowed\nread outside denied\nwrite outside denied\n"
	if linked {
		want += "read through a link out denied\n"
	}
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(outside, "new")); err == nil {
		t.Fatal("the sandboxed process wrote outside its directory")
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	if _, err := (Policy{Write: []string{filepath.Join(dir, "missing")}}).resolve(); err == nil {
		t.Fatal("a missing directory was accepted")
	}
	if _, err := (Policy{}).resolve(); err == nil {
		t.Fatal("an empty policy was accepted")
	}
	p, err := (Policy{Write: []string{dir, dir + string(filepath.Separator) + "."}}).resolve()
	if err != nil || len(p.Write) != 1 || !filepath.IsAbs(p.Write[0]) {
		t.Fatalf("resolve = %+v, %v", p, err)
	}
	if !p.reaches(filepath.Join(p.Write[0], "a", "b")) || p.reaches(filepath.Dir(p.Write[0])) {
		t.Fatal("reaches is wrong")
	}
}
