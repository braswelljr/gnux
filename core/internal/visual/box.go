package visual

import (
	"strings"
	"unicode/utf8"
)

// Width is the number of columns s takes on a terminal, its color escapes
// left out. Every character is taken as one column wide.
func Width(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
				i++
			}
			i++
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		n++
	}
	return n
}

// Box frames lines in a rounded border (+, - and | in ASCII), with title
// set into the top edge, as a terminal UI draws a panel. The border is in
// the punctuation color and the title in the header color. Each line of
// the result ends in a newline.
func (t *Theme) Box(title string, lines []string, unicode bool) string {
	tl, tr, bl, br, h, v := "╭", "╮", "╰", "╯", "─", "│"
	if !unicode {
		tl, tr, bl, br, h, v = "+", "+", "+", "+", "-", "|"
	}
	width := Width(title) + 2
	for _, l := range lines {
		width = max(width, Width(l))
	}
	var b strings.Builder
	if title == "" {
		b.WriteString(t.Paint(Punctuation, tl+strings.Repeat(h, width+2)+tr) + "\n")
	} else {
		b.WriteString(t.Paint(Punctuation, tl+h) + " " + t.Paint(Header, title) + " " +
			t.Paint(Punctuation, strings.Repeat(h, width-1-Width(title))+tr) + "\n")
	}
	for _, l := range lines {
		b.WriteString(t.Paint(Punctuation, v) + " " + l + strings.Repeat(" ", width-Width(l)) + " " + t.Paint(Punctuation, v) + "\n")
	}
	b.WriteString(t.Paint(Punctuation, bl+strings.Repeat(h, width+2)+br) + "\n")
	return b.String()
}
