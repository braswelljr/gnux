package cksum_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cksum"
)

func run(opts cksum.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = cksum.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

// The expected values come from the system's POSIX cksum and from Python's
// zlib and hashlib, for "hello\n".
func TestAlgorithmsOnStdin(t *testing.T) {
	tests := []struct {
		alg, input, want string
	}{
		{"", "hello\n", "3015617425 6\n"},
		{"crc", "", "4294967295 0\n"},
		{"crc", "hello\n", "3015617425 6\n"},
		// Eight bytes at a time, with and without bytes left over.
		{"crc", "12345678", "2796357287 8\n"},
		{"crc", "123456789", "930766865 9\n"},
		{"crc", strings.Repeat("0123456789", 100) + "x", "3426585712 1001\n"},
		{"crc32b", "hello\n", "909783072 6\n"},
		{"sysv", "hello\n", "542 1\n"},
		{"bsd", "hello\n", "36979     1\n"},
		{"md5", "hello\n", "MD5 (-) = b1946ac92492d2347c6235b4d2611184\n"},
		{"sha256", "hello\n", "SHA256 (-) = 5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03\n"},
	}
	for _, tt := range tests {
		t.Run(tt.alg+"/"+tt.input, func(t *testing.T) {
			out, errb, err := run(cksum.Options{Algorithm: tt.alg}, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("err=%v stderr=%q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestNamedFiles(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	out, errb, err := run(cksum.Options{}, "", f, missing, "-")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "3015617425 6 " + f + "\n4294967295 0 -\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	if want := "cksumx: " + missing + ": No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}

	out, _, _ = run(cksum.Options{Algorithm: "bsd"}, "", f)
	if want := "36979     1 " + f + "\n"; out != want {
		t.Fatalf("bsd stdout = %q, want %q", out, want)
	}
}

func TestDigestOptions(t *testing.T) {
	tests := []struct {
		name string
		opts cksum.Options
		want string
	}{
		{"untagged", cksum.Options{Algorithm: "md5", Untagged: true}, "b1946ac92492d2347c6235b4d2611184  -\n"},
		{"base64", cksum.Options{Algorithm: "md5", Base64: true}, "MD5 (-) = sZRqySSS0jR8YjW00mERhA==\n"},
		{"blake2b length", cksum.Options{Algorithm: "blake2b", Length: 256},
			"BLAKE2b-256 (-) = 93becc6e9882211c3ec3708c95bcd69baab7bb59c7f4bc84ce637b88a534b783\n"},
		{"zero", cksum.Options{Zero: true}, "3015617425 6\x00"},
		{"raw crc", cksum.Options{Raw: true}, "\xb3\xbe\xab\x91"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "hello\n")
			if err != nil || out != tt.want {
				t.Fatalf("err=%v stderr=%q stdout=%q, want %q", err, errb, out, tt.want)
			}
		})
	}
}

func TestCheckDetectsAlgorithm(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	list := "MD5 (" + f + ") = b1946ac92492d2347c6235b4d2611184\n" +
		"SHA1 (" + f + ") = f572d396fae9206628714fb2ce00f72e94f2258f\n"
	out, errb, err := run(cksum.Options{Check: true}, list)
	if err != nil || out != f+": OK\n"+f+": OK\n" {
		t.Fatalf("err=%v stdout=%q stderr=%q", err, out, errb)
	}

	// With -a, untagged lines are accepted too.
	out, _, err = run(cksum.Options{Check: true, Algorithm: "md5"}, "b1946ac92492d2347c6235b4d2611184  "+f+"\n")
	if err != nil || out != f+": OK\n" {
		t.Fatalf("-a md5 -c: err=%v stdout=%q", err, out)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		opts cksum.Options
		args []string
		msg  string
	}{
		{cksum.Options{Algorithm: "sha3"}, nil, "invalid argument 'sha3' for '--algorithm'"},
		{cksum.Options{Length: 256}, nil, "--length is only supported with --algorithm=blake2b"},
		{cksum.Options{Check: true, Algorithm: "crc"}, nil, "--check is not supported with --algorithm=crc"},
		{cksum.Options{Raw: true}, []string{"a", "b"}, "the --raw option is not supported with multiple files"},
		{cksum.Options{Quiet: true}, nil, "the --quiet option is meaningful only when verifying checksums"},
	}
	for _, tt := range tests {
		_, _, err := run(tt.opts, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || !strings.HasPrefix(usage.Msg, tt.msg) {
			t.Errorf("%+v: err = %v, want usage error starting %q", tt.opts, err, tt.msg)
		}
	}
}
