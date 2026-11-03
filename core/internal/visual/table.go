package visual

import (
	"strings"
	"unicode/utf8"
)

// Table is a table drawn as a terminal UI draws one: a rounded border, a
// rule under the header and between the columns, and the footer rows (a
// total) set apart by another rule. Cells may already be painted; widths
// are measured without the escapes (Width).
type Table struct {
	Header []string
	Rows   [][]string
	Footer [][]string
	// Right aligns a column's cells to the right, as numbers are.
	Right []bool
	// MaxWidth, when positive, is the terminal's width: a table wider than
	// it narrows its widest left-aligned column, cutting the cells short
	// with an ellipsis, so the frame never wraps.
	MaxWidth int
}

// Render draws the table: its border in the punctuation color, its header
// cells in the header color, with box-drawing characters or, when unicode
// is false, with +, - and |. Each line ends in a newline.
func (t *Theme) Render(tb Table, unicode bool) string {
	type edge struct{ left, mid, right string }
	h, v := "─", "│"
	top, rule, bottom := edge{"╭", "┬", "╮"}, edge{"├", "┼", "┤"}, edge{"╰", "┴", "╯"}
	if !unicode {
		h, v = "-", "|"
		top, rule, bottom = edge{"+", "+", "+"}, edge{"+", "+", "+"}, edge{"+", "+", "+"}
	}
	n := len(tb.Header)
	for _, r := range append(tb.Rows, tb.Footer...) {
		n = max(n, len(r))
	}
	widths := make([]int, n)
	measure := func(rows ...[]string) {
		for _, r := range rows {
			for i, c := range r {
				widths[i] = max(widths[i], Width(c))
			}
		}
	}
	measure(tb.Header)
	measure(tb.Rows...)
	measure(tb.Footer...)
	if over := 3*n + 1 - tb.MaxWidth; tb.MaxWidth > 0 {
		for _, w := range widths {
			over += w
		}
		if wide := widest(widths, tb.Right); over > 0 && wide >= 0 {
			widths[wide] = max(widths[wide]-over, min(widths[wide], minColumn))
		}
	}

	var b strings.Builder
	line := func(e edge) {
		parts := make([]string, n)
		for i, w := range widths {
			parts[i] = strings.Repeat(h, w+2)
		}
		b.WriteString(t.Paint(Punctuation, e.left+strings.Join(parts, e.mid)+e.right) + "\n")
	}
	row := func(cells []string, header bool) {
		b.WriteString(t.Paint(Punctuation, v))
		for i, w := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			cell = Truncate(cell, w, unicode)
			if header {
				cell = t.Paint(Header, cell)
			}
			pad := strings.Repeat(" ", w-Width(cell))
			if i < len(tb.Right) && tb.Right[i] {
				cell = pad + cell
			} else {
				cell += pad
			}
			b.WriteString(" " + cell + " " + t.Paint(Punctuation, v))
		}
		b.WriteString("\n")
	}
	line(top)
	if tb.Header != nil {
		row(tb.Header, true)
		line(rule)
	}
	for _, r := range tb.Rows {
		row(r, false)
	}
	if len(tb.Footer) > 0 {
		line(rule)
		for _, r := range tb.Footer {
			row(r, false)
		}
	}
	line(bottom)
	return b.String()
}

// minColumn is the narrowest a column is cut to when a table must fit.
const minColumn = 8

// widest is the index of the widest left-aligned column, -1 if none.
func widest(widths []int, right []bool) int {
	best := -1
	for i, w := range widths {
		if (i >= len(right) || !right[i]) && (best < 0 || w > widths[best]) {
			best = i
		}
	}
	return best
}

// Truncate cuts s to width columns, the last one an ellipsis ("~" when
// unicode is false), keeping its color escapes whole and ending a cut
// colored string with a reset. A string that fits is returned as it is.
func Truncate(s string, width int, unicode bool) string {
	if Width(s) <= width {
		return s
	}
	mark := "…"
	if !unicode {
		mark = "~"
	}
	var b strings.Builder
	n, colored := 0, false
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			b.WriteString(s[i:min(j+1, len(s))])
			colored = true
			i = j + 1
			continue
		}
		if n == width-1 {
			break
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+size])
		i += size
		n++
	}
	b.WriteString(mark)
	if colored {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}
