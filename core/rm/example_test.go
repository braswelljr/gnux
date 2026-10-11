package rm_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

// The engine is an ordinary Go API: build it with the options the flags would
// set, hand it streams, and run it against paths.
func Example() {
	dir, err := os.MkdirTemp("", "rm-example")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	build := filepath.Join(dir, "build")
	if err := os.MkdirAll(filepath.Join(build, "obj"), 0o755); err != nil {
		panic(err)
	}

	// Equivalent to `rmx -rf build`, with diagnostics discarded.
	engine := rm.New(rm.Options{Recursive: true, Force: true}, core.Discard())
	if err := engine.Run([]string{build}); err != nil {
		fmt.Println("remove failed:", err)
		return
	}

	_, err = os.Stat(build)
	fmt.Println("removed:", os.IsNotExist(err))
	// Output: removed: true
}
