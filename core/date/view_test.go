package date_test

import (
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core/date"
)

func TestPanel(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	out, _, err := run(t, date.Options{Pretty: true, Date: "2026-10-15 12:00", HasDate: true}, "", "+%d %B")
	want := "+- 15 October -------------------------------------+\n" +
		"| Local     Thu Oct 15 12:00:00 UTC 2026           |\n" +
		"| UTC       Thu Oct 15 12:00:00 UTC 2026           |\n" +
		"| ISO 8601  2026-10-15T12:00:00+00:00              |\n" +
		"| Unix      1792065600                             |\n" +
		"| Day       Thursday, day 288 of the year, week 42 |\n" +
		"| Zone      UTC (UTC+00:00)                        |\n" +
		"| From now  in 3 days 2 hours                      |\n" +
		"+--------------------------------------------------+\n"
	if err != nil || out != want {
		t.Fatalf("got (%v)\n%s\nwant\n%s", err, out, want)
	}
	// Now has no "From now"; -f's lines are GNU's.
	if out, _, _ = run(t, date.Options{Pretty: true}, ""); strings.Contains(out, "From now") || !strings.Contains(out, "Mon Oct 12 09:05:03 UTC 2026") {
		t.Fatalf("now:\n%s", out)
	}
	if out, _, _ = run(t, date.Options{Pretty: true, File: "-"}, "2026-10-12\n", "+%F"); out != "2026-10-12\n" {
		t.Fatalf("-f: %q", out)
	}
}
