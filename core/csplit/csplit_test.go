package csplit_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/csplit"
)

func seq(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(strconv.Itoa(i) + "\n")
	}
	return b.String()
}

func run(t *testing.T, opts csplit.Options, stdin string, patterns ...string) (files map[string]string, stdout, stderr string, err error) {
	t.Helper()
	t.Chdir(t.TempDir())
	var out, errb bytes.Buffer
	err = csplit.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(append([]string{"-"}, patterns...))
	files = map[string]string{}
	entries, _ := os.ReadDir(".")
	for _, e := range entries {
		data, _ := os.ReadFile(e.Name())
		files[e.Name()] = string(data)
	}
	return files, out.String(), errb.String(), err
}

func names(files map[string]string) []string {
	var n []string
	for k := range files {
		n = append(n, k)
	}
	slices.Sort(n)
	return n
}

func TestPatterns(t *testing.T) {
	tests := []struct {
		name     string
		opts     csplit.Options
		patterns []string
		sizes    string
		files    map[string]string
	}{
		{"line number", csplit.Options{}, []string{"4"}, "6\n15\n",
			map[string]string{"xx00": "1\n2\n3\n", "xx01": seq(10)[6:]}},
		{"repeated line number", csplit.Options{}, []string{"3", "{1}"}, "4\n6\n11\n",
			map[string]string{"xx00": "1\n2\n", "xx01": "3\n4\n5\n", "xx02": "6\n7\n8\n9\n10\n"}},
		{"regexp", csplit.Options{}, []string{"/5/"}, "8\n13\n",
			map[string]string{"xx00": "1\n2\n3\n4\n", "xx01": "5\n6\n7\n8\n9\n10\n"}},
		{"regexp forever", csplit.Options{}, []string{"/[37]/", "{*}"}, "4\n8\n9\n",
			map[string]string{"xx00": "1\n2\n", "xx01": "3\n4\n5\n6\n", "xx02": "7\n8\n9\n10\n"}},
		{"skip", csplit.Options{}, []string{"%3%"}, "17\n",
			map[string]string{"xx00": "3\n4\n5\n6\n7\n8\n9\n10\n"}},
		{"positive offset", csplit.Options{}, []string{"/5/+2"}, "12\n9\n",
			map[string]string{"xx00": "1\n2\n3\n4\n5\n6\n", "xx01": "7\n8\n9\n10\n"}},
		{"negative offset", csplit.Options{}, []string{"/5/-1"}, "6\n15\n",
			map[string]string{"xx00": "1\n2\n3\n", "xx01": seq(10)[6:]}},
		{"suppress matched", csplit.Options{SuppressMatched: true}, []string{"/3/"}, "4\n15\n",
			map[string]string{"xx00": "1\n2\n", "xx01": "4\n5\n6\n7\n8\n9\n10\n"}},
		{"quiet", csplit.Options{Quiet: true}, []string{"4"}, "",
			map[string]string{"xx00": "1\n2\n3\n", "xx01": seq(10)[6:]}},
		{"empty file kept", csplit.Options{}, []string{"1"}, "0\n21\n",
			map[string]string{"xx00": "", "xx01": seq(10)}},
		{"empty file elided", csplit.Options{ElideEmpty: true}, []string{"1"}, "21\n",
			map[string]string{"xx00": seq(10)}},
		{"prefix and digits", csplit.Options{Prefix: "part", PrefixSet: true, Digits: 3}, []string{"6"}, "10\n11\n",
			map[string]string{"part000": "1\n2\n3\n4\n5\n", "part001": "6\n7\n8\n9\n10\n"}},
		{"suffix format", csplit.Options{Prefix: "ch", PrefixSet: true, SuffixFormat: "%02d.txt"}, []string{"6"}, "10\n11\n",
			map[string]string{"ch00.txt": "1\n2\n3\n4\n5\n", "ch01.txt": "6\n7\n8\n9\n10\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, out, errb, err := run(t, tt.opts, seq(10), tt.patterns...)
			if err != nil || out != tt.sizes {
				t.Fatalf("err = %v (stderr %q), stdout = %q, want %q", err, errb, out, tt.sizes)
			}
			if !slices.Equal(names(files), names(tt.files)) {
				t.Fatalf("files = %v, want %v", names(files), names(tt.files))
			}
			for name, want := range tt.files {
				if files[name] != want {
					t.Errorf("%s = %q, want %q", name, files[name], want)
				}
			}
		})
	}
}

func TestFailureRemovesFiles(t *testing.T) {
	files, out, errb, err := run(t, csplit.Options{}, seq(10), "4", "20")
	if !errors.Is(err, core.ErrFailed) || errb != "csplitx: '20': line number out of range\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if len(files) != 0 || out != "6\n" {
		t.Fatalf("files = %v, stdout = %q", names(files), out)
	}
	files, _, _, _ = run(t, csplit.Options{KeepFiles: true}, seq(10), "4", "20")
	if !slices.Equal(names(files), []string{"xx00", "xx01"}) {
		t.Fatalf("-k: files = %v", names(files))
	}
	_, _, errb, _ = run(t, csplit.Options{}, seq(10), "/5/", "{1}")
	if errb != "csplitx: '/5/': match not found on repetition 1\n" {
		t.Fatalf("repetition: %q", errb)
	}
}

func TestPatternErrors(t *testing.T) {
	tests := []struct {
		patterns []string
		msg      string
	}{
		{[]string{"x"}, "'x': invalid pattern"},
		{[]string{"0"}, "0: line number must be greater than zero"},
		{[]string{"5", "3"}, "line number '3' is smaller than preceding line number, 5"},
		{[]string{"/a"}, "/a: closing delimiter '/' missing"},
		{[]string{"/a/x"}, "'/a/x': integer expected after delimiter"},
		{[]string{"/[/"}, "'/[/': invalid regular expression: Unmatched [, [^, [:, [., or [="},
		{[]string{"/a/", "{x}"}, "'{x}': integer required between '{' and '}'"},
		{[]string{"/a/", "{3"}, "'{3': '}' is required in repeat count"},
	}
	for _, tt := range tests {
		_, _, errb, err := run(t, csplit.Options{}, seq(3), tt.patterns...)
		if !errors.Is(err, core.ErrFailed) || errb != "csplitx: "+tt.msg+"\n" {
			t.Errorf("%q: err = %v, stderr = %q", tt.patterns, err, errb)
		}
	}
	_, _, errb, _ := run(t, csplit.Options{}, seq(10), "3", "3")
	if !strings.HasPrefix(errb, "csplitx: warning: line number '3' is the same as preceding line number\n") {
		t.Fatalf("warning: %q", errb)
	}
}

func TestSuffixFormatErrors(t *testing.T) {
	for format, msg := range map[string]string{
		"x":    "missing % conversion specification in suffix",
		"%d%d": "too many % conversion specifications in suffix",
		"%":    "missing conversion specifier in suffix",
		"%s":   "invalid conversion specifier in suffix: s",
		"%#d":  "invalid flags in conversion specification: %#d",
		"%'x":  "invalid flags in conversion specification: %'x",
	} {
		_, _, errb, err := run(t, csplit.Options{SuffixFormat: format}, seq(3), "2")
		if !errors.Is(err, core.ErrFailed) || errb != "csplitx: "+msg+"\n" {
			t.Errorf("-b %q: err = %v, stderr = %q", format, err, errb)
		}
	}
}

func TestOperands(t *testing.T) {
	var usage *core.UsageError
	if err := csplit.New(csplit.Options{}, core.Discard()).Run(nil); !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("no operands: %v", err)
	}
	if err := csplit.New(csplit.Options{}, core.Discard()).Run([]string{"f"}); !errors.As(err, &usage) || usage.Msg != "missing operand after 'f'" {
		t.Fatalf("one operand: %v", err)
	}
}
