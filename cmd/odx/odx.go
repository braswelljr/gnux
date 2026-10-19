// Package odx wires the odx command (GNU od) to its engine.
package odx

import (
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/od"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the odx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "od",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for odx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    od.Name,
		SeeAlso: []string{"catx", "headx", "wcx"},
		Use:     od.Name + " [OPTION]... [FILE]...",
		Short:   "Dump files in octal and other formats.",
		Long: "odx writes an unambiguous representation of its input, by default as two-byte " +
			"octal words, sixteen bytes per line, each line starting with the input offset in " +
			"octal. The FILEs are read one after another as one input; with no FILE, or when " +
			"FILE is -, it reads standard input. -t TYPE selects the format and may be given " +
			"more than once, each format getting a line of its own per block, in the order " +
			"given: a (named characters), c (printable characters or backslash escapes), " +
			"d, o, u and x (signed decimal, octal, unsigned decimal and hexadecimal) with an " +
			"optional size in bytes (1, 2, 4, 8, or C, S, I, L), and f (floating point, size " +
			"4 or 8, or F, D, L); a trailing z adds the printable characters of the line " +
			"between > and <. The traditional letters -a, -b, -c, -d, -f, -i, -l, -o, -s and " +
			"-x are shorthands for -t a, o1, c, u2, fF, dI, dL, o2, d2 and x2. -A RADIX picks " +
			"the offset base (d, o, x, or n for none), -j BYTES skips input, -N BYTES limits " +
			"it, -w BYTES sets the bytes per line, and " +
			"--endian chooses the byte order of multi-byte values. Unless -v is given, a run " +
			"of lines identical to the one before is shown as a single '*'. -S N prints the " +
			"NUL-terminated strings of at least N printable characters instead " +
			"of a dump. BYTES may be decimal, octal with a leading 0, or hexadecimal with " +
			"0x, followed by a multiplier: b 512, k or K 1024, kB 1000, M, MB, G and so on. " +
			"Differences from GNU od: characters are interpreted in the C locale, so -c " +
			"shows every byte above 127 in octal; long double (fL) is read as an eight-byte " +
			"double; -S and -w need a value (GNU's bare -S means -S3 and bare -w means " +
			"-w32), because an optional value could not also be attached as in -w8; the " +
			"traditional offset operand ([+]OFFSET[.][b]) and --traditional are " +
			"not implemented.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Seeing exactly which bytes a file holds, including invisible ones.",
			"Finding stray carriage returns, tabs or NUL bytes with -c.",
			"Reading binary headers as hexadecimal bytes or words with -t x1 or -t x2.",
			"Checking a file's byte order or a magic number with --endian and -N.",
			"Decoding integers or floating-point values stored in a binary file.",
			"Getting a hexdump-style view with -A x -t x1z.",
			"Extracting printable strings from a binary with -S.",
			"Dumping just a slice of a large file with -j and -N.",
		},
		Examples: []docmeta.Example{
			{Description: "Dump a file as octal words:", Bash: "odx data.bin", PowerShell: "odx data.bin"},
			{Description: "Show every character, with escapes for invisible ones:", Bash: "odx -c notes.txt", PowerShell: "odx -c notes.txt"},
			{Description: "A hexdump with offsets in hex and the text alongside:", Bash: "odx -A x -t x1z image.png", PowerShell: "odx -A x -t x1z image.png"},
			{Description: "Read the first four bytes (a magic number):", Bash: "odx -A n -t x1 -N 4 image.png", PowerShell: "odx -A n -t x1 -N 4 image.png"},
			{Description: "Skip a 512-byte header and dump the next 64 bytes:", Bash: "odx -j 512 -N 64 -t x1 disk.img", PowerShell: "odx -j 512 -N 64 -t x1 disk.img"},
			{Description: "Decode big-endian 32-bit integers:", Bash: "odx --endian=big -t u4 counts.bin", PowerShell: "odx --endian=big -t u4 counts.bin"},
			{Description: "Decode floating-point values:", Bash: "odx -t f8 samples.bin", PowerShell: "odx -t f8 samples.bin"},
			{Description: "Two formats per line, hex bytes over characters:", Bash: "odx -t x1 -t c file.txt", PowerShell: "odx -t x1 -t c file.txt"},
			{Description: "Look for Windows line endings in standard input:", Bash: "printfx 'a\\r\\nb\\n' | odx -c", PowerShell: "'a' | odx -c"},
			{Description: "Print every line, even repeated ones:", Bash: "odx -v -t x1 zeros.bin", PowerShell: "odx -v -t x1 zeros.bin"},
			{Description: "Eight bytes per line:", Bash: "odx -w8 -t x1 data.bin", PowerShell: "odx -w8 -t x1 data.bin"},
			{Description: "List the printable strings of at least 6 characters in a binary:", Bash: "odx -S 6 program.bin", PowerShell: "odx -S 6 program.exe"},
		},
	}
}

// formatList is the -t value, shared with the traditional letters so that
// formats keep the order they were given in on the command line.
type formatList struct {
	items *[]string
	alias string // the -t type a traditional letter stands for
}

func (f formatList) String() string { return strings.Join(*f.items, ",") }
func (f formatList) Type() string   { return "string" }

func (f formatList) Set(v string) error {
	if f.alias != "" {
		if v == "true" {
			*f.items = append(*f.items, f.alias)
		}
		return nil
	}
	*f.items = append(*f.items, v)
	return nil
}

// traditional maps each traditional letter to its -t type.
var traditional = []struct {
	short, long, typ, usage string
}{
	{"a", "named-characters", "a", "same as -t a, select named characters, ignoring high-order bit"},
	{"b", "octal-bytes", "o1", "same as -t o1, select octal bytes"},
	{"c", "characters", "c", "same as -t c, select printable characters or backslash escapes"},
	{"d", "unsigned-decimal-2-bytes", "u2", "same as -t u2, select unsigned decimal 2-byte units"},
	{"f", "floats", "fF", "same as -t fF, select floats"},
	{"i", "decimal-ints", "dI", "same as -t dI, select decimal ints"},
	{"l", "decimal-longs", "dL", "same as -t dL, select decimal longs"},
	{"o", "octal-2-bytes", "o2", "same as -t o2, select octal 2-byte units"},
	{"s", "decimal-2-bytes", "d2", "same as -t d2, select decimal 2-byte units"},
	{"x", "hex-2-bytes", "x2", "same as -t x2, select hexadecimal 2-byte units"},
}

// Register installs the odx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("address-radix", "A", "o", "output format for file offsets; `RADIX` is one of [doxn], for Decimal, Octal, Hex or None")
	fs.String("endian", "", "swap input bytes according the specified `ORDER`: little or big")
	fs.StringP("skip-bytes", "j", "", "skip `BYTES` input bytes first")
	fs.StringP("read-bytes", "N", "", "limit dump to `BYTES` input bytes")
	fs.StringP("strings", "S", "", "show only NUL terminated strings of at least `BYTES` printable characters")
	fs.BoolP("output-duplicates", "v", false, "do not use * to mark line suppression")
	fs.StringP("width", "w", "", "output `BYTES` bytes per output line")

	items := &[]string{}
	fs.VarP(formatList{items: items}, "format", "t", "select output format or formats; `TYPE` is a, c, d, f, o, u or x with an optional size, and z")
	for _, t := range traditional {
		fs.VarP(formatList{items: items, alias: t.typ}, t.long, t.short, t.usage)
		fs.Lookup(t.long).NoOptDefVal = "true"
	}

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "characters", "decimal-2-bytes", "decimal-ints", "decimal-longs", "floats", "hex-2-bytes", "named-characters", "octal-2-bytes", "octal-bytes", "unsigned-decimal-2-bytes")
}

// fatal is an option error GNU od reports without the --help hint.
type fatal struct{ msg string }

func (f fatal) Error() string { return f.msg }

// buildOptions folds the parsed flags into od.Options.
func buildOptions(fs *pflag.FlagSet) (od.Options, error) {
	var o od.Options

	radix, _ := fs.GetString("address-radix")
	if len(radix) != 1 || strings.IndexByte("doxn", radix[0]) < 0 {
		return od.Options{}, fatal{"invalid output address radix '" + radix + "'; it must be one character from [doxn]"}
	}
	if radix != "o" {
		o.Radix = od.Radix(radix[0])
	}

	switch e, _ := fs.GetString("endian"); e {
	case "":
	case "little":
		o.Endian = od.EndianLittle
	case "big":
		o.Endian = od.EndianBig
	default:
		return od.Options{}, core.Usagef("invalid argument '%s' for '--endian'\nValid arguments are:\n  - 'big'\n  - 'little'", e)
	}

	bytesFlag := func(name, short string) (int64, bool, error) {
		if !fs.Changed(name) {
			return 0, false, nil
		}
		v, _ := fs.GetString(name)
		n, err := od.ParseBytes(v)
		if err != nil {
			return 0, false, fatal{"invalid -" + short + " argument '" + v + "'"}
		}
		return n, true, nil
	}
	var err error
	if o.Skip, _, err = bytesFlag("skip-bytes", "j"); err != nil {
		return od.Options{}, err
	}
	if o.Limit, o.LimitSet, err = bytesFlag("read-bytes", "N"); err != nil {
		return od.Options{}, err
	}
	if fs.Changed("strings") {
		v, _ := fs.GetString("strings")
		n, err := od.ParseBytes(v)
		if err != nil || n <= 0 {
			return od.Options{}, fatal{"invalid -S argument '" + v + "'"}
		}
		o.Strings = int(min(n, 1<<30))
	}
	if fs.Changed("width") {
		v, _ := fs.GetString("width")
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return od.Options{}, fatal{"invalid -w argument '" + v + "'"}
		}
		o.Width = n
	}
	o.Verbose, _ = fs.GetBool("output-duplicates")

	if types := *fs.Lookup("format").Value.(formatList).items; len(types) > 0 {
		o.Types = append([]string(nil), types...)
	}
	for _, t := range o.Types {
		if err := od.ParseType(t); err != nil {
			return od.Options{}, err
		}
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if f, ok := err.(fatal); ok {
		core.NewDiag(od.Name, s).Errorf("%s", f.msg)
		return core.ErrFailed
	}
	if err != nil {
		return err
	}
	return od.New(opts, s).Run(args)
}
