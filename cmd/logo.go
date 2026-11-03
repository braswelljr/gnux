package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/tool"
)

// logo is the gnux mark drawn in Unicode half blocks: a keycap tile with a
// terminal prompt and a spanner cut out of it, and the line near its foot
// where the face meets the skirt. It is the website's mark
// (web/src/components/ui/logo.tsx), rasterized from the same 32 by 32
// geometry at 24 by 24 pixels, two to a character cell.
var logo = []string{
	"  ▄▄████████████████▄▄",
	" ██████████████████████",
	"█████▀▀████████▀████████",
	"█████▄  ▀████  ▄████████",
	"██████▀  ████   ▀▀ █████",
	"█████ ▄████▀     ▄██████",
	"█████████▀  ▄███████████",
	"███████▀  ▄█████████████",
	"███████▄▄███████████████",
	"███▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀███",
	" ██████████████████████",
	"  ▀▀████████████████▀▀",
}

// logoWidth is the width of the widest line of the logo, in cells.
const logoWidth = 24

// writeLogo draws the mark with the name, version and what gnux is beside it.
// gnux's own help writes it to a terminal only, before the usage line, so
// that help read by a script or a pipe is unchanged.
func writeLogo(w io.Writer, st tool.Style, tools int) {
	side := map[int]string{
		3: st.Paint("1", common.AppName) + " " + st.Dim(common.Version),
		5: "The GNU core utilities' interface,",
		6: "on every platform, in one Go binary.",
		8: fmt.Sprintf("%d tools: %s", tools, st.Command("gnux --list")),
		9: "A tool's manual: " + st.Command("gnux help rmx"),
	}
	for i, line := range logo {
		fmt.Fprint(w, st.Mark(line))
		if text, ok := side[i]; ok {
			pad := logoWidth - len([]rune(line)) + 4
			fmt.Fprint(w, strings.Repeat(" ", pad), text)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}
