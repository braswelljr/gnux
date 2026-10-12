package expand

import (
	"math"

	"github.com/braswelljr/gnux/core"
)

// TabStops is a list of tab stops in GNU's -t syntax, shared by expand and
// unexpand. Build it with Parse and Add, then call Finalize before Next.
type TabStops struct {
	list      []uint64
	extend    uint64 // "/N": every multiple of N after the last stop
	increment uint64 // "+N": every N columns after the last stop
	size      uint64 // a single, repeating tab size; 0 when list applies
	maxWidth  uint64 // the widest gap between two stops
}

// Parse adds the stops in spec: numbers separated by commas or blanks, the
// last of which may be written /N or +N. Errors are written to diag in GNU's
// words; Parse reports whether spec was valid.
func (t *TabStops) Parse(spec string, diag *core.Diag) bool {
	var (
		have, extend, increment bool
		val                     uint64
		numStart                int
	)
	ok := true
	for i := 0; i < len(spec); i++ {
		c := spec[i]
		switch {
		case c == ',' || c == ' ' || c == '\t':
			if have {
				switch {
				case extend:
					ok = t.setExtend(val, diag)
				case increment:
					ok = t.setIncrement(val, diag)
				default:
					t.Add(val)
				}
				if !ok {
					return false
				}
			}
			have = false
		case c == '/':
			if have {
				diag.Errorf("'/' specifier not at start of number: %s", quote(spec[i:]))
				ok = false
			}
			extend, increment = true, false
		case c == '+':
			if have {
				diag.Errorf("'+' specifier not at start of number: %s", quote(spec[i:]))
				ok = false
			}
			increment, extend = true, false
		case c >= '0' && c <= '9':
			if !have {
				val, have, numStart = 0, true, i
			}
			d := uint64(c - '0')
			if val > (math.MaxUint64-d)/10 {
				end := numStart
				for end < len(spec) && spec[end] >= '0' && spec[end] <= '9' {
					end++
				}
				diag.Errorf("tab stop is too large %s", quote(spec[numStart:end]))
				ok = false
				i = end - 1
				continue
			}
			val = val*10 + d
		default:
			diag.Errorf("tab size contains invalid character(s): %s", quote(spec[i:]))
			return false
		}
	}
	if ok && have {
		switch {
		case extend:
			ok = t.setExtend(val, diag)
		case increment:
			ok = t.setIncrement(val, diag)
		default:
			t.Add(val)
		}
	}
	return ok
}

// Add appends one tab stop.
func (t *TabStops) Add(stop uint64) {
	prev := uint64(0)
	if n := len(t.list); n > 0 {
		prev = t.list[n-1]
	}
	if prev <= stop && stop-prev > t.maxWidth {
		t.maxWidth = stop - prev
	}
	t.list = append(t.list, stop)
}

func (t *TabStops) setExtend(n uint64, diag *core.Diag) bool {
	ok := true
	if t.extend != 0 {
		diag.Errorf("'/' specifier only allowed with the last value")
		ok = false
	}
	t.extend = n
	return ok
}

func (t *TabStops) setIncrement(n uint64, diag *core.Diag) bool {
	ok := true
	if t.increment != 0 {
		diag.Errorf("'+' specifier only allowed with the last value")
		ok = false
	}
	t.increment = n
	return ok
}

// Finalize checks the list (nonzero, ascending, not both /N and +N) and
// fixes the tab size. With no stops at all, tabs are every 8 columns.
func (t *TabStops) Finalize(diag *core.Diag) bool {
	prev := uint64(0)
	for _, s := range t.list {
		if s == 0 {
			diag.Errorf("tab size cannot be 0")
			return false
		}
		if s <= prev {
			diag.Errorf("tab sizes must be ascending")
			return false
		}
		prev = s
	}
	if t.increment != 0 && t.extend != 0 {
		diag.Errorf("'/' specifier is mutually exclusive with '+'")
		return false
	}
	switch {
	case len(t.list) == 0:
		t.size = 8
		if t.extend != 0 {
			t.size = t.extend
		} else if t.increment != 0 {
			t.size = t.increment
		}
		t.maxWidth = t.size
	case len(t.list) == 1 && t.extend == 0 && t.increment == 0:
		t.size = t.list[0]
	default:
		t.size = 0
	}
	return true
}

// MaxWidth returns the widest distance between two tab stops.
func (t *TabStops) MaxWidth() uint64 {
	return t.maxWidth
}

// Next returns the column of the tab stop after col. idx is the caller's
// position in the list, carried from one call to the next on a line. last
// reports that col is past the final stop, where a tab is a single space
// (expand) or blanks are left alone (unexpand).
func (t *TabStops) Next(col uint64, idx *int) (next uint64, last bool) {
	if t.size != 0 {
		return col + (t.size - col%t.size), false
	}
	for ; *idx < len(t.list); *idx++ {
		if col < t.list[*idx] {
			return t.list[*idx], false
		}
	}
	if t.extend != 0 {
		return col + (t.extend - col%t.extend), false
	}
	if t.increment != 0 {
		end := t.list[len(t.list)-1]
		return col + (t.increment - (col-end)%t.increment), false
	}
	return 0, true
}

// quote renders s as GNU's quote() does for plain text: in single quotes.
func quote(s string) string {
	return "'" + s + "'"
}
