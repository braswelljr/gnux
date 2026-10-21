//go:build unix && !darwin

package stdbuf

const preloadVar = "LD_PRELOAD"

var extraEnv []string

// libDirs are where GNU coreutils packages install it: Debian, Fedora,
// Arch, Alpine and openSUSE use /usr/libexec/coreutils, older systems
// /usr/lib/coreutils, and a source build /usr/local/libexec/coreutils.
var libDirs = []string{
	"/usr/libexec/coreutils",
	"/usr/lib/coreutils",
	"/usr/local/libexec/coreutils",
	"/usr/lib64/coreutils",
}
