package hashsum_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/hashsum"
)

// Digests of "hello\n", from Python's hashlib.
var hello = map[hashsum.Algorithm]string{
	hashsum.MD5:     "b1946ac92492d2347c6235b4d2611184",
	hashsum.SHA1:    "f572d396fae9206628714fb2ce00f72e94f2258f",
	hashsum.SHA224:  "2d6d67d91d0badcdd06cbbba1fe11538a68a37ec9c2e26457ceff12b",
	hashsum.SHA256:  "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
	hashsum.SHA384:  "1d0f284efe3edea4b9ca3bd514fa134b17eae361ccc7a1eefeff801b9bd6604e01f21f6bf249ef030599f0c218f2ba8c",
	hashsum.SHA512:  "e7c22b994c59d9cf2b48e549b1e24666636045930d3da7c1acb299d1c3b7f931f94aae41edda2c2b207a36e10f8bcb8d45223e54878f5b316e7ce3b6bc019629",
	hashsum.BLAKE2b: "f60ce482e5cc1229f39d71313171a8d9f4ca3a87d066bf4b205effb528192a75f14f3271e2c1a90e1de53f275b4d4793eef2f5e31ea90d2ce29d2e481c36435f",
}

const b2hello256 = "93becc6e9882211c3ec3708c95bcd69baab7bb59c7f4bc84ce637b88a534b783"

func run(alg hashsum.Algorithm, opts hashsum.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = hashsum.New("sumx", alg, opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDigestOfStdin(t *testing.T) {
	for alg, want := range hello {
		t.Run(alg.String(), func(t *testing.T) {
			out, errb, err := run(alg, hashsum.Options{}, "hello\n")
			if err != nil || errb != "" {
				t.Fatalf("err=%v stderr=%q", err, errb)
			}
			if out != want+"  -\n" {
				t.Fatalf("stdout = %q, want %q", out, want+"  -\n")
			}
		})
	}
}

func TestFormats(t *testing.T) {
	dir := t.TempDir()
	f := write(t, dir, "f", "hello\n")
	md5 := hello[hashsum.MD5]
	tests := []struct {
		name string
		alg  hashsum.Algorithm
		opts hashsum.Options
		want string
	}{
		{"text", hashsum.MD5, hashsum.Options{}, md5 + "  " + f + "\n"},
		{"binary", hashsum.MD5, hashsum.Options{Binary: true}, md5 + " *" + f + "\n"},
		{"tag", hashsum.SHA256, hashsum.Options{Tag: true}, "SHA256 (" + f + ") = " + hello[hashsum.SHA256] + "\n"},
		{"zero", hashsum.MD5, hashsum.Options{Zero: true}, md5 + "  " + f + "\x00"},
		{"base64", hashsum.MD5, hashsum.Options{Base64: true}, "sZRqySSS0jR8YjW00mERhA==  " + f + "\n"},
		{"blake2b length", hashsum.BLAKE2b, hashsum.Options{Length: 256}, b2hello256 + "  " + f + "\n"},
		{"blake2b length tag", hashsum.BLAKE2b, hashsum.Options{Length: 256, Tag: true}, "BLAKE2b-256 (" + f + ") = " + b2hello256 + "\n"},
		{"blake2b full tag", hashsum.BLAKE2b, hashsum.Options{Length: 512, Tag: true}, "BLAKE2b (" + f + ") = " + hello[hashsum.BLAKE2b] + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.alg, tt.opts, "", f)
			if err != nil || errb != "" {
				t.Fatalf("err=%v stderr=%q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestRaw(t *testing.T) {
	out, _, err := run(hashsum.MD5, hashsum.Options{Raw: true}, "hello\n")
	if err != nil || len(out) != 16 {
		t.Fatalf("err=%v, %d raw bytes, want 16", err, len(out))
	}
}

func TestMissingFileContinues(t *testing.T) {
	dir := t.TempDir()
	f := write(t, dir, "f", "hello\n")
	missing := filepath.Join(dir, "missing")
	out, errb, err := run(hashsum.MD5, hashsum.Options{}, "", missing, dir, f)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "sumx: " + missing + ": No such file or directory\nsumx: " + dir + ": Is a directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if out != hello[hashsum.MD5]+"  "+f+"\n" {
		t.Fatalf("stdout = %q", out)
	}
}

func TestOptionErrors(t *testing.T) {
	tests := []struct {
		opts hashsum.Options
		msg  string
	}{
		{hashsum.Options{Check: true, Binary: true}, "the --binary and --text options are meaningless when verifying checksums"},
		{hashsum.Options{Check: true, Tag: true}, "the --tag option is meaningless when verifying checksums"},
		{hashsum.Options{Tag: true, Text: true}, "--tag does not support --text mode"},
		{hashsum.Options{Status: true}, "the --status option is meaningful only when verifying checksums"},
		{hashsum.Options{IgnoreMissing: true}, "the --ignore-missing option is meaningful only when verifying checksums"},
		{hashsum.Options{Strict: true}, "the --strict option is meaningful only when verifying checksums"},
		{hashsum.Options{Warn: true}, "the --warn option is meaningful only when verifying checksums"},
		{hashsum.Options{Quiet: true}, "the --quiet option is meaningful only when verifying checksums"},
	}
	for _, tt := range tests {
		_, _, err := run(hashsum.MD5, tt.opts, "")
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("%+v: err = %v, want usage error %q", tt.opts, err, tt.msg)
		}
	}
}

func TestBadLength(t *testing.T) {
	for l, want := range map[int]string{
		13:  "sumx: invalid length: '13'\nsumx: length is not a multiple of 8\n",
		520: "sumx: invalid length: '520'\nsumx: maximum digest length for 'BLAKE2b' is 512 bits\n",
	} {
		_, errb, err := run(hashsum.BLAKE2b, hashsum.Options{Length: l}, "")
		if !errors.Is(err, core.ErrFailed) || errb != want {
			t.Errorf("-l %d: err=%v stderr=%q, want %q", l, err, errb, want)
		}
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	good := write(t, dir, "good", "hello\n")
	bad := write(t, dir, "bad", "changed\n")
	missing := filepath.Join(dir, "missing")
	md5 := hello[hashsum.MD5]

	list := write(t, dir, "list",
		md5+"  "+good+"\n"+
			"MD5 ("+good+") = "+md5+"\n"+
			md5+" *"+bad+"\n"+
			md5+"  "+missing+"\n"+
			"not a checksum line\n"+
			"# a comment\n")

	out, errb, err := run(hashsum.MD5, hashsum.Options{Check: true}, "", list)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	wantOut := good + ": OK\n" + good + ": OK\n" + bad + ": FAILED\n" + missing + ": FAILED open or read\n"
	if out != wantOut {
		t.Fatalf("stdout = %q, want %q", out, wantOut)
	}
	wantErr := "sumx: " + missing + ": No such file or directory\n" +
		"sumx: WARNING: 1 line is improperly formatted\n" +
		"sumx: WARNING: 1 listed file could not be read\n" +
		"sumx: WARNING: 1 computed checksum did NOT match\n"
	if errb != wantErr {
		t.Fatalf("stderr = %q, want %q", errb, wantErr)
	}
}

func TestCheckModes(t *testing.T) {
	dir := t.TempDir()
	good := write(t, dir, "good", "hello\n")
	missing := filepath.Join(dir, "missing")
	md5 := hello[hashsum.MD5]
	okList := write(t, dir, "ok", md5+"  "+good+"\n")
	withJunk := write(t, dir, "junk", md5+"  "+good+"\nxyz\n")
	withMissing := write(t, dir, "missing-list", md5+"  "+good+"\n"+md5+"  "+missing+"\n")
	onlyMissing := write(t, dir, "only-missing", md5+"  "+missing+"\n")
	empty := write(t, dir, "empty", "junk\n")

	tests := []struct {
		name          string
		opts          hashsum.Options
		list          string
		fail          bool
		stdout, errEq string
	}{
		{"quiet hides OK", hashsum.Options{Check: true, Quiet: true}, okList, false, "", ""},
		{"status prints nothing", hashsum.Options{Check: true, Status: true}, withJunk, false, "", ""},
		{"strict fails on junk", hashsum.Options{Check: true, Strict: true}, withJunk, true, good + ": OK\n",
			"sumx: WARNING: 1 line is improperly formatted\n"},
		{"warn names the line", hashsum.Options{Check: true, Warn: true}, withJunk, false, good + ": OK\n",
			"sumx: " + withJunk + ": 2: improperly formatted MD5 checksum line\nsumx: WARNING: 1 line is improperly formatted\n"},
		{"ignore missing", hashsum.Options{Check: true, IgnoreMissing: true}, withMissing, false, good + ": OK\n", ""},
		{"ignore missing, nothing verified", hashsum.Options{Check: true, IgnoreMissing: true}, onlyMissing, true, "",
			"sumx: " + onlyMissing + ": no file was verified\n"},
		{"no proper lines", hashsum.Options{Check: true}, empty, true, "",
			"sumx: " + empty + ": no properly formatted checksum lines found\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(hashsum.MD5, tt.opts, "", tt.list)
			if (err != nil) != tt.fail {
				t.Fatalf("err = %v, want failure %v", err, tt.fail)
			}
			if out != tt.stdout || errb != tt.errEq {
				t.Fatalf("stdout=%q stderr=%q, want %q and %q", out, errb, tt.stdout, tt.errEq)
			}
		})
	}
}

func TestCheckFromStdinAndAuto(t *testing.T) {
	dir := t.TempDir()
	good := write(t, dir, "good", "hello\n")
	lines := "SHA256 (" + good + ") = " + hello[hashsum.SHA256] + "\n" +
		"BLAKE2b-256 (" + good + ") = " + b2hello256 + "\n" +
		hello[hashsum.MD5] + "  " + good + "\n" // untagged: not valid without an algorithm
	out, errb, err := run(hashsum.Auto, hashsum.Options{Check: true}, lines)
	if err != nil {
		t.Fatalf("err = %v (stderr %q)", err, errb)
	}
	if out != good+": OK\n"+good+": OK\n" || errb != "sumx: WARNING: 1 line is improperly formatted\n" {
		t.Fatalf("stdout=%q stderr=%q", out, errb)
	}

	_, errb, _ = run(hashsum.MD5, hashsum.Options{Check: true}, "junk\n")
	if errb != "sumx: standard input: no properly formatted checksum lines found\n" {
		t.Fatalf("stderr = %q", errb)
	}
}

func TestCheckBlake2bInfersLength(t *testing.T) {
	dir := t.TempDir()
	good := write(t, dir, "good", "hello\n")
	out, errb, err := run(hashsum.BLAKE2b, hashsum.Options{Check: true}, b2hello256+"  "+good+"\n")
	if err != nil || out != good+": OK\n" {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, out, errb)
	}
}

func TestParseTag(t *testing.T) {
	for tag, want := range map[string]int{"MD5": 128, "SHA512": 512, "BLAKE2b": 512, "BLAKE2b-64": 64} {
		if _, bits, ok := hashsum.ParseTag(tag); !ok || bits != want {
			t.Errorf("ParseTag(%q) = %d, %v", tag, bits, ok)
		}
	}
	for _, tag := range []string{"md5", "BLAKE2b-7", "BLAKE2b-1024", "SHA3"} {
		if _, _, ok := hashsum.ParseTag(tag); ok {
			t.Errorf("ParseTag(%q) accepted", tag)
		}
	}
}
