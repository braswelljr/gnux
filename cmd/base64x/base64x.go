// Package base64x wires the base64x command (GNU base64) to its engine. The
// flags and documentation are shared with the other encoding tool
// (internal/codec).
package base64x

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/base64"
	"github.com/braswelljr/gnux/internal/codec"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

var self = codec.Tool{
	Name:     base64.Name,
	GNU:      "base64",
	Encoding: "base64",
	Sibling:  "base32x",
	Example:  "hello",
	Encoded:  "aGVsbG8=",
	New:      base64.New,
}

// Spec describes the base64x tool.
func Spec() tool.Spec { return self.Spec() }

// Meta returns the documentation metadata for base64x.
func Meta() docmeta.Meta { return self.Meta() }

// Register installs the base64x flags on fs.
func Register(fs *pflag.FlagSet) { codec.Register(fs) }

// buildOptions folds the parsed flags into base64.Options.
func buildOptions(fs *pflag.FlagSet) (base64.Options, error) { return codec.BuildOptions(fs) }
