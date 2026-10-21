//go:build linux

package hostid

import (
	"encoding/binary"

	"github.com/braswelljr/gnux/core/fsys"
)

// hostID is glibc's gethostid: the four bytes of /etc/hostid when it holds
// them, otherwise the identifier derived from the host's address.
func hostID() uint32 {
	if b, err := fsys.ReadFile("/etc/hostid"); err == nil && len(b) >= 4 {
		return binary.NativeEndian.Uint32(b)
	}
	return fromHostname()
}
