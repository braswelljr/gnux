// Package sha1sumx wires the sha1sumx command (GNU sha1sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package sha1sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/sha1sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      sha1sum.Name,
	GNU:       "sha1sum",
	Algorithm: "SHA-1",
	Standard:  "FIPS 180-4",
	Bits:      160,
	Weak:      true,
	New:       sha1sum.New,
}

// Spec describes the sha1sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for sha1sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the sha1sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into sha1sum.Options.
func buildOptions(fs *pflag.FlagSet) (sha1sum.Options, error) { return self.BuildOptions(fs) }
