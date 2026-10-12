//go:build !linux && !darwin && !freebsd && !windows

package uname

import (
	"os"
	"runtime"
)

// kernelNames is what uname -s prints on the remaining platforms Go targets.
var kernelNames = map[string]string{
	"openbsd":   "OpenBSD",
	"netbsd":    "NetBSD",
	"dragonfly": "DragonFly",
	"solaris":   "SunOS",
	"illumos":   "SunOS",
	"aix":       "AIX",
	"plan9":     "Plan9",
	"js":        "js",
	"wasip1":    "WASI",
	"android":   "Linux",
	"ios":       "Darwin",
}

// system reports what Go knows about the platform when there is no system
// call gnux can use without cgo: its name and architecture, and the host
// name when there is one.
func system() (Info, error) {
	name := kernelNames[runtime.GOOS]
	if name == "" {
		name = runtime.GOOS
	}
	node, err := os.Hostname()
	if err != nil || node == "" {
		node = unknown
	}
	machine := goarchMachine[runtime.GOARCH]
	if machine == "" {
		machine = runtime.GOARCH
	}
	return Info{
		KernelName:       name,
		Nodename:         node,
		KernelRelease:    unknown,
		KernelVersion:    unknown,
		Machine:          machine,
		Processor:        unknown,
		HardwarePlatform: unknown,
		OperatingSystem:  name,
	}, nil
}
