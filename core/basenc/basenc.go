// Package basenc is the engine shared by base64x and base32x: encode a file
// or standard input in base64 or base32 (RFC 4648), or decode it, as GNU
// base64 and base32 do. Each tool's own package (core/base64, core/base32)
// fixes the encoding and the tool name.
package basenc

import (
	"bufio"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Encoding selects the alphabet.
type Encoding int

const (
	// Base64 is RFC 4648 base64 with padding.
	Base64 Encoding = iota
	// Base32 is RFC 4648 base32 with padding.
	Base32
)

// DefaultWrap is the line length of encoded output unless Options says
// otherwise.
const DefaultWrap = 76

// Options mirrors the GNU base64/base32 flag surface. The zero value encodes,
// wrapping lines at 76 columns.
type Options struct {
	// Decode (-d) decodes rather than encodes.
	Decode bool
	// IgnoreGarbage (-i) skips characters outside the alphabet when decoding.
	IgnoreGarbage bool
	// Wrap (-w COLS) is the encoded line length; 0 means DefaultWrap.
	Wrap int
	// NoWrap (-w 0) writes the encoding as one line, with no newline at all.
	NoWrap bool
}

// Coder is the engine. Construct it with New.
type Coder struct {
	Options

	prog    string
	enc     Encoding
	streams core.Streams
	diag    *core.Diag
}

// New builds an engine for enc whose diagnostics are prefixed with prog.
func New(prog string, enc Encoding, opts Options, s core.Streams) *Coder {
	return &Coder{Options: opts, prog: prog, enc: enc, streams: s, diag: core.NewDiag(prog, s)}
}

// errInvalid is reported as GNU's "invalid input".
var errInvalid = errors.New("invalid input")

// Run encodes or decodes its operand, at most one FILE; none or "-" is
// standard input. It returns core.ErrFailed when the file cannot be read or,
// when decoding, the input is not valid; the output decoded up to that point
// is still written.
func (c *Coder) Run(args []string) error {
	if len(args) > 1 {
		return core.Usagef("extra operand '%s'", args[1])
	}
	name := "-"
	if len(args) == 1 {
		name = args[0]
	}
	r := c.streams.In
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			c.diag.Errorf("%s: %s", name, core.ErrText(err))
			return core.ErrFailed
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			c.diag.Errorf("read error: %s", core.ErrText(syscall.EISDIR))
			return core.ErrFailed
		}
		r = f
	}

	out := core.NewWriter(c.streams.Out)
	var err error
	if c.Decode {
		err = c.decode(r, out)
	} else {
		err = c.encode(r, out)
	}
	_ = out.Flush()
	switch {
	case errors.Is(err, errInvalid):
		c.diag.Errorf("invalid input")
	case err != nil:
		c.diag.Errorf("read error: %s", core.ErrText(err))
	}
	return c.diag.Result()
}

func (c *Coder) quantum() (raw, encoded int) {
	if c.enc == Base32 {
		return 5, 8
	}
	return 3, 4
}

func (c *Coder) encoder() interface {
	EncodeToString([]byte) string
	Decode(dst, src []byte) (int, error)
	DecodedLen(int) int
} {
	if c.enc == Base32 {
		return base32.StdEncoding
	}
	return base64.StdEncoding
}

func (c *Coder) encode(r io.Reader, out *bufio.Writer) error {
	// A whole number of quanta per chunk keeps padding to the very end.
	buf := make([]byte, 3*5*4096)
	cols := c.Wrap
	if cols <= 0 {
		cols = DefaultWrap
	}
	col := 0
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			s := c.encoder().EncodeToString(buf[:n])
			if c.NoWrap {
				_, _ = out.WriteString(s)
			} else {
				for s != "" {
					k := min(cols-col, len(s))
					_, _ = out.WriteString(s[:k])
					s, col = s[k:], col+k
					if col == cols {
						_ = out.WriteByte('\n')
						col = 0
					}
				}
			}
		}
		if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	if !c.NoWrap && col > 0 {
		_ = out.WriteByte('\n')
	}
	return nil
}

func (c *Coder) inAlphabet(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z', b == '=':
		return true
	case c.enc == Base32:
		return b >= '2' && b <= '7'
	default:
		return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '+' || b == '/'
	}
}

// decode reads quantum by quantum, so concatenated encodings (each with its
// own padding) decode as GNU's do. Newlines are always ignored; anything
// else outside the alphabet is invalid unless IgnoreGarbage is set.
func (c *Coder) decode(r io.Reader, out *bufio.Writer) error {
	_, size := c.quantum()
	enc := c.encoder()
	q := make([]byte, 0, size)
	dst := make([]byte, enc.DecodedLen(size))
	br := core.NewReader(r)
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if b == '\n' {
			continue
		}
		if !c.inAlphabet(b) {
			if c.IgnoreGarbage {
				continue
			}
			return errInvalid
		}
		q = append(q, b)
		if len(q) < size {
			continue
		}
		n, derr := enc.Decode(dst, q)
		_, _ = out.Write(dst[:n])
		if derr != nil {
			return errInvalid
		}
		q = q[:0]
	}
	if len(q) == 0 {
		return nil
	}
	// A final quantum without its padding: decode what it holds, then report
	// the input as invalid, as GNU does.
	if c.enc == Base32 {
		n, _ := base32.StdEncoding.WithPadding(base32.NoPadding).Decode(dst, q)
		_, _ = out.Write(dst[:n])
	} else {
		n, _ := base64.RawStdEncoding.Decode(dst, q)
		_, _ = out.Write(dst[:n])
	}
	return errInvalid
}
