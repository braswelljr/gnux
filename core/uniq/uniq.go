// Package uniq implements the engine behind uniqx, a GNU-compatible
// implementation of the GNU/UNIX uniq command: report or omit repeated lines.
package uniq

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "uniqx"

// Delimit says how -D/--all-repeated separates the groups of duplicates.
type Delimit int

const (
	// DelimitNone prints the groups back to back (-D, --all-repeated=none).
	DelimitNone Delimit = iota
	// DelimitPrepend prints an empty line before every group.
	DelimitPrepend
	// DelimitSeparate prints an empty line between groups.
	DelimitSeparate
)

// Group is the --group=METHOD mode, which prints every line and separates
// the groups of equal lines with an empty line.
type Group int

const (
	// GroupNone is plain uniq: --group was not given.
	GroupNone Group = iota
	// GroupSeparate separates the groups with an empty line.
	GroupSeparate
	// GroupPrepend prints an empty line before every group.
	GroupPrepend
	// GroupAppend prints an empty line after every group.
	GroupAppend
	// GroupBoth prints an empty line before and after every group.
	GroupBoth
)

// Options mirrors the GNU uniq flag surface. The zero value is plain uniq:
// one copy of each run of equal adjacent lines, whole lines compared.
type Options struct {
	// Count (-c) prefixes each output line with the number of occurrences.
	Count bool
	// Repeated (-d) prints one copy of each group of duplicate lines only.
	Repeated bool
	// AllRepeated (-D, --all-repeated) prints every line of each group of
	// duplicates; Delimit says how the groups are separated.
	AllRepeated bool
	// Delimit is the --all-repeated=METHOD separator; it applies only with
	// AllRepeated.
	Delimit Delimit
	// Group (--group[=METHOD]) prints every line, the groups separated.
	Group Group
	// SkipFields (-f N) ignores the first N fields when comparing. A field is
	// a run of blanks followed by a run of non-blanks.
	SkipFields int64
	// SkipChars (-s N) ignores the first N bytes, after the skipped fields.
	SkipChars int64
	// CheckChars (-w N) compares at most N bytes; it applies only when
	// CheckCharsSet is true, so the zero value compares whole lines.
	CheckChars int64
	// CheckCharsSet records that -w was given.
	CheckCharsSet bool
	// IgnoreCase (-i) compares ASCII letters without regard to case.
	IgnoreCase bool
	// Unique (-u) prints only the lines that are not repeated.
	Unique bool
	// ZeroTerminated (-z) makes NUL, not newline, the line delimiter.
	ZeroTerminated bool
}

// Uniq is the engine. Construct it with New.
type Uniq struct {
	Options

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	delim   byte
}

// New builds a uniq engine reading from and writing to s.
func New(opts Options, s core.Streams) *Uniq {
	u := &Uniq{
		Options: opts,
		streams: s,
		diag:    core.NewDiag(Name, s),
		delim:   '\n',
	}
	if opts.ZeroTerminated {
		u.delim = 0
	}
	return u
}

// Run filters INPUT (standard input when absent or "-") into OUTPUT
// (standard output when absent or "-"). The returned error is a
// *core.UsageError for an invalid combination of options or operands,
// core.ErrFailed when the input or output failed, otherwise nil.
func (u *Uniq) Run(args []string) error {
	if len(args) > 2 {
		return core.Usagef("extra operand '%s'", args[2])
	}
	if u.Group != GroupNone && (u.Count || u.Repeated || u.AllRepeated || u.Unique) {
		return core.Usagef("--group is mutually exclusive with -c/-d/-D/-u")
	}
	if u.AllRepeated && u.Count {
		return core.Usagef("printing all duplicated lines and repeat counts is meaningless")
	}

	in := "-"
	if len(args) > 0 {
		in = args[0]
	}
	out := "-"
	if len(args) > 1 {
		out = args[1]
	}

	r := u.streams.In
	if in != "-" {
		f, err := os.Open(in)
		if err != nil {
			u.diag.Errorf("%s: %s", in, core.ErrText(err))
			return u.diag.Result()
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			u.diag.Errorf("error reading '%s': %s", in, core.ErrText(errIsDir))
			return u.diag.Result()
		}
		r = f
	}
	if r == nil {
		r = strings.NewReader("")
	}

	w := u.streams.Out
	if out != "-" {
		f, err := os.Create(out)
		if err != nil {
			u.diag.Errorf("%s: %s", out, core.ErrText(err))
			return u.diag.Result()
		}
		defer f.Close()
		w = f
	}
	u.out = bufio.NewWriter(w)

	display := in
	if in == "-" {
		display = "standard input"
	}
	err := u.filter(bufio.NewReader(r))
	if werr := u.out.Flush(); werr != nil {
		if !core.IsClosedPipe(werr) {
			u.diag.Errorf("write error: %s", core.ErrText(werr))
		}
		return u.diag.Result()
	}
	if err != nil {
		u.diag.Errorf("error reading '%s': %s", display, core.ErrText(err))
	}
	return u.diag.Result()
}

// errIsDir is rendered by core.ErrText as GNU's "Is a directory".
var errIsDir = errors.New("is a directory")

// readLine returns the next line without its delimiter. ok is false at the
// end of the input; a last line with no delimiter is still a line.
func (u *Uniq) readLine(r *bufio.Reader) (line []byte, ok bool, err error) {
	line, err = r.ReadBytes(u.delim)
	if len(line) > 0 && line[len(line)-1] == u.delim {
		line = line[:len(line)-1]
		return line, true, nil
	}
	if errors.Is(err, io.EOF) {
		return line, len(line) > 0, nil
	}
	return line, len(line) > 0, err
}

// filter is GNU uniq's main loop. Plain uniq and --group take the simple
// path; the counting and duplicate-selecting modes share the general one,
// whose comparison anchor and group delimiters follow GNU exactly.
func (u *Uniq) filter(r *bufio.Reader) error {
	if !u.Count && !u.Repeated && !u.AllRepeated && !u.Unique {
		return u.simple(r)
	}

	// output_unique / output_first_repeated / output_later_repeated in GNU.
	outUnique := !u.Repeated && !u.AllRepeated
	outFirst := !u.Unique
	outLater := u.AllRepeated

	prev, ok, err := u.readLine(r)
	if !ok {
		return err
	}
	var (
		matchCount     int64
		firstDelimiter = true
	)
	write := func(line []byte, match bool, n int64) {
		var want bool
		switch {
		case n == 0:
			want = outUnique
		case !match:
			want = outFirst
		default:
			want = outLater
		}
		if !want {
			return
		}
		if u.Count {
			fmt.Fprintf(u.out, "%7d ", n+1)
		}
		_, _ = u.out.Write(line)
		_ = u.out.WriteByte(u.delim)
	}

	for {
		line, ok, rerr := u.readLine(r)
		if !ok {
			err = rerr
			break
		}
		match := !u.different(line, prev)
		if match {
			matchCount++
		}

		if u.AllRepeated && u.Delimit != DelimitNone {
			switch {
			case !match:
				if matchCount > 0 {
					firstDelimiter = false
				}
			case matchCount == 1:
				if u.Delimit == DelimitPrepend || (u.Delimit == DelimitSeparate && !firstDelimiter) {
					_ = u.out.WriteByte(u.delim)
				}
			}
		}

		if !match || outLater {
			write(prev, match, matchCount)
			prev = line
			if !match {
				matchCount = 0
			}
		}
		if rerr != nil {
			err = rerr
			break
		}
	}
	write(prev, false, matchCount)
	return err
}

// simple prints the first line of each group, and with --group every line
// with the groups delimited.
func (u *Uniq) simple(r *bufio.Reader) error {
	var (
		prev         []byte
		havePrev     bool
		printedGroup bool
	)
	for {
		line, ok, err := u.readLine(r)
		if !ok {
			if (u.Group == GroupAppend || u.Group == GroupBoth) && printedGroup {
				_ = u.out.WriteByte(u.delim)
			}
			return err
		}
		newGroup := !havePrev || u.different(line, prev)
		if newGroup && u.Group != GroupNone &&
			(u.Group == GroupPrepend || u.Group == GroupBoth ||
				(printedGroup && (u.Group == GroupAppend || u.Group == GroupSeparate))) {
			_ = u.out.WriteByte(u.delim)
		}
		if newGroup || u.Group != GroupNone {
			_, _ = u.out.Write(line)
			_ = u.out.WriteByte(u.delim)
			prev, havePrev = line, true
			printedGroup = true
		}
		if err != nil {
			return err
		}
	}
}

// key returns the part of line that is compared: after the skipped fields
// and characters, at most CheckChars bytes.
func (u *Uniq) key(line []byte) []byte {
	i := 0
	for n := int64(0); n < u.SkipFields && i < len(line); n++ {
		for i < len(line) && isBlank(line[i]) {
			i++
		}
		for i < len(line) && !isBlank(line[i]) {
			i++
		}
	}
	i += int(min(u.SkipChars, int64(len(line)-i)))
	k := line[i:]
	if u.CheckCharsSet && u.CheckChars < int64(len(k)) {
		k = k[:u.CheckChars]
	}
	return k
}

// different reports whether two lines belong to different groups.
func (u *Uniq) different(a, b []byte) bool {
	ka, kb := u.key(a), u.key(b)
	if u.IgnoreCase {
		return !equalFoldASCII(ka, kb)
	}
	return !bytes.Equal(ka, kb)
}

// isBlank is GNU's field separator test: a space, a tab or a newline.
func isBlank(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

// equalFoldASCII compares a and b, folding only ASCII letters, as GNU uniq
// does in the C locale.
func equalFoldASCII(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if lower(a[i]) != lower(b[i]) {
			return false
		}
	}
	return true
}

func lower(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}
