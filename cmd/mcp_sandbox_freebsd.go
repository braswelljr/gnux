//go:build freebsd

package cmd

// sandboxEachCall: under Capsicum a sandboxed process cannot start another,
// so each call of gnux mcp enters the sandbox itself.
const sandboxEachCall = true
