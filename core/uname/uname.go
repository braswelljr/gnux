// Package uname implements the engine behind unamex, a GNU-compatible
// implementation of the GNU uname command: print system information.
package uname

import (
	"fmt"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
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

	// Pretty, Color and Theme are gnux's terminal view, not GNU options:
	// with Pretty, the fields are a panel titled with the host's name, one
	// labeled line each, every field when none is selected, colored with
	// Color in Theme (a built-in theme; "" takes GNUX_THEME's).
	Pretty bool
	Color  bool
	Theme  string
}

// Uname is the engine. Construct it with New.
type Uname struct {
	Options

	// System reads the information; it defaults to the platform's own source
	// (uname(2) on Linux, sysctl on macOS and FreeBSD, the registry on Windows).
	System func() (Info, error)
	// Getenv reads GNUX_THEME and the locale for the view; nil means
	// os.Getenv.
	Getenv func(string) string

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
	o.Pretty, o.Color, o.Theme = false, false, ""
	if o == (Options{}) {
		if u.Pretty {
			o.All = true
		}
		o.KernelName = true
	}
	fields := []struct {
		on, all bool
		label   string
		value   string
	}{
		{o.KernelName, true, "Kernel", info.KernelName},
		{o.Nodename, true, "Host", info.Nodename},
		{o.KernelRelease, true, "Release", info.KernelRelease},
		{o.KernelVersion, true, "Version", info.KernelVersion},
		{o.Machine, true, "Machine", info.Machine},
		{o.Processor, info.Processor != unknown, "Processor", info.Processor},
		{o.HardwarePlatform, info.HardwarePlatform != unknown, "Platform", info.HardwarePlatform},
		{o.OperatingSystem, true, "System", info.OperatingSystem},
	}
	var out, labels []string
	for _, f := range fields {
		if f.on || (o.All && f.all) {
			out = append(out, orUnknown(f.value))
			labels = append(labels, f.label)
		}
	}
	if u.Pretty {
		u.panel(orUnknown(info.Nodename), labels, out)
		return nil
	}
	fmt.Fprintln(u.streams.Out, strings.Join(out, " "))
	return nil
}

// System reads the system information from the platform's own source:
// uname(2) on Linux, sysctl on macOS and FreeBSD, the registry on Windows. It
// is the default of Uname.System, and what archx reads the machine name from.
func System() (Info, error) {
	return system()
}

func orUnknown(s string) string {
	if s == "" {
		return unknown
	}
	return s
}

// panel is gnux's terminal view (BR-13): the fields as one panel titled
// with the host's name, each on a labeled line.
func (u *Uname) panel(host string, labels, values []string) {
	getenv := u.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	t := visual.Use(u.Color, u.Theme, getenv, u.diag.Warnf)
	unicode := visual.UTF8(getenv)
	// The frame takes four columns; a long kernel version is cut to fit
	// (width is -4 off a terminal with COLUMNS unset: no limit).
	width := core.TerminalWidth(u.streams.Out, getenv) - 4
	lines := make([]string, len(values))
	for i, v := range values {
		lines[i] = t.Paint(visual.Punctuation, fmt.Sprintf("%-10s", labels[i])) + v
		if width > 0 {
			lines[i] = visual.Truncate(lines[i], width, unicode)
		}
	}
	fmt.Fprint(u.streams.Out, t.Box(host, lines, unicode))
}
