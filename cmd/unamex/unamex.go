// Package unamex wires the unamex command (GNU uname) to its engine.
package unamex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uname"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the unamex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "uname",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for unamex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    uname.Name,
		SeeAlso: []string{"hostnamex", "nprocx", "uptimex"},
		Use:     uname.Name + " [OPTION]...",
		Short:   "Print system information.",
		Long: "unamex prints information about the machine and operating system. With no " +
			"option it prints the kernel name, as -s does. The selected fields are printed " +
			"on one line, separated by spaces, in a fixed order whatever the order of the " +
			"options: kernel name (-s), network node host name (-n), kernel release (-r), " +
			"kernel version (-v), machine hardware name (-m), processor type (-p), hardware " +
			"platform (-i) and operating system (-o). -a prints them all, leaving out -p and " +
			"-i when they are unknown, as GNU does. The values come from uname(2) on Linux, " +
			"sysctl on macOS and FreeBSD, and the registry on Windows, so unamex prints the " +
			"same fields on every platform: on Windows the kernel name is 'Windows_NT' and " +
			"the operating system 'MS/Windows'; in the browser playground the fields describe " +
			"the WebAssembly runtime.",
		Permissions: "Needs no permissions.",
		UseCases: []string{
			"Branching a portable script on the operating system or architecture.",
			"Recording the platform in bug reports and build logs.",
			"Picking the right download for the machine's architecture with -m.",
			"Checking the kernel release before relying on a kernel feature.",
			"Getting the same answer on Linux, macOS, FreeBSD and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the kernel name:",
				Bash:        "unamex",
				PowerShell:  "unamex",
			},
			{
				Description: "Print everything:",
				Bash:        "unamex -a",
				PowerShell:  "unamex -a",
			},
			{
				Description: "Print the machine architecture:",
				Bash:        "unamex -m",
				PowerShell:  "unamex -m",
			},
			{
				Description: "Print the kernel release:",
				Bash:        "unamex -r",
				PowerShell:  "unamex -r",
			},
			{
				Description: "Print the operating system:",
				Bash:        "unamex -o",
				PowerShell:  "unamex -o",
			},
			{
				Description: "Print the kernel name and release:",
				Bash:        "unamex -sr",
				PowerShell:  "unamex -sr",
			},
			{
				Description: "Print the host name:",
				Bash:        "unamex -n",
				PowerShell:  "unamex -n",
			},
			{
				Description: "Branch a script on the operating system:",
				Bash:        "case \"$(unamex -s)\" in Linux) echox linux ;; Darwin) echox macos ;; esac",
				PowerShell:  "switch (unamex -s) { 'Linux' { 'linux' } 'Darwin' { 'macos' } 'Windows_NT' { 'windows' } }",
			},
			{
				Description: "Pick a download for this machine:",
				Bash:        "curl -LO \"https://example.com/tool-$(unamex -s)-$(unamex -m).tar.gz\"",
				PowerShell:  "Invoke-WebRequest \"https://example.com/tool-$(unamex -s)-$(unamex -m).zip\" -OutFile tool.zip",
			},
			{
				Description: "Record the platform in a log:",
				Bash:        "unamex -a >> build.log",
				PowerShell:  "unamex -a >> build.log",
			},
			{
				Description: "Print system information in the portable gnux shell:",
				Bash:        "gnux shell -c 'unamex -srm'",
				PowerShell:  "gnux shell -c 'unamex -srm'",
			},
		},
	}
}

// Register installs the unamex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("all", "a", false, "print all information, in the following order, except omit -p and -i if unknown")
	fs.BoolP("kernel-name", "s", false, "print the kernel name")
	fs.BoolP("nodename", "n", false, "print the network node hostname")
	fs.BoolP("kernel-release", "r", false, "print the kernel release")
	fs.BoolP("kernel-version", "v", false, "print the kernel version")
	fs.BoolP("machine", "m", false, "print the machine hardware name")
	fs.BoolP("processor", "p", false, "print the processor type (non-portable)")
	fs.BoolP("hardware-platform", "i", false, "print the hardware platform (non-portable)")
	fs.BoolP("operating-system", "o", false, "print the operating system")
}

// buildOptions folds the parsed flags into uname.Options.
func buildOptions(fs *pflag.FlagSet) uname.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	return uname.Options{
		All:              on("all"),
		KernelName:       on("kernel-name"),
		Nodename:         on("nodename"),
		KernelRelease:    on("kernel-release"),
		KernelVersion:    on("kernel-version"),
		Machine:          on("machine"),
		Processor:        on("processor"),
		HardwarePlatform: on("hardware-platform"),
		OperatingSystem:  on("operating-system"),
	}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return uname.New(buildOptions(fs), s).Run(args)
}
