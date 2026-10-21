//go:build !unix

package stdbuf

// Windows has no library preloading of the kind stdbuf relies on, so only
// gnux's own tools can be run.
const preloadVar = ""

var (
	extraEnv []string
	libDirs  []string
)
