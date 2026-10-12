package tr_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tr"
)

// run executes the engine against in-memory streams.
func run(opts tr.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = tr.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestTranslate(t *testing.T) {
	tests := []struct {
		name  string
		opts  tr.Options
		input string
		sets  []string
		want  string
	}{
		{"single bytes", tr.Options{}, "hello", []string{"lo", "01"}, "he001"},
		{"range", tr.Options{}, "hello world", []string{"a-z", "A-Z"}, "HELLO WORLD"},
		{"case classes", tr.Options{}, "Hello World", []string{"[:lower:]", "[:upper:]"}, "HELLO WORLD"},
		{"case classes reversed", tr.Options{}, "Hello World", []string{"[:upper:]", "[:lower:]"}, "hello world"},
		{"class in set1 only", tr.Options{}, "a1b2", []string{"[:digit:]", "x"}, "axbx"},
		{"set2 padded with its last byte", tr.Options{}, "abcde", []string{"a-e", "xy"}, "xyyyy"},
		{"truncate set1", tr.Options{Truncate: true}, "abcde", []string{"a-e", "xy"}, "xycde"},
		{"later mapping wins", tr.Options{}, "a", []string{"aa", "xy"}, "y"},
		{"complement", tr.Options{Complement: true}, "a1b2\n", []string{"a-z", "_"}, "a_b__"},
		{"complement keeps newlines", tr.Options{Complement: true}, "a-b c\n", []string{"a-z\\n", "."}, "a.b.c\n"},
		{"escapes", tr.Options{}, "a\tb", []string{"\\t", "\\n"}, "a\nb"},
		{"octal escape", tr.Options{}, "a b", []string{"\\040", "_"}, "a_b"},
		{"escaped backslash", tr.Options{}, `a\b`, []string{`\\`, "/"}, "a/b"},
		{"escaped dash is literal", tr.Options{}, "a-c", []string{`a\-c`, "xyz"}, "xyz"},
		{"trailing dash is literal", tr.Options{}, "a-", []string{"a-", "xy"}, "xy"},
		{"repeat fill", tr.Options{}, "abcdef", []string{"a-f", "[x*]"}, "xxxxxx"},
		{"repeat count then fill", tr.Options{}, "abcdef", []string{"a-f", "[x*2][y*]"}, "xxyyyy"},
		{"repeat octal count", tr.Options{}, "abcdefghij", []string{"a-j", "[x*010]yz"}, "xxxxxxxxyz"},
		{"equivalence class", tr.Options{}, "banana", []string{"[=a=]", "o"}, "bonono"},
		{"unclosed bracket is literal", tr.Options{}, "[a", []string{"[a", "xy"}, "xy"},
		{"translate and squeeze", tr.Options{Squeeze: true}, "aabbcc", []string{"abc", "xxy"}, "xy"},
		{"multibyte input is bytes", tr.Options{}, "é", []string{"\\303", "X"}, "X\xa9"},
		{"no change outside set1", tr.Options{}, "xyz", []string{"a", "b"}, "xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input, tt.sets...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestDeleteAndSqueeze(t *testing.T) {
	tests := []struct {
		name  string
		opts  tr.Options
		input string
		sets  []string
		want  string
	}{
		{"delete", tr.Options{Delete: true}, "hello world", []string{"lo"}, "he wrd"},
		{"delete class", tr.Options{Delete: true}, "a1b2c3", []string{"[:digit:]"}, "abc"},
		{"delete carriage returns", tr.Options{Delete: true}, "a\r\nb\r\n", []string{"\\r"}, "a\nb\n"},
		{"delete complement", tr.Options{Delete: true, Complement: true}, "a1b2\n", []string{"[:digit:]"}, "12"},
		{"squeeze", tr.Options{Squeeze: true}, "aaabbbccc  d", []string{"a-z "}, "abc d"},
		{"squeeze only the set", tr.Options{Squeeze: true}, "aabb", []string{"a"}, "abb"},
		{"squeeze blank lines", tr.Options{Squeeze: true}, "a\n\n\nb\n", []string{"\\n"}, "a\nb\n"},
		{"squeeze complement", tr.Options{Squeeze: true, Complement: true}, "aa--bb", []string{"a-z"}, "aa-bb"},
		{"delete then squeeze", tr.Options{Delete: true, Squeeze: true}, "a1  b22  c", []string{"[:digit:]", " "}, "a b c"},
		{"squeeze across reads", tr.Options{Squeeze: true}, "xx", []string{"x"}, "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.input, tt.sets...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestSqueezeAcrossReads checks that a run split between two reads is still
// squeezed.
func TestSqueezeAcrossReads(t *testing.T) {
	var out bytes.Buffer
	in := iotest.OneByteReader(strings.NewReader("aaab"))
	err := tr.New(tr.Options{Squeeze: true}, core.Streams{In: in, Out: &out, Err: &bytes.Buffer{}}).Run([]string{"a"})
	if err != nil || out.String() != "ab" {
		t.Fatalf("out = %q, err = %v", out.String(), err)
	}
}

func TestClasses(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	tests := []struct {
		class string
		want  string
	}{
		{"alnum", "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"},
		{"alpha", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"},
		{"blank", "\t "},
		{"digit", "0123456789"},
		{"lower", "abcdefghijklmnopqrstuvwxyz"},
		{"punct", "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"},
		{"space", "\t\n\v\f\r "},
		{"upper", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"xdigit", "0123456789ABCDEFabcdef"},
	}
	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			out, _, err := run(tr.Options{Delete: true, Complement: true}, string(all), "[:"+tt.class+":]")
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v", out, err)
			}
		})
	}
	for class, n := range map[string]int{"cntrl": 33, "graph": 94, "print": 95} {
		out, _, _ := run(tr.Options{Delete: true, Complement: true}, string(all), "[:"+class+":]")
		if len(out) != n {
			t.Errorf("[:%s:] has %d members, want %d", class, len(out), n)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts tr.Options
		args []string
		want string
	}{
		{"no operand", tr.Options{}, nil, "missing operand"},
		{"translate with one", tr.Options{}, []string{"a"},
			"missing operand after 'a'\nTwo strings must be given when translating."},
		{"delete and squeeze with one", tr.Options{Delete: true, Squeeze: true}, []string{"a"},
			"missing operand after 'a'\nTwo strings must be given when both deleting and squeezing repeats."},
		{"delete with two", tr.Options{Delete: true}, []string{"a", "b"},
			"extra operand 'b'\nOnly one string may be given when deleting without squeezing repeats."},
		{"three", tr.Options{}, []string{"a", "b", "c"}, "extra operand 'c'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(tt.opts, "", tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) || usage.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", err, tt.want)
			}
		})
	}
}

func TestSetErrors(t *testing.T) {
	tests := []struct {
		name string
		opts tr.Options
		args []string
		want string
	}{
		{"reverse range", tr.Options{}, []string{"z-a", "x"},
			"trx: range-endpoints of 'z-a' are in reverse collating sequence order\n"},
		{"bad class", tr.Options{Delete: true}, []string{"[:nope:]"},
			"trx: invalid character class 'nope'\n"},
		{"empty set2", tr.Options{}, []string{"a", ""},
			"trx: when not truncating set1, string2 must be non-empty\n"},
		{"class in set2", tr.Options{}, []string{"a-z", "[:digit:]"},
			"trx: when translating, the only character classes that may appear in string2 are 'upper' and 'lower'\n"},
		{"misaligned case", tr.Options{}, []string{"a-z", "[:upper:]"},
			"trx: misaligned [:upper:] and/or [:lower:] construct\n"},
		{"fill in set1", tr.Options{}, []string{"[a*]", "x"},
			"trx: the [c*] repeat construct may not appear in string1\n"},
		{"two fills", tr.Options{}, []string{"abc", "[x*][y*]"},
			"trx: only one [c*] repeat construct may appear in string2\n"},
		{"bad repeat count", tr.Options{}, []string{"abc", "[x*09]"},
			"trx: invalid repeat count '09' in [c*n] construct\n"},
		{"long equivalence", tr.Options{Delete: true}, []string{"[=ab=]"},
			"trx: ab: equivalence class operand must be a single character\n"},
		{"equivalence in set2", tr.Options{}, []string{"a", "[=b=]"},
			"trx: [=c=] expressions may not appear in string2 when translating\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "abc", tt.args...)
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if out != "" || errb != tt.want {
				t.Fatalf("stdout = %q, stderr = %q, want %q", out, errb, tt.want)
			}
		})
	}
}

func TestWarnings(t *testing.T) {
	out, errb, err := run(tr.Options{}, `a\`, `\`, "x")
	if err != nil || out != "ax" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if errb != "trx: warning: an unescaped backslash at end of string is not portable\n" {
		t.Fatalf("stderr = %q", errb)
	}

	out, errb, err = run(tr.Options{}, " 0", `\400`, "xy")
	if err != nil || out != "xy" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if !strings.Contains(errb, "ambiguous octal escape") {
		t.Fatalf("stderr = %q", errb)
	}
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (f failWriter) Write([]byte) (int, error) { return 0, f.err }

func TestWriteErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr string
		failed  bool
	}{
		{"closed pipe is quiet", os.NewSyscallError("write", syscall.EPIPE), "", false},
		{"other errors are reported", errors.New("disk full"), "trx: write error: Disk full\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errb bytes.Buffer
			s := core.Streams{In: strings.NewReader("abc"), Out: failWriter{tt.err}, Err: &errb}
			err := tr.New(tr.Options{}, s).Run([]string{"a", "b"})
			if errors.Is(err, core.ErrFailed) != tt.failed || errb.String() != tt.wantErr {
				t.Fatalf("err = %v, stderr = %q", err, errb.String())
			}
		})
	}
}

func TestReadError(t *testing.T) {
	var out, errb bytes.Buffer
	in := iotest.TimeoutReader(strings.NewReader("abc"))
	s := core.Streams{In: iotest.OneByteReader(in), Out: &out, Err: &errb}
	err := tr.New(tr.Options{}, s).Run([]string{"a", "x"})
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb.String(), "trx: read error: ") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}
