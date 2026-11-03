package uptime

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// loadBar is the width of a load average's bar.
const loadBar = 16

// panel is gnux's terminal view (BR-13): a box titled with how long the
// system has been up, holding the time, the users, each load average as a
// bar of the load against the CPUs (yellow from 75% of them, red from 90%,
// as a machine nearly full should stand out) and the CPU count.
func (u *Uptime) panel(out *bufio.Writer, now, boot time.Time, users int) {
	getenv := u.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	cpus := runtime.NumCPU()
	if u.CPUs != nil {
		cpus = u.CPUs()
	}
	cpus = max(cpus, 1)
	theme := visual.Use(u.Color, u.Theme, getenv, u.diag.Warnf)
	unicode := visual.UTF8(getenv)
	label := func(s string) string { return theme.Paint(visual.Punctuation, fmt.Sprintf("%-6s", s)) + "  " }

	rows := []string{
		label("Time") + now.Format("15:04:05"),
		label("Users") + strconv.Itoa(users),
	}
	if avg, err := u.loadAvg(); err == nil && len(avg) > 0 {
		for i, a := range avg {
			name := ""
			if i == 0 {
				name = "Load"
			}
			span := []string{"1 min", "5 min", "15 min"}[min(i, 2)]
			rows = append(rows, label(name)+theme.Paint(visual.Punctuation, fmt.Sprintf("%6s", span))+"  "+
				theme.Bar(a/float64(cpus), loadBar, unicode)+fmt.Sprintf("  %.2f", a))
		}
	}
	rows = append(rows, label("CPUs")+strconv.Itoa(cpus))
	_, _ = out.WriteString(theme.Box(upTitle(now, boot), rows, unicode))
}

// upTitle is how long the system has been up, as the panel's title.
func upTitle(now, boot time.Time) string {
	up := int64(now.Sub(boot) / time.Second)
	if boot.IsZero() || up < 0 {
		return "up ????"
	}
	days, hours, mins := up/86400, up%86400/3600, up%3600/60
	var b strings.Builder
	b.WriteString("up ")
	switch {
	case days == 1:
		b.WriteString("1 day, ")
	case days > 1:
		fmt.Fprintf(&b, "%d days, ", days)
	}
	fmt.Fprintf(&b, "%d:%02d", hours, mins)
	return b.String()
}
