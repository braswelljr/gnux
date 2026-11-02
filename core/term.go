package core

import (
	"io"
	"os"
	"strconv"
)

// TerminalWidth is the width in columns of the terminal w writes to, as GNU
// tools find it: COLUMNS (through getenv) when it holds a positive number,
// else the terminal's own size; 0 when w is no terminal and COLUMNS is unset.
func TerminalWidth(w io.Writer, getenv func(string) string) int {
	if getenv != nil {
		if n, err := strconv.Atoi(getenv("COLUMNS")); err == nil && n > 0 {
			return n
		}
	}
	if f, ok := w.(*os.File); ok {
		return fileWidth(f)
	}
	return 0
}
