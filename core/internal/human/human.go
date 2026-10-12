// Package human formats sizes the way GNU du, df and ls do (gnulib's
// human_readable) and parses their --block-size argument (gnulib's
// humblock), so "1.5K", "10M" and "-BM" mean exactly what they mean in GNU.
//
// A block size takes the suffixes K, M, G, T, P, E, Z and Y in either case,
// optionally followed by B (powers of 1000) or iB.
package human

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Opts are gnulib's human_readable options.
type Opts int

// The rounding styles (Ceiling, the zero value, is what du and df use) and
// the formatting flags.
const (
	Ceiling Opts = 0
	Nearest Opts = 1
	Floor   Opts = 2

	// GroupDigits groups thousands; in the C locale that is a no-op.
	GroupDigits       Opts = 4
	SuppressPointZero Opts = 8
	// Autoscale picks the unit (K, M, G, ...) that keeps the number small.
	Autoscale Opts = 16
	// Base1024 uses powers of 1024 rather than 1000.
	Base1024        Opts = 32
	SpaceBeforeUnit Opts = 64
	// SI appends the unit letter.
	SI Opts = 128
	// B appends "B" (or "iB" in base 1024) after the unit letter.
	B Opts = 256
)

const powerLetters = "\x00KMGTPEZYRQ"

const exponentMax = len(powerLetters) - 1

// Readable renders n units of fromBlock bytes as a count of toBlock-byte
// units, as human_readable does.
func Readable(n uint64, opts Opts, fromBlock, toBlock uint64) string {
	inexact := opts & (Nearest | Floor)
	base := uint64(1000)
	if opts&Base1024 != 0 {
		base = 1024
	}
	exponent := -1
	var number string

	var amt uint64
	tenths, rounding := 0, 0
	integer := false
	switch {
	case toBlock <= fromBlock:
		if fromBlock%toBlock == 0 {
			mult := fromBlock / toBlock
			if n == 0 || n <= math.MaxUint64/mult {
				amt, integer = n*mult, true
			}
		}
	case fromBlock != 0 && toBlock%fromBlock == 0:
		divisor := toBlock / fromBlock
		r10 := (n % divisor) * 10
		r2 := (r10 % divisor) * 2
		amt = n / divisor
		tenths = int(r10 / divisor)
		switch {
		case r2 < divisor:
			rounding = btoi(r2 > 0)
		default:
			rounding = 2 + btoi(divisor < r2)
		}
		integer = true
	}

	if integer {
		point := ""
		if opts&Autoscale != 0 {
			exponent = 0
			if base <= amt {
				for {
					r10 := (amt%base)*10 + uint64(tenths)
					r2 := (r10%base)*2 + uint64(rounding>>1)
					amt /= base
					tenths = int(r10 / base)
					if r2 < base {
						rounding = btoi(r2+uint64(rounding) != 0)
					} else {
						rounding = 2 + btoi(base < r2+uint64(rounding))
					}
					exponent++
					if base > amt || exponent >= exponentMax {
						break
					}
				}
				if amt < 10 {
					if inexact == Nearest && 2 < rounding+(tenths&1) ||
						inexact == Ceiling && 0 < rounding {
						tenths++
						rounding = 0
						if tenths == 10 {
							amt++
							tenths = 0
						}
					}
					if amt < 10 && (tenths != 0 || opts&SuppressPointZero == 0) {
						point = "." + strconv.Itoa(tenths)
						tenths, rounding = 0, 0
					}
				}
			}
		}
		if inexact == Nearest && 5 < tenths+btoi(0 < rounding+int(amt&1)) ||
			inexact == Ceiling && 0 < tenths+rounding {
			amt++
			if opts&Autoscale != 0 && amt == base && exponent < exponentMax {
				exponent++
				point = ""
				if opts&SuppressPointZero == 0 {
					point = ".0"
				}
				amt = 1
			}
		}
		number = strconv.FormatUint(amt, 10) + point
	} else {
		// Neither block size divides the other: fall back on floating
		// point, as gnulib does.
		damt := float64(n) * (float64(fromBlock) / float64(toBlock))
		if opts&Autoscale == 0 {
			number = fmt.Sprintf("%.0f", adjust(inexact, damt))
		} else {
			e := 1.0
			exponent = 0
			for {
				e *= float64(base)
				exponent++
				if !(e*float64(base) <= damt && exponent < exponentMax) {
					break
				}
			}
			damt /= e
			number = fmt.Sprintf("%.1f", adjust(inexact, damt))
			if 1+2+btoi(opts&Base1024 == 0) < len(number) ||
				opts&SuppressPointZero != 0 && strings.HasSuffix(number, "0") {
				number = fmt.Sprintf("%.0f", adjust(inexact, damt*10)/10)
			}
		}
	}

	if opts&SI == 0 {
		return number
	}
	if exponent < 0 {
		exponent = 0
		for power := uint64(1); power < toBlock; power *= base {
			exponent++
			if exponent == exponentMax {
				break
			}
		}
	}
	var b strings.Builder
	b.WriteString(number)
	if (exponent != 0 || opts&B != 0) && opts&SpaceBeforeUnit != 0 {
		b.WriteByte(' ')
	}
	if exponent != 0 {
		if opts&Base1024 == 0 && exponent == 1 {
			b.WriteByte('k')
		} else {
			b.WriteByte(powerLetters[exponent])
		}
	}
	if opts&B != 0 {
		if opts&Base1024 != 0 && exponent != 0 {
			b.WriteByte('i')
		}
		b.WriteByte('B')
	}
	return b.String()
}

func adjust(inexact Opts, v float64) float64 {
	if inexact != Nearest && v < math.MaxUint64 {
		u := math.Trunc(v)
		if inexact == Ceiling && u != v {
			u++
		}
		return u
	}
	return v
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ErrInvalid is a block size that is not a number with an optional suffix;
// ErrTooLarge one that does not fit.
var (
	ErrInvalid  = errors.New("invalid block size")
	ErrTooLarge = errors.New("block size too large")
)

// multipliers are the suffix letters a block size may end with, in either
// case, each a power of 1024 (or of 1000 when followed by B).
const multipliers = "KMGTPEZY"

// humanReadable is the block size that asks for automatic units; any
// unambiguous abbreviation of it ("human") will do.
const humanReadable = "human-readable"

// ParseBlockSize reads a --block-size argument as gnulib's humblock does:
// "human-readable" or "si"; or a number with an optional suffix (K, KiB,
// KB, M, ...; "1K" means 1024 bytes and "1KB" 1000). A suffix with no number
// ("K", "MB") also asks for the unit to be printed after each size.
func ParseBlockSize(spec string) (uint64, Opts, error) {
	var opts Opts
	if strings.HasPrefix(spec, "'") {
		opts |= GroupDigits
		spec = spec[1:]
	}
	switch {
	case spec == humanReadable || (len(spec) > 1 && len(spec) < len(humanReadable) && humanReadable[:len(spec)] == spec):
		return 1, opts | Autoscale | SI | Base1024, nil
	case spec == "si":
		return 1, opts | Autoscale | SI, nil
	}
	digits := 0
	for digits < len(spec) && spec[digits] >= '0' && spec[digits] <= '9' {
		digits++
	}
	n := uint64(1)
	if digits > 0 {
		v, err := strconv.ParseUint(spec[:digits], 10, 64)
		if err != nil {
			return 0, 0, ErrTooLarge
		}
		n = v
	}
	suffix := spec[digits:]
	if suffix == "" {
		if digits == 0 {
			return 0, 0, ErrInvalid
		}
		if n == 0 {
			return 0, 0, ErrInvalid
		}
		return n, opts, nil
	}
	power := strings.IndexByte(multipliers, upper(suffix[0])) + 1
	if power == 0 {
		return 0, 0, ErrInvalid
	}
	unit := uint64(1024)
	switch suffix[1:] {
	case "":
	case "B":
		unit = 1000
	case "iB":
	default:
		return 0, 0, ErrInvalid
	}
	size := n
	for range power {
		if size > math.MaxUint64/unit {
			return 0, 0, ErrTooLarge
		}
		size *= unit
	}
	if size == 0 {
		return 0, 0, ErrInvalid
	}
	if digits == 0 {
		opts |= SI
		if strings.HasSuffix(suffix, "B") {
			opts |= B
		}
		if !strings.HasSuffix(suffix, "B") || strings.HasSuffix(suffix, "iB") {
			opts |= Base1024
		}
	}
	return size, opts, nil
}

func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// DefaultBlockSize resolves the block size when no option gives one: the
// first of the named environment variables that is set (DU_BLOCK_SIZE or
// DF_BLOCK_SIZE, then BLOCK_SIZE and BLOCKSIZE), else 512 under
// POSIXLY_CORRECT and 1024 otherwise. An invalid variable is ignored.
func DefaultBlockSize(getenv func(string) string, own string) (uint64, Opts) {
	for _, name := range []string{own, "BLOCK_SIZE", "BLOCKSIZE"} {
		if name == "" {
			continue
		}
		if v := getenv(name); v != "" {
			if size, opts, err := ParseBlockSize(v); err == nil {
				return size, opts
			}
			break
		}
	}
	if getenv("POSIXLY_CORRECT") != "" {
		return 512, 0
	}
	return 1024, 0
}
