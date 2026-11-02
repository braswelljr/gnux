package ls

import (
	"os"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// palette colors names as GNU ls does (visual.Names), with what only an
// lsx entry knows: a symbolic link's target.
type palette struct {
	*visual.Names
}

// newPalette builds the palette from the built-in defaults overlaid with an
// LS_COLORS value ("di=01;34:*.tar=01;31:…").
func newPalette(lsColors string) *palette {
	return &palette{visual.NewNames(lsColors)}
}

// entryColor returns the color code for an entry's own name.
func (p *palette) entryColor(e *entry) string {
	if e.mode&os.ModeSymlink == 0 {
		return p.ModeColor(e.mode, e.name)
	}
	switch {
	case e.target == nil:
		if code := p.Lookup("or"); code != "" {
			return code
		}
	case p.LinkTarget():
		return p.ModeColor(e.target.Mode(), e.name)
	}
	return p.Lookup("ln")
}

// targetColor returns the color code for the target shown after "->".
func (p *palette) targetColor(e *entry) string {
	if e.target == nil {
		return p.Lookup("mi")
	}
	return p.ModeColor(e.target.Mode(), e.link)
}
