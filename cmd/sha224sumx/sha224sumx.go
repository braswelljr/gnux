// Package sha224sumx wires the sha224sumx command (GNU sha224sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package sha224sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sha224sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      sha224sum.Name,
	GNU:       "sha224sum",
	Algorithm: "SHA-224",
	Standard:  "FIPS 180-4",
	Bits:      224,
	New:       sha224sum.New,
}

// Spec describes the sha224sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for sha224sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the sha224sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into sha224sum.Options.
func buildOptions(fs *pflag.FlagSet) (sha224sum.Options, error) { return self.BuildOptions(fs) }
