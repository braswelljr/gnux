package uniq_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uniq"
)

func pretty(t *testing.T, opts uniq.Options, input string, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	u := uniq.New(opts, core.Streams{In: strings.NewReader(input), Out: &out, Err: &errb})
	u.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	if err := u.Run(args); err != nil {
		t.Fatalf("Run: %v (%s)", err, errb.String())
	}
	return out.String()
}

func TestCountView(t *testing.T) {
	input := "200\n200\n200\n200\n404\n404\n500\n"
	got := pretty(t, uniq.Options{Count: true, Pretty: true}, input)
	want := "      4 #################### 200\n" +
		"      2 ##########.......... 404\n" +
		"      1 #####............... 500\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}

	// Colored: the count on the gradient by its share; the text is GNU's
	// with the bar added.
	colored := pretty(t, uniq.Options{Count: true, Pretty: true, Color: true}, input)
	if !strings.Contains(colored, "\x1b[") || strip(colored) != want {
		t.Fatalf("colored: %q", colored)
	}

	// Without -c, with -z, or to an OUTPUT file, GNU's bytes.
	if got := pretty(t, uniq.Options{Pretty: true, Color: true}, input); got != "200\n404\n500\n" {
		t.Fatalf("no -c: %q", got)
	}
	if got := pretty(t, uniq.Options{Count: true, ZeroTerminated: true, Pretty: true}, "a\x00a\x00"); got != "      2 a\x00" {
		t.Fatalf("-z: %q", got)
	}
}

func TestCountViewStreamsPastTheLimit(t *testing.T) {
	// 4096 groups are held; the scale is their largest count, and a larger
	// count after them gets a full bar.
	var in strings.Builder
	in.WriteString("big\nbig\n")
	for i := range 4095 {
		fmt.Fprintf(&in, "line%d\n", i)
	}
	in.WriteString("huge\nhuge\nhuge\nhuge\n")
	got := strings.Split(pretty(t, uniq.Options{Count: true, Pretty: true}, in.String()), "\n")
	if got[0] != "      2 #################### big" || got[1] != "      1 ##########.......... line0" {
		t.Fatalf("first rows: %q", got[:2])
	}
	if last := got[len(got)-2]; last != "      4 #################### huge" {
		t.Fatalf("last row: %q", last)
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
