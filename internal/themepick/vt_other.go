//go:build !windows

package themepick

import "os"

// enableVT has nothing to do outside Windows: terminals read escape
// sequences already.
func enableVT(*os.File) func() { return func() {} }
