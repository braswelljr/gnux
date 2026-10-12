// Package sort implements the engine behind sortx, a GNU-compatible
// implementation of the GNU/UNIX sort command: sort, merge or check the lines
// of text files.
package sort

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sortx"

// failure is the exit status of every kind of trouble, as in GNU sort; 1 is
// reserved for "not sorted" under --check.
const failure = 2

// CheckMode says whether, and how, the input is checked instead of sorted
// (-c, -C, --check=WHEN).
type CheckMode int

const (
	// CheckNone sorts as usual.
	CheckNone CheckMode = iota
	// CheckDiagnose (-c, --check=diagnose-first) reports the first line out
	// of order and exits 1.
	CheckDiagnose
	// CheckQuiet (-C, --check=quiet) exits 1 on disorder without a message.
	CheckQuiet
)

// Options mirrors the GNU sort flag surface. The zero value is plain sort:
// whole lines in byte order, ascending.
type Options struct {
	// IgnoreLeadingBlanks (-b) skips blanks at the start of each key.
	IgnoreLeadingBlanks bool
	// DictionaryOrder (-d) considers only blanks and alphanumeric characters.
	DictionaryOrder bool
	// IgnoreCase (-f) folds lowercase letters to uppercase.
	IgnoreCase bool
	// IgnoreNonprinting (-i) considers only printable characters.
	IgnoreNonprinting bool

	// GeneralNumeric (-g) compares as floating-point numbers.
	GeneralNumeric bool
	// HumanNumeric (-h) compares numbers with SI suffixes (2K, 1G).
	HumanNumeric bool
	// Month (-M) compares month names: (unknown) < JAN < ... < DEC.
	Month bool
	// Numeric (-n) compares by string numerical value.
	Numeric bool
	// Version (-V) compares version numbers within text.
	Version bool

	// Reverse (-r) reverses the result of comparisons.
	Reverse bool

	// Keys (-k) are KEYDEFs, POS1[,POS2], each POS being F[.C][OPTS]. With
	// none, the whole line is the key.
	Keys []string
	// FieldSeparator (-t) is the single character that separates fields,
	// or "\0" for NUL. Empty means the transition from a non-blank to a
	// blank character.
	FieldSeparator string

	// Unique (-u) outputs only the first of a run of equal lines.
	Unique bool
	// Stable (-s) disables the last-resort whole-line comparison.
	Stable bool
	// Check (-c, -C) checks for sorted input instead of sorting.
	Check CheckMode
	// Merge (-m) merges already sorted inputs without sorting them.
	Merge bool
	// Output (-o) writes the result to this file instead of standard
	// output. It may be one of the inputs.
	Output string
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
	// Files0From (--files0-from=F), when set, reads the operands as
	// NUL-terminated names from file F ("-" for standard input).
	Files0From string
}

// ignoreMode selects the characters -d and -i leave out of a comparison.
type ignoreMode int

const (
	ignoreNone ignoreMode = iota
	ignoreNondictionary
	ignoreNonprinting
)

// key is one parsed sort key. Field and character positions are zero-based;
// eword < 0 means the key runs to the end of the line, and echar == 0 means
// the end field is included whole.
type key struct {
	sword, schar int
	eword, echar int

	skipsblanks, skipeblanks bool
	ignore                   ignoreMode
	translate                bool
	numeric, general, human  bool
	month, version, reverse  bool
}

// hasOrdering reports whether k carries any option of its own, the test GNU
// uses to decide whether a key inherits the global options.
func (k *key) hasOrdering() bool {
	return k.ignore != ignoreNone || k.translate || k.skipsblanks || k.skipeblanks ||
		k.numeric || k.general || k.human || k.month || k.version
}

// letters spells the key's options the way GNU does in its "incompatible"
// message, leaving out -b and -r, which conflict with nothing.
func (k *key) letters() string {
	var b strings.Builder
	for _, o := range []struct {
		on bool
		c  byte
	}{
		{k.ignore == ignoreNondictionary, 'd'},
		{k.translate, 'f'},
		{k.general, 'g'},
		{k.human, 'h'},
		{k.ignore == ignoreNonprinting, 'i'},
		{k.month, 'M'},
		{k.numeric, 'n'},
		{k.version, 'V'},
	} {
		if o.on {
			b.WriteByte(o.c)
		}
	}
	return b.String()
}

// incompatible reports whether k asks for more than one way of ordering.
func (k *key) incompatible() bool {
	n := 0
	for _, on := range []bool{k.numeric, k.general, k.human, k.month, k.version || k.ignore != ignoreNone} {
		if on {
			n++
		}
	}
	return n > 1
}

// Sort is the engine. Construct it with New.
type Sort struct {
	Options

	streams core.Streams
	diag    *core.Diag

	keys    []*key
	tab     int // separator byte, or -1 for blank transitions
	delim   byte
	blanks  [256]bool
	reverse bool // global -r, applied to the last-resort comparison
}

// New builds a sort engine reading from and writing to s.
func New(opts Options, s core.Streams) *Sort {
	return &Sort{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// input is one operand and the lines read from it.
type input struct {
	name  string
	lines [][]byte
}

// Run sorts, merges or checks the operands; no operands means standard input.
// As in GNU sort, trouble with any input or with the output stops the run with
// exit status 2 before anything is written. With Check, an unsorted input
// returns a *core.ExitError with status 1.
func (s *Sort) Run(args []string) error {
	if err := s.setup(); err != nil {
		return err
	}

	var names []string
	switch {
	case s.Files0From != "":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'\nfile operands cannot be combined with --files0-from", args[0])
		}
		list, err := s.readNames()
		if err != nil {
			return err
		}
		names = list
	case len(args) == 0:
		names = []string{"-"}
	default:
		names = args
	}

	if s.Check != CheckNone {
		letter := byte('c')
		if s.Check == CheckQuiet {
			letter = 'C'
		}
		if len(names) > 1 {
			return core.Usagef("extra operand '%s' not allowed with -%c", names[1], letter)
		}
		if s.Output != "" {
			return s.die("options '-%co' are incompatible", letter)
		}
	}

	// GNU checks that every input can be read before touching the output,
	// so a typo in an operand never truncates the -o file.
	for _, name := range names {
		if name == "-" {
			continue
		}
		f, err := os.Open(name)
		if err != nil {
			return s.die("cannot read: %s: %s", name, core.ErrText(err))
		}
		f.Close()
	}

	var out *os.File
	if s.Output != "" && s.Check == CheckNone {
		// Opened without truncating: the output may also be an input, which
		// is read in full before the file is emptied and rewritten.
		f, err := os.OpenFile(s.Output, os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return s.die("open failed: %s: %s", s.Output, core.ErrText(err))
		}
		defer f.Close()
		out = f
	}

	inputs := make([]input, 0, len(names))
	for _, name := range names {
		lines, err := s.read(name)
		if err != nil {
			return s.die("read failed: %s: %s", name, core.ErrText(err))
		}
		inputs = append(inputs, input{name: name, lines: lines})
	}

	if s.Check != CheckNone {
		return s.check(inputs[0])
	}

	var lines [][]byte
	if s.Merge {
		lines = s.merge(inputs)
	} else {
		for _, in := range inputs {
			lines = append(lines, in.lines...)
		}
		slices.SortStableFunc(lines, s.compare)
	}
	if s.Unique {
		lines = s.uniq(lines)
	}
	return s.write(lines, out)
}

// die reports a fatal problem the way GNU sort does - one message, status 2,
// no "--help" hint - and returns the error that carries the status.
func (s *Sort) die(format string, a ...any) error {
	s.diag.Errorf(format, a...)
	return &core.ExitError{Code: failure}
}

// setup parses the keys and the separator and applies GNU's inheritance of
// the global options.
func (s *Sort) setup() error {
	s.delim = '\n'
	if s.ZeroTerminated {
		s.delim = 0
	}
	s.blanks[' '], s.blanks['\t'] = true, true
	if s.ZeroTerminated {
		// With NUL-terminated records, a newline inside one is just a blank.
		s.blanks['\n'] = true
	}

	s.tab = -1
	switch sep := s.FieldSeparator; {
	case sep == "":
	case sep == `\0`:
		s.tab = 0
	case len(sep) > 1:
		return s.die("multi-character tab '%s'", sep)
	default:
		s.tab = int(sep[0])
	}

	global := &key{
		skipsblanks: s.IgnoreLeadingBlanks,
		skipeblanks: s.IgnoreLeadingBlanks,
		translate:   s.IgnoreCase,
		numeric:     s.Numeric,
		general:     s.GeneralNumeric,
		human:       s.HumanNumeric,
		month:       s.Month,
		version:     s.Version,
		reverse:     s.Reverse,
		eword:       -1,
	}
	switch {
	case s.DictionaryOrder:
		global.ignore = ignoreNondictionary
	case s.IgnoreNonprinting:
		global.ignore = ignoreNonprinting
	}
	s.reverse = s.Reverse

	for _, spec := range s.Keys {
		k, err := parseKey(spec)
		if err != nil {
			return s.die("%s: invalid field specification '%s'", err, spec)
		}
		if !k.hasOrdering() && !k.reverse {
			sword, schar, eword, echar := k.sword, k.schar, k.eword, k.echar
			*k = *global
			k.sword, k.schar, k.eword, k.echar = sword, schar, eword, echar
		}
		s.keys = append(s.keys, k)
	}
	if len(s.keys) == 0 && (global.hasOrdering() || global.reverse) {
		s.keys = []*key{global}
	}

	for _, k := range s.keys {
		if k.incompatible() {
			return s.die("options '-%s' are incompatible", k.letters())
		}
	}
	return nil
}

// parseKey parses a KEYDEF: POS1[,POS2], where POS is F[.C][OPTS]. The error
// is GNU's description of what is wrong.
func parseKey(spec string) (*key, error) {
	k := &key{eword: -1}
	p := spec

	n, rest, ok := number(p)
	if !ok {
		return nil, errors.New("invalid number at field start")
	}
	if n == 0 {
		return nil, errors.New("field number is zero")
	}
	k.sword, p = n-1, rest
	if strings.HasPrefix(p, ".") {
		n, rest, ok = number(p[1:])
		if !ok {
			return nil, errors.New("invalid number after '.'")
		}
		if n == 0 {
			return nil, errors.New("character offset is zero")
		}
		k.schar, p = n-1, rest
	}
	p = k.ordering(p, true)

	if strings.HasPrefix(p, ",") {
		n, rest, ok = number(p[1:])
		if !ok {
			return nil, errors.New("invalid number after ','")
		}
		if n == 0 {
			return nil, errors.New("field number is zero")
		}
		k.eword, p = n-1, rest
		if strings.HasPrefix(p, ".") {
			n, rest, ok = number(p[1:])
			if !ok {
				return nil, errors.New("invalid number after '.'")
			}
			k.echar, p = n, rest
		}
		p = k.ordering(p, false)
	}
	if p != "" {
		return nil, errors.New("stray character in field spec")
	}
	return k, nil
}

// number reads a leading decimal count, clamping one too big to represent.
func number(s string) (n int, rest string, ok bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, s, false
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		v = int(^uint(0) >> 1)
	}
	return v, s[i:], true
}

// ordering applies the option letters at the start of s to k and returns
// what follows them. A 'b' applies to the start or the end of the key,
// depending on which position it follows.
func (k *key) ordering(s string, start bool) string {
	for s != "" {
		switch s[0] {
		case 'b':
			if start {
				k.skipsblanks = true
			} else {
				k.skipeblanks = true
			}
		case 'd':
			k.ignore = ignoreNondictionary
		case 'f':
			k.translate = true
		case 'g':
			k.general = true
		case 'h':
			k.human = true
		case 'i':
			k.ignore = ignoreNonprinting
		case 'M':
			k.month = true
		case 'n':
			k.numeric = true
		case 'r':
			k.reverse = true
		case 'V':
			k.version = true
		default:
			return s
		}
		s = s[1:]
	}
	return s
}

// readNames loads the NUL-separated operand list named by --files0-from.
func (s *Sort) readNames() ([]string, error) {
	var (
		data []byte
		err  error
	)
	fromStdin := s.Files0From == "-"
	if fromStdin {
		if s.streams.In != nil {
			data, err = io.ReadAll(s.streams.In)
		}
	} else {
		data, err = os.ReadFile(s.Files0From)
	}
	if err != nil {
		return nil, s.die("cannot open '%s' for reading: %s", s.Files0From, core.ErrText(err))
	}

	data = bytes.TrimSuffix(data, []byte{0})
	if len(data) == 0 {
		return nil, s.die("no input from '%s'", s.Files0From)
	}
	var names []string
	for i, name := range strings.Split(string(data), "\x00") {
		switch {
		case name == "":
			return nil, s.die("%s:%d: invalid zero-length file name", s.Files0From, i+1)
		case name == "-" && fromStdin:
			return nil, s.die("when reading file names from stdin, no file name of '-' allowed")
		}
		names = append(names, name)
	}
	return names, nil
}

// read loads every line of one operand, without its delimiter. A last line
// with no delimiter is a line all the same.
func (s *Sort) read(name string) ([][]byte, error) {
	var data []byte
	if name == "-" {
		if s.streams.In != nil {
			var err error
			if data, err = io.ReadAll(s.streams.In); err != nil {
				return nil, err
			}
		}
	} else {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			// Reading a directory fails differently on every platform.
			return nil, errIsDir
		}
		if data, err = io.ReadAll(f); err != nil {
			return nil, err
		}
	}

	if len(data) == 0 {
		return nil, nil
	}
	data = bytes.TrimSuffix(data, []byte{s.delim})
	return bytes.Split(data, []byte{s.delim}), nil
}

// check reports the first line of in that is out of order.
func (s *Sort) check(in input) error {
	for i := 1; i < len(in.lines); i++ {
		c := s.compare(in.lines[i-1], in.lines[i])
		if c > 0 || (s.Unique && c == 0) {
			if s.Check == CheckDiagnose {
				s.diag.Warnf("%s:%d: disorder: %s", in.name, i+1, in.lines[i])
			}
			return &core.ExitError{Code: 1}
		}
	}
	return nil
}

// merge combines inputs that are each already sorted. On a tie the line from
// the earlier input comes first, as in GNU sort.
func (s *Sort) merge(inputs []input) [][]byte {
	pos := make([]int, len(inputs))
	var out [][]byte
	for {
		best := -1
		for i, in := range inputs {
			if pos[i] == len(in.lines) {
				continue
			}
			if best < 0 || s.compare(in.lines[pos[i]], inputs[best].lines[pos[best]]) < 0 {
				best = i
			}
		}
		if best < 0 {
			return out
		}
		out = append(out, inputs[best].lines[pos[best]])
		pos[best]++
	}
}

// uniq keeps the first line of every run of lines that compare equal.
func (s *Sort) uniq(lines [][]byte) [][]byte {
	if len(lines) == 0 {
		return lines
	}
	out := lines[:1]
	for _, line := range lines[1:] {
		if s.compare(out[len(out)-1], line) != 0 {
			out = append(out, line)
		}
	}
	return out
}

// write prints the result, to the -o file when one was opened.
func (s *Sort) write(lines [][]byte, file *os.File) error {
	w := s.streams.Out
	name := "standard output"
	if file != nil {
		if err := file.Truncate(0); err != nil {
			return s.die("write failed: %s: %s", s.Output, core.ErrText(err))
		}
		w, name = file, s.Output
	}

	bw := bufio.NewWriter(w)
	for _, line := range lines {
		_, _ = bw.Write(line)
		_ = bw.WriteByte(s.delim)
	}
	err := bw.Flush()
	if err == nil && file != nil {
		err = file.Close()
	}
	if err != nil {
		if core.IsClosedPipe(err) {
			return nil
		}
		return s.die("write failed: %s: %s", name, core.ErrText(err))
	}
	return nil
}

// compare orders two lines: by the keys, then, unless -s or -u, by the whole
// line in byte order.
func (s *Sort) compare(a, b []byte) int {
	if len(s.keys) > 0 {
		if d := s.keyCompare(a, b); d != 0 || s.Unique || s.Stable {
			return d
		}
	}
	d := bytes.Compare(a, b)
	if s.reverse {
		return -d
	}
	return d
}

// keyCompare compares a and b key by key, stopping at the first difference.
func (s *Sort) keyCompare(a, b []byte) int {
	for _, k := range s.keys {
		ta := s.field(a, k)
		tb := s.field(b, k)

		var d int
		switch {
		case k.numeric:
			d = numCompare(s.skipBlanks(ta), s.skipBlanks(tb))
		case k.general:
			d = generalCompare(ta, tb)
		case k.human:
			d = humanCompare(s.skipBlanks(ta), s.skipBlanks(tb))
		case k.month:
			d = s.month(ta) - s.month(tb)
		case k.version:
			d = verCompare(ta, tb)
		default:
			d = textCompare(ta, tb, k.ignore, k.translate)
		}
		if d != 0 {
			if k.reverse {
				return -d
			}
			return d
		}
	}
	return 0
}

// field returns the text of key k within line, as GNU's begfield/limfield
// compute it.
func (s *Sort) field(line []byte, k *key) []byte {
	beg := s.begField(line, k)
	end := len(line)
	if k.eword >= 0 {
		end = s.limField(line, k)
	}
	if end < beg {
		end = beg
	}
	return line[beg:end]
}

func (s *Sort) begField(line []byte, k *key) int {
	p, lim := 0, len(line)
	for sword := k.sword; p < lim && sword > 0; sword-- {
		p = s.nextField(line, p)
		if s.tab >= 0 && p < lim {
			p++
		}
	}
	if k.skipsblanks {
		for p < lim && s.blanks[line[p]] {
			p++
		}
	}
	return min(lim, p+k.schar)
}

func (s *Sort) limField(line []byte, k *key) int {
	p, lim := 0, len(line)
	eword, echar := k.eword, k.echar
	if echar == 0 {
		eword++ // the whole end field
	}
	for p < lim && eword > 0 {
		eword--
		p = s.nextField(line, p)
		if s.tab >= 0 && p < lim && (eword > 0 || echar != 0) {
			p++
		}
	}
	if echar != 0 {
		if k.skipeblanks {
			for p < lim && s.blanks[line[p]] {
				p++
			}
		}
		p = min(lim, p+echar)
	}
	return p
}

// nextField moves p to the end of the field it is in: up to the separator
// with -t, otherwise past the leading blanks and then the non-blanks.
func (s *Sort) nextField(line []byte, p int) int {
	lim := len(line)
	if s.tab >= 0 {
		for p < lim && int(line[p]) != s.tab {
			p++
		}
		return p
	}
	for p < lim && s.blanks[line[p]] {
		p++
	}
	for p < lim && !s.blanks[line[p]] {
		p++
	}
	return p
}

func (s *Sort) skipBlanks(b []byte) []byte {
	i := 0
	for i < len(b) && s.blanks[b[i]] {
		i++
	}
	return b[i:]
}

var months = []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

// month returns 1 to 12 for a field starting with a month's abbreviation, in
// any case and after blanks, and 0 for anything else.
func (s *Sort) month(b []byte) int {
	b = s.skipBlanks(b)
	if len(b) < 3 {
		return 0
	}
	m := strings.ToUpper(string(b[:3]))
	for i, name := range months {
		if m == name {
			return i + 1
		}
	}
	return 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// numCompare compares two numbers the way -n does, as strings and so without
// limits on their size: an optional '-', digits, an optional '.' and more
// digits. Anything that is not a number is zero.
func numCompare(a, b []byte) int {
	na, ia, fa := splitNumber(a)
	nb, ib, fb := splitNumber(b)
	if na != nb {
		if na {
			return -1
		}
		return 1
	}
	d := len(ia) - len(ib)
	if d == 0 {
		d = bytes.Compare(ia, ib)
	}
	if d == 0 {
		d = bytes.Compare(fa, fb)
	}
	if na {
		return -d
	}
	return d
}

// splitNumber returns the sign, integer digits (no leading zeros) and
// fraction digits (no trailing zeros) of the number b starts with.
func splitNumber(b []byte) (neg bool, intPart, frac []byte) {
	i := 0
	if i < len(b) && b[i] == '-' {
		neg = true
		i++
	}
	start := i
	for i < len(b) && isDigit(b[i]) {
		i++
	}
	intPart = bytes.TrimLeft(b[start:i], "0")
	if i < len(b) && b[i] == '.' {
		i++
		start = i
		for i < len(b) && isDigit(b[i]) {
			i++
		}
		frac = bytes.TrimRight(b[start:i], "0")
	}
	if len(intPart) == 0 && len(frac) == 0 {
		neg = false // -0 is 0
	}
	return neg, intPart, frac
}

// units ranks the SI suffixes -h understands.
var units = map[byte]int{'k': 1, 'K': 1, 'M': 2, 'G': 3, 'T': 4, 'P': 5, 'E': 6, 'Z': 7, 'Y': 8, 'R': 9, 'Q': 10}

// humanCompare orders by sign and suffix first, then by value, so 2K sorts
// after 1000 and before 1M.
func humanCompare(a, b []byte) int {
	if d := unitOrder(a) - unitOrder(b); d != 0 {
		return d
	}
	return numCompare(a, b)
}

func unitOrder(b []byte) int {
	neg := len(b) > 0 && b[0] == '-'
	i := 0
	if neg {
		i++
	}
	nonzero := false
	for i < len(b) && (isDigit(b[i]) || b[i] == '.') {
		if b[i] > '0' && b[i] <= '9' {
			nonzero = true
		}
		i++
	}
	if !nonzero || i == len(b) {
		return 0
	}
	if neg {
		return -units[b[i]]
	}
	return units[b[i]]
}

// generalCompare compares the floating-point numbers a and b start with, as
// strtod reads them. Text that is no number sorts first, then NaN, then the
// numbers in order.
func generalCompare(a, b []byte) int {
	x, okA := parseFloat(a)
	y, okB := parseFloat(b)
	switch {
	case !okA && !okB:
		return 0
	case !okA:
		return -1
	case !okB:
		return 1
	case x < y:
		return -1
	case x > y:
		return 1
	case x == y:
		return 0
	case !math.IsNaN(y): // x is NaN
		return -1
	case !math.IsNaN(x): // y is NaN
		return 1
	}
	return 0
}

// parseFloat reads the longest prefix of b, after leading white space, that is
// a floating-point number.
func parseFloat(b []byte) (float64, bool) {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] >= '\t' && b[i] <= '\r') {
		i++
	}
	j := i
	for j < len(b) && floatChar(b[j]) {
		j++
	}
	for end := j; end > i; end-- {
		v, err := strconv.ParseFloat(string(b[i:end]), 64)
		var ne *strconv.NumError
		if err == nil || errors.As(err, &ne) && errors.Is(ne.Err, strconv.ErrRange) {
			return v, true
		}
	}
	return 0, false
}

func floatChar(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '.' || c == '+' || c == '-'
}

// textCompare compares a and b byte by byte, after leaving out what -d or -i
// ignore and folding case for -f.
func textCompare(a, b []byte, ignore ignoreMode, fold bool) int {
	if ignore == ignoreNone && !fold {
		return bytes.Compare(a, b)
	}
	skip := func(c byte) bool {
		switch ignore {
		case ignoreNondictionary:
			return c != ' ' && c != '\t' && !isAlnum(c)
		case ignoreNonprinting:
			return c < 0x20 || c > 0x7e
		}
		return false
	}
	i, j := 0, 0
	for {
		for i < len(a) && skip(a[i]) {
			i++
		}
		for j < len(b) && skip(b[j]) {
			j++
		}
		if i == len(a) || j == len(b) {
			break
		}
		ca, cb := a[i], b[j]
		if fold {
			ca, cb = upper(ca), upper(cb)
		}
		if ca != cb {
			return int(ca) - int(cb)
		}
		i++
		j++
	}
	switch {
	case i < len(a):
		return 1
	case j < len(b):
		return -1
	}
	return 0
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isAlnum(c byte) bool { return isAlpha(c) || isDigit(c) }

func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

// verCompare is gnulib's filevercmp, which GNU sort -V uses: "." and ".."
// first, then other hidden names, then the rest, compared as versions with
// any file suffix considered only to break a tie.
func verCompare(a, b []byte) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return -1
	case len(b) == 0:
		return 1
	}
	if a[0] == '.' {
		if b[0] != '.' {
			return -1
		}
		adot, bdot := len(a) == 1, len(b) == 1
		switch {
		case adot && bdot:
			return 0
		case adot:
			return -1
		case bdot:
			return 1
		}
		adotdot := a[1] == '.' && len(a) == 2
		bdotdot := b[1] == '.' && len(b) == 2
		switch {
		case adotdot && bdotdot:
			return 0
		case adotdot:
			return -1
		case bdotdot:
			return 1
		}
	} else if b[0] == '.' {
		return 1
	}

	ap, bp := prefixLen(a), prefixLen(b)
	d := verrevcmp(a[:ap], b[:bp])
	if d != 0 || (ap == len(a) && bp == len(b)) {
		return d
	}
	return verrevcmp(a, b)
}

// prefixLen is the length of s without its file suffix, a run of
// (\.[A-Za-z~][A-Za-z0-9~]*)* at the end.
func prefixLen(s []byte) int {
	n := len(s)
	prefix := 0
	for i := 0; ; {
		if i == n {
			return prefix
		}
		i++
		prefix = i
		for i+1 < n && s[i] == '.' && (isAlpha(s[i+1]) || s[i+1] == '~') {
			for i += 2; i < n && (isAlnum(s[i]) || s[i] == '~'); i++ {
			}
		}
	}
}

// order ranks one character for verrevcmp: the end and '~' before
// everything, digits, letters, then everything else.
func order(s []byte, pos int) int {
	if pos >= len(s) {
		return -1
	}
	c := s[pos]
	switch {
	case isDigit(c):
		return 0
	case isAlpha(c):
		return int(c)
	case c == '~':
		return -2
	}
	return int(c) + 256
}

func verrevcmp(a, b []byte) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		for (i < len(a) && !isDigit(a[i])) || (j < len(b) && !isDigit(b[j])) {
			ca, cb := order(a, i), order(b, j)
			if ca != cb {
				return ca - cb
			}
			i++
			j++
		}
		for i < len(a) && a[i] == '0' {
			i++
		}
		for j < len(b) && b[j] == '0' {
			j++
		}
		first := 0
		for i < len(a) && j < len(b) && isDigit(a[i]) && isDigit(b[j]) {
			if first == 0 {
				first = int(a[i]) - int(b[j])
			}
			i++
			j++
		}
		if i < len(a) && isDigit(a[i]) {
			return 1
		}
		if j < len(b) && isDigit(b[j]) {
			return -1
		}
		if first != 0 {
			return first
		}
	}
	return 0
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")
