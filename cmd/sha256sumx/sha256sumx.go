// Package sha256sumx wires the sha256sumx command (GNU sha256sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package sha256sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sha256sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      sha256sum.Name,
	GNU:       "sha256sum",
	Algorithm: "SHA-256",
	Standard:  "FIPS 180-4",
	Bits:      256,
	New:       sha256sum.New,
}

// Spec describes the sha256sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for sha256sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the sha256sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into sha256sum.Options.
func buildOptions(fs *pflag.FlagSet) (sha256sum.Options, error) { return self.BuildOptions(fs) }
