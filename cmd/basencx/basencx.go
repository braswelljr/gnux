// Package basencx wires the basencx command (GNU basenc) to its engine,
// core/basenc. The decoding and wrapping flags are those of base64x and
// base32x (internal/codec); basencx adds one switch per encoding.
package basencx

import (
	"errors"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basenc"
	"github.com/braswelljr/gnux/internal/codec"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the basencx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "basenc",
		Flags: Register,
		Run:   run,
	}
}

// encodings are the encoding switches, in GNU's order.
var encodings = []struct {
	flag  string
	enc   basenc.Encoding
	usage string
}{
	{"base64", basenc.Base64, "same as 'base64' program (RFC4648 section 4)"},
	{"base64url", basenc.Base64URL, "file- and url-safe base64 (RFC4648 section 5)"},
	{"base32", basenc.Base32, "same as 'base32' program (RFC4648 section 6)"},
	{"base32hex", basenc.Base32Hex, "extended hex alphabet base32 (RFC4648 section 7)"},
	{"base16", basenc.Base16, "hex encoding (RFC4648 section 8)"},
	{"base2msbf", basenc.Base2MSBF, "bit string with most significant bit (msb) first"},
	{"base2lsbf", basenc.Base2LSBF, "bit string with least significant bit (lsb) first"},
	{"z85", basenc.Z85, "ascii85-like encoding (ZeroMQ spec:32/Z85); when encoding, input length must be a multiple of 4; when decoding, input length must be a multiple of 5"},
}

// choice is an encoding switch. Every switch of a flag set shares one
// selection, which the switch given last sets, so that the last encoding on
// the command line wins, as in GNU basenc.
type choice struct {
	enc basenc.Encoding
	sel *selection
}

type selection struct {
	enc basenc.Encoding
	set bool
}

func (c *choice) Set(s string) error {
	on, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if on {
		*c.sel = selection{c.enc, true}
	} else if c.sel.enc == c.enc {
		c.sel.set = false
	}
	return nil
}

func (c *choice) String() string {
	return strconv.FormatBool(c.sel != nil && c.sel.set && c.sel.enc == c.enc)
}

func (c *choice) Type() string { return "bool" }

// IsBoolFlag marks the switch as taking no argument.
func (c *choice) IsBoolFlag() bool { return true }

// Register installs the basencx flags on fs.
func Register(fs *pflag.FlagSet) {
	sel := new(selection)
	for _, e := range encodings {
		f := fs.VarPF(&choice{enc: e.enc, sel: sel}, e.flag, "", e.usage)
		f.NoOptDefVal = "true"
	}
	codec.Register(fs)
}

// buildOptions folds the parsed flags into the encoding and basenc.Options.
// It is a usage error to give no encoding.
func buildOptions(fs *pflag.FlagSet) (basenc.Encoding, basenc.Options, error) {
	f := fs.Lookup(encodings[0].flag)
	sel := f.Value.(*choice).sel
	opts, err := codec.BuildOptions(fs)
	if err != nil {
		return 0, basenc.Options{}, err
	}
	if !sel.set {
		return 0, basenc.Options{}, core.Usagef("missing encoding type")
	}
	return sel.enc, opts, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	enc, opts, err := buildOptions(fs)
	if err != nil {
		var usage *core.UsageError
		if errors.As(err, &usage) {
			return err
		}
		// An invalid wrap size, which GNU reports without the --help hint.
		d := core.NewDiag(basenc.Name, s)
		d.Errorf("%s", err)
		return d.Result()
	}
	return basenc.New(basenc.Name, enc, opts, s).Run(args)
}

// Meta returns the documentation metadata for basencx.
func Meta() docmeta.Meta {
	n := basenc.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"base64x", "base32x", "odx"},
		Use:     n + " [OPTION]... [FILE]",
		Short:   "Encode or decode data in one of several encodings.",
		Long: n + " encodes FILE in the encoding an option selects and writes the result to " +
			"standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads " +
			"standard input; at most one FILE is accepted. An encoding must be given: " +
			"--base64 and --base32 are base64x and base32x; --base64url is base64 with '-' " +
			"and '_' in place of '+' and '/', safe in file names and URLs; --base32hex is " +
			"base32 over 0-9 and A-V; --base16 is upper-case hexadecimal (decoding takes " +
			"either case); --base2msbf and --base2lsbf write every byte as eight '0' and '1' " +
			"characters, most or least significant bit first; --z85 is ZeroMQ's Z85, which " +
			"turns each four bytes into five characters and refuses input that is not a whole " +
			"number of groups. If several are given, the last one wins. Encoded output is " +
			"wrapped every 76 characters and ends with a newline; -w COLS wraps at COLS " +
			"instead, and -w 0 writes one line with no newline at all. When decoding, " +
			"newlines are ignored and -i also ignores every other character outside the " +
			"alphabet; any other stray character, or a truncated input, stops decoding with " +
			"'invalid input' and exit status 1, after the data decoded so far has been written.",
		Permissions: "Needs read permission on FILE and search permission on the directories " +
			"leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Making tokens and identifiers safe for URLs and file names with --base64url.",
			"Writing a hex dump of a key or digest with --base16.",
			"Looking at the bits of a byte string with --base2msbf.",
			"Decoding Z85 strings used by ZeroMQ, such as CURVE keys.",
		},
		Examples: []docmeta.Example{
			{Description: "Encode a value for a URL, where '+' and '/' would need escaping:",
				Bash: "printfx '>>?' | " + n + " --base64url", PowerShell: "printfx '>>?' | " + n + " --base64url", Output: "Pj4_"},
			{Description: "Print the bytes of a file in hexadecimal on one line:",
				Bash: n + " --base16 -w 0 key.bin", PowerShell: n + " --base16 -w 0 key.bin"},
			{Description: "Turn a hex string back into bytes:",
				Bash: "echox 68656c6c6f | " + n + " --base16 -d", PowerShell: "echox 68656c6c6f | " + n + " --base16 -d", Output: "hello"},
			{Description: "See the bits of a character:",
				Bash: "printfx A | " + n + " --base2msbf", PowerShell: "printfx A | " + n + " --base2msbf", Output: "01000001"},
			{Description: "Decode a ZeroMQ CURVE key written in Z85:",
				Bash: "printfx '%s' \"$KEY\" | " + n + " --z85 -d > key.bin", PowerShell: "printfx '%s' $env:KEY | " + n + " --z85 -d > key.bin"},
			{Description: "Encode in base32 with the hex alphabet, which sorts as the data does:",
				Bash: "printfx hello | " + n + " --base32hex", PowerShell: "printfx hello | " + n + " --base32hex", Output: "D1IMOR3F"},
		},
	}
}
