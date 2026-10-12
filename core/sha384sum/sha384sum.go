// Package sha384sum implements the engine behind sha384sumx, a GNU-compatible
// implementation of the GNU/UNIX sha384sum command: print or check SHA-384
// (384-bit) message digests. The behavior is core/hashsum, shared with the
// other digest tools; this package fixes the algorithm and the tool name.
package sha384sum

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sha384sumx"

// Options mirrors the GNU sha384sum flag surface. The zero value prints one
// untagged, hex line per file.
type Options = hashsum.Options

// New builds a sha384sum engine reading from and writing to s. Call Run on it
// with the operands.
func New(opts Options, s core.Streams) *hashsum.Sum {
	return hashsum.New(Name, hashsum.SHA384, opts, s)
}
