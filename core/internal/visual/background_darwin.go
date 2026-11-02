package visual

import (
	"errors"
	"os/exec"
	"strings"
	"sync"
)

// systemDark reads macOS's appearance: AppleInterfaceStyle is "Dark" in
// dark mode and missing (defaults fails) in light mode. It is asked once
// per process.
func systemDark(func(string) string) (dark, ok bool) {
	return appearance()
}

var appearance = sync.OnceValues(func() (bool, bool) {
	out, err := exec.Command("/usr/bin/defaults", "read", "-g", "AppleInterfaceStyle").Output()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return strings.TrimSpace(string(out)) == "Dark", true
	case errors.As(err, &exit):
		return false, true
	}
	return false, false
})
