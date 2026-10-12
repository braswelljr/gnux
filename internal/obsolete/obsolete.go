// Package obsolete lets an adapter accept GNU's obsolete "-NUM" option form
// (fold -72, fmt -60, split -1000, expand -4) alongside the flag that
// replaced it: either the one given last wins (Number and String), or both
// add to one list in the order given (List).
//
// pflag would reject -72 as an unknown shorthand. Registering each digit as a
// hidden, argument-less shorthand makes pflag deliver "-72" as '7' then '2',
// which Number reassembles.
package obsolete

import (
	"strconv"

	"github.com/spf13/pflag"
)

// Order stamps flags as they are parsed, since pflag does not record the
// order flags appeared in.
type Order struct{ n int }

func (o *Order) next() int {
	o.n++
	return o.n
}

// Number is the obsolete -NUM value: its digits and when it was given (0 if
// it was not).
type Number struct {
	Text string
	At   int
	ord  *Order
}

// Given reports whether -NUM appeared on the command line.
func (n *Number) Given() bool { return n.At > 0 }

// Register adds the hidden digit shorthands to fs.
func Register(fs *pflag.FlagSet, ord *Order) *Number {
	num := &Number{ord: ord}
	registerDigits(fs, num)
	return num
}

func (n *Number) digit(d byte) {
	at := n.ord.next()
	// Digits given after another flag start a new number.
	if n.At != at-1 {
		n.Text = ""
	}
	n.Text += string(d)
	n.At = at
}

// sink receives the digits of -NUM as pflag delivers them.
type sink interface{ digit(d byte) }

func registerDigits(fs *pflag.FlagSet, s sink) {
	for d := range 10 {
		f := fs.VarPF(&digit{sink: s, d: byte('0' + d)}, "obsolete-"+strconv.Itoa(d), strconv.Itoa(d), "")
		f.NoOptDefVal = "true"
		f.Hidden = true
	}
}

type digit struct {
	sink sink
	d    byte
}

func (v *digit) String() string { return "" }

func (v *digit) Type() string { return "bool" }

func (v *digit) Set(string) error {
	v.sink.digit(v.d)
	return nil
}

// String is a string flag value that records when it was set, so it can be
// compared with a Number.
type String struct {
	Text string
	At   int
	ord  *Order
	typ  string
}

// NewString returns a String stamped by ord; typ is the placeholder --help
// shows for its value.
func NewString(ord *Order, typ string) *String { return &String{ord: ord, typ: typ} }

func (v *String) String() string { return v.Text }

// Type is what --help shows as the value placeholder.
func (v *String) Type() string { return v.typ }

// Set records the value and when it was given.
func (v *String) Set(s string) error {
	v.Text, v.At = s, v.ord.next()
	return nil
}

// Last returns the text of whichever of s and n was given last, and whether
// either was given at all.
func Last(s *String, n *Number) (string, bool) {
	switch {
	case s.At == 0 && n.At == 0:
		return "", false
	case s.At > n.At:
		return s.Text, true
	}
	return n.Text, true
}

// Lookup returns the Number Register added to fs.
func Lookup(fs *pflag.FlagSet) *Number {
	return fs.Lookup("obsolete-0").Value.(*digit).sink.(*Number)
}

// List collects the values of one flag and of -NUM, in the order given.
type List struct {
	Items []string
	typ   string
	n     int // stamp of the last Set or digit
	digAt int // stamp of the last digit
}

// RegisterList adds the hidden digit shorthands to fs and returns a List
// they feed; register the flag that shares it with fs.VarP(list, ...). typ is
// the placeholder --help shows for the flag's value.
func RegisterList(fs *pflag.FlagSet, typ string) *List {
	l := &List{typ: typ}
	registerDigits(fs, l)
	return l
}

// LookupList returns the List RegisterList added to fs.
func LookupList(fs *pflag.FlagSet) *List {
	return fs.Lookup("obsolete-0").Value.(*digit).sink.(*List)
}

func (l *List) digit(d byte) {
	l.n++
	// Digits run together until another flag interrupts them.
	if l.digAt == l.n-1 && len(l.Items) > 0 {
		l.Items[len(l.Items)-1] += string(d)
	} else {
		l.Items = append(l.Items, string(d))
	}
	l.digAt = l.n
}

func (l *List) String() string { return "" }

// Type is what --help shows as the value placeholder.
func (l *List) Type() string { return l.typ }

// Set appends one value of the flag.
func (l *List) Set(s string) error {
	l.n++
	l.Items = append(l.Items, s)
	return nil
}
