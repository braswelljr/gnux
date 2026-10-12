package printf_test

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"syscall"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printf"
)

// run executes the engine against in-memory streams.
func run(args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = printf.New(printf.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"plain text", []string{"hello"}, "hello"},
		{"percent", []string{"100%%"}, "100%"},
		{"leading double dash skipped", []string{"--", "%s", "x"}, "x"},

		{"decimal", []string{"%d", "42"}, "42"},
		{"i", []string{"%i", "-7"}, "-7"},
		{"width", []string{"[%5d]", "42"}, "[   42]"},
		{"left", []string{"[%-5d]", "42"}, "[42   ]"},
		{"zero", []string{"%05d", "42"}, "00042"},
		{"zero negative", []string{"%05d", "-42"}, "-0042"},
		{"plus", []string{"%+d", "5"}, "+5"},
		{"space", []string{"% d", "5"}, " 5"},
		{"precision on integers", []string{"%.3d", "7"}, "007"},
		{"zero precision of zero", []string{"[%.0d]", "0"}, "[]"},
		{"grouping flag accepted", []string{"%'d", "1234"}, "1234"},
		{"hex", []string{"%x %X", "255", "255"}, "ff FF"},
		{"alternate hex", []string{"%#x", "255"}, "0xff"},
		{"alternate hex of zero", []string{"%#x", "0"}, "0"},
		{"octal", []string{"%o", "8"}, "10"},
		{"alternate octal", []string{"%#o", "8"}, "010"},
		{"unsigned wraps negatives", []string{"%u", "-1"}, "18446744073709551615"},
		{"unsigned ignores plus", []string{"%+u", "5"}, "5"},
		{"hex input", []string{"%d", "0x1F"}, "31"},
		{"octal input", []string{"%d", "017"}, "15"},
		{"leading space and sign", []string{"%d", "  +3"}, "3"},
		{"char constant", []string{"%d", "'A"}, "65"},
		{"double-quote char constant", []string{"%d", `"a`}, "97"},
		{"unicode char constant", []string{"%d", "'é"}, "233"},
		{"empty is zero", []string{"%d", ""}, "0"},
		{"min int64", []string{"%d", "-9223372036854775808"}, "-9223372036854775808"},

		{"float", []string{"%f", "3.5"}, "3.500000"},
		{"float precision", []string{"%.2f", "3.14159"}, "3.14"},
		{"F", []string{"%F", "1"}, "1.000000"},
		{"exponent", []string{"%e", "1234.5"}, "1.234500e+03"},
		{"exponent upper", []string{"%E", "0.00012"}, "1.200000E-04"},
		{"g", []string{"%g", "0.0001"}, "0.0001"},
		{"g large", []string{"%g", "1000000"}, "1e+06"},
		{"g precision zero", []string{"%.0g", "123"}, "1e+02"},
		{"alternate g", []string{"%#g", "1"}, "1.00000"},
		{"hex float", []string{"%a", "1"}, "0x1p+0"},
		{"hex float fraction", []string{"%a", "10"}, "0x1.4p+3"},
		{"hex float upper", []string{"%A", "-0.5"}, "-0X1P-1"},
		{"hex float zero padded", []string{"%010a", "1"}, "0x00001p+0"},
		{"float from hex", []string{"%g", "0x10"}, "16"},
		{"float from char", []string{"%.1f", "'A"}, "65.0"},
		{"inf", []string{"%f %F", "inf", "-infinity"}, "inf -INF"},
		{"inf padded", []string{"[%5f]", "inf"}, "[  inf]"},
		{"nan", []string{"%+e", "nan"}, "+nan"},

		{"string", []string{"%s", "abc"}, "abc"},
		{"string width", []string{"[%6s]", "abc"}, "[   abc]"},
		{"string left", []string{"[%-6s]", "abc"}, "[abc   ]"},
		{"string precision", []string{"%.2s", "abc"}, "ab"},
		{"width counts bytes", []string{"[%4s]", "é"}, "[  é]"},
		{"char", []string{"%c", "xyz"}, "x"},
		{"char of nothing is NUL", []string{"%c", ""}, "\x00"},
		{"star width", []string{"[%*d]", "4", "7"}, "[   7]"},
		{"negative star width left-justifies", []string{"[%*d]", "-4", "7"}, "[7   ]"},
		{"star precision", []string{"%.*f", "1", "2.25"}, "2.2"},
		{"negative star precision is ignored", []string{"%.*f", "-1", "2"}, "2.000000"},

		{"reused", []string{"%s=%d\n", "a", "1", "b", "2"}, "a=1\nb=2\n"},
		{"missing arguments", []string{"%s=%d;", "a", "1", "b"}, "a=1;b=0;"},
		{"missing everything", []string{"[%s|%d|%f]"}, "[|0|0.000000]"},

		{"escapes", []string{`\a\b\e\f\n\r\t\v\\\"`}, "\a\b\x1b\f\n\r\t\v\\\""},
		{"octal escape", []string{`\101\0\12`}, "A\x00\n"},
		{"hex escape", []string{`\x41\x4a`}, "AJ"},
		{"unicode escape", []string{`\u00e9\U0001F600`}, "é😀"},
		{"unknown escape kept", []string{`\q`}, `\q`},
		{"trailing backslash", []string{`a\`}, `a\`},
		{"\\c stops", []string{`a\cb%s`, "x"}, "a"},

		{"b", []string{"%b", `a\tb\n`}, "a\tb\n"},
		{"b octal with zero", []string{"%b", `\0101`}, "A"},
		{"b octal without zero", []string{"%b", `\101`}, "A"},
		{"b \\c stops everything", []string{"%b|%s", `x\cy`, "z"}, "x"},
		{"b width", []string{"[%4b]", `a`}, "[   a]"},

		{"q plain", []string{"%q", "abc"}, "abc"},
		{"q space", []string{"%q", "a b"}, "'a b'"},
		{"q quote", []string{"%q", "it's"}, `"it's"`},
		{"q quote and dollar", []string{"%q", "it's $x"}, `'it'\''s $x'`},
		{"q empty", []string{"%q", ""}, "''"},
		{"q control", []string{"%q", "a\nb"}, `'a'$'\n''b'`},
		{"q tilde", []string{"%q", "~x"}, "'~x'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.args...)
			if err != nil {
				t.Fatalf("Run(%q): %v (stderr %q)", tt.args, err, errb)
			}
			if out != tt.want {
				t.Fatalf("Run(%q) = %q, want %q", tt.args, out, tt.want)
			}
			if errb != "" {
				t.Fatalf("stderr = %q, want empty", errb)
			}
		})
	}
}

// TestNumericErrors checks that a bad number is diagnosed, printed as far as
// it was converted, and fails the run without stopping it.
func TestNumericErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		out     string
		wantErr string
	}{
		{"not a number", []string{"%d;", "abc", "2"}, "0;2;", "printfx: 'abc': expected a numeric value\n"},
		{"partly converted", []string{"%d", "12x"}, "12", "printfx: '12x': value not completely converted\n"},
		{"overflow", []string{"%d", "99999999999999999999"}, "9223372036854775807", "printfx: '99999999999999999999': Numerical result out of range\n"},
		{"negative overflow", []string{"%d", "-99999999999999999999"}, "-9223372036854775808", "printfx: '-99999999999999999999': Numerical result out of range\n"},
		{"unsigned overflow", []string{"%u", "99999999999999999999"}, "18446744073709551615", "printfx: '99999999999999999999': Numerical result out of range\n"},
		{"float not a number", []string{"%.1f", "x1"}, "0.0", "printfx: 'x1': expected a numeric value\n"},
		{"float partly converted", []string{"%.1f", "1.5kg"}, "1.5", "printfx: '1.5kg': value not completely converted\n"},
		{"quote in operand", []string{"%d", "it's"}, "0", "printfx: \"it's\": expected a numeric value\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.args...)
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if out != tt.out || errb != tt.wantErr {
				t.Fatalf("got (%q, %q), want (%q, %q)", out, errb, tt.out, tt.wantErr)
			}
		})
	}
}

func TestCharConstantWarning(t *testing.T) {
	out, errb, err := run("%d", "'AB")
	if err != nil || out != "65" {
		t.Fatalf("got (%q, %v), want (\"65\", nil)", out, err)
	}
	if want := "printfx: warning: B: character(s) following character constant have been ignored\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

// TestFatal checks the errors that stop printf at once.
func TestFatal(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		out     string
		wantErr string
	}{
		{"length modifier", []string{"a%ld", "1"}, "a", "printfx: %l: invalid conversion specification\n"},
		{"unknown conversion", []string{"%y"}, "", "printfx: %y: invalid conversion specification\n"},
		{"percent at the end", []string{"ab%"}, "ab", "printfx: %: invalid conversion specification\n"},
		{"flagged percent", []string{"%5%"}, "", "printfx: %5%: invalid conversion specification\n"},
		{"missing hex digits", []string{`a\xz`}, "a", "printfx: missing hexadecimal number in escape\n"},
		{"short unicode", []string{`\u12`}, "", "printfx: missing hexadecimal number in escape\n"},
		{"invalid universal character", []string{`\u0041`}, "", "printfx: invalid universal character name \\u0041\n"},
		{"surrogate", []string{`\ud800`}, "", "printfx: invalid universal character name \\ud800\n"},
		{"bad star width", []string{"%*d", "x", "1"}, "", "printfx: 'x': expected a numeric value\nprintfx: invalid field width: 'x'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.args...)
			if !errors.Is(err, core.ErrFailed) {
				t.Fatalf("err = %v, want ErrFailed", err)
			}
			if out != tt.out || errb != tt.wantErr {
				t.Fatalf("got (%q, %q), want (%q, %q)", out, errb, tt.out, tt.wantErr)
			}
		})
	}
}

func TestExcessArguments(t *testing.T) {
	out, errb, err := run(`hi\n`, "extra", "more")
	if err != nil || out != "hi\n" {
		t.Fatalf("got (%q, %v)", out, err)
	}
	if want := "printfx: warning: ignoring excess arguments, starting with 'extra'\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestMissingOperand(t *testing.T) {
	for _, args := range [][]string{nil, {"--"}} {
		_, _, err := run(args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "missing operand" {
			t.Fatalf("Run(%q) = %v, want the usage error 'missing operand'", args, err)
		}
	}
}

// brokenPipe fails every write as a pipe whose reader has gone.
type brokenPipe struct{}

func (brokenPipe) Write([]byte) (int, error) { return 0, syscall.EPIPE }

// failing fails every write with an error worth reporting.
type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, syscall.ENOSPC }

func TestWriteErrors(t *testing.T) {
	var errb bytes.Buffer
	err := printf.New(printf.Options{}, core.Streams{In: strings.NewReader(""), Out: brokenPipe{}, Err: &errb}).Run([]string{"x"})
	if err != nil || errb.Len() != 0 {
		t.Fatalf("closed pipe: err = %v, stderr = %q; want a quiet success", err, errb.String())
	}

	errb.Reset()
	err = printf.New(printf.Options{}, core.Streams{In: strings.NewReader(""), Out: failing{}, Err: &errb}).Run([]string{"x"})
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb.String(), "printfx: write error: ") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		directive string
		v         float64
		want      string
		ok        bool
	}{
		{"%f", 1.5, "1.500000", true},
		{"%.2e", 1234, "1.23e+03", true},
		{"%-8.1f|", 0, "", false},
		{"%08.3f", -1.5, "-001.500", true},
		{"%g", 100, "100", true},
		{"%a", 2, "0x1p+1", true},
		{"%G", math.Inf(1), "INF", true},
		{"%d", 1, "", false},
		{"%*f", 1, "", false},
		{"f", 1, "", false},
	}
	for _, tt := range tests {
		got, ok := printf.FormatFloat(tt.directive, tt.v)
		if got != tt.want || ok != tt.ok {
			t.Errorf("FormatFloat(%q, %v) = (%q, %v), want (%q, %v)", tt.directive, tt.v, got, ok, tt.want, tt.ok)
		}
	}
}

func TestParseRat(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"1", "1", true},
		{"-2.5", "-5/2", true},
		{"+.5", "1/2", true},
		{"1e3", "1000", true},
		{"0x10", "16", true},
		{"0x1.8p1", "3", true},
		{" 7", "7", true},
		{"inf", "", false},
		{"nan", "", false},
		{"1/2", "", false},
		{"1_000", "", false},
		{"0b101", "", false},
		{"abc", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		r, ok := printf.ParseRat(tt.in)
		if ok != tt.ok || (ok && r.RatString() != tt.want) {
			t.Errorf("ParseRat(%q) = (%v, %v), want (%s, %v)", tt.in, r, ok, tt.want, tt.ok)
		}
	}
}
