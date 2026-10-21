package find

import shellpattern "github.com/braswelljr/gnux/core/internal/fnmatch"

// fnmatch reports whether name matches the shell pattern as GNU find asks
// fnmatch(3) to, with case folding for the -i tests.
func fnmatch(pattern, name string, fold bool) bool {
	return shellpattern.Match(pattern, name, fold)
}
