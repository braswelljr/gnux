// Package hashsum is the engine shared by the GNU message digest tools:
// md5sumx, sha1sumx, sha224sumx, sha256sumx, sha384sumx, sha512sumx, b2sumx,
// and the digest algorithms of cksumx. It prints a digest line per file, or,
// with Check, reads such lines back and verifies each file against them.
//
// Each tool's own package (core/md5sum, core/sha256sum, ...) fixes the
// algorithm and the diagnostic name; this package holds the behavior.
package hashsum

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/blake2b"

	"github.com/braswelljr/gnux/core"
)

// Algorithm is a message digest algorithm.
type Algorithm int

const (
	// Auto detects the algorithm from each tagged line; it is valid only when
	// checking (cksumx -c without -a).
	Auto Algorithm = iota
	// MD5 is RFC 1321, 128 bits.
	MD5
	// SHA1 is FIPS 180-4 SHA-1, 160 bits.
	SHA1
	// SHA224 is FIPS 180-4 SHA-224.
	SHA224
	// SHA256 is FIPS 180-4 SHA-256.
	SHA256
	// SHA384 is FIPS 180-4 SHA-384.
	SHA384
	// SHA512 is FIPS 180-4 SHA-512.
	SHA512
	// BLAKE2b is RFC 7693 BLAKE2b, 512 bits unless a length is given.
	BLAKE2b
)

var (
	tags = [...]string{Auto: "", MD5: "MD5", SHA1: "SHA1", SHA224: "SHA224", SHA256: "SHA256",
		SHA384: "SHA384", SHA512: "SHA512", BLAKE2b: "BLAKE2b"}
	sizes = [...]int{Auto: 0, MD5: 128, SHA1: 160, SHA224: 224, SHA256: 256, SHA384: 384,
		SHA512: 512, BLAKE2b: 512}
)

// String returns the algorithm's tag, as written in BSD-style lines ("SHA256").
func (a Algorithm) String() string { return tags[a] }

// Bits returns the algorithm's default digest length in bits.
func (a Algorithm) Bits() int { return sizes[a] }

// NewHash returns a fresh hash. bits is the digest length, used by BLAKE2b
// only; 0 means the algorithm's default. It returns nil for Auto.
func (a Algorithm) NewHash(bits int) hash.Hash {
	switch a {
	case MD5:
		return md5.New()
	case SHA1:
		return sha1.New()
	case SHA224:
		return sha256.New224()
	case SHA256:
		return sha256.New()
	case SHA384:
		return sha512.New384()
	case SHA512:
		return sha512.New()
	case BLAKE2b:
		if bits == 0 {
			bits = 512
		}
		h, err := blake2b.New(bits/8, nil)
		if err != nil {
			return nil
		}
		return h
	case Auto:
	}
	return nil
}

// ParseTag reads a BSD-style tag ("SHA256", "BLAKE2b", "BLAKE2b-256") into
// its algorithm and digest length in bits.
func ParseTag(tag string) (Algorithm, int, bool) {
	if rest, ok := strings.CutPrefix(tag, "BLAKE2b-"); ok {
		n, err := strconv.Atoi(rest)
		if err != nil || n <= 0 || n%8 != 0 || n > 512 {
			return Auto, 0, false
		}
		return BLAKE2b, n, true
	}
	for a := MD5; a <= BLAKE2b; a++ {
		if tags[a] == tag {
			return a, sizes[a], true
		}
	}
	return Auto, 0, false
}

// TagName returns the tag a digest of bits bits is written with: the
// algorithm's name, with the length appended for a shortened BLAKE2b.
func TagName(a Algorithm, bits int) string {
	if a == BLAKE2b && bits != 0 && bits != 512 {
		return fmt.Sprintf("BLAKE2b-%d", bits)
	}
	return a.String()
}

// Options mirrors the flags the GNU digest tools share. The zero value prints
// one untagged, hex, text-mode line per file.
type Options struct {
	// Binary (-b) marks each line with '*', GNU's binary-mode indicator. The
	// digest is the same: files are always read as bytes.
	Binary bool
	// Text (-t) is the default text mode, given explicitly; it only matters
	// for the option checks (it cannot be combined with Tag or Check).
	Text bool
	// Tag (--tag) prints BSD-style lines: "SHA256 (file) = digest".
	Tag bool
	// Zero (-z) ends each output line with NUL and disables name escaping.
	Zero bool
	// Length (-l) is the BLAKE2b digest length in bits, a multiple of 8 up to
	// 512; 0 means 512.
	Length int
	// Base64 prints digests in base64 rather than hex (cksumx --base64).
	Base64 bool
	// Raw prints the bare binary digest (cksumx --raw).
	Raw bool

	// Check (-c) reads digest lines from the operands and verifies each file.
	Check bool
	// IgnoreMissing (--ignore-missing) skips listed files that do not exist.
	IgnoreMissing bool
	// Quiet (--quiet) does not print OK for each verified file.
	Quiet bool
	// Status (--status) prints nothing; the exit status tells the result.
	Status bool
	// Strict (--strict) fails when a line is improperly formatted.
	Strict bool
	// Warn (-w) diagnoses each improperly formatted line.
	Warn bool
}

// Sum is the engine. Construct it with New.
type Sum struct {
	Options

	prog    string
	alg     Algorithm
	streams core.Streams
	diag    *core.Diag
}

// New builds an engine that digests with alg and prefixes its diagnostics
// with prog (the tool name).
func New(prog string, alg Algorithm, opts Options, s core.Streams) *Sum {
	return &Sum{Options: opts, prog: prog, alg: alg, streams: s, diag: core.NewDiag(prog, s)}
}

// Run digests each operand, or with Check verifies the files the operands
// list. No operand, or "-", is standard input. It returns a *core.UsageError
// for an invalid option combination, core.ErrFailed when anything failed or
// did not match, and nil otherwise.
func (h *Sum) Run(args []string) error {
	if err := h.validate(); err != nil {
		return err
	}
	if len(args) == 0 {
		args = []string{"-"}
	}
	if h.Check {
		return h.check(args)
	}
	if h.alg == Auto {
		return core.Usagef("an algorithm is required to compute a digest")
	}
	for _, name := range args {
		sum, err := h.digest(h.alg, h.Length, name)
		if err != nil {
			h.diag.Errorf("%s: %s", name, core.ErrText(err))
			continue
		}
		if h.Raw {
			_, _ = h.streams.Out.Write(sum)
			continue
		}
		_, _ = io.WriteString(h.streams.Out, h.line(h.alg, h.Length, sum, name))
	}
	return h.diag.Result()
}

func (h *Sum) validate() error {
	o := h.Options
	if o.Check {
		if o.Binary || o.Text {
			return core.Usagef("the --binary and --text options are meaningless when verifying checksums")
		}
		if o.Tag {
			return core.Usagef("the --tag option is meaningless when verifying checksums")
		}
	} else {
		for _, f := range []struct {
			on   bool
			name string
		}{
			{o.IgnoreMissing, "ignore-missing"}, {o.Status, "status"}, {o.Warn, "warn"},
			{o.Quiet, "quiet"}, {o.Strict, "strict"},
		} {
			if f.on {
				return core.Usagef("the --%s option is meaningful only when verifying checksums", f.name)
			}
		}
	}
	if o.Tag && o.Text {
		return core.Usagef("--tag does not support --text mode")
	}
	switch {
	case o.Length == 0:
	case o.Length < 0 || o.Length%8 != 0:
		h.diag.Errorf("invalid length: '%d'", o.Length)
		h.diag.Errorf("length is not a multiple of 8")
		return core.ErrFailed
	case o.Length > 512:
		h.diag.Errorf("invalid length: '%d'", o.Length)
		h.diag.Errorf("maximum digest length for 'BLAKE2b' is 512 bits")
		return core.ErrFailed
	}
	return nil
}

// digest hashes the file called name ("-" is standard input).
func (h *Sum) digest(alg Algorithm, bits int, name string) ([]byte, error) {
	r := h.streams.In
	if name != "-" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			return nil, syscall.EISDIR
		}
		r = f
	}
	sum := alg.NewHash(bits)
	if _, err := io.Copy(sum, r); err != nil {
		return nil, err
	}
	return sum.Sum(nil), nil
}

// line formats one output line for a digest of name.
func (h *Sum) line(alg Algorithm, bits int, sum []byte, name string) string {
	d := hex.EncodeToString(sum)
	if h.Base64 {
		d = base64.StdEncoding.EncodeToString(sum)
	}
	prefix, end := "", "\n"
	if h.Zero {
		end = "\x00"
	} else if needsEscape(name) {
		prefix, name = `\`, escape(name)
	}
	if h.Tag {
		return prefix + TagName(alg, bits) + " (" + name + ") = " + d + end
	}
	mark := " "
	if h.Binary {
		mark = "*"
	}
	return prefix + d + " " + mark + name + end
}

// needsEscape reports whether a file name must be escaped on its line: a
// newline or carriage return would make the line ambiguous, and on Unix a
// backslash too (escapeTriggers, per platform).
func needsEscape(name string) bool {
	return strings.ContainsAny(name, escapeTriggers)
}

var escaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`)

func escape(name string) string { return escaper.Replace(name) }

// unescape reverses escape, reporting false for an unknown escape.
func unescape(s string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			return "", false
		}
		i++
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		default:
			return "", false
		}
	}
	return b.String(), true
}

// entry is one parsed checksum line.
type entry struct {
	alg  Algorithm
	bits int
	sum  []byte
	name string
}

// checkResult tallies one checksum list.
type checkResult struct {
	proper, improper, unreadable, mismatched, verified int
}

func (h *Sum) check(lists []string) error {
	failed := false
	for _, list := range lists {
		if !h.checkList(list) {
			failed = true
		}
	}
	if failed || h.diag.Failed() {
		return core.ErrFailed
	}
	return nil
}

// checkList verifies every file one list names, reporting whether all went
// well.
func (h *Sum) checkList(list string) bool {
	display := list
	r := h.streams.In
	if list == "-" {
		display = "standard input"
	} else {
		f, err := os.Open(list)
		if err != nil {
			h.diag.Errorf("%s: %s", list, core.ErrText(err))
			return false
		}
		defer f.Close()
		r = f
	}

	var res checkResult
	br := core.NewReader(r)
	for lineNo := 1; ; lineNo++ {
		raw, err := br.ReadString('\n')
		if raw == "" && err != nil {
			if !errors.Is(err, io.EOF) {
				h.diag.Errorf("%s: read error: %s", display, core.ErrText(err))
				return false
			}
			break
		}
		line := strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
		line = strings.TrimLeft(line, " \t")
		if line == "" || line[0] == '#' {
			continue
		}
		e, ok := h.parse(line)
		if !ok {
			res.improper++
			if h.Warn {
				h.diag.Warnf("%s: %d: improperly formatted %schecksum line", display, lineNo, h.algLabel())
			}
			continue
		}
		res.proper++
		h.verify(e, &res)
	}

	ok := true
	switch {
	case res.proper == 0:
		h.diag.Errorf("%s: no properly formatted checksum lines found", display)
		return false
	case !h.Status:
		if res.improper > 0 {
			h.diag.Warnf("WARNING: %d %s improperly formatted", res.improper, plural(res.improper, "line is", "lines are"))
		}
		if res.unreadable > 0 {
			h.diag.Warnf("WARNING: %d listed %s could not be read", res.unreadable, plural(res.unreadable, "file", "files"))
		}
		if res.mismatched > 0 {
			h.diag.Warnf("WARNING: %d computed %s did NOT match", res.mismatched, plural(res.mismatched, "checksum", "checksums"))
		}
	}
	// GNU's condition: with --ignore-missing, at least one file must match.
	if h.IgnoreMissing && res.verified-res.mismatched == 0 {
		h.diag.Errorf("%s: no file was verified", display)
		ok = false
	}
	if res.mismatched > 0 || res.unreadable > 0 || (h.Strict && res.improper > 0) {
		ok = false
	}
	return ok
}

func (h *Sum) algLabel() string {
	if h.alg == Auto {
		return ""
	}
	return h.alg.String() + " "
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// verify digests one listed file and reports the result.
func (h *Sum) verify(e entry, res *checkResult) {
	shown := e.name
	prefix := ""
	if needsEscape(shown) {
		prefix, shown = `\`, escape(shown)
	}
	sum, err := h.digest(e.alg, e.bits, e.name)
	switch {
	case err != nil:
		if h.IgnoreMissing && errors.Is(err, fs.ErrNotExist) {
			return
		}
		res.unreadable++
		h.diag.Warnf("%s: %s", e.name, core.ErrText(err))
		if !h.Status {
			fmt.Fprintf(h.streams.Out, "%s%s: FAILED open or read\n", prefix, shown)
		}
	case bytes.Equal(sum, e.sum):
		res.verified++
		if !h.Quiet && !h.Status {
			fmt.Fprintf(h.streams.Out, "%s%s: OK\n", prefix, shown)
		}
	default:
		res.verified++
		res.mismatched++
		if !h.Status {
			fmt.Fprintf(h.streams.Out, "%s%s: FAILED\n", prefix, shown)
		}
	}
}

// parse reads one checksum line, tagged ("SHA256 (name) = digest") or
// untagged ("digest  name", "digest *name"), with an optional leading
// backslash marking an escaped name.
func (h *Sum) parse(line string) (entry, bool) {
	escaped := false
	if strings.HasPrefix(line, `\`) {
		escaped, line = true, line[1:]
	}
	e, ok := h.parseTagged(line)
	if !ok {
		e, ok = h.parseUntagged(line)
	}
	if !ok || e.name == "" {
		return entry{}, false
	}
	if escaped {
		name, ok := unescape(e.name)
		if !ok {
			return entry{}, false
		}
		e.name = name
	}
	return e, true
}

func (h *Sum) parseTagged(line string) (entry, bool) {
	open := strings.Index(line, " (")
	if open <= 0 {
		return entry{}, false
	}
	alg, bits, ok := ParseTag(line[:open])
	if !ok {
		return entry{}, false
	}
	if h.alg != Auto && alg != h.alg {
		return entry{}, false
	}
	if h.alg == BLAKE2b && h.Length != 0 && bits != h.Length {
		return entry{}, false
	}
	rest := line[open+2:]
	closeAt := strings.LastIndex(rest, ") = ")
	if closeAt < 0 {
		return entry{}, false
	}
	sum, ok := decodeDigest(rest[closeAt+4:], bits)
	if !ok {
		return entry{}, false
	}
	return entry{alg: alg, bits: bits, sum: sum, name: rest[:closeAt]}, true
}

func (h *Sum) parseUntagged(line string) (entry, bool) {
	if h.alg == Auto {
		return entry{}, false
	}
	sp := strings.IndexByte(line, ' ')
	if sp <= 0 || sp+2 > len(line) || (line[sp+1] != ' ' && line[sp+1] != '*') {
		return entry{}, false
	}
	d := line[:sp]
	bits := h.alg.Bits()
	if h.alg == BLAKE2b {
		switch {
		case h.Length != 0:
			bits = h.Length
		case isHex(d):
			bits = len(d) * 4
			if bits%8 != 0 || bits > 512 {
				return entry{}, false
			}
		}
	}
	sum, ok := decodeDigest(d, bits)
	if !ok {
		return entry{}, false
	}
	return entry{alg: h.alg, bits: bits, sum: sum, name: line[sp+2:]}, true
}

// decodeDigest reads a digest of bits bits written in hex or base64.
func decodeDigest(s string, bits int) ([]byte, bool) {
	if len(s) == bits/4 && isHex(s) {
		b, err := hex.DecodeString(s)
		return b, err == nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != bits/8 || base64.StdEncoding.EncodeToString(b) != s {
		return nil, false
	}
	return b, true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
