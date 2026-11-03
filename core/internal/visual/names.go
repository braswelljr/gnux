package visual

import (
	"os"
	"strings"
)

// defaultNameColors is GNU ls's built-in palette for file names, keyed as
// in LS_COLORS.
var defaultNameColors = map[string]string{
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

// Names colors file names as GNU ls does: by file type, then by the
// name's suffix, from GNU's defaults overlaid with LS_COLORS. Every tool
// that shows a file name in its type's color (lsx, findx) uses it, so a
// name looks the same everywhere.
type Names struct {
	types    map[string]string
	suffixes []suffixColor // in LS_COLORS order; later entries win
}

type suffixColor struct {
	suffix, code string
}

// NewNames builds the palette from the built-in defaults overlaid with an
// LS_COLORS value ("di=01;34:*.tar=01;31:…").
func NewNames(lsColors string) *Names {
	p := &Names{types: make(map[string]string, len(defaultNameColors))}
	for k, v := range defaultNameColors {
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

// Lookup returns the code for a type key, or "" when it is unset or a
// no-op.
func (p *Names) Lookup(key string) string {
	switch code := p.types[key]; code {
	case "0", "00":
		return ""
	default:
		return code
	}
}

// LinkTarget reports whether LS_COLORS asks for a symbolic link to take
// its target's color (ln=target).
func (p *Names) LinkTarget() bool {
	return p.types["ln"] == "target"
}

// ModeColor picks a color from a file mode, falling back to the name's
// suffix for ordinary files.
func (p *Names) ModeColor(mode os.FileMode, name string) string {
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
		if code := p.Lookup(key); code != "" {
			return code
		}
	}

	lower := strings.ToLower(name)
	for i := len(p.suffixes) - 1; i >= 0; i-- {
		if strings.HasSuffix(lower, p.suffixes[i].suffix) {
			return p.suffixes[i].code
		}
	}
	return p.Lookup("fi")
}
