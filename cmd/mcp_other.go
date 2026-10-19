//go:build !unix && !windows

package cmd

import "os/exec"

// stopGroup leaves the default: on js and wasip1 there are no processes to
// start, and gnux mcp refuses to run there anyway (it has no sandbox).
func stopGroup(*exec.Cmd) {}
