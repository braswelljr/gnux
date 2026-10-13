//go:build !windows

package grep

// needTerm says whether --color=auto requires TERM to be set, as GNU grep
// does everywhere but on Windows: no TERM means no known terminal.
const needTerm = true
