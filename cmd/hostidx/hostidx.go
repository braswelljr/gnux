// Package hostidx wires the hostidx command (GNU hostid) to its engine.
package hostidx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hostid"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the hostidx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "hostid",
		Run:  run,
	}
}

// Meta returns the documentation metadata for hostidx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    hostid.Name,
		SeeAlso: []string{"hostnamex", "unamex"},
		Use:     hostid.Name + " [OPTION]",
		Short:   "Print the numeric identifier for the current host.",
		Long: "hostidx prints the host's 32-bit identifier in hexadecimal, eight digits wide, " +
			"as gethostid(3) returns it. On Linux that is the contents of /etc/hostid when the " +
			"file exists, and otherwise a value glibc derives from the first IPv4 address the " +
			"host name resolves to. On macOS and the BSDs it is the kern.hostid sysctl, which " +
			"may be zero. Windows has no gethostid, so hostidx derives the value from the " +
			"host's address there too, as glibc does; with no address it prints 00000000. " +
			"The identifier is not guaranteed to be unique.",
		Permissions: "Reads /etc/hostid on Linux when it exists. Needs no permissions.",
		UseCases: []string{
			"Reporting the identifier that some license managers bind to.",
			"Telling machines apart in a log when the hostname is not enough.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the host identifier a license server asks for:",
				Bash:        "hostidx",
				PowerShell:  "hostidx",
				Output:      "a8c002d7",
			},
			{
				Description: "Record the host name and identifier together in a log line:",
				Bash:        "echox \"$(hostnamex) $(hostidx)\" >> machines.log",
				PowerShell:  "echox \"$(hostnamex) $(hostidx)\" >> machines.log",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return hostid.New(hostid.Options{}, s).Run(args)
}
