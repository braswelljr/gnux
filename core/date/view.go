package date

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// print writes t in format, or with the view on, as a panel.
func (d *Date) print(out *bufio.Writer, format string, t, now time.Time) {
	if !d.Pretty {
		fmt.Fprintln(out, Format(format, t))
		return
	}
	getenv := d.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	th := visual.Use(d.Color, d.Theme, getenv, d.diag.Warnf)
	row := func(label, value string) string {
		return th.Paint(visual.Punctuation, fmt.Sprintf("%-10s", label)) + value
	}
	lines := []string{
		row("Local", th.Paint(visual.Date, Format(DefaultFormat, t))),
		row("UTC", Format(DefaultFormat, t.UTC())),
		row("ISO 8601", Format("%Y-%m-%dT%H:%M:%S%:z", t)),
		row("Unix", th.Paint(visual.Size, Format("%s", t))),
		row("Day", Format("%A, day %-j of the year, week %V", t)),
		row("Zone", Format("%Z (UTC%:z)", t)),
	}
	if !t.Equal(now) {
		lines = append(lines, row("From now", relative(t.Sub(now))))
	}
	fmt.Fprint(out, th.Box(Format(format, t), lines, visual.UTF8(getenv)))
}

// relative says how far a moment is from now in its two largest units:
// "in 3 days 4 hours", "2 minutes 5 seconds ago", "now".
func relative(d time.Duration) string {
	ago := d < 0
	if ago {
		d = -d
	}
	units := []struct {
		name string
		size time.Duration
	}{
		{"year", 365 * 24 * time.Hour},
		{"day", 24 * time.Hour},
		{"hour", time.Hour},
		{"minute", time.Minute},
		{"second", time.Second},
	}
	var parts []string
	for _, u := range units {
		if n := d / u.size; n > 0 && len(parts) < 2 {
			name := u.name
			if n != 1 {
				name += "s"
			}
			parts = append(parts, strconv.FormatInt(int64(n), 10)+" "+name)
			d -= n * u.size
		} else if len(parts) > 0 {
			break
		}
	}
	switch {
	case len(parts) == 0:
		return "now"
	case ago:
		return join(parts) + " ago"
	}
	return "in " + join(parts)
}

func join(parts []string) string {
	if len(parts) == 2 {
		return parts[0] + " " + parts[1]
	}
	return parts[0]
}
