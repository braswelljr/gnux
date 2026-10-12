// Package md5sumx wires the md5sumx command (GNU md5sum) to its engine. The flags
// and documentation are shared with the other digest tools (internal/digest).
package md5sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/md5sum"
	"github.com/braswelljr/gnux/internal/digest"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = digest.Tool{
	Name:      md5sum.Name,
	GNU:       "md5sum",
	Algorithm: "MD5",
	Standard:  "RFC 1321",
	Bits:      128,
	Weak:      true,
	New:       md5sum.New,
}

// Spec describes the md5sumx tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for md5sumx.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the md5sumx flags on fs.
func Register(fs *pflag.FlagSet) { self.Register(fs) }

// buildOptions folds the parsed flags into md5sum.Options.
func buildOptions(fs *pflag.FlagSet) (md5sum.Options, error) { return self.BuildOptions(fs) }
