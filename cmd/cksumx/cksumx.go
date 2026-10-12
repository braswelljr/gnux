// Package cksumx wires the cksumx command (GNU cksum) to its engine.
package cksumx

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cksum"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the cksumx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "cksum",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for cksumx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    cksum.Name,
		SeeAlso: []string{"md5sumx", "sha256sumx", "b2sumx", "wcx"},
		Use:     cksum.Name + " [OPTION]... [FILE]...",
		Short:   "Print or check checksums and byte counts.",
		Long: "cksumx prints a checksum and the byte count of each FILE. With no FILE, or when " +
			"FILE is -, it reads standard input (and, with no FILE at all, prints no name). " +
			"By default it computes the POSIX CRC (the 'crc' algorithm), as plain cksum " +
			"always has. -a ALGORITHM picks another: sysv and bsd (the legacy sum " +
			"checksums, with a block count rather than a byte count), crc32b (the CRC-32 of " +
			"zlib and gzip), or the message digests md5, sha1, sha224, sha256, sha384, sha512 " +
			"and blake2b. Digests are printed as BSD-style tagged lines ('SHA256 (FILE) = " +
			"DIGEST'); --untagged prints them in the md5sumx style instead, and when both " +
			"--tag and --untagged are given, --tag wins. --base64 prints digests in base64, " +
			"--raw writes the bare binary checksum of a single input, -l BITS shortens a " +
			"blake2b digest, and -z ends lines with NUL. -c checks the files listed in " +
			"tagged or untagged digest lines, detecting the algorithm from each tagged line " +
			"when -a is not given, with --ignore-missing, --quiet, --status, --strict and -w " +
			"as in md5sumx. The legacy algorithms cannot be checked. On Windows a backslash in a file name is the path separator, so it is not escaped as " +
			"GNU escapes it on Unix; a name with a newline is still escaped.",
		Permissions: "Needs read permission on each FILE (and, with -c, on the checksum lists and " +
			"every file they name) and search permission on the directories leading to them. " +
			"Nothing is ever modified.",
		UseCases: []string{
			"Comparing a file with the CRC a POSIX system's cksum printed for it.",
			"Computing any of the supported digests with one tool and one syntax.",
			"Verifying mixed checksum lists that name their algorithm per line.",
			"Getting a gzip-compatible CRC-32 with -a crc32b.",
			"Printing a digest in base64, as package managers and SRI hashes use.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the POSIX CRC and size of a file:",
				Bash:        "cksumx file.iso",
				PowerShell:  "cksumx file.iso",
			},
			{
				Description: "Checksum standard input:",
				Bash:        "echox hello | cksumx",
				PowerShell:  "echox hello | cksumx",
			},
			{
				Description: "Print a SHA-256 digest as a tagged line:",
				Bash:        "cksumx -a sha256 file.iso",
				PowerShell:  "cksumx -a sha256 file.iso",
			},
			{
				Description: "Print it in the sha256sumx style:",
				Bash:        "cksumx -a sha256 --untagged file.iso",
				PowerShell:  "cksumx -a sha256 --untagged file.iso",
			},
			{
				Description: "Print a digest in base64 (as SRI hashes use):",
				Bash:        "cksumx -a sha384 --base64 app.js",
				PowerShell:  "cksumx -a sha384 --base64 app.js",
			},
			{
				Description: "Print a 256-bit BLAKE2b digest:",
				Bash:        "cksumx -a blake2b -l 256 file.iso",
				PowerShell:  "cksumx -a blake2b -l 256 file.iso",
			},
			{
				Description: "Print the gzip-compatible CRC-32:",
				Bash:        "cksumx -a crc32b data.bin",
				PowerShell:  "cksumx -a crc32b data.bin",
			},
			{
				Description: "Print the legacy BSD sum checksum:",
				Bash:        "cksumx -a bsd file.txt",
				PowerShell:  "cksumx -a bsd file.txt",
			},
			{
				Description: "Save tagged digests and check them later:",
				Bash:        "cksumx -a sha256 *.iso > SUMS && cksumx -c SUMS",
				PowerShell:  "cksumx -a sha256 (Get-ChildItem *.iso -Name) > SUMS; cksumx -c SUMS",
			},
			{
				Description: "Check silently and branch on the result:",
				Bash:        "cksumx -c --status SUMS && echox verified",
				PowerShell:  "cksumx -c --status SUMS; if ($LASTEXITCODE -eq 0) { 'verified' }",
			},
			{
				Description: "Write the raw binary digest of one file:",
				Bash:        "cksumx -a sha256 --raw file.iso > file.sha256.bin",
				PowerShell:  "cksumx -a sha256 --raw file.iso > file.sha256.bin",
			},
			{
				Description: "Checksum in the portable gnux shell:",
				Bash:        "gnux shell -c 'cksumx -a sha256 file.iso > file.iso.sum'",
				PowerShell:  "gnux shell -c 'cksumx -a sha256 file.iso > file.iso.sum'",
			},
		},
	}
}

// Register installs the cksumx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("algorithm", "a", "", "select the digest `TYPE` to use: sysv, bsd, crc, crc32b, md5, sha1, sha224, sha256, sha384, sha512, blake2b")
	fs.Bool("base64", false, "emit base64-encoded digests, not hexadecimal")
	fs.BoolP("check", "c", false, "read checksums from the FILEs and check them")
	fs.StringP("length", "l", "", "digest length in `bits`; must not exceed the maximum for the blake2 algorithm and must be a multiple of 8")
	fs.Bool("raw", false, "emit a raw binary digest, not hexadecimal")
	fs.Bool("tag", false, "create a BSD-style checksum (the default)")
	fs.Bool("untagged", false, "create a reversed style checksum, without digest type")
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline, and disable file name escaping")
	fs.Bool("ignore-missing", false, "don't fail or report status for missing files")
	fs.Bool("quiet", false, "don't print OK for each successfully verified file")
	fs.Bool("status", false, "don't output anything, status code shows success")
	fs.Bool("strict", false, "exit non-zero for improperly formatted checksum lines")
	fs.BoolP("warn", "w", false, "warn about improperly formatted checksum lines")
}

// fatalError is a mistake GNU reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// buildOptions folds the parsed flags into cksum.Options.
func buildOptions(fs *pflag.FlagSet) (cksum.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := cksum.Options{
		Untagged:      on("untagged") && !on("tag"),
		Base64:        on("base64"),
		Raw:           on("raw"),
		Zero:          on("zero"),
		Check:         on("check"),
		IgnoreMissing: on("ignore-missing"),
		Quiet:         on("quiet"),
		Status:        on("status"),
		Strict:        on("strict"),
		Warn:          on("warn"),
	}
	alg, _ := fs.GetString("algorithm")
	alg, err := expand(alg)
	if err != nil {
		return cksum.Options{}, err
	}
	o.Algorithm = alg
	if fs.Changed("length") {
		text, _ := fs.GetString("length")
		n, err := strconv.ParseUint(text, 10, 31)
		if err != nil {
			return cksum.Options{}, &fatalError{"invalid length: '" + text + "'"}
		}
		o.Length = int(n)
	}
	return o, nil
}

// expand completes an unambiguous abbreviation of an algorithm name, as GNU's
// argmatch does. An unknown name is left for the engine to reject.
func expand(alg string) (string, error) {
	if alg == "" {
		return "", nil
	}
	var found []string
	for _, a := range cksum.Algorithms {
		if a == alg {
			return a, nil
		}
		if strings.HasPrefix(a, alg) {
			found = append(found, a)
		}
	}
	switch len(found) {
	case 0:
		return alg, nil
	case 1:
		return found[0], nil
	}
	return "", core.Usagef("ambiguous argument '%s' for '--algorithm'\nValid arguments are: '%s'",
		alg, strings.Join(cksum.Algorithms, "', '"))
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(cksum.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return cksum.New(opts, s).Run(args)
}
