package stat_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/stat"
)

func TestPretty(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	f := fixture(t)
	out, _, err := run(stat.Options{Pretty: true}, f)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if !strings.HasPrefix(lines[0], "╭─ "+f+" ─") || !strings.HasPrefix(lines[len(lines)-1], "╰") {
		t.Fatalf("not a box titled with the name:\n%s", out)
	}
	for _, want := range []string{
		"│ Type      regular file ",
		"│ Size      5 bytes, ",
		"│ Modified  2023-11-14 22:13:20.500000000 +0000 ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	if runtime.GOOS != "windows" && !strings.Contains(out, "│ Mode      0644 -rw-r--r-- ") {
		t.Errorf("mode:\n%s", out)
	}
	width := len([]rune(lines[0]))
	for _, l := range lines {
		if len([]rune(l)) != width {
			t.Errorf("ragged line %q", l)
		}
	}

	// A format of the user's own, or -t, is printed as it is.
	if out, _, _ := run(stat.Options{Pretty: true, Format: "%s", FormatSet: true}, f); out != "5\n" {
		t.Errorf("-c: %q", out)
	}
	if out, _, _ := run(stat.Options{Pretty: true, Terse: true}, f); strings.Contains(out, "╭") {
		t.Errorf("-t: %q", out)
	}

	// The file system's panel carries a bar of the blocks in use.
	out, _, _ = run(stat.Options{Pretty: true, FileSystem: true}, f)
	if !strings.Contains(out, "│ Used ") || !strings.Contains(out, "%") {
		t.Errorf("-f:\n%s", out)
	}
}
