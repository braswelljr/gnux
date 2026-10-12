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

	streams core.Streams
	diag    *core.Diag
}

// New builds a date engine reading from and writing to s.
func New(opts Options, s core.Streams) *Date {
	return &Date{
		Options: opts,
		Now:     time.Now,
		Getenv:  os.Getenv,
		streams: s,
		diag:    core.NewDiag(Name, s),
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

	loc := d.location()
	now := d.Now().In(loc)
	out := bufio.NewWriter(d.streams.Out)
	defer core.FlushOutput(out, d.diag, &err)

	switch {
	case d.File != "":
		d.batch(out, format, now)
	case d.Reference != "":
		info, err := os.Stat(d.Reference)
		if err != nil {
			d.diag.Errorf("%s: %s", d.Reference, core.ErrText(err))
			break
		}
		fmt.Fprintln(out, Format(format, info.ModTime().In(loc)))
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
	for _, given := range []bool{d.HasDate, d.File != "", d.HasSet, d.Reference != ""} {
		if given {
			sources++
		}
	}
	if sources > 1 {
		if d.HasSet && (d.File != "" || d.Reference != "") {
			return "", core.Usagef("the options to print and set the time may not be used together")
		}
		return "", core.Usagef("the options to specify dates for printing are mutually exclusive")
	}

	formats := 0
	format := DefaultFormat
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
		} else if d.HasDate || d.File != "" || d.Reference != "" || d.HasSet {
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
		f, err := os.Open(d.File)
		if err != nil {
			d.diag.Errorf("%s: %s", d.File, core.ErrText(err))
			return
		}
		defer f.Close()
		in = f
	}
	r := bufio.NewReader(in)
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
