// Package tsort implements the engine behind tsortx, a GNU-compatible
// implementation of the GNU tsort command: print a total ordering consistent
// with the partial ordering given as pairs of strings.
package tsort

import (
	"bufio"
	"errors"
	"io"
	"slices"
	"syscall"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "tsortx"

// Options mirrors the GNU tsort flag surface, which has no options of its
// own. It is here so that tsort has the same shape as every other engine.
type Options struct{}

// Tsort is the engine. Construct it with New.
type Tsort struct {
	Options

	streams core.Streams
	diag    *core.Diag
}

// New builds a tsort engine reading from and writing to s.
func New(opts Options, s core.Streams) *Tsort {
	return &Tsort{Options: opts, streams: s, diag: core.NewDiag(Name, s)}
}

// item is one string of the input: a node of the graph.
type item struct {
	str     string
	count   int        // predecessors not yet printed
	top     *successor // successors, the most recently read first
	qlink   *item      // the next item in the output queue, or on a loop's path
	printed bool
}

type successor struct {
	suc  *item
	next *successor
}

// Run reads pairs of whitespace-separated strings from the operand (standard
// input when there is none, or for "-") and prints every string once, each
// after every string paired before it. It runs GNU's algorithm (Knuth's
// algorithm T), so the order matches GNU's: strings with no predecessor are
// taken in byte order, then in the order they become free. A loop is
// reported on standard error, one of its relations is dropped and sorting
// carries on, and the run fails. An odd number of strings prints nothing.
func (t *Tsort) Run(args []string) (err error) {
	if len(args) > 1 {
		return core.Usagef("extra operand '%s'", args[1])
	}
	name := "-"
	if len(args) == 1 {
		name = args[0]
	}

	r := t.streams.In
	if name != "-" {
		f, err := fsys.Open(name)
		if err != nil {
			t.diag.Errorf("%s: %s", name, core.ErrText(err))
			return t.diag.Result()
		}
		defer f.Close()
		if info, err := f.Stat(); err == nil && info.IsDir() {
			t.diag.Errorf("%s: %s", name, core.ErrText(syscall.EISDIR))
			return t.diag.Result()
		}
		r = f
	}

	items := map[string]*item{}
	lookup := func(s string) *item {
		it := items[s]
		if it == nil {
			it = &item{str: s}
			items[s] = it
		}
		return it
	}

	in := core.NewReader(r)
	var j *item
	var tok []byte
	for {
		c, rerr := in.ReadByte()
		if rerr == nil && c != ' ' && c != '\t' && c != '\n' {
			tok = append(tok, c)
			continue
		}
		if len(tok) > 0 {
			k := lookup(string(tok))
			tok = tok[:0]
			if j == nil {
				j = k
			} else {
				if j != k {
					k.count++
					j.top = &successor{suc: k, next: j.top}
				}
				j = nil
			}
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				t.diag.Errorf("%s: %s", name, core.ErrText(rerr))
				return t.diag.Result()
			}
			break
		}
	}
	if j != nil {
		t.diag.Errorf("%s: input contains an odd number of tokens", name)
		return t.diag.Result()
	}

	// GNU keeps the strings in a search tree and walks it in order, which
	// is byte order.
	sorted := make([]*item, 0, len(items))
	for _, it := range items {
		sorted = append(sorted, it)
	}
	slices.SortFunc(sorted, func(a, b *item) int {
		switch {
		case a.str < b.str:
			return -1
		case a.str > b.str:
			return 1
		}
		return 0
	})

	out := core.NewWriter(t.streams.Out)
	defer core.FlushOutput(out, t.diag, &err)
	t.sort(out, sorted, name)
	return t.diag.Result()
}

// sort is algorithm T over the items, in byte order.
func (t *Tsort) sort(out *bufio.Writer, sorted []*item, name string) {
	left := len(sorted)
	for left > 0 {
		// T4: queue every item that has no predecessor left.
		var head, zeros *item
		for _, k := range sorted {
			if k.count == 0 && !k.printed {
				if head == nil {
					head = k
				} else {
					zeros.qlink = k
				}
				zeros = k
			}
		}

		for head != nil {
			// T5: output the front of the queue.
			_, _ = out.WriteString(head.str)
			_ = out.WriteByte('\n')
			head.printed = true
			left--
			// T6: erase its relations.
			for p := head.top; p != nil; p = p.next {
				p.suc.count--
				if p.suc.count == 0 {
					zeros.qlink = p.suc
					zeros = p.suc
				}
			}
			// T7: remove it from the queue.
			head = head.qlink
		}

		// T8: whatever is left is part of a loop, or after one.
		if left > 0 {
			// The output goes first, so that it and the report stay in
			// order on a shared terminal.
			_ = out.Flush()
			t.diag.Errorf("%s: input contains a loop:", name)
			var loop *item
			for {
				found := false
				for _, k := range sorted {
					if loop, found = t.detectLoop(k, loop); found {
						break
					}
				}
				if loop == nil {
					break
				}
			}
		}
	}
}

// detectLoop is GNU's detect_loop, called for each item in order until a
// loop is found: it follows predecessors back from the first item still
// waiting until it meets one it has already passed, reports that loop and
// removes the relation that closes it. It returns the new start of the path
// and whether the walk should stop.
func (t *Tsort) detectLoop(k, loop *item) (*item, bool) {
	if k.count == 0 {
		return loop, false
	}
	if loop == nil {
		// Start the walk at k.
		return k, false
	}
	for p := &k.top; *p != nil; p = &(*p).next {
		if (*p).suc != loop {
			continue
		}
		if k.qlink == nil {
			// k leads to the start of the path: extend the path back to k.
			k.qlink = loop
			return k, false
		}
		// k is already on the path: what lies between is a loop.
		for loop != nil {
			next := loop.qlink
			t.diag.Errorf("%s", loop.str)
			if loop == k {
				(*p).suc.count--
				*p = (*p).next
				break
			}
			loop.qlink = nil
			loop = next
		}
		for loop != nil {
			next := loop.qlink
			loop.qlink = nil
			loop = next
		}
		return nil, true
	}
	return loop, false
}
