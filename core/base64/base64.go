// Package base64 implements the engine behind base64x, a GNU-compatible
// implementation of the GNU/UNIX base64 command: base64 encode or decode a
// file or standard input (RFC 4648). The behavior is core/basenc, shared with
// the other encoding tool; this package fixes the alphabet and the tool name.
package base64

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basenc"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "base64x"

// Options mirrors the GNU base64 flag surface. The zero value encodes,
// wrapping lines at 76 columns.
type Options = basenc.Options

// New builds a base64 engine reading from and writing to s. Call Run on it
// with at most one operand.
func New(opts Options, s core.Streams) *basenc.Coder {
	return basenc.New(Name, basenc.Base64, opts, s)
}
