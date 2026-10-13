//go:build unix

package grep_test

import (
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core/grep"
)

// A FIFO nobody writes to would block the search forever, so these cases
// only pass if the FIFO is never opened.
func TestDevicesSkipped(t *testing.T) {
	tree(t, map[string]string{"d/file.txt": "match\n"})
	if err := syscall.Mkfifo("d/pipe", 0o644); err != nil {
		t.Skipf("cannot make a FIFO: %v", err)
	}

	t.Run("met while descending", func(t *testing.T) {
		got := run(t, grep.Options{Directories: grep.DirectoriesRecurse}, "", "match", "d")
		check(t, got, p("d/file.txt")+":match\n", "", 0)
	})
	t.Run("named, with -D skip", func(t *testing.T) {
		got := run(t, grep.Options{Devices: grep.DevicesSkip}, "", "match", "d/pipe", "d/file.txt")
		check(t, got, "d/file.txt:match\n", "", 0)
	})
	t.Run("-D skip while descending", func(t *testing.T) {
		got := run(t, grep.Options{Directories: grep.DirectoriesRecurse, Devices: grep.DevicesSkip}, "", "match", "d")
		check(t, got, p("d/file.txt")+":match\n", "", 0)
	})
}

// Outside Windows a terminal is only known to take colors when TERM says so.
func TestColorAutoNeedsTerm(t *testing.T) {
	got := runWith(t, grep.Options{Color: grep.ColorAuto}, "abc\n", func(g *grep.Grep) {
		g.Getenv = func(string) string { return "" }
		g.IsTerminal = true
	}, "b")
	check(t, got, "abc\n", "", 0)
}
