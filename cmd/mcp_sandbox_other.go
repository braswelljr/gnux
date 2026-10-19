//go:build !freebsd

package cmd

// sandboxEachCall is false: gnux mcp enters the sandbox once, and every call
// it starts inherits it.
const sandboxEachCall = false
