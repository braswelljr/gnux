// Package sha224sum implements the engine behind sha224sumx, a GNU-compatible
// implementation of the GNU/UNIX sha224sum command: print or check SHA-224
// (224-bit) message digests. The behavior is core/hashsum, shared with the
// other digest tools; this package fixes the algorithm and the tool name.
package sha224sum

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sha224sumx"

// Options mirrors the GNU sha224sum flag surface. The zero value prints one
// untagged, hex line per file.
type Options = hashsum.Options

// New builds a sha224sum engine reading from and writing to s. Call Run on it
// with the operands.
func New(opts Options, s core.Streams) *hashsum.Sum {
	return hashsum.New(Name, hashsum.SHA224, opts, s)
}
