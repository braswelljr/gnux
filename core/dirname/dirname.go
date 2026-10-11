// Package dirname implements the engine behind dirnamex, a GNU-compatible
// implementation of the GNU/UNIX dirname command: strip the last component from file names.
package dirname

import (
	"fmt"
	"os"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "dirnamex"

// Options mirrors the GNU dirname flag surface.
type Options struct {
	// Zero (-z) ends each output line with NUL instead of a newline.
	Zero bool
}

// Dirname is the engine. Construct it with New.
type Dirname struct {
	Options

	streams core.Streams
}

// New builds a dirname engine writing to s.
func New(opts Options, s core.Streams) *Dirname {
	return &Dirname{Options: opts, streams: s}
}

// Run prints the directory part of each name.
func (d *Dirname) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	end := "\n"
	if d.Zero {
		end = "\x00"
	}
	for _, name := range args {
		fmt.Fprint(d.streams.Out, Dir(name)+end)
	}
	return nil
}

// Dir returns name with its last component removed, following the POSIX
// dirname rules: a name without a separator is ".", and anything directly
// under the root is the root. Unlike filepath.Dir it does not clean the rest
// of the path, so "a/./b" stays "a/.". '/' separates components everywhere;
// the native separator is honored too, so a backslash is an ordinary character
// on Unix.
func Dir(name string) string {
	end := len(name)
	for end > 0 && os.IsPathSeparator(name[end-1]) {
		end--
	}
	if end == 0 {
		if name == "" {
			return "."
		}
		// Only separators: the root, spelled as it was given.
		return name[:1]
	}

	for end > 0 && !os.IsPathSeparator(name[end-1]) {
		end--
	}
	if end == 0 {
		return "."
	}

	for end > 0 && os.IsPathSeparator(name[end-1]) {
		end--
	}
	if end == 0 {
		return name[:1]
	}
	return name[:end]
}
