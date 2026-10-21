// Package basenc implements the engine behind basencx, a GNU-compatible
// implementation of the GNU/UNIX basenc command: encode a file or standard
// input in one of several encodings (RFC 4648 base64, base64url, base32,
// base32hex and base16, a bit string either way round, and Z85), or decode
// it. base64x and base32x are this engine with the encoding fixed: their own
// packages (core/base64, core/base32) set it and the tool name.
package basenc

import (
	"bufio"
	"errors"
	"io"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name of basencx, used as the prefix of its
// diagnostics. base64x and base32x pass their own.
const Name = "basencx"

// Encoding selects the alphabet.
type Encoding int

const (
	// Base64 is RFC 4648 base64 with padding (--base64).
	Base64 Encoding = iota
	// Base32 is RFC 4648 base32 with padding (--base32).
	Base32
	// Base64URL is RFC 4648's file- and URL-safe base64, with padding
	// (--base64url).
	Base64URL
	// Base32Hex is RFC 4648's base32 with the extended hex alphabet
	// (--base32hex).
	Base32Hex
	// Base16 is upper-case hexadecimal (--base16); decoding takes either case.
	Base16
	// Base2MSBF writes each byte as eight '0' and '1' characters, most
	// significant bit first (--base2msbf).
	Base2MSBF
	// Base2LSBF is Base2MSBF with the least significant bit first
	// (--base2lsbf).
	Base2LSBF
	// Z85 is ZeroMQ's Z85 (spec 32/Z85): four bytes to five characters. The
	// input must be a whole number of groups (--z85).
	Z85
)

// DefaultWrap is the line length of encoded output unless Options says
// otherwise.
const DefaultWrap = 76

// Options mirrors the GNU basenc/base64/base32 flag surface. The zero value
// encodes, wrapping lines at 76 columns.
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

// errInvalid is reported as GNU's "invalid input", and errZ85Length as the
// message GNU gives Z85 input that is not a whole number of groups.
var (
	errInvalid   = errors.New("invalid input")
	errZ85Length = errors.New("invalid z85 length")
)

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
		f, err := fsys.Open(name)
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
	switch {
	case c.Decode:
		err = c.decode(r, out)
	default:
		err = c.encode(r, out)
	}
	_ = out.Flush()
	switch {
	case errors.Is(err, errInvalid):
		c.diag.Errorf("invalid input")
	case errors.Is(err, errZ85Length):
		c.diag.Errorf("invalid input (length must be multiple of 4 characters)")
	case err != nil:
		c.diag.Errorf("read error: %s", core.ErrText(err))
	}
	return c.diag.Result()
}

// chunk is how much input is encoded at a time: a whole number of every
// codec's input group, so that padding only ever comes at the very end.
const chunk = 3 * 5 * 4 * 4096

func (c *Coder) encode(r io.Reader, out *bufio.Writer) error {
	cd := codecs[c.enc]
	buf := make([]byte, chunk)
	var text []byte
	cols := c.Wrap
	if cols <= 0 {
		cols = DefaultWrap
	}
	col := 0
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			if c.enc == Z85 && n%4 != 0 {
				return errZ85Length
			}
			text = cd.encode(text[:0], buf[:n])
			if c.NoWrap {
				_, _ = out.Write(text)
			} else {
				s := text
				for len(s) > 0 {
					k := min(cols-col, len(s))
					_, _ = out.Write(s[:k])
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

// decode reads group by group, so concatenated encodings (each with its own
// padding) decode as GNU's do. Newlines are always ignored; anything else
// outside the alphabet is invalid unless IgnoreGarbage is set.
func (c *Coder) decode(r io.Reader, out *bufio.Writer) error {
	cd := codecs[c.enc]
	q := make([]byte, 0, cd.group)
	dst := make([]byte, 8)
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
		if !cd.alphabet(b) {
			if c.IgnoreGarbage {
				continue
			}
			return errInvalid
		}
		q = append(q, b)
		if len(q) < cd.group {
			continue
		}
		n, ok := cd.decode(dst, q)
		_, _ = out.Write(dst[:n])
		if !ok {
			return errInvalid
		}
		q = q[:0]
	}
	if len(q) == 0 {
		return nil
	}
	// A final group cut short: base64 and base32 decode what it holds, as
	// GNU does, and the input is invalid either way.
	if cd.partial != nil {
		_, _ = out.Write(dst[:cd.partial(dst, q)])
	}
	return errInvalid
}
