package od_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/od"
)

func dump(t *testing.T, opts od.Options, input string) string {
	t.Helper()
	var out, errb bytes.Buffer
	o := od.New(opts, core.Streams{In: strings.NewReader(input), Out: &out, Err: &errb})
	o.Getenv = func(string) string { return "" }
	if err := o.Run(nil); err != nil {
		t.Fatalf("Run: %v (%s)", err, errb.String())
	}
	return out.String()
}

func TestByteClasses(t *testing.T) {
	input := "A \x01\xff\x00"
	opts := od.Options{Radix: od.RadixHex, Types: []string{"x1z"}, Pretty: true, Color: true}
	got := dump(t, opts, input)
	paint := func(code, s string) string { return "\x1b[" + code + "m" + s + "\x1b[0m" }
	want := paint("90", "000000") +
		" " + paint("36", "41") + " " + paint("32", "20") + " " + paint("35", "01") +
		" " + paint("33", "ff") + " " + paint("90", "00") +
		strings.Repeat(" ", 33) + "  >" +
		paint("36", "A") + paint("32", " ") + paint("35", ".") + paint("33", ".") + paint("90", ".") + "<\n" +
		paint("90", "000005") + "\n"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}

	// The text is GNU's, and without Color nothing is painted.
	plain := dump(t, od.Options{Radix: od.RadixHex, Types: []string{"x1z"}}, input)
	if strip(got) != plain || dump(t, od.Options{Radix: od.RadixHex, Types: []string{"x1z"}, Pretty: true}, input) != plain {
		t.Fatalf("text changed: %q, want %q", strip(got), plain)
	}

	// A word takes the most unusual class among its bytes: zero only when
	// both are; a repeated block's '*' is dimmed.
	got = dump(t, od.Options{Pretty: true, Color: true}, "\x00\x00a\x00"+strings.Repeat("\x00", 44))
	if !strings.Contains(got, paint("90", "000000")+" "+paint("36", "000141")) {
		t.Fatalf("words: %q", got)
	}
	if !strings.Contains(got, "\n"+paint("90", "*")+"\n") {
		t.Fatalf("no dimmed '*': %q", got)
	}
}

func strip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
