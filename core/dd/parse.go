package dd

import (
	"math"
	"math/bits"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// ParseError is a bad operand value. GNU dd reports it without the
// "Try --help" hint that follows a usage error.
type ParseError struct{ Msg string }

func (e *ParseError) Error() string { return e.Msg }

// Parse turns dd's KEY=VALUE operands into Options. An unknown operand or a
// conflicting pair of conversions is a *core.UsageError; a bad number or an
// unknown conversion or flag is a *ParseError.
func Parse(args []string) (Options, error) {
	var o Options
	for _, arg := range args {
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			return Options{}, core.Usagef("unrecognized operand '%s'", arg)
		}
		var err error
		switch key {
		case "if":
			o.Input = val
		case "of":
			o.Output = val
		case "bs":
			o.BlockSize, err = blockSize(val)
		case "ibs":
			o.InputBlockSize, err = blockSize(val)
		case "obs":
			o.OutputBlockSize, err = blockSize(val)
		case "cbs":
			o.ConversionBlockSize, err = blockSize(val)
		case "skip", "iseek":
			o.Skip, err = Size(val)
		case "seek", "oseek":
			o.Seek, err = Size(val)
		case "count":
			o.Count, err = Size(val)
			o.HasCount = true
		case "status":
			switch val {
			case "none":
				o.Status = StatusNone
			case "noxfer":
				o.Status = StatusNoXfer
			case "progress":
				o.Status = StatusProgress
			default:
				err = &ParseError{"invalid status level: '" + val + "'"}
			}
		case "conv":
			err = o.parseConv(val)
		case "iflag":
			err = o.parseFlags(val, true)
		case "oflag":
			err = o.parseFlags(val, false)
		default:
			return Options{}, core.Usagef("unrecognized operand '%s'", arg)
		}
		if err != nil {
			return Options{}, err
		}
	}

	if (o.Excl && o.NoCreat) || (o.Block && o.Unblock) || (o.LCase && o.UCase) {
		return Options{}, core.Usagef("cannot combine %s", combo(o))
	}
	return o, nil
}

// combo names the first conflicting pair, for the "cannot combine" error.
func combo(o Options) string {
	switch {
	case o.Excl && o.NoCreat:
		return "excl and nocreat"
	case o.Block && o.Unblock:
		return "block and unblock"
	}
	return "lcase and ucase"
}

func (o *Options) parseConv(list string) error {
	for _, c := range strings.Split(list, ",") {
		switch c {
		case "notrunc":
			o.NoTrunc = true
		case "noerror":
			o.NoError = true
		case "sync":
			o.Sync = true
		case "fsync":
			o.Fsync = true
		case "fdatasync":
			o.Fdatasync = true
		case "excl":
			o.Excl = true
		case "nocreat":
			o.NoCreat = true
		case "lcase":
			o.LCase = true
		case "ucase":
			o.UCase = true
		case "swab":
			o.Swab = true
		case "block":
			o.Block = true
		case "unblock":
			o.Unblock = true
		default:
			return &ParseError{"invalid conversion: '" + c + "'"}
		}
	}
	return nil
}

func (o *Options) parseFlags(list string, input bool) error {
	for _, f := range strings.Split(list, ",") {
		switch {
		case f == "binary" || f == "text":
			// Meaningful only to Windows text mode in GNU dd; gnux never
			// translates line endings.
		case input && f == "fullblock":
			o.FullBlock = true
		case input && f == "count_bytes":
			o.CountBytes = true
		case input && f == "skip_bytes":
			o.SkipBytes = true
		case !input && f == "seek_bytes":
			o.SeekBytes = true
		case !input && f == "append":
			o.Append = true
		default:
			if input {
				return &ParseError{"invalid input flag: '" + f + "'"}
			}
			return &ParseError{"invalid output flag: '" + f + "'"}
		}
	}
	return nil
}

func blockSize(s string) (int64, error) {
	n, err := Size(s)
	if err != nil {
		return 0, err
	}
	if n <= 0 || n > math.MaxInt32 {
		return 0, &ParseError{"invalid number: '" + s + "'"}
	}
	return n, nil
}

// Size parses a dd number: decimal digits with an optional multiplier
// suffix, and products of such numbers joined by 'x' (2x512). The suffixes
// are c (1), w (2), b (512), kB KB MB GB TB PB EB (powers of 1000), and K M G
// T P E alone or followed by iB (powers of 1024).
func Size(s string) (int64, error) {
	bad := &ParseError{"invalid number: '" + s + "'"}
	total := uint64(1)
	for _, part := range strings.Split(s, "x") {
		n, err := factor(part)
		if err != nil {
			return 0, bad
		}
		hi, lo := bits.Mul64(total, n)
		if hi != 0 || lo > math.MaxInt64 {
			return 0, &ParseError{"invalid number: '" + s + "': Value too large for defined data type"}
		}
		total = lo
	}
	return int64(total), nil
}

func factor(s string) (uint64, error) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, strconv.ErrSyntax
	}
	n, err := strconv.ParseUint(s[:i], 10, 64)
	if err != nil {
		return 0, err
	}
	mult, ok := suffix(s[i:])
	if !ok {
		return 0, strconv.ErrSyntax
	}
	hi, lo := bits.Mul64(n, mult)
	if hi != 0 {
		return 0, strconv.ErrRange
	}
	return lo, nil
}

func suffix(s string) (uint64, bool) {
	switch s {
	case "":
		return 1, true
	case "c":
		return 1, true
	case "w":
		return 2, true
	case "b":
		return 512, true
	}
	const letters = "KMGTPE"
	l := s[0]
	if l == 'k' {
		l = 'K'
	}
	p := strings.IndexByte(letters, l)
	if p < 0 {
		return 0, false
	}
	var base uint64
	switch s[1:] {
	case "", "iB":
		base = 1024
	case "B":
		base = 1000
	default:
		return 0, false
	}
	m := uint64(1)
	for range p + 1 {
		m *= base
	}
	return m, true
}
