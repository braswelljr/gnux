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

// TestEncodings checks each encoding against what GNU basenc 9.7 prints for
// the same input.
func TestEncodings(t *testing.T) {
	tests := []struct {
		enc     basenc.Encoding
		in, out string
	}{
		{basenc.Base64URL, ">>?", "Pj4_\n"},
		{basenc.Base32Hex, "hello", "D1IMOR3F\n"},
		{basenc.Base16, "hello", "68656C6C6F\n"},
		{basenc.Base2MSBF, "hello", "0110100001100101011011000110110001101111\n"},
		{basenc.Base2LSBF, "hello", "0001011010100110001101100011011011110110\n"},
		{basenc.Z85, "hell", "xK#0@\n"},
		{basenc.Z85, "hellohel", "xK#0@z!JY#\n"},
	}
	for _, tt := range tests {
		out, errb, err := run(tt.enc, basenc.Options{}, tt.in)
		if err != nil || out != tt.out {
			t.Fatalf("encoding %d of %q: err=%v stderr=%q stdout=%q, want %q", tt.enc, tt.in, err, errb, out, tt.out)
		}
		back, errb, err := run(tt.enc, basenc.Options{Decode: true}, out)
		if err != nil || back != tt.in {
			t.Fatalf("decoding %d of %q: err=%v stderr=%q stdout=%q", tt.enc, out, err, errb, back)
		}
	}
}

func TestEncodingMistakes(t *testing.T) {
	tests := []struct {
		name     string
		enc      basenc.Encoding
		opts     basenc.Options
		in       string
		out, err string
	}{
		{"z85 needs whole groups", basenc.Z85, basenc.Options{}, "hello", "", "encx: invalid input (length must be multiple of 4 characters)\n"},
		{"z85 short group", basenc.Z85, basenc.Options{Decode: true}, "xK#0", "", "encx: invalid input\n"},
		{"base16 either case", basenc.Base16, basenc.Options{Decode: true}, "68656c6C6F", "hello", ""},
		{"base16 odd length", basenc.Base16, basenc.Options{Decode: true}, "68656C6C6", "hell", "encx: invalid input\n"},
		{"base16 garbage ignored", basenc.Base16, basenc.Options{Decode: true, IgnoreGarbage: true}, "68 65", "he", ""},
		{"base2 short byte", basenc.Base2MSBF, basenc.Options{Decode: true}, "0110100001", "h", "encx: invalid input\n"},
		{"base64url refuses /", basenc.Base64URL, basenc.Options{Decode: true}, "Pj4/", "", "encx: invalid input\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.enc, tt.opts, tt.in)
			if out != tt.out || errb != tt.err || (err != nil) != (tt.err != "") {
				t.Fatalf("stdout=%q stderr=%q err=%v; want %q %q", out, errb, err, tt.out, tt.err)
			}
		})
	}
}

func TestWrapBase2(t *testing.T) {
	out, _, _ := run(basenc.Base2LSBF, basenc.Options{Wrap: 10}, "abc")
	if want := "1000011001\n0001101100\n0110\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}
