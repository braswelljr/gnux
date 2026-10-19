// Package date implements the engine behind datex, a GNU-compatible
// implementation of the GNU/UNIX date command: print a date and time in a
// given format, now or as described by -d, -f or -r.
package date

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "datex"

// DefaultFormat is GNU date's output format in the C locale.
const DefaultFormat = "%a %b %e %H:%M:%S %Z %Y"

// ISO8601Formats maps each -I precision to its format.
var ISO8601Formats = map[string]string{
	"date":    "%Y-%m-%d",
	"hours":   "%Y-%m-%dT%H%:z",
	"minutes": "%Y-%m-%dT%H:%M%:z",
	"seconds": "%Y-%m-%dT%H:%M:%S%:z",
	"ns":      "%Y-%m-%dT%H:%M:%S,%N%:z",
}

// RFC3339Formats maps each --rfc-3339 precision to its format.
var RFC3339Formats = map[string]string{
	"date":    "%Y-%m-%d",
	"seconds": "%Y-%m-%d %H:%M:%S%:z",
	"ns":      "%Y-%m-%d %H:%M:%S.%N%:z",
}

// RFC5322Format is the -R (--rfc-email) format.
const RFC5322Format = "%a, %d %b %Y %H:%M:%S %z"

// Options mirrors the GNU date flag surface. The zero value prints the
// current time in the default format.
type Options struct {
	// Date (-d) is a date string to display instead of now; HasDate says it
	// was given (an empty string means the start of today).
	Date    string
	HasDate bool
	// File (-f) names a file of date strings, one per line ("-" is standard
	// input); each is printed in turn.
	File string
	// Reference (-r) displays the last modification time of this file.
	Reference string
	// Set (-s) asks to set the system clock, which gnux does not do.
	Set    string
	HasSet bool
	// UTC (-u) prints Coordinated Universal Time.
	UTC bool
	// ISO8601 (-I) is the precision of ISO 8601 output: date, hours, minutes,
	// seconds or ns. Empty means not requested.
	ISO8601 string
	// RFC3339 (--rfc-3339) is the precision of RFC 3339 output: date, seconds
	// or ns. Empty means not requested.
	RFC3339 string
	// RFC5322 (-R) prints RFC 5322 (email) format.
	RFC5322 bool
	// Resolution (--resolution) prints the resolution of the system clock
	// instead of a date: the time it shows is that many seconds after the
	// epoch, by default in the format "%s.%N".
	Resolution bool
}

// Date is the engine. Construct it with New.
type Date struct {
	Options

	// Now returns the current time; it is the injection point for tests.
	Now func() time.Time
	// Location, when set, is the zone dates are shown in; otherwise TZ (read
	// through Getenv) or the local zone is used, and UTC under -u.
	Location *time.Location
	// Getenv reads TZ; it defaults to os.Getenv.
	Getenv func(string) string
	// ClockResolution returns the resolution of the system clock in
	// nanoseconds, for --resolution and "%-N"; it defaults to measuring it.
	ClockResolution func() int64

	streams core.Streams
	diag    *core.Diag
}

// New builds a date engine reading from and writing to s.
func New(opts Options, s core.Streams) *Date {
	return &Date{
		Options:         opts,
		Now:             time.Now,
		Getenv:          os.Getenv,
		ClockResolution: clockResolution,
		streams:         s,
		diag:            core.NewDiag(Name, s),
	}
}

// Run prints the date. args holds at most one operand: +FORMAT, or a
// [MMDDhhmm[[CC]YY][.ss]] date to set the clock to, which is refused.
func (d *Date) Run(args []string) (err error) {
	format, err := d.format(args)
	if err != nil {
		return err
	}
	if d.HasSet || (len(args) == 1 && !strings.HasPrefix(args[0], "+")) {
		d.diag.Errorf("cannot set date: Operation not supported")
		return d.diag.Result()
	}

	format = d.adjustResolution(format)
	loc := d.location()
	now := d.Now().In(loc)
	out := core.NewWriter(d.streams.Out)
	defer core.FlushOutput(out, d.diag, &err)

	switch {
	case d.File != "":
		d.batch(out, format, now)
	case d.Reference != "":
		info, err := fsys.Stat(d.Reference)
		if err != nil {
			d.diag.Errorf("%s: %s", d.Reference, core.ErrText(err))
			break
		}
		fmt.Fprintln(out, Format(format, info.ModTime().In(loc)))
	case d.Resolution:
		fmt.Fprintln(out, Format(format, time.Unix(0, d.ClockResolution()).In(loc)))
	case d.HasDate:
		t, err := Parse(d.Date, now)
		if err != nil {
			d.diag.Errorf("invalid date '%s'", d.Date)
			break
		}
		fmt.Fprintln(out, Format(format, t))
	default:
		fmt.Fprintln(out, Format(format, now))
	}
	return d.diag.Result()
}

// format checks the operands and options and returns the output format.
func (d *Date) format(args []string) (string, error) {
	if len(args) > 1 {
		return "", core.Usagef("extra operand '%s'", args[1])
	}
	sources := 0
	for _, given := range []bool{d.HasDate, d.File != "", d.Reference != "", d.Resolution} {
		if given {
			sources++
		}
	}
	if sources > 1 {
		return "", core.Usagef("the options to specify dates for printing are mutually exclusive")
	}
	if d.HasSet && sources > 0 {
		return "", core.Usagef("the options to print and set the time may not be used together")
	}

	formats := 0
	format := DefaultFormat
	if d.Resolution {
		format = "%s.%N"
	}
	if d.ISO8601 != "" {
		f, ok := ISO8601Formats[d.ISO8601]
		if !ok {
			return "", core.Usagef("invalid argument '%s' for '--iso-8601'", d.ISO8601)
		}
		format = f
		formats++
	}
	if d.RFC3339 != "" {
		f, ok := RFC3339Formats[d.RFC3339]
		if !ok {
			return "", core.Usagef("invalid argument '%s' for '--rfc-3339'", d.RFC3339)
		}
		format = f
		formats++
	}
	if d.RFC5322 {
		format = RFC5322Format
		formats++
	}
	if len(args) == 1 {
		if op := args[0]; strings.HasPrefix(op, "+") {
			format = op[1:]
			formats++
		} else if sources > 0 || d.HasSet {
			return "", core.Usagef("the argument '%s' lacks a leading '+';\n"+
				"when using an option to specify date(s), any non-option\n"+
				"argument must be a format string beginning with '+'", op)
		}
	}
	if formats > 1 {
		return "", core.Usagef("multiple output formats specified")
	}
	return format, nil
}

// adjustResolution rewrites each "%-N" in format as the number of digits
// the clock's resolution needs ("%9N" for nanoseconds, "%6N" for
// microseconds), as GNU date does.
func (d *Date) adjustResolution(format string) string {
	var b strings.Builder
	digits := byte(0)
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			b.WriteByte(format[i])
			continue
		}
		if strings.HasPrefix(format[i+1:], "-N") {
			if digits == 0 {
				digits = '0' + byte(resolutionWidth(d.ClockResolution()))
			}
			b.WriteByte('%')
			b.WriteByte(digits)
			b.WriteByte('N')
			i += 2
			continue
		}
		b.WriteByte('%')
		if format[i+1] == '%' {
			b.WriteByte('%')
			i++
		}
	}
	return b.String()
}

// resolutionWidth is the number of decimal digits a time with resolution
// res nanoseconds needs to lose nothing.
func resolutionWidth(res int64) int {
	w := 9
	for r := int64(10); r <= res && w > 0; r *= 10 {
		w--
	}
	return w
}

// clockResolution measures the resolution of the system clock in
// nanoseconds, as gnulib's gettime_res does: the greatest common divisor of
// the sub-second parts of up to 32 distinct timestamps.
func clockResolution() int64 {
	r := int64(time.Second)
	earlier := time.Time{}
	for i := 0; r > 1 && i < 32; i++ {
		now := time.Now()
		// Successive equal timestamps say little; wait for a different one,
		// then stop sampling after it.
		for now.Equal(earlier) {
			i = 32
			now = time.Now()
		}
		earlier = now
		if ns := int64(now.Nanosecond()); ns > 0 {
			r = gcd(r, ns)
		}
	}
	return r
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// location is the zone dates are displayed in.
func (d *Date) location() *time.Location {
	switch {
	case d.UTC:
		return time.UTC
	case d.Location != nil:
		return d.Location
	}
	if tz, ok := d.lookupTZ(); ok {
		if tz == "" || tz == "UTC" || tz == "UTC0" || tz == ":UTC" {
			return time.UTC
		}
		if loc, err := time.LoadLocation(strings.TrimPrefix(tz, ":")); err == nil {
			return loc
		}
		// An unknown zone is UTC, as in the C library.
		return time.UTC
	}
	return time.Local
}

func (d *Date) lookupTZ() (string, bool) {
	if d.Getenv == nil {
		return "", false
	}
	tz := d.Getenv("TZ")
	return tz, tz != ""
}

// batch prints one line for each date string in d.File.
func (d *Date) batch(out *bufio.Writer, format string, now time.Time) {
	var in io.Reader
	if d.File == "-" {
		in = d.streams.In
	} else {
		f, err := fsys.Open(d.File)
		if err != nil {
			d.diag.Errorf("%s: %s", d.File, core.ErrText(err))
			return
		}
		defer f.Close()
		in = f
	}
	r := core.NewReader(in)
	for {
		line, err := r.ReadString('\n')
		if line != "" || err == nil {
			line = strings.TrimSuffix(line, "\n")
			if t, perr := Parse(line, now); perr != nil {
				_ = out.Flush()
				d.diag.Errorf("invalid date '%s'", line)
			} else {
				fmt.Fprintln(out, Format(format, t))
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				d.diag.Errorf("%s: %s", d.File, core.ErrText(err))
			}
			return
		}
	}
}
