// Package digest is the command-line adapter shared by the message digest
// tools (md5sumx, sha1sumx, sha224sumx, sha256sumx, sha384sumx, sha512sumx,
// b2sumx): their flags, their documentation and the mapping onto
// hashsum.Options. Each cmd/<tool>x package describes its tool with a Tool
// and delegates to it.
package digest

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Tool describes one digest tool.
type Tool struct {
	Name      string // command name, "sha256sumx"
	GNU       string // GNU name, "sha256sum"
	Algorithm string // what it computes, "SHA-256"
	Standard  string // where it is defined, "FIPS 180-4"
	Bits      int    // digest length in bits
	// Length adds b2sum's -l/--length option.
	Length bool
	// Weak marks an algorithm that must not be relied on for security.
	Weak bool
	New  func(hashsum.Options, core.Streams) *hashsum.Sum
}

// Spec describes the tool.
func (t Tool) Spec() tool.Spec {
	return tool.Spec{
		Meta:  t.Meta(),
		GNU:   t.GNU,
		Flags: t.Register,
		Run:   t.run,
	}
}

// Register installs the tool's flags on fs.
func (t Tool) Register(fs *pflag.FlagSet) {
	fs.BoolP("binary", "b", false, "read in binary mode")
	fs.BoolP("check", "c", false, "read checksums from the FILEs and check them")
	if t.Length {
		fs.StringP("length", "l", "", "digest length in `bits`; must not exceed the maximum for the blake2 algorithm and must be a multiple of 8")
	}
	fs.Bool("tag", false, "create a BSD-style checksum")
	fs.BoolP("text", "t", false, "read in text mode (default)")
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline, and disable file name escaping")
	fs.Bool("ignore-missing", false, "don't fail or report status for missing files")
	fs.Bool("quiet", false, "don't print OK for each successfully verified file")
	fs.Bool("status", false, "don't output anything, status code shows success")
	fs.Bool("strict", false, "exit non-zero for improperly formatted checksum lines")
	fs.BoolP("warn", "w", false, "warn about improperly formatted checksum lines")
}

// fatalError is a mistake GNU reports without the --help hint.
type fatalError struct{ msgs []string }

func (e *fatalError) Error() string { return strings.Join(e.msgs, "\n") }

// BuildOptions folds the parsed flags into hashsum.Options.
func (t Tool) BuildOptions(fs *pflag.FlagSet) (hashsum.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := hashsum.Options{
		Binary:        on("binary"),
		Text:          on("text"),
		Tag:           on("tag"),
		Zero:          on("zero"),
		Check:         on("check"),
		IgnoreMissing: on("ignore-missing"),
		Quiet:         on("quiet"),
		Status:        on("status"),
		Strict:        on("strict"),
		Warn:          on("warn"),
	}
	// -b and -t only choose the mark printed before each name; GNU lets the
	// last win. Text, the default, wins in any order.
	if o.Binary && o.Text {
		o.Binary = false
	}
	if t.Length && fs.Changed("length") {
		text, _ := fs.GetString("length")
		n, err := strconv.ParseUint(text, 10, 31)
		if err != nil {
			return hashsum.Options{}, &fatalError{[]string{"invalid length: '" + text + "'"}}
		}
		o.Length = int(n)
	}
	return o, nil
}

func (t Tool) run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := t.BuildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(t.Name, s)
		for _, m := range fatal.msgs {
			d.Errorf("%s", m)
		}
		return d.Result()
	}
	return t.New(opts, s).Run(args)
}

// Meta returns the tool's documentation.
func (t Tool) Meta() docmeta.Meta {
	n := t.Name
	sums := strings.TrimSuffix(strings.TrimSuffix(n, "x"), "sum") + ".sums"
	long := n + " prints or checks " + t.Algorithm + " message digests (" + t.Standard + ", " +
		strconv.Itoa(t.Bits) + " bits). For each FILE it prints one line: the digest in hexadecimal, a space, a mode " +
		"mark (a space for text mode, '*' with -b), and the file name. With no FILE, or when " +
		"FILE is -, it reads standard input. Files are always read as bytes, so -b and -t " +
		"change only the mark; when both are given, -t wins. --tag prints BSD-style lines " +
		"instead ('ALGORITHM (FILE) = DIGEST'). A file name containing a backslash, newline " +
		"or carriage return is escaped and its line starts with a backslash, unless -z ends " +
		"lines with NUL instead. -c reads such lines (in either style) from the FILEs and " +
		"checks each listed file, printing 'FILE: OK' or 'FILE: FAILED' and a summary warning " +
		"of mismatches, unreadable files and improperly formatted lines; --quiet drops the OK " +
		"lines, --status prints nothing, --ignore-missing skips listed files that do not " +
		"exist, -w warns about each improperly formatted line and --strict fails because of " +
		"them. The exit status is 0 when everything was read (and, with -c, matched), 1 " +
		"otherwise."
	if t.Length {
		long += " -l BITS shortens the digest to BITS bits, a multiple of 8 up to 512; " +
			"tagged lines then name the length ('BLAKE2b-256')."
	}
	if t.Weak {
		long += " " + t.Algorithm + " is broken for security purposes: use it to detect " +
			"accidental corruption, not tampering, and prefer sha256sumx or b2sumx otherwise."
	}
	examples := []docmeta.Example{
		{Description: "Print the digest of a file:", Bash: n + " file.iso", PowerShell: n + " file.iso"},
		{Description: "Digest several files:", Bash: n + " *.tar.gz", PowerShell: n + " (Get-ChildItem *.tar.gz -Name)"},
		{Description: "Digest standard input:", Bash: "echox -n hello | " + n, PowerShell: "echox -n hello | " + n},
		{Description: "Save digests to a file:", Bash: n + " *.iso > " + sums, PowerShell: n + " (Get-ChildItem *.iso -Name) > " + sums},
		{Description: "Verify files against saved digests:", Bash: n + " -c " + sums, PowerShell: n + " -c " + sums},
		{Description: "Verify quietly, showing only failures:", Bash: n + " -c --quiet " + sums, PowerShell: n + " -c --quiet " + sums},
		{Description: "Verify silently and branch on the result:", Bash: n + " -c --status " + sums + " && echox verified",
			PowerShell: n + " -c --status " + sums + "; if ($LASTEXITCODE -eq 0) { 'verified' }"},
		{Description: "Verify only the listed files that exist:", Bash: n + " -c --ignore-missing " + sums, PowerShell: n + " -c --ignore-missing " + sums},
		{Description: "Fail on malformed lines in the list:", Bash: n + " -c --strict -w " + sums, PowerShell: n + " -c --strict -w " + sums},
		{Description: "Print BSD-style tagged lines:", Bash: n + " --tag file.iso", PowerShell: n + " --tag file.iso"},
		{Description: "Mark the files as read in binary mode:", Bash: n + " -b file.iso", PowerShell: n + " -b file.iso"},
		{Description: "Digest in the portable gnux shell:", Bash: "gnux shell -c '" + n + " file.iso > file.iso." + strings.TrimSuffix(n, "x") + "'",
			PowerShell: "gnux shell -c '" + n + " file.iso > file.iso." + strings.TrimSuffix(n, "x") + "'"},
	}
	if t.Length {
		examples = append(examples[:len(examples)-1],
			docmeta.Example{Description: "Print a 256-bit digest:", Bash: n + " -l 256 file.iso", PowerShell: n + " -l 256 file.iso"},
			examples[len(examples)-1])
	}
	see := []string{"cksumx"}
	for _, s := range []string{"md5sumx", "sha1sumx", "sha256sumx", "sha512sumx", "b2sumx"} {
		if s != n {
			see = append(see, s)
		}
	}
	return docmeta.Meta{
		Name:    n,
		SeeAlso: see,
		Use:     n + " [OPTION]... [FILE]...",
		Short:   "Print or check " + t.Algorithm + " (" + strconv.Itoa(t.Bits) + "-bit) checksums.",
		Long:    long,
		Permissions: "Needs read permission on each FILE (and, with -c, on the checksum lists and " +
			"every file they name) and search permission on the directories leading to them. " +
			"Nothing is ever modified.",
		UseCases: []string{
			"Verifying a download against the checksum its publisher lists.",
			"Recording digests of release artifacts so users can check them.",
			"Detecting whether files changed between two points in time.",
			"Finding duplicate files by comparing their digests.",
			"Checking a backup or a copy against the original.",
		},
		Examples: examples,
	}
}
