//go:build !unix

package stdbuf

import "os"

func signalOf(*os.ProcessState) int { return 0 }
