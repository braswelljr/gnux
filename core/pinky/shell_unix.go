//go:build unix

package pinky

import (
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// shellOf reads a user's login shell from /etc/passwd. It is empty for a
// user kept elsewhere (macOS directory services, LDAP), which os/user can
// find without cgo but whose shell it does not report.
func shellOf(name string) string {
	data, err := fsys.ReadFile("/etc/passwd")
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 7 && fields[0] == name {
			return fields[6]
		}
	}
	return ""
}
