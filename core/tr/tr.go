// Package tr implements the engine behind trx, a GNU-compatible implementation
// of the GNU/UNIX tr command: translate, squeeze and delete bytes read from
// standard input, writing the result to standard output.
//
// Like GNU tr, it works on bytes: a multibyte UTF-8 character is several
// separate bytes to it, and the character classes are those of the C locale.
package tr

import (
	"bufio"
	"errors"
	"io"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "trx"

// bufSize is the read chunk.
const bufSize = 64 << 10

// Options mirrors the GNU tr flag surface.
type Options struct {
	// Complement (-c, -C) uses every byte not in SET1 instead of SET1, in
	// ascending order.
	Complement bool
	// Delete (-d) deletes the bytes in SET1 instead of translating them.
	Delete bool
	// Squeeze (-s) replaces each run of a repeated byte from the last SET
	// given with a single occurrence.
	Squeeze bool
	// Truncate (-t) first truncates SET1 to the length of SET2.
	Truncate bool
}

// Tr is the engine. Construct it with New.
type Tr struct {
	Options

	streams core.Streams
	diag    *core.Diag

	translate bool
	table     [256]byte
	del       [256]bool
	squeeze   [256]bool
}

// New builds a tr engine reading from and writing to s.
func New(opts Options, s core.Streams) *Tr {
	return &Tr{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// Run validates the SET operands and filters standard input to standard
// output. It returns a *core.UsageError for a wrong number of operands,
// core.ErrFailed for an invalid SET or an I/O error, otherwise nil.
func (t *Tr) Run(args []string) error {
	if err := t.checkOperands(args); err != nil {
		return err
	}
	if err := t.build(args); err != nil {
		var se *setError
		if errors.As(err, &se) {
			t.diag.Errorf("%s", se.msg)
			return t.diag.Result()
		}
		return err
	}
	t.filter()
	return t.diag.Result()
}

// checkOperands enforces GNU's operand counts for the chosen mode.
func (t *Tr) checkOperands(args []string) error {
	twoSets := t.Squeeze && t.Delete || !t.Squeeze && !t.Delete
	switch {
	case len(args) == 0:
		return core.Usagef("missing operand")
	case len(args) == 1 && twoSets:
		why := "Two strings must be given when translating."
		if t.Delete {
			why = "Two strings must be given when both deleting and squeezing repeats."
		}
		return core.Usagef("missing operand after '%s'\n%s", args[0], why)
	case len(args) == 2 && t.Delete && !t.Squeeze:
		return core.Usagef("extra operand '%s'\nOnly one string may be given when deleting without squeezing repeats.",
			args[1])
	case len(args) > 2:
		return core.Usagef("extra operand '%s'", args[2])
	}
	return nil
}

// build parses the SETs and fills the translation, deletion and squeeze
// tables.
func (t *Tr) build(args []string) error {
	warn := func(msg string) { t.diag.Warnf("%s", msg) }

	elems1, err := parse(args[0], warn)
	if err != nil {
		return err
	}
	for _, e := range elems1 {
		if e.kind == kindRepeat && e.fill {
			return setErrorf("the [c*] repeat construct may not appear in string1")
		}
	}
	set1 := expand(elems1, 0)
	if t.Complement {
		set1 = complement(set1.bytes)
	}

	var elems2 []element
	if len(args) == 2 {
		if elems2, err = parse(args[1], warn); err != nil {
			return err
		}
	}
	t.translate = len(args) == 2 && !t.Delete

	switch {
	case t.Delete:
		for _, c := range set1.bytes {
			t.del[c] = true
		}
		if t.Squeeze {
			mark(&t.squeeze, expand(elems2, 0).bytes)
		}
	case t.translate:
		set2, err := t.translation(set1, elems2)
		if err != nil {
			return err
		}
		if t.Squeeze {
			mark(&t.squeeze, set2)
		}
	default: // -s with one SET
		mark(&t.squeeze, set1.bytes)
	}
	return nil
}

// translation builds the translation table from SET1 to SET2 and returns
// SET2 as written out (before it is padded to the length of SET1).
func (t *Tr) translation(set1 expanded, elems2 []element) ([]byte, error) {
	fills := 0
	for _, e := range elems2 {
		switch e.kind {
		case kindClass:
			if e.class != "upper" && e.class != "lower" {
				return nil, setErrorf("when translating, the only character classes that may appear in string2 are 'upper' and 'lower'")
			}
		case kindEquiv:
			return nil, setErrorf("[=c=] expressions may not appear in string2 when translating")
		case kindRepeat:
			if e.fill {
				fills++
			}
		}
	}
	if fills > 1 {
		return nil, setErrorf("only one [c*] repeat construct may appear in string2")
	}
	fixed := len(expand(elems2, 0).bytes)
	set2 := expand(elems2, max(0, len(set1.bytes)-fixed))

	// [:upper:] and [:lower:] in SET2 must line up with a case class in
	// SET1, so that each letter maps to its counterpart.
	for at := range set2.caseAt {
		if _, ok := set1.caseAt[at]; !ok {
			return nil, setErrorf("misaligned [:upper:] and/or [:lower:] construct")
		}
	}

	s1, s2 := set1.bytes, set2.bytes
	switch {
	case t.Truncate:
		if len(s1) > len(s2) {
			s1 = s1[:len(s2)]
		}
	case len(s2) == 0 && len(s1) > 0:
		return nil, setErrorf("when not truncating set1, string2 must be non-empty")
	}
	for i := range t.table {
		t.table[i] = byte(i)
	}
	for i, c := range s1 {
		if i < len(s2) {
			t.table[c] = s2[i]
		} else {
			t.table[c] = s2[len(s2)-1]
		}
	}
	return s2, nil
}

// complement returns every byte not in set, in ascending order.
func complement(set []byte) expanded {
	var in [256]bool
	mark(&in, set)
	x := expanded{caseAt: map[int]string{}}
	for c := range 256 {
		if !in[c] {
			x.bytes = append(x.bytes, byte(c))
		}
	}
	return x
}

func mark(table *[256]bool, set []byte) {
	for _, c := range set {
		table[c] = true
	}
}

// filter copies standard input to standard output through the tables.
func (t *Tr) filter() {
	out := bufio.NewWriterSize(t.streams.Out, bufSize)
	buf := make([]byte, bufSize)
	last, haveLast := byte(0), false
	for {
		n, rerr := t.streams.In.Read(buf)
		for _, c := range buf[:n] {
			if t.del[c] {
				continue
			}
			if t.translate {
				c = t.table[c]
			}
			if t.squeeze[c] && haveLast && last == c {
				continue
			}
			last, haveLast = c, true
			_ = out.WriteByte(c)
		}
		// Write errors are sticky in the buffered writer.
		if _, werr := out.Write(nil); werr != nil {
			t.writeError(werr)
			return
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				t.diag.Errorf("read error: %s", core.ErrText(rerr))
			}
			break
		}
	}
	if err := out.Flush(); err != nil {
		t.writeError(err)
	}
}

func (t *Tr) writeError(err error) {
	if !core.IsClosedPipe(err) {
		t.diag.Errorf("write error: %s", core.ErrText(err))
	}
}
