// Package sha512sum implements the engine behind sha512sumx, a GNU-compatible
// implementation of the GNU/UNIX sha512sum command: print or check SHA-512
// (512-bit) message digests. The behavior is core/hashsum, shared with the
// other digest tools; this package fixes the algorithm and the tool name.
package sha512sum

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sha512sumx"

// Options mirrors the GNU sha512sum flag surface. The zero value prints one
// untagged, hex line per file.
type Options = hashsum.Options

// New builds a sha512sum engine reading from and writing to s. Call Run on it
// with the operands.
func New(opts Options, s core.Streams) *hashsum.Sum {
	return hashsum.New(Name, hashsum.SHA512, opts, s)
}
