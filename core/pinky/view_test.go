package pinky_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/pinky"
)

func TestTable(t *testing.T) {
	p, out, _ := newEngine(t, pinky.Options{Pretty: true}, "")
	p.Self = func() string { return "bob" }
	p.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	if err := p.Run(nil); err != nil {
		t.Fatal(err)
	}
	// The whole name, which GNU cuts to 19 columns; the TTY with GNU's
	// '*' for a terminal that refuses messages.
	for _, want := range []string{
		"| Login       | Name                                | TTY    | Idle  | When         | Where    |",
		"| alice       | Alice Alice Liddell                 | *pts/0 |",
		"| bob         | Robert Tables the Third of His Name | tty1   |       | Oct 20 11:30 |          |",
		"| carolineabc | ???                                 | pts/12 | 2d    | Oct 20 11:59 | host:0   |",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in\n%s", want, out.String())
		}
	}

	// The caller's login in the "yours" color, and an empty column left
	// out (no one here logged in from elsewhere).
	p, out, _ = newEngine(t, pinky.Options{Pretty: true, Color: true}, "")
	p.Self = func() string { return "bob" }
	p.Getenv = func(string) string { return "" }
	if err := p.Run([]string{"bob"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[1;33mbob\x1b[0m") || strings.Contains(out.String(), "Where") {
		t.Fatalf("colors and columns: %q", out.String())
	}
}

func TestLongPanel(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".plan"), []byte("Ship 2.0.\nThen rest.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, out, _ := newEngine(t, pinky.Options{Long: true, Pretty: true, NoProject: true}, home)
	p.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	if err := p.Run([]string{"alice", "nobody"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"+- alice ",
		"| In real life  Alice Alice Liddell ",
		"| Shell         /bin/bash ",
		"| Plan          Ship 2.0. ",
		"|               Then rest. ",
		"+- nobody ",
		"| In real life  ??? ",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in\n%s", want, out.String())
		}
	}
}
