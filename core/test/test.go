// Package test implements the engine behind testx, a GNU-compatible
// implementation of the GNU/UNIX test command: evaluate a conditional
// expression and report the result as the exit status.
package test

import (
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "testx"

// Exit statuses, as in GNU test.
const (
	ExitTrue  = 0 // the expression is true
	ExitFalse = 1 // the expression is false or missing
	ExitError = 2 // the expression is malformed
)

// Options configures the engine. The zero value is plain test.
type Options struct {
	// Bracket runs test as '[': the expression must be followed by a ']'
	// argument, which is not part of it, and diagnostics are prefixed with
	// '['.
	Bracket bool
}

// BracketName is the name test answers to when it is written '[ EXPRESSION ]'.
const BracketName = "["

// Test is the engine. Construct it with New.
type Test struct {
	Options

	// IsTerminal answers -t FD. It defaults to checking the engine's own
	// streams for descriptors 0, 1 and 2; every other descriptor is not a
	// terminal.
	IsTerminal func(fd int) bool

	// Stat and Lstat read file metadata; they default to os.Stat and
	// os.Lstat.
	Stat  func(string) (os.FileInfo, error)
	Lstat func(string) (os.FileInfo, error)

	// Access answers -r, -w and -x (mode 4, 2 and 1). It defaults to
	// access(2) on Unix and to the permission bits elsewhere.
	Access func(path string, mode uint32) bool

	streams core.Streams
	diag    *core.Diag
}

// New builds a test engine reading from and writing to s.
func New(opts Options, s core.Streams) *Test {
	t := &Test{
		Options: opts,
		Stat:    os.Stat,
		Lstat:   os.Lstat,
		Access:  access,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
	if opts.Bracket {
		t.diag = core.NewDiag(BracketName, s)
	}
	t.IsTerminal = func(fd int) bool {
		switch fd {
		case 0:
			return core.IsTerminal(s.In)
		case 1:
			return core.IsTerminal(s.Out)
		case 2:
			return core.IsTerminal(s.Err)
		}
		return false
	}
	return t
}

// syntaxError is a malformed expression; its message is already worded as
// GNU test words it.
type syntaxError struct{ msg string }

func (e *syntaxError) Error() string { return e.msg }

// Run evaluates the expression in args. It returns nil when the expression is
// true, a *core.ExitError with status 1 when it is false, and one with status
// 2, after a diagnostic, when it is malformed. With Bracket, a missing
// closing ']' is malformed.
func (t *Test) Run(args []string) error {
	if t.Bracket {
		if len(args) == 0 || args[len(args)-1] != "]" {
			t.diag.Errorf("missing ']'")
			return &core.ExitError{Code: ExitError}
		}
		args = args[:len(args)-1]
	}
	ok, err := t.Eval(args)
	if err != nil {
		t.diag.Errorf("%s", err.Error())
		return &core.ExitError{Code: ExitError}
	}
	if !ok {
		return &core.ExitError{Code: ExitFalse}
	}
	return nil
}

// Eval evaluates the expression and reports its value, or the syntax error
// that makes it malformed.
func (t *Test) Eval(args []string) (bool, error) {
	p := &parser{t: t, args: args}
	return p.top()
}

type parser struct {
	t    *Test
	args []string
	pos  int
}

func quote(s string) string { return "'" + s + "'" }

// top applies POSIX's rules for one to four arguments, which decide the
// meaning by the number of arguments rather than by precedence, and falls
// back to the full grammar beyond that.
func (p *parser) top() (bool, error) {
	a := p.args
	var (
		v   bool
		err error
	)
	switch len(a) {
	case 0:
		return false, nil
	case 1:
		return a[0] != "", nil
	case 2:
		v, err = p.two(a)
	case 3:
		v, err = p.three(a)
	case 4:
		switch {
		case a[0] == "!":
			v, err = p.three(a[1:])
			v = !v
		case a[0] == "(" && a[3] == ")":
			v, err = p.two(a[1:3])
		default:
			return p.full()
		}
	default:
		return p.full()
	}
	return v, err
}

func (p *parser) two(a []string) (bool, error) {
	if a[0] == "!" {
		return a[1] == "", nil
	}
	if isUnary(a[0]) {
		return p.unary(a[0], a[1])
	}
	return false, &syntaxError{quote(a[0]) + ": unary operator expected"}
}

func (p *parser) three(a []string) (bool, error) {
	switch {
	case isBinary(a[1]):
		return p.binary(a[0], a[1], a[2])
	case a[1] == "-a":
		return a[0] != "" && a[2] != "", nil
	case a[1] == "-o":
		return a[0] != "" || a[2] != "", nil
	case a[0] == "!":
		v, err := p.two(a[1:])
		return !v, err
	case a[0] == "(" && a[2] == ")":
		return a[1] != "", nil
	}
	return false, &syntaxError{quote(a[1]) + ": binary operator expected"}
}

// full parses the general grammar:
//
//	expr := and ( -o and )*
//	and  := term ( -a term )*
//	term := ! term | ( expr ) | UNARY arg | arg BINARY arg | arg
func (p *parser) full() (bool, error) {
	v, err := p.or()
	if err != nil {
		return false, err
	}
	if p.pos < len(p.args) {
		return false, &syntaxError{"extra argument " + quote(p.args[p.pos])}
	}
	return v, nil
}

func (p *parser) or() (bool, error) {
	v, err := p.and()
	if err != nil {
		return false, err
	}
	for p.pos < len(p.args) && p.args[p.pos] == "-o" {
		p.pos++
		r, err := p.and()
		if err != nil {
			return false, err
		}
		v = v || r
	}
	return v, nil
}

func (p *parser) and() (bool, error) {
	v, err := p.term()
	if err != nil {
		return false, err
	}
	for p.pos < len(p.args) && p.args[p.pos] == "-a" {
		p.pos++
		r, err := p.term()
		if err != nil {
			return false, err
		}
		v = v && r
	}
	return v, nil
}

func (p *parser) term() (bool, error) {
	if p.pos >= len(p.args) {
		return false, &syntaxError{"missing argument after " + quote(p.args[p.pos-1])}
	}
	a := p.args[p.pos]
	rest := len(p.args) - p.pos

	switch {
	case a == "!":
		p.pos++
		v, err := p.term()
		return !v, err
	case a == "(":
		p.pos++
		if p.pos >= len(p.args) {
			return false, &syntaxError{"')' expected"}
		}
		v, err := p.or()
		if err != nil {
			return false, err
		}
		if p.pos >= len(p.args) {
			return false, &syntaxError{"')' expected"}
		}
		if p.args[p.pos] != ")" {
			return false, &syntaxError{"')' expected, found " + quote(p.args[p.pos])}
		}
		p.pos++
		return v, nil
	case a == "-l" && rest >= 4 && isBinary(p.args[p.pos+2]):
		// -l STRING stands for the length of STRING, as an integer.
		op, str := p.args[p.pos+2], p.args[p.pos+1]
		p.pos += 3
		return p.binaryRight("", str, true, op)
	case rest >= 3 && isBinary(p.args[p.pos+1]):
		op := p.args[p.pos+1]
		p.pos += 2
		return p.binaryRight(a, "", false, op)
	case isUnary(a):
		if rest < 2 {
			return false, &syntaxError{"missing argument after " + quote(a)}
		}
		p.pos += 2
		return p.unary(a, p.args[p.pos-1])
	}
	p.pos++
	return a != "", nil
}

func isUnary(op string) bool {
	if len(op) != 2 || op[0] != '-' {
		return false
	}
	return strings.IndexByte("bcdefgGhLknOprsStuwxz", op[1]) >= 0
}

func isBinary(op string) bool {
	switch op {
	case "=", "==", "!=", "<", ">",
		"-eq", "-ne", "-lt", "-le", "-gt", "-ge", "-nt", "-ot", "-ef":
		return true
	}
	return false
}

func (p *parser) unary(op, arg string) (bool, error) {
	switch op {
	case "-n":
		return arg != "", nil
	case "-z":
		return arg == "", nil
	case "-t":
		n, err := integer(arg)
		if err != nil {
			return false, err
		}
		if !n.IsInt64() {
			return false, nil
		}
		return p.t.IsTerminal(int(n.Int64())), nil
	case "-h", "-L":
		info, err := p.t.Lstat(arg)
		return err == nil && info.Mode()&os.ModeSymlink != 0, nil
	case "-r":
		return p.exists(arg) && p.t.Access(arg, 4), nil
	case "-w":
		return p.exists(arg) && p.t.Access(arg, 2), nil
	case "-x":
		return p.exists(arg) && p.t.Access(arg, 1), nil
	}

	info, err := p.t.Stat(arg)
	if err != nil {
		return false, nil
	}
	m := info.Mode()
	switch op {
	case "-b":
		return m&os.ModeDevice != 0 && m&os.ModeCharDevice == 0, nil
	case "-c":
		return m&os.ModeCharDevice != 0, nil
	case "-d":
		return m.IsDir(), nil
	case "-e":
		return true, nil
	case "-f":
		return m.IsRegular(), nil
	case "-g":
		return m&os.ModeSetgid != 0, nil
	case "-u":
		return m&os.ModeSetuid != 0, nil
	case "-k":
		return m&os.ModeSticky != 0, nil
	case "-p":
		return m&os.ModeNamedPipe != 0, nil
	case "-S":
		return m&os.ModeSocket != 0, nil
	case "-s":
		return info.Size() > 0, nil
	case "-O":
		uid, _, ok := owner(info)
		return ok && uid == euid(), nil
	case "-G":
		_, gid, ok := owner(info)
		return ok && gid == egid(), nil
	}
	return false, nil
}

func (p *parser) exists(path string) bool {
	_, err := p.t.Stat(path)
	return err == nil
}

// binaryRight reads the right operand of op and applies op. Either operand
// may be -l STRING, the length of STRING, which GNU test means for the
// integer comparisons; -nt, -ot and -ef refuse it. The string comparisons
// accept it too, and then compare what GNU's parser has moved on to: STRING
// itself for a -l on the left, and the operator with STRING for one on the
// right (test 3 = -l abc compares "=" with "abc").
func (p *parser) binaryRight(left, leftString string, leftIsLength bool, op string) (bool, error) {
	right := p.args[p.pos]
	rightIsLength := right == "-l" && p.pos+1 < len(p.args)
	if rightIsLength {
		right = p.args[p.pos+1]
		p.pos++
	}
	p.pos++
	switch {
	case !leftIsLength && !rightIsLength:
	case isInteger(op):
		if leftIsLength {
			left = strconv.Itoa(len(leftString))
		}
		if rightIsLength {
			right = strconv.Itoa(len(right))
		}
	case op == "-nt" || op == "-ot" || op == "-ef":
		return false, &syntaxError{op + " does not accept -l"}
	case rightIsLength:
		left = op
	default:
		left = leftString
	}
	return p.binary(left, op, right)
}

func isInteger(op string) bool {
	switch op {
	case "-eq", "-ne", "-lt", "-le", "-gt", "-ge":
		return true
	}
	return false
}

func (p *parser) binary(left, op, right string) (bool, error) {
	switch op {
	case "=", "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	case "<":
		return left < right, nil
	case ">":
		return left > right, nil
	case "-nt", "-ot":
		l, lerr := p.t.Stat(left)
		r, rerr := p.t.Stat(right)
		if op == "-ot" {
			l, r, lerr, rerr = r, l, rerr, lerr
		}
		switch {
		case lerr != nil:
			return false, nil
		case rerr != nil:
			return true, nil
		}
		return l.ModTime().After(r.ModTime()), nil
	case "-ef":
		l, lerr := p.t.Stat(left)
		r, rerr := p.t.Stat(right)
		return lerr == nil && rerr == nil && os.SameFile(l, r), nil
	}

	a, err := integer(left)
	if err != nil {
		return false, err
	}
	b, err := integer(right)
	if err != nil {
		return false, err
	}
	c := a.Cmp(b)
	switch op {
	case "-eq":
		return c == 0, nil
	case "-ne":
		return c != 0, nil
	case "-lt":
		return c < 0, nil
	case "-le":
		return c <= 0, nil
	case "-gt":
		return c > 0, nil
	}
	return c >= 0, nil // -ge
}

// integer parses an integer operand as GNU test does: optional surrounding
// blanks, an optional sign and at least one digit, of any length.
func integer(s string) (*big.Int, error) {
	t := strings.Trim(s, " \t\n\v\f\r")
	digits := strings.TrimPrefix(strings.TrimPrefix(t, "-"), "+")
	if t != "" && (t[0] == '+' || t[0] == '-') && len(t)-len(digits) != 1 {
		digits = ""
	}
	ok := digits != ""
	for i := 0; i < len(digits) && ok; i++ {
		ok = digits[i] >= '0' && digits[i] <= '9'
	}
	if !ok {
		return nil, &syntaxError{"invalid integer " + quote(s)}
	}
	n, _ := new(big.Int).SetString(strings.TrimPrefix(t, "+"), 10)
	return n, nil
}
