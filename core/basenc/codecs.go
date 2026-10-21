package basenc

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
)

// codec is one encoding: how a chunk is encoded, which bytes belong to its
// alphabet, and how one group of encoded characters decodes.
type codec struct {
	// group is the number of encoded characters that decode on their own.
	group int
	// encode appends the encoding of src to dst. src is a whole number of
	// input groups, except at the end of the input.
	encode func(dst, src []byte) []byte
	// alphabet reports whether b may appear in the encoding, padding included.
	alphabet func(b byte) bool
	// decode decodes one whole group into dst, reporting whether it was valid.
	decode func(dst, group []byte) (int, bool)
	// partial decodes a final group cut short, where the encoding allows it.
	partial func(dst, group []byte) int
}

// stdlib wraps an encoding/base64 or encoding/base32 encoding.
type stdlib interface {
	AppendEncode(dst, src []byte) []byte
	Decode(dst, src []byte) (int, error)
}

func std(enc, raw stdlib, group int, alphabet func(byte) bool) codec {
	return codec{
		group:    group,
		encode:   enc.AppendEncode,
		alphabet: alphabet,
		decode: func(dst, g []byte) (int, bool) {
			n, err := enc.Decode(dst, g)
			return n, err == nil
		},
		partial: func(dst, g []byte) int {
			n, _ := raw.Decode(dst, g)
			return n
		},
	}
}

func between(b, lo, hi byte) bool { return b >= lo && b <= hi }

var codecs = map[Encoding]codec{
	Base64: std(base64.StdEncoding, base64.RawStdEncoding, 4, func(b byte) bool {
		return between(b, 'A', 'Z') || between(b, 'a', 'z') || between(b, '0', '9') || b == '+' || b == '/' || b == '='
	}),
	Base64URL: std(base64.URLEncoding, base64.RawURLEncoding, 4, func(b byte) bool {
		return between(b, 'A', 'Z') || between(b, 'a', 'z') || between(b, '0', '9') || b == '-' || b == '_' || b == '='
	}),
	Base32: std(base32.StdEncoding, base32.StdEncoding.WithPadding(base32.NoPadding), 8, func(b byte) bool {
		return between(b, 'A', 'Z') || between(b, '2', '7') || b == '='
	}),
	Base32Hex: std(base32.HexEncoding, base32.HexEncoding.WithPadding(base32.NoPadding), 8, func(b byte) bool {
		return between(b, '0', '9') || between(b, 'A', 'V') || b == '='
	}),
	Base16: {
		group: 2,
		encode: func(dst, src []byte) []byte {
			const digits = "0123456789ABCDEF"
			for _, b := range src {
				dst = append(dst, digits[b>>4], digits[b&15])
			}
			return dst
		},
		alphabet: func(b byte) bool {
			return between(b, '0', '9') || between(b, 'A', 'F') || between(b, 'a', 'f')
		},
		decode: func(dst, g []byte) (int, bool) {
			n, err := hex.Decode(dst, g)
			return n, err == nil
		},
	},
	Base2MSBF: base2(true),
	Base2LSBF: base2(false),
	Z85: {
		group: 5,
		encode: func(dst, src []byte) []byte {
			for ; len(src) >= 4; src = src[4:] {
				v := binary.BigEndian.Uint32(src)
				var five [5]byte
				for i := 4; i >= 0; i-- {
					five[i] = z85Alphabet[v%85]
					v /= 85
				}
				dst = append(dst, five[:]...)
			}
			return dst
		},
		alphabet: func(b byte) bool { return z85Value[b] != 0xff },
		decode: func(dst, g []byte) (int, bool) {
			var v uint64
			for _, b := range g {
				v = v*85 + uint64(z85Value[b])
			}
			if v > 0xffffffff {
				return 0, false
			}
			binary.BigEndian.PutUint32(dst, uint32(v))
			return 4, true
		},
	},
}

// base2 writes each byte as eight bits.
func base2(msbFirst bool) codec {
	bit := func(i int) uint { // the bit written at position i of a byte
		if msbFirst {
			return uint(7 - i)
		}
		return uint(i)
	}
	return codec{
		group: 8,
		encode: func(dst, src []byte) []byte {
			for _, b := range src {
				for i := range 8 {
					dst = append(dst, '0'+(b>>bit(i))&1)
				}
			}
			return dst
		},
		alphabet: func(b byte) bool { return b == '0' || b == '1' },
		decode: func(dst, g []byte) (int, bool) {
			var v byte
			for i, c := range g {
				v |= (c - '0') << bit(i)
			}
			dst[0] = v
			return 1, true
		},
	}
}

// z85Alphabet is ZeroMQ's Z85 alphabet; z85Value maps a byte back to its
// digit, 0xff for a byte outside it.
const z85Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ.-:+=^!/*?&<>()[]{}@%$#"

var z85Value = func() (t [256]byte) {
	for i := range t {
		t[i] = 0xff
	}
	for i := range len(z85Alphabet) {
		t[z85Alphabet[i]] = byte(i)
	}
	return t
}()
