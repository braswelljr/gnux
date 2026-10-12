package od_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/od"
)

func run(opts od.Options, stdin string, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	err := od.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func le(o od.Options) od.Options { o.Endian = od.EndianLittle; return o }

// TestFormats checks output against what GNU od prints for the same input.
func TestFormats(t *testing.T) {
	tests := []struct {
		name  string
		opts  od.Options
		input string
		want  string
	}{
		{"default octal words", le(od.Options{}), "hello\n", "0000000 062550 066154 005157\n0000006\n"},
		{"characters", od.Options{Types: []string{"c"}}, "hello\n", "0000000   h   e   l   l   o  \\n\n0000006\n"},
		{"hex bytes", od.Options{Types: []string{"x1"}}, "hello\n", "0000000 68 65 6c 6c 6f 0a\n0000006\n"},
		{"signed words", le(od.Options{Types: []string{"d2"}}), "hello\n", "0000000  25960  27756   2671\n0000006\n"},
		{"unsigned bytes, no address", od.Options{Types: []string{"u1"}, Radix: od.RadixNone}, "hello\n", " 104 101 108 108 111  10\n"},
		{"named characters", od.Options{Types: []string{"a"}}, "a\n\x00 \x7f\x81", "0000000   a  nl nul  sp del soh\n0000006\n"},
		{"escapes and octal", od.Options{Types: []string{"c"}}, "\t\x00\xff", "0000000  \\t  \\0 377\n0000003\n"},
		{"decimal address", od.Options{Types: []string{"x1"}, Radix: od.RadixDecimal}, strings.Repeat("a", 17), "0000000 61 61 61 61 61 61 61 61 61 61 61 61 61 61 61 61\n0000016 61\n0000017\n"},
		{"hex address", od.Options{Types: []string{"x1"}, Radix: od.RadixHex}, "a", "000000 61\n000001\n"},
		{"big endian", od.Options{Types: []string{"x2"}, Endian: od.EndianBig}, "\x01\x02", "0000000 0102\n0000002\n"},
		{"little endian", od.Options{Types: []string{"x2"}, Endian: od.EndianLittle}, "\x01\x02", "0000000 0201\n0000002\n"},
		{"odd byte padded with zero", le(od.Options{Types: []string{"x2"}}), "\x01\x02\x03", "0000000 0201 0003\n0000003\n"},
		{"sizes by letter", le(od.Options{Types: []string{"xC"}}), "\x01", "0000000 01\n0000001\n"},
		{"eight-byte unsigned", le(od.Options{Types: []string{"u8"}}), "\xff\xff\xff\xff\xff\xff\xff\xff", "0000000 18446744073709551615\n0000010\n"},
		{"eight-byte signed", le(od.Options{Types: []string{"d8"}}), "\xff\xff\xff\xff\xff\xff\xff\xff", "0000000                   -1\n0000010\n"},
		{"float", le(od.Options{Types: []string{"fF"}}), "\x00\x00\x80\x3f", "0000000               1\n0000004\n"},
		{"double", le(od.Options{Types: []string{"f8"}}), "\x00\x00\x00\x00\x00\x00\xf8\x3f", "0000000                      1.5\n0000010\n"},
		{"empty input", od.Options{}, "", "0000000\n"},
		{"width", od.Options{Types: []string{"x1"}, Width: 4}, "abcdef", "0000000 61 62 63 64\n0000004 65 66\n0000006\n"},
		{"verbose keeps duplicates", od.Options{Types: []string{"x1"}, Width: 2, Verbose: true}, "aaaa", "0000000 61 61\n0000002 61 61\n0000004\n"},
		{"duplicates starred", od.Options{Types: []string{"x1"}, Width: 2}, "aaaaaab", "0000000 61 61\n*\n0000006 62\n0000007\n"},
		{"skip", od.Options{Types: []string{"c"}, Skip: 2}, "abcd", "0000002   c   d\n0000004\n"},
		{"limit", od.Options{Types: []string{"c"}, Limit: 2, LimitSet: true}, "abcd", "0000000   a   b\n0000002\n"},
		{"trailer", od.Options{Types: []string{"x1z"}, Radix: od.RadixHex}, "a\nc", "000000 61 0a 63" + strings.Repeat(" ", 39) + "  >a.c<\n000003\n"},
		{"two formats align", le(od.Options{Types: []string{"x1", "x2"}}), "ab", "0000000 61 62\n         6261\n0000002\n"},
		{"two formats in one type", le(od.Options{Types: []string{"x1x2"}}), "ab", "0000000 61 62\n         6261\n0000002\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("err=%v stderr=%q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout =\n%q\nwant\n%q", out, tt.want)
			}
		})
	}
}

func TestStrings(t *testing.T) {
	out, _, err := run(od.Options{Strings: 3}, "ab\x00hello\x00\x01xyz\x00tail")
	if err != nil {
		t.Fatal(err)
	}
	if want := "0000003 hello\n0000012 xyz\n0000022\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestFilesConcatenate(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	_ = os.WriteFile(a, []byte("ab"), 0o644)
	_ = os.WriteFile(b, []byte("cd"), 0o644)
	out, errb, err := run(od.Options{Types: []string{"c"}}, "", a, filepath.Join(dir, "missing"), b)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if out != "0000000   a   b   c   d\n0000004\n" {
		t.Fatalf("stdout = %q", out)
	}
	if !strings.HasPrefix(errb, "odx: "+filepath.Join(dir, "missing")+": No such file or directory") {
		t.Fatalf("stderr = %q", errb)
	}
}

func TestErrors(t *testing.T) {
	_, errb, err := run(od.Options{Skip: 10}, "abc")
	if !errors.Is(err, core.ErrFailed) || errb != "odx: cannot skip past end of combined input\n" {
		t.Fatalf("skip: err=%v stderr=%q", err, errb)
	}
	_, errb, err = run(od.Options{Radix: 'q'}, "")
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "invalid output address radix 'q'") {
		t.Fatalf("radix: err=%v stderr=%q", err, errb)
	}
	for typ, want := range map[string]string{
		"q":  "invalid character 'q' in type string 'q'",
		"d3": "invalid type string 'd3';\nthis system doesn't provide a 3-byte integral type",
		"f2": "invalid type string 'f2';\nthis system doesn't provide a 2-byte floating point type",
	} {
		var u *core.UsageError
		if _, _, err := run(od.Options{Types: []string{typ}}, ""); !errors.As(err, &u) || u.Msg != want {
			t.Fatalf("type %q: err = %v", typ, err)
		}
	}
	_, errb, _ = run(od.Options{Types: []string{"x2"}, Width: 3}, "abcd")
	if errb != "odx: warning: invalid width 3; using 2 instead\n" {
		t.Fatalf("width warning = %q", errb)
	}
}

func TestParseBytes(t *testing.T) {
	for in, want := range map[string]int64{"10": 10, "010": 8, "0x10": 16, "2k": 2048, "1b": 512, "3kB": 3000, "1M": 1 << 20, "0": 0} {
		got, err := od.ParseBytes(in)
		if err != nil || got != want {
			t.Fatalf("ParseBytes(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "x", "1q", "-1", "08"} {
		if _, err := od.ParseBytes(in); err == nil {
			t.Fatalf("ParseBytes(%q) succeeded", in)
		}
	}
}
