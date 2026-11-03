package visual

import "strings"

// Report colors one line of a tool's -v report ("removed 'a'", "'a' ->
// 'b'", "mkdirx: created directory 'd'"): the tool's name prefix and the
// arrows in the punctuation color, each quoted name in the path color, and
// the words before the first name by what they report, a removal in the
// failure color, a creation in the success color. The text is unchanged.
func (t *Theme) Report(line string) string {
	if t == nil || line == "" {
		return line
	}
	var b strings.Builder
	rest := line
	// "prog: " before the message.
	if i := strings.Index(rest, ": "); i > 0 && !strings.ContainsAny(rest[:i], " '") {
		b.WriteString(t.Paint(Punctuation, rest[:i+2]))
		rest = rest[i+2:]
	}
	if q := strings.IndexByte(rest, '\''); q > 0 {
		b.WriteString(t.Paint(reportVerb(rest[:q]), rest[:q]))
		rest = rest[q:]
	}
	for rest != "" {
		if rest[0] != '\'' {
			q := strings.IndexByte(rest, '\'')
			if q < 0 {
				q = len(rest)
			}
			between := rest[:q]
			if strings.TrimSpace(between) == "->" {
				b.WriteString(t.Paint(Punctuation, between))
			} else {
				b.WriteString(between)
			}
			rest = rest[q:]
			continue
		}
		// A quoted name ends at a quote that ends a word, so one that
		// holds a quote of its own ('it's') stays whole.
		end := len(rest)
		for i := 1; i < len(rest); i++ {
			if rest[i] == '\'' && (i+1 == len(rest) || strings.IndexByte(" ,):", rest[i+1]) >= 0) {
				end = i + 1
				break
			}
		}
		b.WriteString(t.Paint(Path, rest[:end]))
		rest = rest[end:]
	}
	return b.String()
}

// reportVerb picks the color of the words that start a report line.
func reportVerb(words string) string {
	switch w := strings.ToLower(words); {
	case strings.Contains(w, "fail"):
		return Bad
	case strings.Contains(w, "remov"):
		return Bad
	case strings.Contains(w, "created"):
		return Good
	case strings.Contains(w, "retained"):
		return Punctuation
	}
	return Header
}
