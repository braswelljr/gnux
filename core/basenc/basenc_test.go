package basenc_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/basenc"
)

func run(enc basenc.Encoding, opts basenc.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = basenc.New("encx", enc, opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestEncode(t *testing.T) {
	long := strings.Repeat("a", 60) // 80 base64 characters: wraps once at 76
	tests := []struct {
		name  string
		enc   basenc.Encoding
		opts  basenc.Options
		input string
		want  string
	}{
		{"base64", basenc.Base64, basenc.Options{}, "hello world", "aGVsbG8gd29ybGQ=\n"},
		{"base32", basenc.Base32, basenc.Options{}, "hello world", "NBSWY3DPEB3W64TMMQ======\n"},
		{"empty input prints nothing", basenc.Base64, basenc.Options{}, "", ""},
		{"wraps at 76", basenc.Base64, basenc.Options{}, long,
			strings.Repeat("YWFh", 19) + "\n" + "YWFh\n"},
		{"custom wrap", basenc.Base64, basenc.Options{Wrap: 4}, "hello", "aGVs\nbG8=\n"},
		{"no wrap has no newline", basenc.Base64, basenc.Options{NoWrap: true}, long, strings.Repeat("YWFh", 20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.enc, tt.opts, tt.input)
			if err != nil || errb != "" || out != tt.want {
				t.Fatalf("err=%v stderr=%q stdout=%q, want %q", err, errb, out, tt.want)
			}
		})
	}
}

func TestEncodeLargeInputMatchesOneShot(t *testing.T) {
	input := strings.Repeat("0123456789abcdef", 20000) // several read chunks
	out, _, err := run(basenc.Base64, basenc.Options{NoWrap: true}, input)
	if err != nil {
		t.Fatal(err)
	}
	dec, _, err := run(basenc.Base64, basenc.Options{Decode: true}, out)
	if err != nil || dec != input {
		t.Fatalf("round trip failed: err=%v, %d bytes back, want %d", err, len(dec), len(input))
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name  string
		enc   basenc.Encoding
		opts  basenc.Options
		input string
		want  string
		fail  bool
	}{
		{"base64", basenc.Base64, basenc.Options{Decode: true}, "aGVsbG8gd29ybGQ=\n", "hello world", false},
		{"base32", basenc.Base32, basenc.Options{Decode: true}, "NBSWY3DPEB3W64TMMQ======\n", "hello world", false},
		{"newlines ignored", basenc.Base64, basenc.Options{Decode: true}, "aGVs\nbG8=\n", "hello", false},
		{"concatenated", basenc.Base64, basenc.Options{Decode: true}, "aGk=aGk=", "hihi", false},
		{"garbage is invalid", basenc.Base64, basenc.Options{Decode: true}, "aGVs bG8=", "hel", true},
		{"garbage ignored with -i", basenc.Base64, basenc.Options{Decode: true, IgnoreGarbage: true}, "aGVs !bG8=", "hello", false},
		{"missing padding", basenc.Base64, basenc.Options{Decode: true}, "aGk", "hi", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.enc, tt.opts, tt.input)
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
			if tt.fail {
				if !errors.Is(err, core.ErrFailed) || errb != "encx: invalid input\n" {
					t.Fatalf("err=%v stderr=%q, want invalid input", err, errb)
				}
			} else if err != nil || errb != "" {
				t.Fatalf("err=%v stderr=%q", err, errb)
			}
		})
	}
}

func TestOperands(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, _, err := run(basenc.Base64, basenc.Options{}, "", f); err != nil || out != "aGk=\n" {
		t.Fatalf("file: err=%v stdout=%q", err, out)
	}

	_, _, err := run(basenc.Base64, basenc.Options{}, "", "a", "b")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("two operands: err = %v", err)
	}

	missing := filepath.Join(dir, "missing")
	_, errb, err := run(basenc.Base64, basenc.Options{}, "", missing)
	if !errors.Is(err, core.ErrFailed) || errb != "encx: "+missing+": No such file or directory\n" {
		t.Fatalf("missing: err=%v stderr=%q", err, errb)
	}
}
