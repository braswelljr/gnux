package ls

import (
	"os"
	"strings"
)

// defaultColors is GNU ls's built-in palette, keyed as in LS_COLORS.
var defaultColors = map[string]string{
	"di": "01;34", // directory
	"ln": "01;36", // symbolic link
	"pi": "33",    // fifo
	"so": "01;35", // socket
	"bd": "01;33", // block device
	"cd": "01;33", // character device
	"ex": "01;32", // executable
	"su": "37;41", // setuid
	"sg": "30;43", // setgid
	"tw": "30;42", // sticky and other-writable directory
	"ow": "34;42", // other-writable directory
	"st": "37;44", // sticky directory
}

// palette maps file types and name suffixes to SGR color codes.
type palette struct {
	types    map[string]string
	suffixes []suffixColor // in LS_COLORS order; later entries win
}

type suffixColor struct {
	suffix, code string
}

// newPalette builds the palette from the built-in defaults overlaid with an
// LS_COLORS value ("di=01;34:*.tar=01;31:…").
func newPalette(lsColors string) *palette {
	p := &palette{types: make(map[string]string, len(defaultColors))}
	for k, v := range defaultColors {
		p.types[k] = v
	}
	for _, item := range strings.Split(lsColors, ":") {
		key, code, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		if suffix, isGlob := strings.CutPrefix(key, "*"); isGlob {
			p.suffixes = append(p.suffixes, suffixColor{strings.ToLower(suffix), code})
			continue
		}
		p.types[key] = code
	}
	return p
}

// lookup returns the code for a type key, or "" when it is unset or a no-op.
func (p *palette) lookup(key string) string {
	switch code := p.types[key]; code {
	case "0", "00":
		return ""
	default:
		return code
	}
}

// entryColor returns the color code for an entry's own name.
func (p *palette) entryColor(e *entry) string {
	if e.mode&os.ModeSymlink == 0 {
		return p.modeColor(e.mode, e.name)
	}
	switch {
	case e.target == nil:
		if code := p.lookup("or"); code != "" {
			return code
		}
	case p.types["ln"] == "target":
		return p.modeColor(e.target.Mode(), e.name)
	}
	return p.lookup("ln")
}

// targetColor returns the color code for the target shown after "->".
func (p *palette) targetColor(e *entry) string {
	if e.target == nil {
		return p.lookup("mi")
	}
	return p.modeColor(e.target.Mode(), e.link)
}

// modeColor picks a color from a file mode, falling back to the name's suffix
// for ordinary files.
func (p *palette) modeColor(mode os.FileMode, name string) string {
	var key string
	switch {
	case mode.IsDir():
		// Where permission bits are not real (Windows), every directory is plain.
		sticky := permissionBits && mode&os.ModeSticky != 0
		otherWritable := permissionBits && mode&0o002 != 0
		switch {
		case sticky && otherWritable:
			key = "tw"
		case otherWritable:
			key = "ow"
		case sticky:
			key = "st"
		default:
			key = "di"
		}
	case mode&os.ModeSymlink != 0:
		key = "ln"
	case mode&os.ModeNamedPipe != 0:
		key = "pi"
	case mode&os.ModeSocket != 0:
		key = "so"
	case mode&os.ModeCharDevice != 0:
		key = "cd"
	case mode&os.ModeDevice != 0:
		key = "bd"
	case mode&os.ModeSetuid != 0:
		key = "su"
	case mode&os.ModeSetgid != 0:
		key = "sg"
	case mode&0o111 != 0:
		key = "ex"
	}
	if key != "" {
		// A type whose color has been switched off falls through to the
		// suffix rules, as in GNU ls.
		if code := p.lookup(key); code != "" {
			return code
		}
	}

	lower := strings.ToLower(name)
	for i := len(p.suffixes) - 1; i >= 0; i-- {
		if strings.HasSuffix(lower, p.suffixes[i].suffix) {
			return p.suffixes[i].code
		}
	}
	return p.lookup("fi")
}

// putColored writes text wrapped in the given SGR code, or bare when code is
// empty. Like GNU ls, the first colored name is preceded by a reset so that
// whatever attributes the terminal was left in cannot bleed into the listing.
func (l *Ls) putColored(text, code string) {
	if code == "" {
		l.put(text)
		return
	}
	if !l.usedColor {
		l.usedColor = true
		l.put("\x1b[0m")
	}
	l.put("\x1b[" + code + "m" + text + "\x1b[0m")
}
