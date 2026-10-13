package grep

// needTerm says whether --color=auto requires TERM to be set. A Windows
// console colors without one, and the variable is normally absent there.
const needTerm = false
