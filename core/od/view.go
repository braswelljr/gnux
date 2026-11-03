package od

import (
	"os"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view of a dump (BR-13): every value in the color of its
// bytes' class (visual.ByteKey: zero, printable, whitespace, other control,
// above 0x7f), and the offsets and the "*" of a repeated block dimmed. The
// text is GNU's. -S's strings are not colored.

func (o *Od) prepareView() {
	o.theme = nil
	if !o.Pretty || !o.Color || o.Strings > 0 {
		return
	}
	getenv := o.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	o.theme = visual.Use(true, o.Theme, getenv, o.diag.Warnf)
}

// appendPainted appends v in the color of key, or as it is uncolored.
func (o *Od) appendPainted(dst []byte, key string, v []byte) []byte {
	code := o.theme.Code(key)
	if code == "" {
		return append(dst, v...)
	}
	dst = append(dst, "\x1b["...)
	dst = append(dst, code...)
	dst = append(dst, 'm')
	dst = append(dst, v...)
	return append(dst, "\x1b[0m"...)
}

// dim is s in the punctuation color.
func (o *Od) dim(s string) string {
	return o.theme.Paint(visual.Punctuation, s)
}
