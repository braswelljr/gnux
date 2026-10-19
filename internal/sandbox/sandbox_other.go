//go:build !linux && !darwin && !windows && !freebsd

package sandbox

import (
	"fmt"
	"runtime"
)

// enter refuses: there is no kernel sandbox gnux can use here, and running
// unconfined instead would make the sandbox a lie.
func enter(Policy) error {
	return fmt.Errorf("sandbox: %w: gnux can sandbox itself on Linux (Landlock) and macOS (Seatbelt), not on %s", ErrUnsupported, runtime.GOOS)
}
