//go:build !freebsd

package sandbox

// refusedHere reports whether err is this system's own way of refusing a
// path in the sandbox; elsewhere that is permission denied, checked anyway.
func refusedHere(error) bool { return false }
