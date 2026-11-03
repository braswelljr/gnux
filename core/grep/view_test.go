package grep_test

import (
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/grep"
)

func TestPrettyHeadings(t *testing.T) {
	tree(t, map[string]string{"a.txt": "one\nfoo 1\nfoo 2\n", "b.txt": "foo 3\n", "c.txt": "none\n"})

	got := run(t, grep.Options{Pretty: true, LineNumber: true}, "", "foo", "a.txt", "b.txt", "c.txt")
	check(t, got, "a.txt\n2:foo 1\n3:foo 2\n\nb.txt\n1:foo 3\n", "", 0)

	// Context groups inside a file keep their separator; a new file does
	// not get one.
	tree(t, map[string]string{"a.txt": "foo\nx\nx\nx\nfoo\n", "b.txt": "foo\n"})
	got = run(t, grep.Options{Pretty: true, After: 1, Context: true}, "", "foo", "a.txt", "b.txt")
	check(t, got, "a.txt\nfoo\nx\n--\nfoo\n\nb.txt\nfoo\n", "", 0)

	// One file has no name to head with.
	got = run(t, grep.Options{Pretty: true}, "", "foo", "b.txt")
	check(t, got, "foo\n", "", 0)

	// The machine formats are GNU's.
	got = run(t, grep.Options{Pretty: true, Count: true}, "", "foo", "a.txt", "b.txt")
	check(t, got, "a.txt:2\nb.txt:1\n", "", 0)
	got = run(t, grep.Options{Pretty: true, Null: true}, "", "foo", "b.txt", "a.txt")
	check(t, got, "b.txt\x00foo\na.txt\x00foo\na.txt\x00foo\n", "", 0)
}

func TestPrettyHeadingColor(t *testing.T) {
	tree(t, map[string]string{"a.txt": "foo\n", "b.txt": "foo\n"})
	got := run(t, grep.Options{Pretty: true, Color: grep.ColorAlways}, "", "foo", "a.txt", "b.txt")
	if !strings.HasPrefix(got.out, "\x1b[1;35ma.txt\x1b[0m\n") {
		t.Errorf("heading = %q", got.out)
	}
}
