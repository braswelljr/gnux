// Package regex is the regular expression engine of the tools that take POSIX
// patterns (grep, sed, expr, nl, csplit, find). It is Go's regexp, which runs
// in linear time, for every pattern it can express. A pattern with a back
// reference (\1 to \9), which RE2 cannot express, runs on a backtracking
// matcher behind the same methods instead, so a caller never needs to know
// which one it holds.
//
// A translator writes a back reference into the Go syntax it produces with
// BackRef, which is an empty named group that regexp/syntax accepts and this
// package recognizes. Groups are numbered as in POSIX: by their opening
// parenthesis, counting only unnamed groups, so that a caller may wrap a
// pattern in a named group of its own (grep -w does) without renumbering it.
package regex

import (
	"errors"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"sync/atomic"
)

// markerName starts the name of every group BackRef and Look write, and
// backRefName that of a back reference. No translator emits a named group of
// its own with this prefix, and a pattern cannot spell one: a POSIX pattern's
// "(?" is always escaped, and the Perl translator renames named groups.
const (
	markerName  = "gnux"
	backRefName = markerName + "backref"
)

// Lookaround is the kind of a lookaround assertion.
type Lookaround string

// The four lookaround assertions of Perl.
const (
	Ahead     Lookaround = markerName + "ahead"     // (?=...)
	NotAhead  Lookaround = markerName + "notahead"  // (?!...)
	Behind    Lookaround = markerName + "behind"    // (?<=...)
	NotBehind Lookaround = markerName + "notbehind" // (?<!...)
)

// Look returns the Go syntax for a lookaround assertion over body, which is
// Go syntax itself. It matches the empty string where body does (or, for the
// negative kinds, does not) match just after or just before.
func Look(kind Lookaround, body string) string {
	id := backRefCount.Add(1)
	return "(?P<" + string(kind) + "_" + strconv.FormatUint(id, 10) + ">" + body + ")"
}

// BackRef returns the Go syntax for a back reference to group n (1 to 9).
// Each call gives the group a name of its own, since Go refuses two groups of
// one name and a pattern may repeat a reference (\1\1).
func BackRef(n int) string {
	id := backRefCount.Add(1)
	return "(?P<" + backRefName + strconv.Itoa(n) + "_" + strconv.FormatUint(id, 10) + ">)"
}

var backRefCount atomic.Uint64

// ErrBackReference is the error for a back reference to a group the pattern
// does not have, with the wording of the C library.
var ErrBackReference = errors.New("Invalid back reference") //nolint:revive,staticcheck // regcomp's message, shown as is

// ErrTooComplex is what Err reports once a back-reference match was given up
// because it needed more steps than any reasonable pattern does.
var ErrTooComplex = errors.New("regular expression too complex to match with back references")

// Regexp is a compiled regular expression. Its methods mean what the methods
// of the same name on *regexp.Regexp mean.
type Regexp struct {
	re *regexp.Regexp // the Go engine; it also validates every pattern
	bt *program       // set when the pattern has a back reference
	// err is set (to ErrTooComplex) when a search was abandoned; the
	// abandoned search reported no match.
	err atomic.Pointer[error]
}

// Compile parses a Go regular expression, which may hold back references
// written with BackRef. A syntax error is the *syntax.Error regexp.Compile
// returns, so callers can map it to their own messages as before.
func Compile(expr string) (*Regexp, error) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	r := &Regexp{re: re}
	if !HasMarkers(expr) {
		return r, nil
	}
	tree, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return nil, err
	}
	prog, err := newProgram(tree)
	if err != nil {
		return nil, err
	}
	if prog.backRefs > 0 || prog.looks > 0 {
		r.bt = prog
	}
	return r, nil
}

// HasMarkers reports whether a Go expression holds a back reference or a
// lookaround. Such an expression must not be given to regexp alone for
// anything but validation: regexp reads the marker groups as plain groups, so
// its LiteralPrefix, for one, runs the text around them together.
func HasMarkers(expr string) bool {
	return strings.Contains(expr, "(?P<"+markerName)
}

// MustCompile is Compile for expressions known to be valid.
func MustCompile(expr string) *Regexp {
	r, err := Compile(expr)
	if err != nil {
		panic(err)
	}
	return r
}

// HasBackRefs reports whether the expression uses the backtracking matcher:
// it has a back reference or a lookaround.
func (r *Regexp) HasBackRefs() bool { return r.bt != nil }

// Err returns ErrTooComplex once a search has been abandoned, and nil before.
func (r *Regexp) Err() error {
	if p := r.err.Load(); p != nil {
		return *p
	}
	return nil
}

// Longest makes later searches prefer the longest of the leftmost matches,
// as POSIX requires.
func (r *Regexp) Longest() {
	r.re.Longest()
	if r.bt != nil {
		r.bt.longest = true
	}
}

// String returns the source text.
func (r *Regexp) String() string { return r.re.String() }

// NumSubexp returns the number of groups, not counting back references.
func (r *Regexp) NumSubexp() int {
	if r.bt != nil {
		return r.bt.groups
	}
	return r.re.NumSubexp()
}

// SubexpIndex returns the index of the group with the given name, or -1.
func (r *Regexp) SubexpIndex(name string) int {
	if r.bt != nil {
		return r.bt.named(name)
	}
	return r.re.SubexpIndex(name)
}

// LiteralPrefix returns a literal every match must start with. A pattern with
// back references reports none.
func (r *Regexp) LiteralPrefix() (prefix string, complete bool) {
	if r.bt != nil {
		return "", false
	}
	return r.re.LiteralPrefix()
}

// Match reports whether b contains a match.
func (r *Regexp) Match(b []byte) bool {
	if r.bt != nil {
		return r.find(b, 0, false) != nil
	}
	return r.re.Match(b)
}

// MatchString reports whether s contains a match.
func (r *Regexp) MatchString(s string) bool {
	if r.bt != nil {
		return r.Match([]byte(s))
	}
	return r.re.MatchString(s)
}

// FindIndex returns the location of the leftmost match in b, or nil.
func (r *Regexp) FindIndex(b []byte) []int {
	if r.bt != nil {
		if loc := r.find(b, 0, false); loc != nil {
			return loc[:2]
		}
		return nil
	}
	return r.re.FindIndex(b)
}

// FindSubmatchIndex returns the locations of the leftmost match and its
// groups in b, or nil.
func (r *Regexp) FindSubmatchIndex(b []byte) []int {
	if r.bt != nil {
		return r.find(b, 0, true)
	}
	return r.re.FindSubmatchIndex(b)
}

// FindStringSubmatchIndex is FindSubmatchIndex for a string.
func (r *Regexp) FindStringSubmatchIndex(s string) []int {
	if r.bt != nil {
		return r.find([]byte(s), 0, true)
	}
	return r.re.FindStringSubmatchIndex(s)
}

// FindAllIndex returns the successive non-overlapping matches in b, at most
// n of them when n >= 0.
func (r *Regexp) FindAllIndex(b []byte, n int) [][]int {
	if r.bt != nil {
		return r.findAll(b, n, false)
	}
	return r.re.FindAllIndex(b, n)
}

// FindAllSubmatchIndex is FindAllIndex with the groups' locations.
func (r *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	if r.bt != nil {
		return r.findAll(b, n, true)
	}
	return r.re.FindAllSubmatchIndex(b, n)
}

// find runs the backtracking matcher from pos and returns the match with its
// groups (or only its bounds when groups is false), or nil.
func (r *Regexp) find(b []byte, pos int, groups bool) []int {
	loc, ok := r.bt.search(b, pos, groups)
	if !ok {
		err := ErrTooComplex
		r.err.CompareAndSwap(nil, &err)
	}
	return loc
}

// findAll steps through b as regexp does: after an empty match the search
// moves on by one character, and an empty match right after the previous
// match is skipped.
func (r *Regexp) findAll(b []byte, n int, groups bool) [][]int {
	var out [][]int
	prevEnd := -1
	for pos := 0; pos <= len(b) && (n < 0 || len(out) < n); {
		loc := r.find(b, pos, groups)
		if loc == nil {
			break
		}
		accept := true
		if loc[1] == pos {
			if loc[0] == prevEnd {
				accept = false
			}
			pos += runeWidth(b, pos)
		} else {
			pos = loc[1]
		}
		prevEnd = loc[1]
		if accept {
			out = append(out, loc)
		}
	}
	return out
}

// runeWidth is the width of the character at b[i], at least 1.
func runeWidth(b []byte, i int) int {
	if i >= len(b) {
		return 1
	}
	_, w := decode(b, i)
	return w
}
