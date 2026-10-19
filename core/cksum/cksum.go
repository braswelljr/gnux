// Package cksum implements the engine behind cksumx, a GNU-compatible
// implementation of the GNU/UNIX cksum command: print or check a CRC, a
// legacy checksum or a message digest for each file. The digest algorithms
// are core/hashsum, shared with md5sumx and its siblings.
package cksum

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "cksumx"

// Algorithms lists the values Options.Algorithm accepts, in GNU's order.
var Algorithms = []string{
	"sysv", "bsd", "crc", "crc32b", "md5", "sha1", "sha224", "sha256", "sha384", "sha512", "blake2b",
}

var digests = map[string]hashsum.Algorithm{
	"md5": hashsum.MD5, "sha1": hashsum.SHA1, "sha224": hashsum.SHA224, "sha256": hashsum.SHA256,
	"sha384": hashsum.SHA384, "sha512": hashsum.SHA512, "blake2b": hashsum.BLAKE2b,
}

// Options mirrors the GNU cksum flag surface. The zero value prints the POSIX
// CRC and byte count of each file.
type Options struct {
	// Algorithm (-a) is one of Algorithms; "" means crc (and, when checking,
	// detect the algorithm from each tagged line).
	Algorithm string
	// Untagged (--untagged) prints digests in the md5sum style rather than
	// the BSD-style tagged lines that are cksum's default.
	Untagged bool
	// Base64 (--base64) prints digests in base64.
	Base64 bool
	// Raw (--raw) prints the bare binary checksum of a single file.
	Raw bool
	// Length (-l) is the blake2b digest length in bits.
	Length int
	// Zero (-z) ends each output line with NUL.
	Zero bool

	// Check (-c) verifies the files listed in the operands.
	Check bool
	// IgnoreMissing, Quiet, Status, Strict and Warn tune Check as in md5sumx.
	IgnoreMissing, Quiet, Status, Strict, Warn bool
}

// Cksum is the engine. Construct it with New.
type Cksum struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds a cksum engine reading from and writing to s.
func New(opts Options, s core.Streams) *Cksum {
	return &Cksum{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run checksums each operand (standard input when there is none, or for
// "-"), or with Check verifies the listed files. It returns a
// *core.UsageError for an invalid option, core.ErrFailed when anything
// failed, and nil otherwise.
func (c *Cksum) Run(args []string) (err error) {
	alg := c.Algorithm
	if alg != "" && !valid(alg) {
		return core.Usagef("invalid argument '%s' for '--algorithm'\nValid arguments are: '%s'",
			alg, strings.Join(Algorithms, "', '"))
	}
	if c.Length != 0 && alg != "blake2b" {
		return core.Usagef("--length is only supported with --algorithm=blake2b")
	}
	if c.Raw && len(args) > 1 {
		return core.Usagef("the --raw option is not supported with multiple files")
	}

	if d, ok := digests[alg]; ok || (alg == "" && c.Check) {
		if !ok {
			d = hashsum.Auto
		}
		return hashsum.New(Name, d, hashsum.Options{
			Tag: !c.Untagged && !c.Check, Zero: c.Zero, Length: c.Length, Base64: c.Base64, Raw: c.Raw,
			Check: c.Check, IgnoreMissing: c.IgnoreMissing, Quiet: c.Quiet, Status: c.Status,
			Strict: c.Strict, Warn: c.Warn,
		}, c.streams).Run(args)
	}

	if alg == "" {
		alg = "crc"
	}
	if c.Check {
		return core.Usagef("--check is not supported with --algorithm=%s", alg)
	}
	for _, f := range []struct {
		on   bool
		name string
	}{
		{c.IgnoreMissing, "ignore-missing"}, {c.Status, "status"}, {c.Warn, "warn"},
		{c.Quiet, "quiet"}, {c.Strict, "strict"},
	} {
		if f.on {
			return core.Usagef("the --%s option is meaningful only when verifying checksums", f.name)
		}
	}

	out := core.NewWriter(c.streams.Out)
	defer core.FlushOutput(out, c.diag, &err)
	if len(args) == 0 {
		c.legacy(out, alg, "-", false)
	}
	for _, name := range args {
		c.legacy(out, alg, name, true)
		// Keep output and diagnostics in operand order on a shared terminal.
		_ = out.Flush()
	}
	return c.diag.Result()
}

func valid(alg string) bool {
	for _, a := range Algorithms {
		if a == alg {
			return true
		}
	}
	return false
}

// legacy prints the sysv, bsd, crc or crc32b checksum of one input. A file
// read because no operand was given prints no name, as in GNU.
func (c *Cksum) legacy(out *bufio.Writer, alg, name string, named bool) {
	r := c.streams.In
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			c.diag.Errorf("%s: %s", name, core.ErrText(err))
			return
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			c.diag.Errorf("%s: %s", name, core.ErrText(syscall.EISDIR))
			return
		}
		r = f
	}

	var sum, n uint64
	var err error
	switch alg {
	case "sysv":
		sum, n, err = SysV(r)
	case "bsd":
		sum, n, err = BSD(r)
	case "crc32b":
		sum, n, err = crc32b(r)
	default:
		sum, n, err = posixCRC(r)
	}
	if err != nil {
		c.diag.Errorf("%s: %s", name, core.ErrText(err))
		return
	}

	if c.Raw {
		switch alg {
		case "sysv", "bsd":
			_ = binary.Write(out, binary.BigEndian, uint16(sum))
		default:
			_ = binary.Write(out, binary.BigEndian, uint32(sum))
		}
		return
	}

	var line string
	switch alg {
	case "sysv":
		line = fmt.Sprintf("%d %d", sum, (n+511)/512)
	case "bsd":
		line = fmt.Sprintf("%05d %5d", sum, (n+1023)/1024)
	default:
		line = fmt.Sprintf("%d %d", sum, n)
	}
	if named {
		line += " " + name
	}
	end := "\n"
	if c.Zero {
		end = "\x00"
	}
	_, _ = out.WriteString(line + end)
}

// crcTables are the tables for the POSIX cksum polynomial 0x04C11DB7,
// processed most significant bit first. crcTables[0] advances the CRC by one
// byte; crcTables[k] advances it by that byte followed by k zero bytes, which
// lets posixCRC fold eight input bytes into the CRC in one step
// ("slicing-by-8") instead of one table lookup after another.
var crcTables = func() (t [8][256]uint32) {
	for i := range t[0] {
		c := uint32(i) << 24
		for range 8 {
			if c&0x80000000 != 0 {
				c = c<<1 ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		t[0][i] = c
	}
	for k := 1; k < len(t); k++ {
		for i, c := range t[k-1] {
			t[k][i] = c<<8 ^ t[0][c>>24]
		}
	}
	return t
}()

// posixCRC is the CRC POSIX specifies for cksum: the data, then its length
// in as few bytes as hold it (least significant first), then complemented.
func posixCRC(r io.Reader) (sum, n uint64, err error) {
	var crc uint32
	t := &crcTables
	err = each(r, func(b []byte) {
		n += uint64(len(b))
		for ; len(b) >= 8; b = b[8:] {
			hi := crc ^ binary.BigEndian.Uint32(b)
			lo := binary.BigEndian.Uint32(b[4:])
			crc = t[7][hi>>24] ^ t[6][byte(hi>>16)] ^ t[5][byte(hi>>8)] ^ t[4][byte(hi)] ^
				t[3][lo>>24] ^ t[2][byte(lo>>16)] ^ t[1][byte(lo>>8)] ^ t[0][byte(lo)]
		}
		for _, c := range b {
			crc = crc<<8 ^ t[0][byte(crc>>24)^c]
		}
	})
	for l := n; l > 0; l >>= 8 {
		crc = crc<<8 ^ t[0][byte(crc>>24)^byte(l)]
	}
	return uint64(^crc), n, err
}

// crc32b is the ISO-HDLC CRC-32 (zlib, gzip), printed in decimal.
func crc32b(r io.Reader) (sum, n uint64, err error) {
	var crc uint32
	err = each(r, func(b []byte) {
		crc = crc32.Update(crc, crc32.IEEETable, b)
		n += uint64(len(b))
	})
	return uint64(crc), n, err
}

// SysV is the System V sum (sum -s, cksum -a sysv): the byte total folded
// to 16 bits. It returns the checksum and the number of bytes read.
func SysV(r io.Reader) (sum, n uint64, err error) {
	var s uint64
	err = each(r, func(b []byte) {
		for _, c := range b {
			s += uint64(c)
		}
		n += uint64(len(b))
	})
	v := (s & 0xffff) + (s&0xffffffff)>>16
	return (v & 0xffff) + v>>16, n, err
}

// BSD is the BSD sum (sum, cksum -a bsd): a 16-bit checksum rotated right
// before each byte. It returns the checksum and the number of bytes read.
func BSD(r io.Reader) (sum, n uint64, err error) {
	var s uint32
	err = each(r, func(b []byte) {
		for _, c := range b {
			s = (s >> 1) + ((s & 1) << 15)
			s += uint32(c)
			s &= 0xffff
		}
		n += uint64(len(b))
	})
	return uint64(s), n, err
}

// each feeds r to fn in chunks.
func each(r io.Reader, fn func([]byte)) error {
	buf := make([]byte, 64<<10)
	for {
		k, err := r.Read(buf)
		if k > 0 {
			fn(buf[:k])
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
