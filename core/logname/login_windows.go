package logname

import (
	"os/user"
	"strings"
)

// login is the user the Windows session was signed in as, without its
// domain, as whoamix prints it.
func login(any) (string, bool) {
	u, err := user.Current()
	if err != nil || u.Username == "" {
		return "", false
	}
	name := u.Username
	if i := strings.LastIndexByte(name, '\\'); i >= 0 {
		name = name[i+1:]
	}
	return name, true
}
