// Package b2sumx wires the b2sumx command (GNU b2sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package b2sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/b2sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      b2sum.Name,
	GNU:       "b2sum",
	Algorithm: "BLAKE2b",
	Standard:  "RFC 7693",
	Bits:      512,
	Length:    true,
	New:       b2sum.New,
}

// Spec describes the b2sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for b2sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the b2sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into b2sum.Options.
func buildOptions(fs *pflag.FlagSet) (b2sum.Options, error) { return self.BuildOptions(fs) }
