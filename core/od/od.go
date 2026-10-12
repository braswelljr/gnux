// Package od implements the engine behind odx, a GNU-compatible
// implementation of the GNU/UNIX od command: dump files in octal and other
// formats.
package od

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "odx"

// Radix is the base of the offsets printed at the start of each line (-A).
type Radix byte

const (
	// RadixOctal prints offsets in octal, seven digits wide (the default).
	RadixOctal Radix = 0
	// RadixDecimal prints offsets in decimal, seven digits wide.
	RadixDecimal Radix = 'd'
	// RadixHex prints offsets in hexadecimal, six digits wide.
	RadixHex Radix = 'x'
	// RadixNone prints no offsets.
	RadixNone Radix = 'n'
)

// Endian is the byte order multi-byte values are read in (--endian).
type Endian int

const (
	// EndianNative reads values in the machine's own order (the default).
	EndianNative Endian = iota
	// EndianLittle reads values least significant byte first.
	EndianLittle
	// EndianBig reads values most significant byte first.
	EndianBig
)

// Options mirrors the GNU od flag surface. The zero value dumps the input as
// two-byte octal words, sixteen bytes per line, with octal offsets.
type Options struct {
	// Radix (-A) selects the offset base.
	Radix Radix
	// Skip (-j) skips this many bytes of the combined input first.
	Skip int64
	// Limit (-N) dumps at most this many bytes when LimitSet is true.
	Limit    int64
	LimitSet bool
	// Strings (-S N), when positive, prints the NUL-terminated strings of at
	// least that many printable characters instead of a dump.
	Strings int
	// Types (-t) are the output formats, one output line each per block, in
	// order: a, c, d[SIZE], f[SIZE], o[SIZE], u[SIZE], x[SIZE], any of them
	// followed by z. Empty means o2.
	Types []string
	// Verbose (-v) prints every line; otherwise a run of lines identical to
	// the one before is shown as a single '*'.
	Verbose bool
	// Width (-w) is the number of bytes per output line; 0 means 16.
	Width int
	// Endian (--endian) is the byte order of multi-byte values.
	Endian Endian
}

// kind is the conversion of one output format.
type kind byte

const (
	kindNamed  kind = 'a' // named characters
	kindChar   kind = 'c' // printable characters or backslash escapes
	kindSigned kind = 'd'
	kindFloat  kind = 'f'
	kindOctal  kind = 'o'
	kindUnsign kind = 'u'
	kindHex    kind = 'x'
)

// format is one parsed -t type.
type format struct {
	kind    kind
	size    int  // bytes per value
	width   int  // characters per printed value
	trailer bool // z: append the printable characters
	pad     int  // extra spaces spread across a line so every format aligns
}

// Od is the engine. Construct it with New.
type Od struct {
	Options

	streams core.Streams
	diag    *core.Diag
	formats []format
	order   binary.ByteOrder
}

// New builds an od engine reading from and writing to s.
func New(opts Options, s core.Streams) *Od {
	return &Od{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// widths of a printed value, by conversion and size.
var widths = map[kind]map[int]int{
	kindOctal:  {1: 3, 2: 6, 4: 11, 8: 22},
	kindUnsign: {1: 3, 2: 5, 4: 10, 8: 20},
	kindSigned: {1: 4, 2: 6, 4: 11, 8: 20},
	kindHex:    {1: 2, 2: 4, 4: 8, 8: 16},
	kindFloat:  {4: 15, 8: 24},
}

// ParseType parses one -t TYPE string, which may hold several formats
// ("x1z", "d2u4"), and returns a usage error in GNU's words for a bad one.
func ParseType(s string) error {
	_, err := parseType(s)
	return err
}

func parseType(s string) ([]format, error) {
	if s == "" {
		return nil, core.Usagef("invalid type string ''")
	}
	var out []format
	rest := s
	for rest != "" {
		f := format{kind: kind(rest[0])}
		rest = rest[1:]
		switch f.kind {
		case kindNamed, kindChar:
			f.size, f.width = 1, 3
		case kindSigned, kindOctal, kindUnsign, kindHex:
			f.size = 4
			if rest != "" {
				switch rest[0] {
				case 'C':
					f.size, rest = 1, rest[1:]
				case 'S':
					f.size, rest = 2, rest[1:]
				case 'I':
					f.size, rest = 4, rest[1:]
				case 'L':
					f.size, rest = 8, rest[1:]
				default:
					n, r, ok := digits(rest)
					if ok {
						if widths[f.kind][n] == 0 {
							return nil, core.Usagef("invalid type string '%s';\nthis system doesn't provide a %d-byte integral type", s, n)
						}
						f.size, rest = n, r
					}
				}
			}
			f.width = widths[f.kind][f.size]
		case kindFloat:
			f.size = 8
			if rest != "" {
				switch rest[0] {
				case 'F':
					f.size, rest = 4, rest[1:]
				case 'D', 'L':
					f.size, rest = 8, rest[1:]
				default:
					n, r, ok := digits(rest)
					if ok {
						if widths[kindFloat][n] == 0 {
							return nil, core.Usagef("invalid type string '%s';\nthis system doesn't provide a %d-byte floating point type", s, n)
						}
						f.size, rest = n, r
					}
				}
			}
			f.width = widths[kindFloat][f.size]
		default:
			return nil, core.Usagef("invalid character '%c' in type string '%s'", f.kind, s)
		}
		if rest != "" && rest[0] == 'z' {
			f.trailer, rest = true, rest[1:]
		}
		out = append(out, f)
	}
	return out, nil
}

func digits(s string) (n int, rest string, ok bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, s, false
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		v = math.MaxInt
	}
	return v, s[i:], true
}

// Run dumps the operands, concatenated; no operands, or "-", is standard
// input. A file that cannot be read is diagnosed and skipped, and the
// returned error is core.ErrFailed.
func (o *Od) Run(args []string) error {
	types := o.Types
	if len(types) == 0 {
		types = []string{"o2"}
	}
	for _, t := range types {
		fs, err := parseType(t)
		if err != nil {
			return err
		}
		o.formats = append(o.formats, fs...)
	}
	switch o.Radix {
	case RadixOctal, RadixDecimal, RadixHex, RadixNone, 'o':
	default:
		o.diag.Errorf("invalid output address radix '%c'; it must be one character from [doxn]", byte(o.Radix))
		return core.ErrFailed
	}

	switch o.Endian {
	case EndianLittle:
		o.order = binary.LittleEndian
	case EndianBig:
		o.order = binary.BigEndian
	default:
		o.order = binary.NativeEndian
	}

	if len(args) == 0 {
		args = []string{"-"}
	}
	in := &concat{names: args, od: o}

	if o.Skip > 0 {
		n, err := io.CopyN(io.Discard, in, o.Skip)
		if n < o.Skip {
			if err != nil && !errors.Is(err, io.EOF) {
				o.diag.Errorf("%s", core.ErrText(err))
			}
			o.diag.Errorf("cannot skip past end of combined input")
			return core.ErrFailed
		}
	}
	var r io.Reader = in
	if o.LimitSet {
		r = io.LimitReader(in, max(o.Limit, 0))
	}

	w := bufio.NewWriter(o.streams.Out)
	var err error
	if o.Strings > 0 {
		err = o.dumpStrings(w, bufio.NewReader(r))
	} else {
		err = o.dump(w, r)
	}
	if ferr := w.Flush(); err == nil {
		err = ferr
	}
	if err != nil && !core.IsClosedPipe(err) {
		o.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return o.diag.Result()
}

// lcm of every format's size: a line must hold a whole number of each.
func (o *Od) lcm() int {
	l := 1
	for _, f := range o.formats {
		a, b := l, f.size
		for b != 0 {
			a, b = b, a%b
		}
		l = l / a * f.size
	}
	return l
}

func (o *Od) blockSize() int {
	l := o.lcm()
	if o.Width == 0 {
		if l < 16 {
			return l * (16 / l)
		}
		return l
	}
	if o.Width < 0 || o.Width%l != 0 {
		o.diag.Warnf("warning: invalid width %d; using %d instead", o.Width, l)
		return l
	}
	return o.Width
}

func (o *Od) dump(w *bufio.Writer, r io.Reader) error {
	block := o.blockSize()

	// Spread padding so every format's fields end at the same column.
	lineWidth := 0
	for _, f := range o.formats {
		lineWidth = max(lineWidth, (f.width+1)*(block/f.size))
	}
	for i := range o.formats {
		f := &o.formats[i]
		f.pad = lineWidth - (f.width+1)*(block/f.size)
	}

	buf := make([]byte, block)
	var prev []byte
	addr := o.Skip
	starred := false
	for {
		n, err := io.ReadFull(r, buf)
		if n == 0 {
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return err
			}
			break
		}
		data := buf[:n]
		if !o.Verbose && n == block && prev != nil && bytes.Equal(prev, data) {
			if !starred {
				if _, err := w.WriteString("*\n"); err != nil {
					return err
				}
				starred = true
			}
			addr += int64(n)
			continue
		}
		starred = false
		if err := o.line(w, addr, data, block); err != nil {
			return err
		}
		prev = append(prev[:0], data...)
		addr += int64(n)
		if n < block {
			break
		}
	}
	if o.Radix != RadixNone {
		if _, err := w.WriteString(o.address(addr) + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (o *Od) address(a int64) string {
	switch o.Radix {
	case RadixDecimal:
		return fmt.Sprintf("%07d", a)
	case RadixHex:
		return fmt.Sprintf("%06x", a)
	case RadixNone:
		return ""
	default:
		return fmt.Sprintf("%07o", a)
	}
}

// line prints one block: a line per format, the first prefixed with the
// offset and the others with as many spaces.
func (o *Od) line(w *bufio.Writer, addr int64, data []byte, block int) error {
	// A partial last block is padded with zero bytes up to a whole value.
	padded := make([]byte, block)
	copy(padded, data)
	n := len(data)

	lead := o.address(addr)
	for i, f := range o.formats {
		if i == 0 {
			_, _ = w.WriteString(lead)
		} else {
			_, _ = w.WriteString(strings.Repeat(" ", len(lead)))
		}
		fields := block / f.size
		blank := (block - n) / f.size
		padLeft := f.pad
		for k := fields; k > blank; k-- {
			idx := fields - k
			next := f.pad * (k - 1) / fields
			adj := padLeft - next + f.width
			padLeft = next
			v := o.value(f, padded[idx*f.size:(idx+1)*f.size])
			_ = w.WriteByte(' ')
			_, _ = w.WriteString(strings.Repeat(" ", max(adj-len(v), 0)))
			_, _ = w.WriteString(v)
		}
		if f.trailer {
			// Space out the missing fields so the trailer lines up.
			_, _ = w.WriteString(strings.Repeat(" ", blank*(f.width+1)+f.pad*blank/fields))
			_, _ = w.WriteString("  >")
			for _, c := range data {
				if c >= 0x20 && c < 0x7f {
					_ = w.WriteByte(c)
				} else {
					_ = w.WriteByte('.')
				}
			}
			_ = w.WriteByte('<')
		}
		if err := w.WriteByte('\n'); err != nil {
			return err
		}
	}
	return nil
}

var named = [...]string{
	"nul", "soh", "stx", "etx", "eot", "enq", "ack", "bel",
	"bs", "ht", "nl", "vt", "ff", "cr", "so", "si",
	"dle", "dc1", "dc2", "dc3", "dc4", "nak", "syn", "etb",
	"can", "em", "sub", "esc", "fs", "gs", "rs", "us",
	"sp",
}

// value formats one value, unpadded except for the zero-filled octal and
// hexadecimal conversions.
func (o *Od) value(f format, b []byte) string {
	switch f.kind {
	case kindNamed:
		c := b[0] & 0x7f
		switch {
		case int(c) < len(named):
			return named[c]
		case c == 0x7f:
			return "del"
		default:
			return string(rune(c))
		}
	case kindChar:
		switch c := b[0]; c {
		case 0:
			return `\0`
		case '\a':
			return `\a`
		case '\b':
			return `\b`
		case '\f':
			return `\f`
		case '\n':
			return `\n`
		case '\r':
			return `\r`
		case '\t':
			return `\t`
		case '\v':
			return `\v`
		default:
			if c >= 0x20 && c < 0x7f {
				return string(rune(c))
			}
			return fmt.Sprintf("%03o", c)
		}
	case kindFloat:
		if f.size == 4 {
			return formatFloat(float64(math.Float32frombits(o.order.Uint32(b))), 32)
		}
		return formatFloat(math.Float64frombits(o.order.Uint64(b)), 64)
	}

	var u uint64
	switch f.size {
	case 1:
		u = uint64(b[0])
	case 2:
		u = uint64(o.order.Uint16(b))
	case 4:
		u = uint64(o.order.Uint32(b))
	default:
		u = o.order.Uint64(b)
	}
	switch f.kind {
	case kindOctal:
		return fmt.Sprintf("%0*o", f.width, u)
	case kindHex:
		return fmt.Sprintf("%0*x", f.width, u)
	case kindUnsign:
		return strconv.FormatUint(u, 10)
	default:
		var v int64
		switch f.size {
		case 1:
			v = int64(int8(u))
		case 2:
			v = int64(int16(u))
		case 4:
			v = int64(int32(u))
		default:
			v = int64(u)
		}
		return strconv.FormatInt(v, 10)
	}
}

// formatFloat prints x as C's "%.*g" does at the smallest precision, from
// FLT_DIG or DBL_DIG up, that reads back as the same value: the way GNU od
// prints floating-point values.
func formatFloat(x float64, bits int) string {
	switch {
	case math.IsNaN(x):
		if math.Signbit(x) {
			return "-nan"
		}
		return "nan"
	case math.IsInf(x, 1):
		return "inf"
	case math.IsInf(x, -1):
		return "-inf"
	}
	prec := 6
	if bits == 64 {
		prec = 15
	}
	for ; prec < 40; prec++ {
		s := strconv.FormatFloat(x, 'g', prec, bits)
		if v, err := strconv.ParseFloat(s, bits); err == nil && v == x {
			return s
		}
	}
	return strconv.FormatFloat(x, 'g', -1, bits)
}

// dumpStrings prints each run of at least Strings printable characters that
// ends in a NUL byte, preceded by its offset.
func (o *Od) dumpStrings(w *bufio.Writer, r *bufio.Reader) error {
	addr := o.Skip
	var run []byte
	start := addr
	for {
		c, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		switch {
		case c == 0:
			if len(run) >= o.Strings {
				if o.Radix != RadixNone {
					_, _ = w.WriteString(o.address(start) + " ")
				}
				for _, b := range run {
					writeEscaped(w, b)
				}
				if err := w.WriteByte('\n'); err != nil {
					return err
				}
			}
			run = run[:0]
		case isPrint(c):
			if len(run) == 0 {
				start = addr
			}
			run = append(run, c)
		default:
			run = run[:0]
		}
		addr++
	}
	if o.Radix != RadixNone {
		_, err := w.WriteString(o.address(addr) + "\n")
		return err
	}
	return nil
}

// isPrint is C's isprint in the C locale, plus the whitespace escapes GNU od
// writes with a backslash.
func isPrint(c byte) bool {
	return (c >= 0x20 && c < 0x7f) || c == '\a' || c == '\b' || c == '\f' || c == '\n' || c == '\r' || c == '\t' || c == '\v'
}

func writeEscaped(w *bufio.Writer, c byte) {
	switch c {
	case '\a':
		_, _ = w.WriteString(`\a`)
	case '\b':
		_, _ = w.WriteString(`\b`)
	case '\f':
		_, _ = w.WriteString(`\f`)
	case '\n':
		_, _ = w.WriteString(`\n`)
	case '\r':
		_, _ = w.WriteString(`\r`)
	case '\t':
		_, _ = w.WriteString(`\t`)
	case '\v':
		_, _ = w.WriteString(`\v`)
	default:
		_ = w.WriteByte(c)
	}
}

// concat reads the operands one after another. An operand that cannot be
// opened or read is diagnosed and skipped, as GNU od does.
type concat struct {
	names []string
	od    *Od
	cur   io.Reader
	file  *os.File

	curName string
}

func (c *concat) Read(p []byte) (int, error) {
	for {
		if c.cur == nil {
			if len(c.names) == 0 {
				return 0, io.EOF
			}
			name := c.names[0]
			c.names = c.names[1:]
			if name == "-" {
				c.cur = c.od.streams.In
			} else {
				f, err := os.Open(name)
				if err != nil {
					c.od.diag.Errorf("%s: %s", name, core.ErrText(err))
					continue
				}
				c.file, c.cur = f, f
			}
			c.curName = name
		}
		n, err := c.cur.Read(p)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				c.od.diag.Errorf("%s: read error: %s", c.curName, core.ErrText(err))
			}
			if c.file != nil {
				c.file.Close()
				c.file = nil
			}
			c.cur = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, nil
	}
}

// ParseBytes parses a -j or -N BYTES operand as GNU od does: decimal, octal
// with a leading 0 or hexadecimal with 0x, then an optional multiplier suffix
// (b 512, k/K 1024, kB 1000, M, MB, G, ... ).
func ParseBytes(s string) (int64, error) {
	bad := fmt.Errorf("invalid argument '%s'", s)
	base, i := 10, 0
	switch {
	case strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X"):
		base, i = 16, 2
	case len(s) > 1 && s[0] == '0':
		base, i = 8, 1
	}
	j := i
	for j < len(s) && strings.IndexByte("0123456789abcdefABCDEF"[:digitsFor(base)], s[j]) >= 0 {
		j++
	}
	if j == i && base != 8 {
		return 0, bad
	}
	var v uint64
	if j > i {
		var err error
		v, err = strconv.ParseUint(s[i:j], base, 64)
		if err != nil {
			return math.MaxInt64, nil
		}
	}
	mult, ok := multiplier(s[j:])
	if !ok {
		return 0, bad
	}
	if v != 0 && (mult == 0 || v > math.MaxInt64/mult) {
		return math.MaxInt64, nil
	}
	return int64(v * mult), nil
}

func digitsFor(base int) int {
	switch base {
	case 16:
		return 22
	case 8:
		return 8
	}
	return 10
}

func multiplier(suffix string) (uint64, bool) {
	switch suffix {
	case "":
		return 1, true
	case "b":
		return 512, true
	}
	power := strings.IndexByte("KMGTPEZYRQ", suffix[0]) + 1
	if suffix[0] == 'k' || suffix[0] == 'm' {
		power = strings.IndexByte("km", suffix[0]) + 1
	}
	if power == 0 {
		return 0, false
	}
	var base uint64
	switch suffix[1:] {
	case "", "iB":
		base = 1024
	case "B":
		base = 1000
	default:
		return 0, false
	}
	m := uint64(1)
	for range power {
		if m > math.MaxInt64/base {
			return 0, true
		}
		m *= base
	}
	return m, true
}
