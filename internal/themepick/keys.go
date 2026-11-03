package themepick

import "unicode/utf8"

// Key is one key press the picker understands.
type Key struct {
	Code Code
	Rune rune // for Char
}

// Code names a key.
type Code int

// The keys the picker reads.
const (
	Char Code = iota
	Up
	Down
	PageUp
	PageDown
	Home
	End
	Enter
	Escape
	Backspace
	ClearFilter // Ctrl-U
	Quit        // Ctrl-C, Ctrl-D
	NextTab     // Tab, Right
	PrevTab     // Shift-Tab, Left
)

// ParseKeys decodes what a terminal in raw mode sent: printable characters,
// the control keys and the escape sequences of the arrow and paging keys.
// An escape sequence it does not know is skipped whole; a lone ESC (nothing
// after it in the same read) is the Escape key.
func ParseKeys(b []byte) []Key {
	var keys []Key
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 0x1b && len(b) == 1:
			keys = append(keys, Key{Code: Escape})
			b = b[1:]
		case c == 0x1b && (b[1] == '[' || b[1] == 'O'):
			n, code := escapeSequence(b)
			if code >= 0 {
				keys = append(keys, Key{Code: code})
			}
			b = b[n:]
		case c == 0x1b:
			// Alt plus a key: take it as Escape, then the key.
			keys = append(keys, Key{Code: Escape})
			b = b[1:]
		case c == '\r' || c == '\n':
			keys = append(keys, Key{Code: Enter})
			b = b[1:]
		case c == 0x7f || c == 0x08:
			keys = append(keys, Key{Code: Backspace})
			b = b[1:]
		case c == 0x03 || c == 0x04:
			keys = append(keys, Key{Code: Quit})
			b = b[1:]
		case c == 0x15:
			keys = append(keys, Key{Code: ClearFilter})
			b = b[1:]
		case c == 0x10: // Ctrl-P
			keys = append(keys, Key{Code: Up})
			b = b[1:]
		case c == 0x0e: // Ctrl-N
			keys = append(keys, Key{Code: Down})
			b = b[1:]
		case c == '\t':
			keys = append(keys, Key{Code: NextTab})
			b = b[1:]
		case c < 0x20:
			b = b[1:]
		default:
			r, n := utf8.DecodeRune(b)
			if r != utf8.RuneError {
				keys = append(keys, Key{Code: Char, Rune: r})
			}
			b = b[n:]
		}
	}
	return keys
}

// escapeSequence measures the CSI or SS3 sequence at the start of b and
// names its key, or -1 for one the picker does not use.
func escapeSequence(b []byte) (int, Code) {
	end := 2
	for end < len(b) && (b[end] < 0x40 || b[end] > 0x7e) {
		end++
	}
	if end == len(b) {
		return len(b), -1 // cut short: drop it
	}
	switch string(b[2 : end+1]) {
	case "A":
		return end + 1, Up
	case "B":
		return end + 1, Down
	case "C":
		return end + 1, NextTab
	case "D":
		return end + 1, PrevTab
	case "Z":
		return end + 1, PrevTab
	case "5~":
		return end + 1, PageUp
	case "6~":
		return end + 1, PageDown
	case "H", "1~", "7~":
		return end + 1, Home
	case "F", "4~", "8~":
		return end + 1, End
	}
	return end + 1, -1
}
