// Package sha384sumx wires the sha384sumx command (GNU sha384sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package sha384sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sha384sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      sha384sum.Name,
	GNU:       "sha384sum",
	Algorithm: "SHA-384",
	Standard:  "FIPS 180-4",
	Bits:      384,
	New:       sha384sum.New,
}

// Spec describes the sha384sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for sha384sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the sha384sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into sha384sum.Options.
func buildOptions(fs *pflag.FlagSet) (sha384sum.Options, error) { return self.BuildOptions(fs) }
