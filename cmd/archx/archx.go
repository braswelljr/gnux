// Package archx wires the archx command (GNU arch) to its engine.
package archx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/arch"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the archx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "arch",
		Run:  run,
	}
}

// Meta returns the documentation metadata for archx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    arch.Name,
		SeeAlso: []string{"unamex", "nprocx", "hostnamex"},
		Use:     arch.Name + " [OPTION]...",
		Short:   "Print machine architecture.",
		Long: "archx prints the machine hardware name, exactly as unamex -m does, and takes no " +
			"operands. The name comes from the operating system: x86_64 or aarch64 on Linux, " +
			"x86_64 or arm64 on macOS, amd64 or arm64 on FreeBSD. On Windows, and in the " +
			"browser playground, gnux names the architecture it was built for the way Linux " +
			"does (x86_64, aarch64, wasm32).",
		Permissions: "Needs no permissions. Reads only system information.",
		UseCases: []string{
			"Picking the right binary to download in an install script.",
			"Recording the machine's architecture in a CI log or a bug report.",
			"Skipping a step that only works on one architecture.",
			"Naming build output after the architecture it was built on.",
			"Checking whether a shell runs under emulation (Rosetta, QEMU).",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the machine architecture:",
				Bash:        "archx",
				PowerShell:  "archx",
				Output:      "x86_64",
			},
			{
				Description: "Print the same name through unamex:",
				Bash:        "unamex -m",
				PowerShell:  "unamex -m",
			},
			{
				Description: "Pick a download for this machine:",
				Bash:        "curl -LO \"https://example.com/tool-$(archx).tar.gz\"",
				PowerShell:  "curl.exe -LO \"https://example.com/tool-$(archx).zip\"",
			},
			{
				Description: "Run a step only on 64-bit ARM:",
				Bash:        "case \"$(archx)\" in aarch64|arm64) echox 'ARM build' ;; esac",
				PowerShell:  "if ((archx) -in 'aarch64','arm64') { 'ARM build' }",
			},
			{
				Description: "Name an output directory after the architecture:",
				Bash:        "mkdirx -p \"build/$(archx)\"",
				PowerShell:  "mkdirx -p \"build/$(archx)\"",
			},
			{
				Description: "Record the system in a log:",
				Bash:        "echox \"$(unamex -s) $(archx)\" >> build.log",
				PowerShell:  "\"$(unamex -s) $(archx)\" >> build.log",
			},
			{
				Description: "Save the architecture in a variable:",
				Bash:        "arch=$(archx)",
				PowerShell:  "$arch = archx",
			},
			{
				Description: "Print the architecture and the number of processors:",
				Bash:        "echox \"$(archx), $(nprocx) cores\"",
				PowerShell:  "\"$(archx), $(nprocx) cores\"",
			},
			{
				Description: "Compare with the expected architecture:",
				Bash:        "[ \"$(archx)\" = x86_64 ] || echox 'not x86_64'",
				PowerShell:  "if ((archx) -ne 'x86_64') { 'not x86_64' }",
			},
			{
				Description: "Branch on the architecture in the portable gnux shell:",
				Bash:        "gnux shell -c 'case $(archx) in x86_64) echox intel ;; *) echox other ;; esac'",
				PowerShell:  "gnux shell -c 'case $(archx) in x86_64) echox intel ;; *) echox other ;; esac'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return arch.New(arch.Options{}, s).Run(args)
}
