package grep

import "strings"

// palette holds the SGR parameters output is colored with, GNU's GREP_COLORS
// capabilities. An empty capability means "not colored".
type palette struct {
	on bool

	ms string // a match in a selected line
	mc string // a match in a context line
	sl string // the rest of a selected line
	cx string // the rest of a context line
	fn string // file names
	ln string // line numbers
	bn string // byte offsets
	se string // separators

	rv bool // with -v, swap sl and cx
	ne bool // do not erase to the end of the line after each sequence
}

// defaultPalette is GNU grep's built-in GREP_COLORS:
// ms=01;31:mc=01;31:sl=:cx=:fn=35:ln=32:bn=32:se=36.
func defaultPalette() palette {
	return palette{ms: "01;31", mc: "01;31", fn: "35", ln: "32", bn: "32", se: "36"}
}

// parse applies a GREP_COLORS value. As in GNU grep, a malformed entry ends
// the parsing and keeps what came before it; unknown names are skipped.
func (p *palette) parse(spec string) {
	for _, entry := range strings.Split(spec, ":") {
		name, value, hasValue := strings.Cut(entry, "=")
		if !hasValue {
			switch name {
			case "rv":
				p.rv = true
			case "ne":
				p.ne = true
			case "":
			default:
				return
			}
			continue
		}
		if strings.Trim(value, "0123456789;") != "" {
			return
		}
		switch name {
		case "mt":
			p.ms, p.mc = value, value
		case "ms":
			p.ms = value
		case "mc":
			p.mc = value
		case "sl":
			p.sl = value
		case "cx":
			p.cx = value
		case "fn":
			p.fn = value
		case "ln":
			p.ln = value
		case "bn":
			p.bn = value
		case "se":
			p.se = value
		}
	}
}

// start appends the sequence that turns the capability on: "\33[...m\33[K".
func (p *palette) start(b []byte, capability string) []byte {
	if !p.on || capability == "" {
		return b
	}
	b = append(b, "\x1b["...)
	b = append(b, capability...)
	b = append(b, 'm')
	return p.erase(b)
}

// end appends the sequence that turns the capability off: "\33[m\33[K".
func (p *palette) end(b []byte, capability string) []byte {
	if !p.on || capability == "" {
		return b
	}
	b = append(b, "\x1b[m"...)
	return p.erase(b)
}

// erase appends EL, which GNU emits after every sequence so that a colored
// background does not leak when the terminal scrolls.
func (p *palette) erase(b []byte) []byte {
	if p.ne {
		return b
	}
	return append(b, "\x1b[K"...)
}

// paint appends text in the capability's color.
func (p *palette) paint(b []byte, capability, text string) []byte {
	b = p.start(b, capability)
	b = append(b, text...)
	return p.end(b, capability)
}
