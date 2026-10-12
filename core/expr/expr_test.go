package expr_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expr"
)

func run(args ...string) (stdout, stderr string, status int, err error) {
	var out, errb bytes.Buffer
	err = expr.New(expr.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	status = 0
	var exit *core.ExitError
	if errors.As(err, &exit) {
		status = exit.Code
	} else if err != nil {
		status = -1
	}
	return out.String(), errb.String(), status, err
}

func TestValues(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   string
		status int
	}{
		{"integer", []string{"5"}, "5", 0},
		{"string keeps its form", []string{"007"}, "007", 0},
		{"zero is false", []string{"0"}, "0", 1},
		{"empty is false", []string{""}, "", 1},
		{"zeros are false", []string{"-00"}, "-00", 1},
		{"add", []string{"2", "+", "3"}, "5", 0},
		{"subtract to zero", []string{"3", "-", "3"}, "0", 1},
		{"precedence", []string{"2", "+", "3", "*", "4"}, "14", 0},
		{"parentheses", []string{"(", "2", "+", "3", ")", "*", "4"}, "20", 0},
		{"division truncates toward zero", []string{"-7", "/", "2"}, "-3", 0},
		{"remainder has the dividend's sign", []string{"-7", "%", "2"}, "-1", 0},
		{"leading zeros as numbers", []string{"007", "+", "1"}, "8", 0},
		{"big numbers", []string{"99999999999999999999", "*", "10"}, "999999999999999999990", 0},
		{"numeric compare", []string{"10", ">", "9"}, "1", 0},
		{"string compare", []string{"10", ">", "9a"}, "0", 1},
		{"equal", []string{"abc", "=", "abc"}, "1", 0},
		{"== equal", []string{"1", "==", "01"}, "1", 0},
		{"not equal", []string{"a", "!=", "a"}, "0", 1},
		{"less or equal", []string{"2", "<=", "2"}, "1", 0},
		{"greater or equal", []string{"b", ">=", "a"}, "1", 0},
		{"or takes the first non-null", []string{"a", "|", "b"}, "a", 0},
		{"or falls through", []string{"0", "|", "b"}, "b", 0},
		{"or of nulls is 0", []string{"", "|", "0"}, "0", 1},
		{"and returns the first", []string{"a", "&", "b"}, "a", 0},
		{"and with a null is 0", []string{"a", "&", ""}, "0", 1},
		{"length", []string{"length", "hello"}, "5", 0},
		{"length counts bytes", []string{"length", "héllo"}, "6", 0},
		{"index", []string{"index", "abcdef", "fd"}, "4", 0},
		{"index none", []string{"index", "abc", "x"}, "0", 1},
		{"substr", []string{"substr", "hello", "2", "3"}, "ell", 0},
		{"substr past the end", []string{"substr", "hello", "4", "10"}, "lo", 0},
		{"substr out of range", []string{"substr", "hello", "9", "1"}, "", 1},
		{"substr bad position", []string{"substr", "hello", "0", "1"}, "", 1},
		{"+ quotes a keyword", []string{"+", "length"}, "length", 0},
		{"length of a keyword", []string{"length", "+", "index"}, "5", 0},
		{"leading -- is skipped", []string{"--", "-5", "+", "1"}, "-4", 0},
		{"match length", []string{"abcabc", ":", "a.c"}, "3", 0},
		{"match keyword", []string{"match", "abc", "ab"}, "2", 0},
		{"match is anchored", []string{"xabc", ":", "abc"}, "0", 1},
		{"match group", []string{"file.txt", ":", `\(.*\)\.txt`}, "file", 0},
		{"match group no match", []string{"file.go", ":", `\(.*\)\.txt`}, "", 1},
		{"match longest", []string{"aaa", ":", "a*"}, "3", 0},
		{"match star at start is literal", []string{"*x", ":", `*x`}, "2", 0},
		{"match caret star", []string{"*x", ":", `^*x`}, "2", 0},
		{"match interval", []string{"aaab", ":", `a\{2\}`}, "2", 0},
		{"match plus extension", []string{"aaab", ":", `a\+`}, "3", 0},
		{"match alternation extension", []string{"bc", ":", `a\|b`}, "1", 0},
		{"match plain plus is literal", []string{"a+b", ":", `a+b`}, "3", 0},
		{"match class", []string{"abc1", ":", `[[:alpha:]]*`}, "3", 0},
		{"match bracket with ]", []string{"]]a", ":", `[]]*`}, "2", 0},
		{"match literal braces", []string{"{x}", ":", `{x}`}, "3", 0},
		{"match dollar", []string{"ab", ":", `ab$`}, "2", 0},
		{"match whole number", []string{"123x", ":", `[0-9]*$`}, "0", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errs, status, _ := run(tt.args...)
			if out != tt.want+"\n" || status != tt.status || errs != "" {
				t.Fatalf("expr %q = %q status %d (stderr %q), want %q status %d",
					tt.args, out, status, errs, tt.want, tt.status)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"1", "+", "a"}, "exprx: non-integer argument\n"},
		{[]string{"1", "/", "0"}, "exprx: division by zero\n"},
		{[]string{"1", "%", "0"}, "exprx: division by zero\n"},
		{[]string{"1", "2"}, "exprx: syntax error: unexpected argument '2'\n"},
		{[]string{"1", "+"}, "exprx: syntax error: missing argument after '+'\n"},
		{[]string{"length"}, "exprx: syntax error: missing argument after 'length'\n"},
		{[]string{"substr", "a", "1"}, "exprx: syntax error: missing argument after '1'\n"},
		{[]string{"(", "1"}, "exprx: syntax error: expecting ')' after '1'\n"},
		{[]string{"(", "1", "2"}, "exprx: syntax error: expecting ')' instead of '2'\n"},
		{[]string{")"}, "exprx: syntax error: unexpected ')'\n"},
		{[]string{"a", ":", `\(`}, "exprx: Unmatched ( or \\(\n"},
		{[]string{"a", ":", `\)`}, "exprx: Unmatched ) or \\)\n"},
		{[]string{"a", ":", `[a`}, "exprx: Unmatched [, [^, [:, [., or [=\n"},
		{[]string{"a", ":", `\(a\)\1`}, "exprx: back references are not supported\n"},
	}
	for _, tt := range tests {
		out, errs, status, _ := run(tt.args...)
		if out != "" || errs != tt.want || status != 2 {
			t.Errorf("expr %q: out %q stderr %q status %d; want stderr %q status 2",
				tt.args, out, errs, status, tt.want)
		}
	}
}

func TestMissingOperand(t *testing.T) {
	for _, args := range [][]string{nil, {"--"}} {
		_, _, _, err := run(args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "missing operand" {
			t.Fatalf("expr %q: err = %v, want usage error 'missing operand'", args, err)
		}
	}
}

func TestEval(t *testing.T) {
	v, err := expr.Eval([]string{"6", "*", "7"})
	if err != nil || v.String() != "42" {
		t.Fatalf("Eval = %v, %v", v, err)
	}
	if _, err := expr.Eval([]string{"a", "+", "1"}); err == nil {
		t.Fatal("Eval accepted a non-integer argument")
	}
}
