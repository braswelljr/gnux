// Package codec is the command-line adapter shared by base64x and base32x:
// their flags, their documentation and the mapping onto basenc.Options. Each
// cmd/<tool>x package describes its tool with a Tool and delegates to it.
package codec

import (
	"errors"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basenc"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Tool describes one encoding tool.
type Tool struct {
	Name     string // command name, "base64x"
	GNU      string // GNU name, "base64"
	Encoding string // "base64"
	Sibling  string // the other encoding tool, for SeeAlso
	Example  string // a short input and its encoding, for the documentation
	Encoded  string
	New      func(basenc.Options, core.Streams) *basenc.Coder
}

// Spec describes the tool.
func (t Tool) Spec() tool.Spec {
	return tool.Spec{
		Meta:  t.Meta(),
		GNU:   t.GNU,
		Flags: Register,
		Run:   t.run,
	}
}

// Register installs the flags of an encoding tool on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("decode", "d", false, "decode data")
	fs.BoolP("ignore-garbage", "i", false, "when decoding, ignore non-alphabet characters")
	fs.StringP("wrap", "w", "", "wrap encoded lines after `COLS` character (default 76); use 0 to disable line wrapping")
}

// fatalError is a mistake GNU reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// BuildOptions folds the parsed flags into basenc.Options.
func BuildOptions(fs *pflag.FlagSet) (basenc.Options, error) {
	var o basenc.Options
	o.Decode, _ = fs.GetBool("decode")
	o.IgnoreGarbage, _ = fs.GetBool("ignore-garbage")
	if fs.Changed("wrap") {
		text, _ := fs.GetString("wrap")
		n, err := strconv.ParseUint(text, 10, 31)
		if err != nil {
			return basenc.Options{}, &fatalError{"invalid wrap size: '" + text + "'"}
		}
		o.Wrap, o.NoWrap = int(n), n == 0
	}
	return o, nil
}

func (t Tool) run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := BuildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(t.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return t.New(opts, s).Run(args)
}

// Meta returns the tool's documentation.
func (t Tool) Meta() docmeta.Meta {
	n, e := t.Name, t.Encoding
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{t.Sibling, "odx", "cksumx"},
		Use:     n + " [OPTION]... [FILE]",
		Short:   "Encode or decode data in " + e + ".",
		Long: n + " encodes FILE in " + e + " (RFC 4648, with padding) and writes the result to " +
			"standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads " +
			"standard input; at most one FILE is accepted. Encoded output is wrapped every 76 " +
			"characters and ends with a newline; -w COLS wraps at COLS instead, and -w 0 writes " +
			"one line with no newline at all. When decoding, newlines are ignored, and -i also " +
			"ignores every other character outside the alphabet; any other stray character, " +
			"or a truncated input, stops decoding with 'invalid input' and exit status 1, " +
			"after the data decoded so far has been written.",
		Permissions: "Needs read permission on FILE and search permission on the directories " +
			"leading to it. Nothing is ever modified.",
		UseCases: []string{
			"Embedding binary data such as images or keys in text formats like JSON or YAML.",
			"Decoding " + e + " found in configuration, tokens or email attachments.",
			"Passing binary data through channels that only carry text.",
			"Producing single-line encodings for environment variables with -w 0.",
			"Inspecting encoded payloads in a pipeline.",
		},
		Examples: []docmeta.Example{
			{Description: "Put a small logo straight into an HTML or CSS data: URL:", Bash: "echox \"data:image/png;" + e + ",$(" + n + " -w 0 logo.png)\"",
				PowerShell: "\"data:image/png;" + e + ",$(" + n + " -w 0 logo.png)\""},
			{Description: "Encode a short string to see what it looks like:", Bash: "printfx '" + t.Example + "' | " + n,
				PowerShell: "printfx '" + t.Example + "' | " + n, Output: t.Encoded},
			{Description: "Decode a value you found in a config file or an API response:", Bash: "echox " + t.Encoded + " | " + n + " -d",
				PowerShell: "echox " + t.Encoded + " | " + n + " -d", Output: t.Example},
			{Description: "Store a binary key in a CI secret, which only takes one line of text:", Bash: n + " -w 0 deploy-key.bin > deploy-key.txt",
				PowerShell: n + " -w 0 deploy-key.bin > deploy-key.txt"},
			{Description: "Turn that secret back into the key file in a CI job:", Bash: "printfx '%s' \"$DEPLOY_KEY\" | " + n + " -d > deploy-key.bin",
				PowerShell: "printfx '%s' $env:DEPLOY_KEY | " + n + " -d > deploy-key.bin"},
			{Description: "Wrap at 64 characters, as PEM certificates are:", Bash: n + " -w 64 cert.der",
				PowerShell: n + " -w 64 cert.der"},
			{Description: "Decode text pasted from an email, where line breaks and stray characters crept in:", Bash: n + " -d -i pasted.txt > attachment.pdf",
				PowerShell: n + " -d -i pasted.txt > attachment.pdf"},
			{Description: "Send a binary file through a chat or a terminal that only carries text:", Bash: n + " report.pdf > report.txt",
				PowerShell: n + " report.pdf > report.txt"},
			{Description: "Make sure a round trip gives back the same bytes:", Bash: n + " file.bin | " + n + " -d > copy.bin && cksumx file.bin copy.bin",
				PowerShell: n + " file.bin | " + n + " -d > copy.bin; cksumx file.bin copy.bin"},
			{Description: "Reject a payload that is not valid " + e + " before using it:", Bash: n + " -d payload.txt > /dev/null && echox valid",
				PowerShell: n + " -d payload.txt > $null; if ($LASTEXITCODE -eq 0) { 'valid' }"},
			{Description: "Encode in the portable gnux shell, the same on every OS:", Bash: "gnux shell -c '" + n + " -w 0 key.bin > key.txt'",
				PowerShell: "gnux shell -c '" + n + " -w 0 key.bin > key.txt'"},
		},
	}
}
