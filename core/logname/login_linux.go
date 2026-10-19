package logname

import (
	"os/user"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
)

// login is glibc's getlogin: the user the kernel's audit subsystem records as
// having logged in (/proc/self/loginuid, which survives su and sudo), and
// failing that the login record of the terminal on stdin.
func login(stdin any) (string, bool) {
	if data, err := fsys.ReadFile("/proc/self/loginuid"); err == nil {
		// 4294967295, (uid_t)-1, means no login has been recorded.
		if uid := strings.TrimSpace(string(data)); uid != "" && uid != "4294967295" {
			if u, err := user.LookupId(uid); err == nil && u.Username != "" {
				return u.Username, true
			}
		}
	}
	return fromTerminal(stdin)
}
