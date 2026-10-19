package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCappedBufferStopsACopy guards the output limit against io.Copy, which
// prefers a ReadFrom to Write and so would read an endless stream to the end.
func TestCappedBufferStopsACopy(t *testing.T) {
	b := &cappedBuffer{limit: 10}
	endless := io.MultiReader(strings.NewReader(strings.Repeat("y\n", 1<<20)), strings.NewReader("tail"))
	if _, err := io.Copy(b, endless); err == nil {
		t.Fatal("the copy ran past the limit")
	}
	if b.String() != "y\ny\ny\ny\ny\n" || !b.cut {
		t.Fatalf("buffer = %q, cut = %v", b.String(), b.cut)
	}
}

func TestRunnerDirectory(t *testing.T) {
	root := t.TempDir()
	if real, err := filepath.EvalSymlinks(root); err == nil {
		root = real
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := &mcpRunner{roots: []string{root}}
	for in, want := range map[string]string{"": root, "sub": filepath.Join(root, "sub"), filepath.Join(root, "sub"): filepath.Join(root, "sub")} {
		if got, err := r.directory(in); err != nil || got != want {
			t.Errorf("directory(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"..", filepath.Join("sub", "..", ".."), filepath.Dir(root)} {
		if _, err := r.directory(in); err == nil {
			t.Errorf("directory(%q) was allowed", in)
		}
	}
}

func TestRunResultText(t *testing.T) {
	for _, tt := range []struct {
		res  runResult
		want string
	}{
		{runResult{Stdout: "a\n", Status: 0}, "exit status 0\n--- stdout ---\na\n"},
		{runResult{Stderr: "boom", Status: 1}, "exit status 1\n--- stderr ---\nboom\n"},
		{runResult{Status: -1, TimedOut: true}, "stopped after 2s, the time limit\n"},
		{runResult{Stdout: "yyy", Status: -1, Truncated: true}, "stopped when its output reached the limit\n--- stdout ---\nyyy\n(output cut off at the limit)\n"},
	} {
		if got := tt.res.text(2 * time.Second); got != tt.want {
			t.Errorf("%+v: got %q, want %q", tt.res, got, tt.want)
		}
	}
}
