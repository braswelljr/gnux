package head

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidSize reports a count that is not a decimal number with an optional
// GNU multiplier suffix.
var ErrInvalidSize = errors.New("invalid size")

// ParseSize parses a GNU count: a non-negative decimal integer optionally
// followed by a multiplier suffix. "b" is 512; K, M, G, T, P, E, Z, Y, R and Q
// are powers of 1024 (also spelled KiB, MiB, …; "k" is accepted for K); KB,
// MB, … (and kB) are powers of 1000. It carries no sign: callers strip a
// leading '+' or '-' themselves, because its meaning differs per tool.
//
// A value too large for int64 is clamped to math.MaxInt64 rather than rejected,
// as GNU head and tail do - "more than the file can hold" is still meaningful.
// The parser is shared by head, tail and any other tool taking such counts.
func ParseSize(s string) (int64, error) {
	digits := 0
	for digits < len(s) && s[digits] >= '0' && s[digits] <= '9' {
		digits++
	}
	if digits == 0 {
		return 0, ErrInvalidSize
	}

	mult, ok := multiplier(s[digits:])
	if !ok {
		return 0, ErrInvalidSize
	}

	n, err := strconv.ParseUint(s[:digits], 10, 64)
	if err != nil {
		// Only out-of-range is possible here: the text is all digits.
		return math.MaxInt64, nil
	}
	if n == 0 {
		return 0, nil
	}
	if mult == 0 || n > math.MaxInt64/mult {
		return math.MaxInt64, nil
	}
	return int64(n * mult), nil
}

// multiplier returns the factor for a suffix. A factor of 0 with ok=true means
// the suffix is valid but too large for 64 bits.
func multiplier(suffix string) (mult uint64, ok bool) {
	switch suffix {
	case "":
		return 1, true
	case "b":
		return 512, true
	}

	power := strings.IndexByte("KMGTPEZYRQ", suffix[0]) + 1
	if suffix[0] == 'k' {
		power = 1
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

	mult = 1
	for range power {
		if mult > math.MaxInt64/base {
			return 0, true
		}
		mult *= base
	}
	return mult, true
}
