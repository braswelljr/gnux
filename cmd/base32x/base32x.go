// Package base32x wires the base32x command (GNU base32) to its engine. The
// flags and documentation are shared with the other encoding tool
// (internal/codec).
package base32x

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/base32"
	"github.com/braswelljr/gnux/internal/codec"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = codec.Tool{
	Name:     base32.Name,
	GNU:      "base32",
	Encoding: "base32",
	Sibling:  "base64x",
	Example:  "hello",
	Encoded:  "NBSWY3DP",
	New:      base32.New,
}

// Spec describes the base32x tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for base32x.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the base32x flags on fs.
func Register(fs *pflag.FlagSet) { codec.Register(fs) }

// buildOptions folds the parsed flags into base32.Options.
func buildOptions(fs *pflag.FlagSet) (base32.Options, error) { return codec.BuildOptions(fs) }
