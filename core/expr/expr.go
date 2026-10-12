// Package expr implements the engine behind exprx, a GNU-compatible
// implementation of the GNU/UNIX expr command: evaluate an expression given
// as separate arguments and print its value.
package expr

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/bre"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "exprx"

// Exit statuses, as in GNU expr.
const (
	ExitTrue    = 0 // the value is neither null nor zero
	ExitFalse   = 1 // the value is null or zero
	ExitInvalid = 2 // the expression is invalid
	ExitFailure = 3 // another error, such as a failed write
)

// Options configures the engine. The zero value is plain expr.
type Options struct{}

// Expr is the engine. Construct it with New.
type Expr struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds an expr engine writing to s.
func New(opts Options, s core.Streams) *Expr {
	return &Expr{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// exprError is an invalid expression, worded as GNU expr words it.
type exprError struct{ msg string }

func (e *exprError) Error() string { return e.msg }

// value is an integer or a string; GNU expr keeps the two apart because a
// string that looks like a number ("007") prints as written.
type value struct {
	num *big.Int
	str string
}

func intValue(n *big.Int) value { return value{num: n} }
func intOf(n int64) value       { return value{num: big.NewInt(n)} }
func strValue(s string) value   { return value{str: s} }

func (v value) String() string {
	if v.num != nil {
		return v.num.String()
	}
	return v.str
}

// toInt returns v as an integer when it is one or looks like one.
func (v value) toInt() (*big.Int, bool) {
	if v.num != nil {
		return v.num, true
	}
	return parseInt(v.str)
}

// null reports whether v counts as false: zero, the empty string, or a
// string of zeros with an optional minus sign.
func (v value) null() bool {
	if v.num != nil {
		return v.num.Sign() == 0
	}
	s := strings.TrimPrefix(v.str, "-")
	if v.str == "" {
		return true
	}
	return s != "" && strings.Trim(s, "0") == ""
}

// parseInt accepts what GNU expr calls an integer: an optional minus sign
// followed by one or more decimal digits.
func parseInt(s string) (*big.Int, bool) {
	d := strings.TrimPrefix(s, "-")
	if d == "" {
		return nil, false
	}
	for i := 0; i < len(d); i++ {
		if d[i] < '0' || d[i] > '9' {
			return nil, false
		}
	}
	n, ok := new(big.Int).SetString(s, 10)
	return n, ok
}

// Run evaluates args and prints the value. The returned error is nil when
// the value is neither null nor zero; otherwise it is a *core.ExitError
// carrying GNU's status, after a diagnostic for an invalid expression. No
// arguments is a usage error.
func (e *Expr) Run(args []string) error {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	v, err := eval(args)
	if err != nil {
		e.diag.Errorf("%s", err.Error())
		return &core.ExitError{Code: ExitInvalid}
	}
	if _, err := fmt.Fprintln(e.streams.Out, v); err != nil && !core.IsClosedPipe(err) {
		e.diag.Errorf("write error: %s", core.ErrText(err))
		return &core.ExitError{Code: ExitFailure}
	}
	if v.null() {
		return &core.ExitError{Code: ExitFalse}
	}
	return nil
}

// Eval evaluates the expression in args.
func Eval(args []string) (fmt.Stringer, error) {
	v, err := eval(args)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func eval(args []string) (value, error) {
	p := &parser{args: args}
	v, err := p.or()
	if err != nil {
		return value{}, err
	}
	if p.pos < len(args) {
		return value{}, &exprError{"syntax error: unexpected argument " + quote(args[p.pos])}
	}
	return v, nil
}

func quote(s string) string { return "'" + s + "'" }

type parser struct {
	args []string
	pos  int
}

// next consumes the next argument when it equals tok.
func (p *parser) next(tok string) bool {
	if p.pos < len(p.args) && p.args[p.pos] == tok {
		p.pos++
		return true
	}
	return false
}

// need fails when the arguments ran out after an operator or keyword.
func (p *parser) need() error {
	if p.pos >= len(p.args) {
		return &exprError{"syntax error: missing argument after " + quote(p.args[p.pos-1])}
	}
	return nil
}

// or: and ( | and )*
func (p *parser) or() (value, error) {
	l, err := p.and()
	if err != nil {
		return l, err
	}
	for p.next("|") {
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.and()
		if err != nil {
			return l, err
		}
		if l.null() {
			l = r
			if r.null() {
				l = intOf(0)
			}
		}
	}
	return l, nil
}

// and: cmp ( & cmp )*
func (p *parser) and() (value, error) {
	l, err := p.cmp()
	if err != nil {
		return l, err
	}
	for p.next("&") {
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.cmp()
		if err != nil {
			return l, err
		}
		if l.null() || r.null() {
			l = intOf(0)
		}
	}
	return l, nil
}

var cmpOps = []string{"<=", ">=", "!=", "==", "<", ">", "="}

// cmp: add ( OP add )*, comparing as integers when both sides are integers
// and as strings otherwise.
func (p *parser) cmp() (value, error) {
	l, err := p.add()
	if err != nil {
		return l, err
	}
	for {
		op := ""
		for _, o := range cmpOps {
			if p.next(o) {
				op = o
				break
			}
		}
		if op == "" {
			return l, nil
		}
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.add()
		if err != nil {
			return l, err
		}
		var c int
		a, aok := l.toInt()
		b, bok := r.toInt()
		if aok && bok {
			c = a.Cmp(b)
		} else {
			c = strings.Compare(l.String(), r.String())
		}
		var t bool
		switch op {
		case "<":
			t = c < 0
		case "<=":
			t = c <= 0
		case "=", "==":
			t = c == 0
		case "!=":
			t = c != 0
		case ">=":
			t = c >= 0
		case ">":
			t = c > 0
		}
		l = intOf(0)
		if t {
			l = intOf(1)
		}
	}
}

// add: mul ( [+-] mul )*
func (p *parser) add() (value, error) {
	l, err := p.mul()
	if err != nil {
		return l, err
	}
	for {
		var op string
		switch {
		case p.next("+"):
			op = "+"
		case p.next("-"):
			op = "-"
		default:
			return l, nil
		}
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.mul()
		if err != nil {
			return l, err
		}
		if l, err = arith(op, l, r); err != nil {
			return l, err
		}
	}
}

// mul: match ( [*/%] match )*
func (p *parser) mul() (value, error) {
	l, err := p.match()
	if err != nil {
		return l, err
	}
	for {
		var op string
		switch {
		case p.next("*"):
			op = "*"
		case p.next("/"):
			op = "/"
		case p.next("%"):
			op = "%"
		default:
			return l, nil
		}
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.match()
		if err != nil {
			return l, err
		}
		if l, err = arith(op, l, r); err != nil {
			return l, err
		}
	}
}

func arith(op string, l, r value) (value, error) {
	a, aok := l.toInt()
	b, bok := r.toInt()
	if !aok || !bok {
		return value{}, &exprError{"non-integer argument"}
	}
	n := new(big.Int)
	switch op {
	case "+":
		n.Add(a, b)
	case "-":
		n.Sub(a, b)
	case "*":
		n.Mul(a, b)
	default:
		if b.Sign() == 0 {
			return value{}, &exprError{"division by zero"}
		}
		if op == "/" {
			n.Quo(a, b)
		} else {
			n.Rem(a, b)
		}
	}
	return intValue(n), nil
}

// match: unary ( : unary )*
func (p *parser) match() (value, error) {
	l, err := p.unary()
	if err != nil {
		return l, err
	}
	for p.next(":") {
		if err := p.need(); err != nil {
			return l, err
		}
		r, err := p.unary()
		if err != nil {
			return l, err
		}
		if l, err = regexMatch(l.String(), r.String()); err != nil {
			return l, err
		}
	}
	return l, nil
}

// unary handles the keywords and "+ TOKEN"; their operands are themselves
// unary expressions, as in GNU expr.
func (p *parser) unary() (value, error) {
	if p.pos >= len(p.args) {
		return value{}, &exprError{"syntax error: missing argument after " + quote(p.args[p.pos-1])}
	}
	operands := func(n int) ([]value, error) {
		out := make([]value, n)
		for i := range out {
			if err := p.need(); err != nil {
				return nil, err
			}
			v, err := p.unary()
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}

	switch {
	case p.next("+"):
		if err := p.need(); err != nil {
			return value{}, err
		}
		p.pos++
		return strValue(p.args[p.pos-1]), nil
	case p.next("length"):
		v, err := operands(1)
		if err != nil {
			return value{}, err
		}
		return intOf(int64(len(v[0].String()))), nil
	case p.next("match"):
		v, err := operands(2)
		if err != nil {
			return value{}, err
		}
		return regexMatch(v[0].String(), v[1].String())
	case p.next("index"):
		v, err := operands(2)
		if err != nil {
			return value{}, err
		}
		return intOf(int64(strings.IndexAny(v[0].String(), v[1].String()) + 1)), nil
	case p.next("substr"):
		v, err := operands(3)
		if err != nil {
			return value{}, err
		}
		return substr(v[0].String(), v[1], v[2]), nil
	}
	return p.primary()
}

func (p *parser) primary() (value, error) {
	tok := p.args[p.pos]
	switch tok {
	case "(":
		p.pos++
		if err := p.need(); err != nil {
			return value{}, err
		}
		v, err := p.or()
		if err != nil {
			return v, err
		}
		if p.pos >= len(p.args) {
			return v, &exprError{"syntax error: expecting ')' after " + quote(p.args[p.pos-1])}
		}
		if p.args[p.pos] != ")" {
			return v, &exprError{"syntax error: expecting ')' instead of " + quote(p.args[p.pos])}
		}
		p.pos++
		return v, nil
	case ")":
		return value{}, &exprError{"syntax error: unexpected ')'"}
	}
	p.pos++
	return strValue(tok), nil
}

// substr returns the LEN bytes of s from position POS (1-based), or the
// empty string when either is not a positive integer or POS is past the end.
func substr(s string, pos, length value) value {
	i, iok := pos.toInt()
	n, nok := length.toInt()
	if !iok || !nok || i.Sign() <= 0 || n.Sign() <= 0 || !i.IsInt64() {
		return strValue("")
	}
	start := i.Int64() - 1
	if start >= int64(len(s)) {
		return strValue("")
	}
	end := int64(len(s))
	if n.IsInt64() && start+n.Int64() < end {
		end = start + n.Int64()
	}
	return strValue(s[start:end])
}

// regexMatch anchors pattern, a POSIX basic regular expression, at the start
// of s. With a \( \) group it returns the text the first group matched (or
// the empty string); without one, the number of bytes matched.
func regexMatch(s, pattern string) (value, error) {
	re, err := compileBRE(pattern)
	if err != nil {
		return value{}, err
	}
	m := re.FindStringSubmatchIndex(s)
	if re.NumSubexp() > 0 {
		if m == nil || m[2] < 0 {
			return strValue(""), nil
		}
		return strValue(s[m[2]:m[3]]), nil
	}
	if m == nil {
		return intOf(0), nil
	}
	return intOf(int64(m[1])), nil
}

// compileBRE compiles a POSIX basic regular expression anchored at the
// start, matching leftmost-longest as POSIX requires.
func compileBRE(pattern string) (*regexp.Regexp, error) {
	re, err := bre.CompileAnchored(pattern)
	var e *bre.Error
	if errors.As(err, &e) {
		return nil, &exprError{e.Msg}
	}
	return re, err
}
