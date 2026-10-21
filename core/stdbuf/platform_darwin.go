//go:build darwin

package stdbuf

// macOS preloads libraries through dyld, which needs a flat namespace for
// libstdbuf to replace the stdio setup.
const preloadVar = "DYLD_INSERT_LIBRARIES"

var extraEnv = []string{"DYLD_FORCE_FLAT_NAMESPACE=y"}

// libDirs are where Homebrew's and MacPorts' GNU coreutils install it.
var libDirs = []string{
	"/opt/homebrew/opt/coreutils/libexec/coreutils",
	"/usr/local/opt/coreutils/libexec/coreutils",
	"/opt/local/libexec/coreutils",
}
