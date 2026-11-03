// Package themepick is the theme picker of gnux theme: the built-in themes in
// a list, grouped by tabs and filtered as you type, beside a preview of the
// one under the cursor, in a frame drawn to fit the terminal. Model holds the
// state and draws a frame; Run puts it on a terminal.
package themepick

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The tabs group the themes: every one, the ones that follow the terminal
// (a family's own name takes the variant for the background; default and
// mono use the terminal's colors), and the dark and light ones.
const (
	tabAll = iota
	tabAdaptive
	tabDark
	tabLight
	tabCount
)

var tabNames = [tabCount]string{"All", "Adaptive", "Dark", "Light"}

// Model is the picker's state. It does no I/O, so a test drives it with keys
// and reads its frames.
type Model struct {
	names   []string
	group   map[string]int // the tab each name belongs to, besides All
	matches []string
	filter  []rune
	tab     int
	cursor  int  // in matches
	top     int  // first match shown
	placed  bool // the first frame has centered the cursor
	saved   string
	preview func(name string) []string
	resolve func(name string) string
	unicode bool
	listW   int // the list pane's width: the longest name and its kind
}

// New starts a picker over names with the cursor on start. saved is marked as
// the theme in use; preview draws a theme's sample lines and resolve names the
// variant a family stands for.
func New(names []string, start, saved string, preview func(string) []string, resolve func(string) string, unicode bool) *Model {
	m := &Model{names: names, group: groups(names), saved: saved, preview: preview, resolve: resolve, unicode: unicode}
	longest := 0
	for _, n := range names {
		longest = max(longest, utf8.RuneCountInString(n))
	}
	// " ▸ " + mark + name + gap + kind ("light") + the right padding.
	m.listW = min(3+2+longest+2+5+padding, 40)
	m.refilter()
	for i, n := range m.matches {
		if n == start {
			m.cursor = i
		}
	}
	return m
}

// groups sorts each name into a tab: a family (NAME with NAME-dark and
// NAME-light beside it), default and mono are adaptive; a -light variant is
// light; every other theme is dark.
func groups(names []string) map[string]int {
	has := make(map[string]bool, len(names))
	for _, n := range names {
		has[n] = true
	}
	g := make(map[string]int, len(names))
	for _, n := range names {
		switch {
		case n == "default" || n == "mono" || (has[n+"-dark"] && has[n+"-light"]):
			g[n] = tabAdaptive
		case strings.HasSuffix(n, "-light"):
			g[n] = tabLight
		default:
			g[n] = tabDark
		}
	}
	return g
}

// Selected is the theme under the cursor, "" when nothing matches.
func (m *Model) Selected() string {
	if len(m.matches) == 0 {
		return ""
	}
	return m.matches[m.cursor]
}

// Handle applies a key. done ends the picker; chosen says whether Enter
// picked Selected (otherwise it was canceled). page is how far Page Up and
// Page Down move.
func (m *Model) Handle(k Key, page int) (done, chosen bool) {
	page = max(page, 1)
	switch k.Code {
	case Up:
		m.move(-1)
	case Down:
		m.move(1)
	case PageUp:
		m.move(-page)
	case PageDown:
		m.move(page)
	case Home:
		m.cursor = 0
	case End:
		m.cursor = max(len(m.matches)-1, 0)
	case NextTab:
		m.tab = (m.tab + 1) % tabCount
		m.keepSelection()
	case PrevTab:
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.keepSelection()
	case Enter:
		return m.Selected() != "", m.Selected() != ""
	case Escape:
		if len(m.filter) > 0 {
			m.filter = m.filter[:0]
			m.keepSelection()
			return false, false
		}
		return true, false
	case Quit:
		return true, false
	case Backspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.keepSelection()
		}
	case ClearFilter:
		m.filter = m.filter[:0]
		m.keepSelection()
	case Char:
		// Only what prints is kept and echoed: a control character (a C1 CSI
		// among them) would reach the terminal as a command.
		if len(m.filter) < 32 && unicode.IsPrint(k.Rune) {
			m.filter = append(m.filter, k.Rune)
			m.keepSelection()
		}
	}
	return false, false
}

func (m *Model) move(by int) {
	if len(m.matches) == 0 {
		return
	}
	m.cursor = min(max(m.cursor+by, 0), len(m.matches)-1)
}

// keepSelection filters again and keeps the cursor on the same theme while it
// still matches.
func (m *Model) keepSelection() {
	was := m.Selected()
	m.refilter()
	m.cursor = 0
	for i, n := range m.matches {
		if n == was {
			m.cursor = i
		}
	}
	m.placed = false
}

func (m *Model) refilter() {
	f := strings.ToLower(string(m.filter))
	m.matches = m.matches[:0]
	for _, n := range m.names {
		if (m.tab == tabAll || m.group[n] == m.tab) && strings.Contains(n, f) {
			m.matches = append(m.matches, n)
		}
	}
}

// count is how many themes a tab holds, before the filter.
func (m *Model) count(tab int) int {
	if tab == tabAll {
		return len(m.names)
	}
	n := 0
	for _, g := range m.group {
		if g == tab {
			n++
		}
	}
	return n
}

// glyphs are the frame's drawing characters, in Unicode or plain ASCII.
type glyphs struct {
	tl, tr, bl, br, h, v, lt, rt, tt, bt, cross, cursor, mark, arrow, search, caret string
}

var (
	unicodeGlyphs = glyphs{"╭", "╮", "╰", "╯", "─", "│", "├", "┤", "┬", "┴", "┼", "▸", "●", "→", "›", "▏"}
	asciiGlyphs   = glyphs{"+", "+", "+", "+", "-", "|", "+", "+", "+", "+", "+", ">", "*", "->", ">", "_"}
)

// The frame's layout, in cells.
const (
	maxWidth = 112 // a frame wider than this only spreads the text apart
	minSide  = 70  // narrower than this, the preview goes under the list
	padding  = 2   // the space inside each pane's border
)

// Frame draws the picker for a terminal width by height cells: a frame of
// whole lines from the top-left corner, each cut to the width and cleared to
// its end, so a frame never wraps or leaves the last one's text behind.
func (m *Model) Frame(width, height int) string {
	if width < 24 || height < 8 {
		// No room for a frame: say so, within what there is.
		return "\x1b[H" + clip("Enlarge the terminal (esc quits).", max(width, 1)) + "\x1b[K\x1b[J"
	}
	w := min(width, maxWidth)
	g := asciiGlyphs
	if m.unicode {
		g = unicodeGlyphs
	}
	inner := w - 2 // inside the outer border

	var lines []string
	// Title bar.
	title := " " + bold("gnux theme") + " "
	lines = append(lines, g.tl+g.h+title+strings.Repeat(g.h, max(inner-1-visible(title), 0))+g.tr)
	// Tabs.
	var tabs []string
	for i, name := range tabNames {
		label := " " + name + " " + dim(strconv.Itoa(m.count(i))) + " "
		if i == m.tab {
			label = "\x1b[7m " + name + " " + strconv.Itoa(m.count(i)) + " \x1b[0m"
		}
		tabs = append(tabs, label)
	}
	lines = append(lines, row(g, inner, strings.Join(tabs, " ")))
	// Search, with the count of matches and the cursor's place on the right.
	query := dim("type to filter")
	if len(m.filter) > 0 {
		query = string(m.filter) + g.caret
	}
	place := dim(strconv.Itoa(min(m.cursor+1, len(m.matches))) + "/" + strconv.Itoa(len(m.matches)))
	search := bold(g.search) + " " + query
	gap := inner - 2*padding - visible(search) - visible(place)
	lines = append(lines, row(g, inner, search+strings.Repeat(" ", max(gap, 1))+place))

	// The preview pane.
	var preview []string
	if sel := m.Selected(); sel != "" {
		head := bold(sel)
		if r := m.resolve(sel); r != sel {
			head += dim(" " + g.arrow + " " + r)
		}
		switch sel {
		case m.saved:
			head += "  " + dim("(in use)")
		case "default":
			head += "  " + dim("(Enter resets)")
		}
		preview = append([]string{head, ""}, m.preview(sel)...)
	} else {
		preview = []string{dim("No theme matches."), "", dim("Backspace or Esc clears the filter.")}
	}

	// Footer: the keys, each bold beside what it does.
	// In order of need, since a narrow frame shows only the first few.
	hints := [][2]string{{"↑↓", "move"}, {"enter", "save"}, {"esc", "quit"}, {"tab", "group"}, {"type", "filter"}}
	if !m.unicode {
		hints[0][0] = "up/down"
	}
	// As many hints as fit, whole: a cut one would read as another key.
	footer := ""
	for _, h := range hints {
		next := bold(h[0]) + " " + dim(h[1])
		if footer != "" {
			next = "   " + next
		}
		if visible(footer)+visible(next) > inner-2*padding {
			break
		}
		footer += next
	}

	// Body: the list beside the preview, or above it when the frame is narrow.
	bodyRows := height - len(lines) - 4 // two separators, footer, bottom border
	side := w >= minSide
	listRows := bodyRows
	if !side {
		// Half for the list, the rest (less a separator) for the preview.
		listRows = max(bodyRows/2, 3)
		preview = preview[:min(len(preview), max(bodyRows-listRows-1, 1))]
	}
	listRows = max(listRows, 1)
	m.scroll(listRows)
	list := m.listCells(listRows, g)

	if side {
		left := m.listW
		right := inner - left - 1
		lines = append(lines, g.lt+strings.Repeat(g.h, left)+g.tt+strings.Repeat(g.h, right)+g.rt)
		for i := range max(bodyRows, 1) {
			l, r := "", ""
			if i < len(list) {
				l = list[i]
			}
			if i < len(preview) {
				r = preview[i]
			}
			lines = append(lines, g.v+cell(l, left)+g.v+cell(strings.Repeat(" ", padding)+r, right)+g.v)
		}
		lines = append(lines, g.lt+strings.Repeat(g.h, left)+g.bt+strings.Repeat(g.h, right)+g.rt)
	} else {
		lines = append(lines, g.lt+strings.Repeat(g.h, inner)+g.rt)
		for _, l := range list {
			lines = append(lines, g.v+cell(l, inner)+g.v)
		}
		lines = append(lines, g.lt+strings.Repeat(g.h, inner)+g.rt)
		for _, p := range preview {
			lines = append(lines, row(g, inner, p))
		}
		lines = append(lines, g.lt+strings.Repeat(g.h, inner)+g.rt)
	}
	lines = append(lines, row(g, inner, footer))
	lines = append(lines, g.bl+strings.Repeat(g.h, inner)+g.br)

	if len(lines) > height {
		// Keep the bottom border: drop body lines above it.
		lines = append(lines[:height-2], lines[len(lines)-2:]...)
	}
	var b strings.Builder
	b.Grow(len(lines) * (w + 32))
	b.WriteString("\x1b[H")
	for i, l := range lines {
		b.WriteString(clip(l, width))
		b.WriteString("\x1b[0m\x1b[K")
		if i < len(lines)-1 {
			b.WriteString("\r\n")
		}
	}
	b.WriteString("\x1b[J")
	return b.String()
}

// listCells are the list pane's rows: the theme under the cursor reversed, the
// one in use marked, and each theme's kind dimmed on the right.
func (m *Model) listCells(rows int, g glyphs) []string {
	cells := make([]string, 0, rows)
	width := m.listW - padding - 1 // the leading space and the right padding aside
	for i := m.top; i < len(m.matches) && len(cells) < rows; i++ {
		n := m.matches[i]
		lead := "  "
		if i == m.cursor {
			lead = g.cursor + " "
		}
		mark := "  "
		if n == m.saved {
			mark = g.mark + " "
		}
		kind := map[int]string{tabAdaptive: "auto", tabDark: "dark", tabLight: "light"}[m.group[n]]
		name := n
		room := width - visible(lead) - visible(mark) - len(kind) - 1
		if utf8.RuneCountInString(name) > room {
			name = string([]rune(name)[:max(room-1, 0)]) + "…"
		}
		text := mark + name + strings.Repeat(" ", max(room-utf8.RuneCountInString(name), 0)+1)
		if i == m.cursor {
			cells = append(cells, " "+lead+"\x1b[7m"+text+kind+"\x1b[0m")
		} else {
			cells = append(cells, " "+lead+text+dim(kind))
		}
	}
	return cells
}

// scroll keeps the cursor inside the rows shown; the first frame after a
// change of tab or filter puts it in the middle, so the theme in use is seen
// with its neighbors.
func (m *Model) scroll(rows int) {
	if !m.placed {
		m.placed = true
		m.top = m.cursor - rows/2
	}
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+rows {
		m.top = m.cursor - rows + 1
	}
	m.top = max(min(m.top, len(m.matches)-rows), 0)
}

// row is a full-width line inside the outer border, padded on both sides.
func row(g glyphs, inner int, s string) string {
	pad := strings.Repeat(" ", padding)
	return g.v + cell(pad+s, inner-padding) + pad + g.v
}

// cell is s cut or padded to exactly width visible cells.
func cell(s string, width int) string {
	s = clip(s, width)
	return s + "\x1b[0m" + strings.Repeat(" ", max(width-visible(s), 0))
}

func bold(s string) string { return "\x1b[1m" + s + "\x1b[0m" }
func dim(s string) string  { return "\x1b[2m" + s + "\x1b[0m" }

// visible is how many cells s takes on screen, escape sequences aside.
func visible(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			i = escapeEnd(s, i) + 1
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		n++
		i += size
	}
	return n
}

// escapeEnd is the index of the last byte of the escape sequence at i.
func escapeEnd(s string, i int) int {
	if i+1 < len(s) && s[i+1] == '[' {
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
			j++
		}
		return min(j, len(s)-1)
	}
	return min(i+1, len(s)-1)
}

// clip cuts s to width visible cells, keeping its escape sequences, so a line
// never wraps and pushes the frame out of place.
func clip(s string, width int) string {
	var b strings.Builder
	cells := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			end := escapeEnd(s, i)
			b.WriteString(s[i : end+1])
			i = end + 1
			continue
		}
		_, n := utf8.DecodeRuneInString(s[i:])
		if cells < width {
			b.WriteString(s[i : i+n])
			cells++
		}
		i += n
	}
	return b.String()
}
