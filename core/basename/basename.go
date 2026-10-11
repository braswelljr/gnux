// Package basename implements the engine behind basenamex, a GNU-compatible
// implementation of the GNU/UNIX basename command: strip the directory and,
// optionally, a trailing suffix from file names.
package basename

import (
	"fmt"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "basenamex"

// Options mirrors the GNU basename flag surface.
type Options struct {
	// Multiple (-a) treats every operand as a NAME instead of reading the
	// second operand as a suffix.
	Multiple bool
	// Suffix (-s) is removed from the end of each name. A non-empty Suffix
	// implies Multiple, as in GNU basename.
	Suffix string
	// Zero (-z) ends each output line with NUL instead of a newline.
	Zero bool
}

// Basename is the engine. Construct it with New.
type Basename struct {
	Options

	streams core.Streams
}

// New builds a basename engine writing to s.
func New(opts Options, s core.Streams) *Basename {
	return &Basename{Options: opts, streams: s}
}

// Run prints the final component of each name. Without -a/-s it accepts the
// traditional "NAME [SUFFIX]" form.
func (b *Basename) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	names, suffix := args, b.Suffix
	if !b.Multiple && b.Suffix == "" {
		if len(args) > 2 {
			return core.Usagef("extra operand '%s'", args[2])
		}
		names = args[:1]
		if len(args) == 2 {
			suffix = args[1]
		}
	}

	end := "\n"
	if b.Zero {
		end = "\x00"
	}
	for _, name := range names {
		fmt.Fprint(b.streams.Out, Base(name, suffix)+end)
	}
	return nil
}

// Base returns the final component of name with suffix removed, following the
// POSIX basename rules rather than filepath.Base: an empty name stays empty,
// a name made only of separators is the root, and a suffix equal to the whole
// component is left alone. '/' separates components everywhere; the native
// separator is honored too, so a backslash is an ordinary character on Unix.
func Base(name, suffix string) string {
	if name == "" {
		return ""
	}

	end := len(name)
	for end > 0 && os.IsPathSeparator(name[end-1]) {
		end--
	}
	if end == 0 {
		// Only separators: the result is the root, spelled as it was given.
		return name[:1]
	}

	start := end
	for start > 0 && !os.IsPathSeparator(name[start-1]) {
		start--
	}
	base := name[start:end]

	if suffix != "" && len(base) > len(suffix) && strings.HasSuffix(base, suffix) {
		base = base[:len(base)-len(suffix)]
	}
	return base
}
