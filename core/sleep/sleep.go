// Package sleep implements the engine behind sleepx, a GNU-compatible implementation of
// the GNU/UNIX sleep command: pause for the sum of the given time intervals.
package sleep

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "sleepx"

// Forever is the duration Sleep receives for an endless pause: an "infinity"
// operand, or a total too large for a time.Duration to hold (about 292 years).
const Forever = time.Duration(math.MaxInt64)

// Sleep is the engine. Construct it with New.
type Sleep struct {
	// Sleep pauses for d, or without end when d is Forever. It is the
	// injection point tests use to observe the computed duration without
	// waiting; it defaults to a timer-based pause.
	Sleep func(d time.Duration)

	diag *core.Diag
}

// New builds a sleep engine writing its diagnostics to s.
func New(s core.Streams) *Sleep {
	return &Sleep{Sleep: pause, diag: core.NewDiag(Name, s)}
}

// pause blocks for d. Forever is honored by sleeping again each time the
// longest representable timer runs out; a bare `select {}` would instead trip
// the runtime's deadlock detector, as no other goroutine is running.
func pause(d time.Duration) {
	time.Sleep(d)
	for d == Forever {
		time.Sleep(d)
	}
}

// Run pauses for the sum of the NUMBER[SUFFIX] operands. Like GNU sleep it
// validates every operand before sleeping at all, reporting each invalid one.
func (s *Sleep) Run(args []string) error {
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}

	d, invalid := Total(args)
	if len(invalid) > 0 {
		// GNU prints one diagnostic per bad operand, then the usage hint once.
		// The caller appends that hint to the returned error, so all but the
		// last diagnostic are written here.
		last := len(invalid) - 1
		for _, arg := range invalid[:last] {
			s.diag.Warnf("invalid time interval '%s'", arg)
		}
		return core.Usagef("invalid time interval '%s'", invalid[last])
	}

	s.Sleep(d)
	return nil
}

// Total adds up the intervals named by args and returns the operands that are
// not valid intervals. The sum is Forever when it is infinite or does not fit
// in a time.Duration.
func Total(args []string) (total time.Duration, invalid []string) {
	// Summed in floating-point seconds, as GNU does, so that many small
	// fractions and very large values behave the same way.
	seconds := 0.0
	for _, arg := range args {
		v, ok := parseInterval(arg)
		if !ok {
			invalid = append(invalid, arg)
			continue
		}
		seconds += v
	}
	if len(invalid) > 0 {
		return 0, invalid
	}

	ns := math.Round(seconds * float64(time.Second))
	if ns >= float64(Forever) {
		return Forever, nil
	}
	return time.Duration(ns), nil
}

// parseInterval converts one NUMBER[SUFFIX] operand to seconds. NUMBER is a
// non-negative decimal or hexadecimal floating-point number, or "inf" /
// "infinity"; SUFFIX is s (seconds, the default), m, h or d.
func parseInterval(arg string) (seconds float64, ok bool) {
	number, unit := arg, 1.0
	if arg != "" {
		switch arg[len(arg)-1] {
		case 's':
			number = arg[:len(arg)-1]
		case 'm':
			number, unit = arg[:len(arg)-1], 60
		case 'h':
			number, unit = arg[:len(arg)-1], 60*60
		case 'd':
			number, unit = arg[:len(arg)-1], 24*60*60
		}
	}

	// ParseFloat accepts Go's digit-separating underscores in hexadecimal
	// numbers; strtod, which GNU uses, does not.
	if strings.Contains(number, "_") {
		return 0, false
	}
	v, err := strconv.ParseFloat(number, 64)
	// A range error still carries the right answer: ±Inf or (signed) zero.
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	if math.IsNaN(v) || v < 0 {
		return 0, false
	}
	return v * unit, true
}
