//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package nice

// Off the systems with Unix niceness, the niceness reads as 0 and cannot be
// changed; nicex warns and runs the command at the normal priority.
func getNiceness() (int, error) {
	return 0, nil
}

func setNiceness(int, int) error {
	return ErrUnsupported
}
