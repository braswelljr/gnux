package echo_test

import (
	"bytes"
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/echo"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantOpts echo.Options
		wantOps  []string
	}{
		{"no arguments", nil, echo.Options{}, nil},
		{"plain operands", []string{"a", "b"}, echo.Options{}, []string{"a", "b"}},
		{"-n", []string{"-n", "a"}, echo.Options{NoNewline: true}, []string{"a"}},
		{"-e", []string{"-e", "a"}, echo.Options{Escapes: true}, []string{"a"}},
		{"cluster -ne", []string{"-ne", "a"}, echo.Options{NoNewline: true, Escapes: true}, []string{"a"}},
		{"cluster -en", []string{"-en", "a"}, echo.Options{NoNewline: true, Escapes: true}, []string{"a"}},
		{"separate -n -e", []string{"-n", "-e", "a"}, echo.Options{NoNewline: true, Escapes: true}, []string{"a"}},
		{"-E after -e wins", []string{"-e", "-E", "a"}, echo.Options{}, []string{"a"}},
		{"-e after -E wins", []string{"-Ee", "a"}, echo.Options{Escapes: true}, []string{"a"}},
		{"unknown letter is an operand", []string{"-x", "a"}, echo.Options{}, []string{"-x", "a"}},
		{"mixed cluster is an operand", []string{"-nx", "a"}, echo.Options{}, []string{"-nx", "a"}},
		{"lone dash is an operand", []string{"-", "-n"}, echo.Options{}, []string{"-", "-n"}},
		{"double dash is an operand", []string{"--", "a"}, echo.Options{}, []string{"--", "a"}},
		{"double dash after an option", []string{"-n", "--", "a"}, echo.Options{NoNewline: true}, []string{"--", "a"}},
		{"options stop at the first operand", []string{"a", "-n"}, echo.Options{}, []string{"a", "-n"}},
		{"long options are operands", []string{"--help", "x"}, echo.Options{}, []string{"--help", "x"}},
		{"only options", []string{"-n"}, echo.Options{NoNewline: true}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, ops := echo.Parse(tt.args)
			if opts != tt.wantOpts {
				t.Errorf("options = %+v, want %+v", opts, tt.wantOpts)
			}
			if len(ops) != len(tt.wantOps) || (len(ops) > 0 && !reflect.DeepEqual(ops, tt.wantOps)) {
				t.Errorf("operands = %q, want %q", ops, tt.wantOps)
			}
		})
	}
}

func TestRun(t *testing.T) {
	esc := echo.Options{Escapes: true}
	tests := []struct {
		name     string
		opts     echo.Options
		operands []string
		want     string
	}{
		{"no operands", echo.Options{}, nil, "\n"},
		{"no operands, no newline", echo.Options{NoNewline: true}, nil, ""},
		{"joins with single spaces", echo.Options{}, []string{"a", "b  c", ""}, "a b  c \n"},
		{"no newline", echo.Options{NoNewline: true}, []string{"a", "b"}, "a b"},
		{"escapes are literal by default", echo.Options{}, []string{`a\tb\n`}, `a\tb\n` + "\n"},
		{"control escapes", esc, []string{`\a\b\e\f\n\r\t\v`}, "\a\b\x1b\f\n\r\t\v\n"},
		{"escaped backslash", esc, []string{`a\\nb`}, `a\nb` + "\n"},
		{"trailing backslash", esc, []string{`a\`}, `a\` + "\n"},
		{"unknown escape is kept", esc, []string{`\q\"`}, `\q\"` + "\n"},
		{"octal with leading zero", esc, []string{`\0101\060`}, "A0\n"},
		{"octal stops at three digits", esc, []string{`\01011`}, "A1\n"},
		{"bare \\0 is NUL", esc, []string{`a\0b`}, "a\x00b\n"},
		{"octal stops at a non-octal digit", esc, []string{`\0189`}, "\x0189\n"},
		{"octal without leading zero", esc, []string{`\101\7`}, "A\a\n"},
		{"octal wraps to a byte", esc, []string{`\0777`}, "\xff\n"},
		{"hex", esc, []string{`\x41\x7a\xA`}, "Az\n\n"},
		{"hex takes two digits at most", esc, []string{`\x414`}, "A4\n"},
		{"hex without digits is literal", esc, []string{`\xg`}, `\xg` + "\n"},
		{"\\c stops output", esc, []string{`ab\cde`, "f"}, "ab"},
		{"\\c in a later operand", esc, []string{"a", `b\c`, "c"}, "a b"},
		{"\\c with -n", echo.Options{Escapes: true, NoNewline: true}, []string{`\c`}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			e := echo.New(tt.opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
			if err := e.Run(tt.operands); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if out.String() != tt.want {
				t.Errorf("stdout = %q, want %q", out.String(), tt.want)
			}
			if errb.Len() != 0 {
				t.Errorf("stderr = %q, want empty", errb.String())
			}
		})
	}
}

type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteError(t *testing.T) {
	var errb bytes.Buffer
	e := echo.New(echo.Options{}, core.Streams{Out: failWriter{fs.ErrPermission}, Err: &errb})
	if err := e.Run([]string{"a"}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "echox: write error: Permission denied\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}
