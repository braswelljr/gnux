// Package uname implements the engine behind unamex, a GNU-compatible
// implementation of the GNU uname command: print system information.
package uname

import (
	"fmt"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "unamex"

// unknown is what GNU prints for a field the system does not report.
const unknown = "unknown"

// Info is the system information uname reports, one field per option.
type Info struct {
	KernelName       string // -s
	Nodename         string // -n
	KernelRelease    string // -r
	KernelVersion    string // -v
	Machine          string // -m
	Processor        string // -p
	HardwarePlatform string // -i
	OperatingSystem  string // -o
}

// Options mirrors the GNU uname flag surface. With no field selected, uname
// prints the kernel name, as -s does.
type Options struct {
	// All (-a) prints every field, leaving out -p and -i when unknown.
	All              bool
	KernelName       bool
	Nodename         bool
	KernelRelease    bool
	KernelVersion    bool
	Machine          bool
	Processor        bool
	HardwarePlatform bool
	OperatingSystem  bool
}

// Uname is the engine. Construct it with New.
type Uname struct {
	Options

	// System reads the information; it defaults to the platform's own source
	// (uname(2) on Linux, sysctl on macOS and FreeBSD, the registry on Windows).
	System func() (Info, error)

	streams core.Streams
	diag    *core.Diag
}

// New builds a uname engine writing to s.
func New(opts Options, s core.Streams) *Uname {
	return &Uname{
		Options: opts,
		System:  system,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run prints the selected fields, separated by spaces, in GNU's fixed order
// whatever the order of the options. uname takes no operands.
func (u *Uname) Run(args []string) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	info, err := u.System()
	if err != nil {
		u.diag.Errorf("cannot get system name: %s", core.ErrText(err))
		return u.diag.Result()
	}

	o := u.Options
	if o == (Options{}) {
		o.KernelName = true
	}
	fields := []struct {
		on, all bool
		value   string
	}{
		{o.KernelName, true, info.KernelName},
		{o.Nodename, true, info.Nodename},
		{o.KernelRelease, true, info.KernelRelease},
		{o.KernelVersion, true, info.KernelVersion},
		{o.Machine, true, info.Machine},
		{o.Processor, info.Processor != unknown, info.Processor},
		{o.HardwarePlatform, info.HardwarePlatform != unknown, info.HardwarePlatform},
		{o.OperatingSystem, true, info.OperatingSystem},
	}
	var out []string
	for _, f := range fields {
		if f.on || (o.All && f.all) {
			out = append(out, orUnknown(f.value))
		}
	}
	fmt.Fprintln(u.streams.Out, strings.Join(out, " "))
	return nil
}

func orUnknown(s string) string {
	if s == "" {
		return unknown
	}
	return s
}
