package visual

// ByteKey is the theme key of a byte's class in a dump: zero, printable
// ASCII, ASCII whitespace (space, \t, \n, \v, \f, \r), any other ASCII
// control, or above 0x7f.
func ByteKey(c byte) string {
	switch {
	case c == 0:
		return ByteZero
	case c == ' ' || (c >= '\t' && c <= '\r'):
		return ByteSpace
	case c > ' ' && c < 0x7f:
		return BytePrint
	case c < 0x80:
		return ByteControl
	}
	return ByteHigh
}

// byteRank orders the classes from the least to the most unusual.
var byteRank = map[string]int{ByteZero: 0, BytePrint: 1, ByteSpace: 2, ByteControl: 3, ByteHigh: 4}

// BytesKey is the class of a value several bytes wide: zero when every
// byte is, else the most unusual class among its bytes, so a word holding
// one control character is marked as one.
func BytesKey(b []byte) string {
	key := ByteZero
	for _, c := range b {
		if k := ByteKey(c); byteRank[k] > byteRank[key] {
			key = k
		}
	}
	return key
}
