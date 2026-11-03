package utmp

import (
	"os/user"
	"strings"
)

// Self is the login name of the user running the process, for a view that
// marks the caller's own entries: the account's name, without a Windows
// domain, else LOGNAME or USER from getenv.
func Self(getenv func(string) string) string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		name := u.Username
		if i := strings.LastIndexByte(name, '\\'); i >= 0 {
			name = name[i+1:]
		}
		return name
	}
	if name := getenv("LOGNAME"); name != "" {
		return name
	}
	return getenv("USER")
}
