package shell_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/shell"
)

func TestPrompt(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "proj"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(home)
	atHome := func(sh *shell.Shell) {
		sh.HomeDir = func() (string, error) { return home, nil }
		sh.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	}

	// The directory, ~ for home, and a failed command's status; a line that
	// continues an unfinished command gets a prompt of its own.
	opts := shell.Options{Interactive: true, Pretty: true}
	_, _, errb := run(t, opts, "cd proj\nfail\nif true\nthen echo ok\nfi\n", atHome)
	sep := string(filepath.Separator)
	want := "gnux ~ > gnux ~" + sep + "proj > gnux ~" + sep + "proj [3] > " +
		"     > " + "     > " + "gnux ~" + sep + "proj > \n"
	if errb != want {
		t.Fatalf("prompts\n got %q\nwant %q", errb, want)
	}

	// Colored: the status in the failure color.
	opts.Color = true
	_, _, errb = run(t, opts, "fail\n", atHome)
	if !strings.Contains(errb, "\x1b[") || !strings.Contains(errb, "[3]") {
		t.Fatalf("colored prompt %q", errb)
	}

	// Without the view, sh's plain prompt.
	_, _, errb = run(t, shell.Options{Interactive: true}, "true\n", atHome)
	if errb != shell.Prompt+shell.Prompt+"\n" {
		t.Fatalf("plain prompt %q", errb)
	}
}

func TestHelpColored(t *testing.T) {
	opts := shell.Options{Pretty: true, Color: true}
	_, out, _ := run(t, opts, "help")
	if !strings.Contains(out, "\x1b[") || !strings.Contains(out, "a | b\x1b[0m                     pipe a's output into b") {
		t.Fatalf("help %q", out)
	}
	// Redirected, help is data.
	dir := t.TempDir()
	setDir := func(sh *shell.Shell) { sh.Set("D", dir) }
	run(t, opts, "help > $D/help.txt", setDir)
	b, err := os.ReadFile(filepath.Join(dir, "help.txt"))
	if err != nil || strings.Contains(string(b), "\x1b[") {
		t.Fatalf("redirected help colored: %v %q", err, b)
	}
}
