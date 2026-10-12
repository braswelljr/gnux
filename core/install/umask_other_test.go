//go:build !unix

package install_test

func umask() uint32 { return 0 }
