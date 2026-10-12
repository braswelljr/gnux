// Package b2sum implements the engine behind b2sumx, a GNU-compatible
// implementation of the GNU/UNIX b2sum command: print or check BLAKE2b
// (512-bit by default) message digests. The behavior is core/hashsum, shared with the
// other digest tools; this package fixes the algorithm and the tool name.
package b2sum

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "b2sumx"

// Options mirrors the GNU b2sum flag surface. The zero value prints one
// untagged, hex line per file.
type Options = hashsum.Options

// New builds a b2sum engine reading from and writing to s. Call Run on it
// with the operands.
func New(opts Options, s core.Streams) *hashsum.Sum {
	return hashsum.New(Name, hashsum.BLAKE2b, opts, s)
}
