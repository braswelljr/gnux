// Package sha512sumx wires the sha512sumx command (GNU sha512sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package sha512sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sha512sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      sha512sum.Name,
	GNU:       "sha512sum",
	Algorithm: "SHA-512",
	Standard:  "FIPS 180-4",
	Bits:      512,
	New:       sha512sum.New,
}

// Spec describes the sha512sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for sha512sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the sha512sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into sha512sum.Options.
func buildOptions(fs *pflag.FlagSet) (sha512sum.Options, error) { return self.BuildOptions(fs) }
