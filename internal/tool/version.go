package tool

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/internal/common"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// registerStandard installs the two flags every GNU tool accepts. --help is
// defined here (without a shorthand) so cobra does not claim -h, which several
// tools use for something else (ls -h, for one).
func registerStandard(fs *pflag.FlagSet) {
	fs.Bool("help", false, "display this help and exit")
	fs.Bool("version", false, "output version information and exit")
}

// PrintVersion writes the version line for the tool called name to w.
func PrintVersion(w io.Writer, name string) {
	if name == common.AppName || strings.HasPrefix(name, common.AppName+" ") {
		fmt.Fprintf(w, "%s %s\n", name, common.Version)
		return
	}
	fmt.Fprintf(w, "%s (%s) %s\n", name, common.AppName, common.Version)
}

// VersionTopic returns the per-flag documentation topic for --version, for
// tools that document their flags individually.
func VersionTopic(name string) Topic {
	return Topic{
		Meta: docmeta.Meta{
			Name:  "version",
			Use:   name + " --version",
			Short: "Print version information and exit (--version).",
			Long: "--version prints the " + name + " version string (injected at build time) and " +
				"exits without doing anything else.",
			Permissions: "Performs no filesystem access and needs no permissions; nothing is " +
				"touched, even when operands are also supplied.",
			UseCases: []string{
				"Confirming which " + name + " build is installed.",
				"Capturing the version in CI logs or bug reports.",
			},
			Examples: []docmeta.Example{
				{
					Description: "Print the " + name + " version:",
					Bash:        name + " --version",
					PowerShell:  name + " --version",
				},
			},
		},
		Register: func(fs *pflag.FlagSet) {
			fs.Bool("version", false, "output version information and exit")
		},
	}
}
