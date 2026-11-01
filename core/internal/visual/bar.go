package visual

import "strings"

// Bar levels: a bar filled to Warn or more is drawn in the warning color,
// and to Alert or more in the alert color, as a disk nearly full should
// stand out.
const (
	Warn  = 0.75
	Alert = 0.90
)

// Bar draws a horizontal meter width columns wide, filled to fraction
// (clamped to 0..1), in eighths of a column with the Unicode block
// characters or in whole columns of '#' and '.' when unicode is false. The
// filled part takes the theme's BarFull, BarWarn or BarAlert color by
// fraction and the rest BarEmpty; a nil theme draws it uncolored.
func (t *Theme) Bar(fraction float64, width int, unicode bool) string {
	if width <= 0 {
		return ""
	}
	fraction = min(max(fraction, 0), 1)
	key := BarFull
	switch {
	case fraction >= Alert:
		key = BarAlert
	case fraction >= Warn:
		key = BarWarn
	}

	var full, empty string
	if unicode {
		eighths := int(fraction*float64(width*8) + 0.5)
		whole, part := eighths/8, eighths%8
		full = strings.Repeat("█", whole)
		rest := width - whole
		if part > 0 {
			full += string([]rune("▏▎▍▌▋▊▉")[part-1])
			rest--
		}
		empty = strings.Repeat("░", rest)
	} else {
		whole := int(fraction*float64(width) + 0.5)
		full = strings.Repeat("#", whole)
		empty = strings.Repeat(".", width-whole)
	}
	return t.Paint(key, full) + t.Paint(BarEmpty, empty)
}
