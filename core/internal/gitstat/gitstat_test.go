package gitstat_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/internal/gitstat"
)

// repo builds a repository with real git, which these tests check this
// package against; without git they are skipped.
func repo(t *testing.T, objectFormat string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	args := []string{"init", "-q", "-b", "main"}
	if objectFormat != "" {
		args = append(args, "--object-format="+objectFormat)
	}
	run(t, dir, args...)
	run(t, dir, "config", "user.email", "t@example.com")
	run(t, dir, "config", "user.name", "t")
	run(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func noEnv(string) string { return "" }

func status(t *testing.T, r *gitstat.Repo, dir, name string, isDir bool) string {
	t.Helper()
	s, ok := r.Status(filepath.Join(dir, filepath.FromSlash(name)), isDir)
	if !ok {
		t.Fatalf("%s: outside the working tree", name)
	}
	return string([]byte{byte(s.Staged), byte(s.Unstaged)})
}

func scenario(t *testing.T, objectFormat string, pack bool) {
	dir := repo(t, objectFormat)
	write(t, dir, "clean.txt", "same\n")
	write(t, dir, "edit.txt", "before\n")
	write(t, dir, "gone.txt", "bye\n")
	write(t, dir, "src/a.go", "package a\n")
	write(t, dir, "src/deep/b.go", "package b\n")
	write(t, dir, ".gitignore", "*.log\nbuild/\n!keep.log\n/rooted.txt\n**/cache/**\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "one")
	if pack {
		// A second commit that edits a file gives the pack a delta.
		write(t, dir, "edit.txt", "before\nand a line more to make a delta\n")
		run(t, dir, "commit", "-q", "-am", "two")
		run(t, dir, "gc", "-q", "--aggressive")
	}

	// Everything after the commit happens a moment later, so that git's
	// "racily clean" check never decides the outcome.
	time.Sleep(10 * time.Millisecond)
	write(t, dir, "edit.txt", "after\n")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "staged.txt", "new\n")
	run(t, dir, "add", "staged.txt")
	write(t, dir, "src/deep/b.go", "package b // changed\n")
	run(t, dir, "add", "src/deep/b.go")
	write(t, dir, "untracked.txt", "?\n")
	write(t, dir, "debug.log", "x\n")
	write(t, dir, "keep.log", "x\n")
	write(t, dir, "build/out.bin", "x\n")
	write(t, dir, "rooted.txt", "x\n")
	write(t, dir, "src/rooted.txt", "x\n")
	write(t, dir, "src/cache/c.txt", "x\n")
	write(t, dir, "fresh/only.txt", "x\n")

	r, err := gitstat.Find(filepath.Join(dir, "src"), noEnv)
	if err != nil || r == nil {
		t.Fatalf("Find: %v, %v", r, err)
	}
	defer r.Close()

	files := map[string]string{
		"clean.txt": "--", "edit.txt": "-M", "staged.txt": "N-", "src/a.go": "--",
		"src/deep/b.go": "M-", "untracked.txt": "-N", "debug.log": "-I", "keep.log": "-N",
		"rooted.txt": "-I", "src/rooted.txt": "-N", "src/cache/c.txt": "-I",
	}
	for name, want := range files {
		if got := status(t, r, dir, name, false); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}
	dirs := map[string]string{
		"src": "MN", "src/deep": "M-", "build": "-I", "fresh": "-N", "": "MM", ".git": "--",
	}
	for name, want := range dirs {
		if got := status(t, r, dir, name, true); got != want {
			t.Errorf("%s/: %s, want %s", name, got, want)
		}
	}
	if !r.IsIgnored(filepath.Join(dir, "build"), true) || r.IsIgnored(filepath.Join(dir, "keep.log"), false) {
		t.Error("IsIgnored disagrees with the rules")
	}
	if _, ok := r.Status(t.TempDir(), true); ok {
		t.Error("a path outside the tree should have no status")
	}
}

func TestStatusLoose(t *testing.T)  { scenario(t, "", false) }
func TestStatusPacked(t *testing.T) { scenario(t, "", true) }
func TestStatusSHA256(t *testing.T) { scenario(t, "sha256", true) }

func TestDeletedDirectory(t *testing.T) {
	dir := repo(t, "")
	write(t, dir, "old/x.txt", "x\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "one")
	run(t, dir, "rm", "-q", "-r", "old")
	r, err := gitstat.Find(dir, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if got := status(t, r, dir, "", true); got != "D-" {
		t.Errorf("top: %s, want D-", got)
	}
}

func TestUnbornBranch(t *testing.T) {
	dir := repo(t, "")
	write(t, dir, "a.txt", "a\n")
	run(t, dir, "add", "a.txt")
	r, err := gitstat.Find(dir, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if got := status(t, r, dir, "a.txt", false); got != "N-" {
		t.Errorf("a.txt: %s, want N-", got)
	}
}

func TestNoRepository(t *testing.T) {
	r, err := gitstat.Find(t.TempDir(), noEnv)
	if err != nil {
		t.Fatal(err)
	}
	// A temporary directory may itself sit inside a repository (a CI
	// checkout); either way, Find must not fail.
	r.Close()
}
