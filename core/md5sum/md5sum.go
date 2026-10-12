// Package md5sum implements the engine behind md5sumx, a GNU-compatible
// implementation of the GNU/UNIX md5sum command: print or check MD5
// (128-bit) message digests. The behavior is core/hashsum, shared with the
// other digest tools; this package fixes the algorithm and the tool name.
package md5sum

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "md5sumx"

// Options mirrors the GNU md5sum flag surface. The zero value prints one
// untagged, hex line per file.
type Options = hashsum.Options

// New builds a md5sum engine reading from and writing to s. Call Run on it
// with the operands.
func New(opts Options, s core.Streams) *hashsum.Sum {
	return hashsum.New(Name, hashsum.MD5, opts, s)
}
