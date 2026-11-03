package stty

import (
	"fmt"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// displayPretty is gnux's terminal view (BR-13) of -a: one panel titled
// with the device, holding the speed and size, the control characters in
// a grid, and each group of flags on lines of its own, a set flag in the
// "good" color and an unset one (-flag) dimmed. The settings are GNU's.
func (s *Stty) displayPretty(m *termios) {
	getenv := s.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	t := visual.Use(s.Color, s.Theme, getenv, s.diag.Warnf)
	label := func(l string) string { return t.Paint(visual.Punctuation, fmt.Sprintf("%-12s", l)) }
	// The panel's frame takes four columns of the screen.
	width := max(s.maxCol-4, 40)

	var lines []string
	in, out := cfgetispeed(m), cfgetospeed(m)
	speed := fmt.Sprintf("%d baud", baudToValue(out))
	if in != 0 && in != out {
		speed = fmt.Sprintf("%d baud in, %d baud out", baudToValue(in), baudToValue(out))
	}
	lines = append(lines, label("Speed")+t.Paint(visual.Size, speed))
	if rows, cols, err := getWinsize(s.fd); err == nil {
		lines = append(lines, label("Size")+t.Paint(visual.Size, fmt.Sprintf("%d rows, %d columns", rows, cols)))
	}
	if hasLine {
		lines = append(lines, label("Line")+fmt.Sprint(m.line))
	}

	var chars []string
	for _, ci := range controls {
		if ci.name == "min" {
			break
		}
		if shownControl(ci) {
			chars = append(chars, fmt.Sprintf("%-8s", ci.name)+t.Paint(visual.Octal, fmt.Sprintf("%-7s", visible(m.cc[ci.index]))))
		}
	}
	chars = append(chars, fmt.Sprintf("%-8s", "min")+t.Paint(visual.Octal, fmt.Sprintf("%-7d", m.cc[c.VMIN])),
		fmt.Sprintf("%-8s", "time")+t.Paint(visual.Octal, fmt.Sprintf("%-7d", m.cc[c.VTIME])))
	lines = append(lines, "")
	lines = append(lines, grid(label("Characters"), chars, width)...)

	names := map[kind]string{control: "Control", input: "Input", output: "Output", local: "Local"}
	var flags []string
	prev := control
	flush := func(k kind) {
		if len(flags) > 0 {
			lines = append(lines, grid(label(names[k]), flags, width)...)
		}
		flags = nil
	}
	lines = append(lines, "")
	for i := range modes {
		info := &modes[i]
		if info.flags&omit != 0 {
			continue
		}
		if info.kind != prev {
			flush(prev)
			prev = info.kind
		}
		mask := info.mask
		if mask == 0 {
			mask = info.bits
		}
		switch {
		case *m.flag(info.kind)&mask == info.bits:
			flags = append(flags, t.Paint(visual.Good, info.name))
		case info.flags&rev != 0:
			flags = append(flags, t.Paint(visual.Punctuation, "-"+info.name))
		}
	}
	flush(prev)
	_, _ = fmt.Fprint(s.out, t.Box(s.device, lines, visual.UTF8(getenv)))
}

// grid lays items out after a label, as many to a line as fit in width,
// the lines after the first indented under it.
func grid(label string, items []string, width int) []string {
	indent := strings.Repeat(" ", visual.Width(label))
	var lines []string
	line, w := label, visual.Width(label)
	for _, it := range items {
		iw := visual.Width(it)
		if w > visual.Width(label) && w+1+iw > width {
			lines = append(lines, strings.TrimRight(line, " "))
			line, w = indent, len(indent)
		}
		if w > len(indent) {
			line += " "
			w++
		}
		line += it
		w += iw
	}
	return append(lines, strings.TrimRight(line, " "))
}
