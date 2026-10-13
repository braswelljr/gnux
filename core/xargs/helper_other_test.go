//go:build !unix

package xargs_test

// helperUnix has nothing to add where there are no Unix signals.
func helperUnix([]string) {}
