package comm_test

import (
	"testing"

	"github.com/braswelljr/gnux/core/comm"
)

func TestView(t *testing.T) {
	t.Setenv("GNUX_THEME", "")
	t.Setenv("LC_ALL", "C.UTF-8")
	dir := t.TempDir()
	a := write(t, dir, "old", "apple\nbanana\n")
	b := write(t, dir, "new", "banana\ndate\n")

	// A header row names the files; the lines and their tabs are GNU's.
	out, _, err := run(comm.Options{Pretty: true, Suppress3: true}, "banana\n", a, "-")
	if want := "old\tstanda…\napple\n"; err != nil || out != want {
		t.Fatalf("got %q (%v), want %q", out, err, want)
	}

	// Each column in its color, the totals too.
	out, _, _ = run(comm.Options{Pretty: true, Color: true, Total: true}, "", a, b)
	want := "\x1b[4mold\x1b[0m\t\x1b[4mnew\x1b[0m\t\x1b[4mboth\x1b[0m\n" +
		"\x1b[36mapple\x1b[0m\n" +
		"\t\t\x1b[1mbanana\x1b[0m\n" +
		"\t\x1b[35mdate\x1b[0m\n" +
		"\x1b[36m1\x1b[0m\t\x1b[35m1\x1b[0m\t\x1b[1m1\x1b[0m\ttotal\n"
	if out != want {
		t.Fatalf("got  %q\nwant %q", out, want)
	}

	// With a delimiter of its own, the names are whole.
	out, _, _ = run(comm.Options{Pretty: true, OutputDelimiterSet: true, OutputDelimiter: ",", Suppress3: true}, "", a, b)
	if want := a + "," + b + "\napple\n,date\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}

	// -z is never viewed: the files hold one NUL-ended record each.
	za := write(t, dir, "za", "x\x00")
	zb := write(t, dir, "zb", "y\x00")
	if out, _, _ = run(comm.Options{Pretty: true, Color: true, ZeroTerminated: true}, "", za, zb); out != "x\x00\ty\x00" {
		t.Fatalf("-z: %q", out)
	}
}
