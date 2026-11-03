package who

import (
	"strings"
	"testing"
)

func TestTable(t *testing.T) {
	self := func(w *Who) {
		w.Self = func() string { return "bob" }
		w.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	}
	viewing = self
	defer func() { viewing = nil }()

	got, err := run(t, Options{Pretty: true})
	if err != nil || !strings.Contains(got, "| NAME  | LINE  | TIME         | COMMENT    |") ||
		!strings.Contains(got, "| alice | pts/0 | Oct 12 09:30 | (10.0.0.5) |") ||
		!strings.Contains(got, "| bob   | pts/1 | Oct 12 13:02 | (laptop:0) |") {
		t.Fatalf("got (%v)\n%s", err, got)
	}

	// -u adds the idle time and the PID, right-aligned; -H's heading is
	// the table's header, not a line of its own.
	got, _ = run(t, Options{Pretty: true, Users: true, Heading: true})
	if !strings.Contains(got, "| NAME  | LINE  | TIME         | IDLE  |  PID | COMMENT    |") ||
		!strings.Contains(got, "| alice | pts/0 | Oct 12 09:30 | 01:30 | 1234 | (10.0.0.5) |") {
		t.Fatalf("-u:\n%s", got)
	}

	// The caller's own logins are in the "yours" color.
	got, _ = run(t, Options{Pretty: true, Color: true})
	if !strings.Contains(got, "\x1b[1;33mbob\x1b[0m") || strings.Contains(got, "\x1b[1;33malice") {
		t.Fatalf("colors: %q", got)
	}

	// -q is GNU's.
	if got, _ = run(t, Options{Pretty: true, Count: true}); got != "alice bob\n# users=2\n" {
		t.Fatalf("-q: %q", got)
	}
}
