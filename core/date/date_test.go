package date_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/date"
)

// fixed is Monday 12 October 2026, 09:05:03.123456789 UTC.
var fixed = time.Date(2026, time.October, 12, 9, 5, 3, 123456789, time.UTC)

func run(t *testing.T, opts date.Options, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	d := date.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	d.Now = func() time.Time { return fixed }
	d.Location = time.UTC
	err := d.Run(args)
	return out.String(), errb.String(), err
}

func TestDefault(t *testing.T) {
	out, _, err := run(t, date.Options{}, "")
	if err != nil || out != "Mon Oct 12 09:05:03 UTC 2026\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
}

func TestFormatConversions(t *testing.T) {
	tests := []struct{ format, want string }{
		{"%Y-%m-%d %H:%M:%S", "2026-10-12 09:05:03"},
		{"%a %A %b %B %h", "Mon Monday Oct October Oct"},
		{"%c", "Mon Oct 12 09:05:03 2026"},
		{"%C %y %g %G %V", "20 26 26 2026 42"},
		{"%d %e %j", "12 12 285"},
		{"%D %F %x %X %T %R %r", "10/12/26 2026-10-12 10/12/26 09:05:03 09:05:03 09:05 09:05:03 AM"},
		{"%H %I %k %l %M %S", "09 09  9  9 05 03"},
		{"%p %P %#p", "AM am am"},
		{"%u %w %U %W %q", "1 1 41 41 4"},
		{"%s", "1791795903"},
		{"%N %3N %6N %12N", "123456789 123 123456 123456789000"},
		{"%z %:z %::z %:::z %Z %#Z", "+0000 +00:00 +00:00:00 +00 UTC utc"},
		{"%n%t%%", "\n\t%"},
		{"%-d %-m %-H %_m %_H %05d %10Y %-j", "12 10 9 10  9 00012 0000002026 285"},
		{"%^a %^B %#a %#b %10A %-10A %^10b", "MON OCTOBER MON OCT     Monday Monday        OCT"},
		{"%Ey %OH %EY", "26 09 2026"},
		{"%!%Q%", "%!%Q%"},
		{"%::::z %:H", "%::::z %:H"},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		if got := date.Format(tt.format, fixed); got != tt.want {
			t.Errorf("Format(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestFormatZonesAndEdges(t *testing.T) {
	kolkata := time.FixedZone("IST", 5*3600+30*60)
	ny := time.FixedZone("EDT", -4*3600)
	odd := time.FixedZone("LMT", -(3600 + 2*60 + 3))
	tests := []struct {
		t      time.Time
		format string
		want   string
	}{
		{fixed.In(kolkata), "%z %:z %:::z %Z", "+0530 +05:30 +05:30 IST"},
		{fixed.In(ny), "%z %:::z %H", "-0400 -04 05"},
		{fixed.In(odd), "%::z %:::z", "-01:02:03 -01:02:03"},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "%U %W %V %G %j %u", "00 00 01 2026 001 4"},
		{time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), "%V %G %g", "53 2026 26"},
		{time.Date(2026, 1, 4, 13, 0, 0, 0, time.UTC), "%U %W %I %l %p", "01 00 01  1 PM"},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), "%I %p %q", "12 AM 4"},
		{time.Unix(-1, 0).UTC(), "%s %Y", "-1 1969"},
		{time.Date(99, 1, 2, 0, 0, 0, 0, time.UTC), "%Y %C %y", "0099 00 99"},
	}
	for _, tt := range tests {
		if got := date.Format(tt.format, tt.t); got != tt.want {
			t.Errorf("Format(%q, %v) = %q, want %q", tt.format, tt.t, got, tt.want)
		}
	}
}

func TestParse(t *testing.T) {
	ts := func(y int, mo time.Month, d, h, m, s int) time.Time {
		return time.Date(y, mo, d, h, m, s, 0, time.UTC)
	}
	tests := []struct {
		in   string
		want time.Time
	}{
		{"", ts(2026, 10, 12, 0, 0, 0)},
		{"now", fixed},
		{"today", fixed},
		{"@0", ts(1970, 1, 1, 0, 0, 0)},
		{"@1791795903.5", time.Date(2026, 10, 12, 9, 5, 3, 500000000, time.UTC)},
		{"2026-03-04", ts(2026, 3, 4, 0, 0, 0)},
		{"2026-03-04 10:11", ts(2026, 3, 4, 10, 11, 0)},
		{"2026-03-04T10:11:12Z", ts(2026, 3, 4, 10, 11, 12)},
		{"2026-03-04T10:11:12+02:00", ts(2026, 3, 4, 8, 11, 12)},
		{"2026-03-04 10:11 +0200", ts(2026, 3, 4, 8, 11, 0)},
		{"2026-03-04 10:11 utc", ts(2026, 3, 4, 10, 11, 0)},
		{"yesterday", fixed.AddDate(0, 0, -1)},
		{"tomorrow", fixed.AddDate(0, 0, 1)},
		{"tomorrow 08:30", ts(2026, 10, 13, 8, 30, 0)},
		{"1 day ago", fixed.AddDate(0, 0, -1)},
		{"2 days ago", fixed.AddDate(0, 0, -2)},
		{"+2 hours", fixed.Add(2 * time.Hour)},
		{"-90 minutes", fixed.Add(-90 * time.Minute)},
		{"3 weeks", fixed.AddDate(0, 0, 21)},
		{"fortnight ago", fixed.AddDate(0, 0, -14)},
		{"next month", fixed.AddDate(0, 1, 0)},
		{"last year", fixed.AddDate(-1, 0, 0)},
		{"2days", fixed.AddDate(0, 0, 2)},
		{"hour ago", fixed.Add(-time.Hour)},
		{"2026-01-31 +1 month", ts(2026, 3, 3, 0, 0, 0)},
		{"monday", ts(2026, 10, 12, 0, 0, 0)},
		{"next monday", ts(2026, 10, 19, 0, 0, 0)},
		{"last monday", ts(2026, 10, 5, 0, 0, 0)},
		{"friday", ts(2026, 10, 16, 0, 0, 0)},
		{"next fri", ts(2026, 10, 16, 0, 0, 0)},
		{"last sunday", ts(2026, 10, 11, 0, 0, 0)},
		{"sunday 10am", ts(2026, 10, 18, 10, 0, 0)},
		{"midnight", ts(2026, 10, 12, 0, 0, 0)},
		{"noon", ts(2026, 10, 12, 12, 0, 0)},
		{"3pm", ts(2026, 10, 12, 15, 0, 0)},
		{"12am", ts(2026, 10, 12, 0, 0, 0)},
		{"10:30 pm", ts(2026, 10, 12, 22, 30, 0)},
		{"7:45:30.25", time.Date(2026, 10, 12, 7, 45, 30, 250000000, time.UTC)},
		{"10/31/2026", ts(2026, 10, 31, 0, 0, 0)},
		{"12/25", ts(2026, 12, 25, 0, 0, 0)},
		{"Dec 25", ts(2026, 12, 25, 0, 0, 0)},
		{"25 December 2027", ts(2027, 12, 25, 0, 0, 0)},
		{"Jan 2, 2028 14:00", ts(2028, 1, 2, 14, 0, 0)},
		{"2026-02-28 12:00 1 day", ts(2026, 3, 1, 12, 0, 0)},
	}
	for _, tt := range tests {
		got, err := date.Parse(tt.in, fixed)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.in, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	for _, in := range []string{"xyz", "2026-02-30", "25:00", "13pm", "next", "ago", "1 day 2", "@x", "monday tuesday", "10:00 11:00", "utc utc", "Feb 30"} {
		if got, err := date.Parse(in, fixed); err == nil {
			t.Errorf("Parse(%q) = %v, want an error", in, got)
		}
	}
}

func TestParseKeepsLocalZone(t *testing.T) {
	ist := time.FixedZone("IST", 5*3600+30*60)
	got, err := date.Parse("2026-03-04 10:00 utc", fixed.In(ist))
	if err != nil {
		t.Fatal(err)
	}
	if got.Location() != ist || got.Hour() != 15 || got.Minute() != 30 {
		t.Fatalf("got %v, want 15:30 IST", got)
	}
}

func TestOutputFormats(t *testing.T) {
	tests := []struct {
		name string
		opts date.Options
		args []string
		want string
	}{
		{"format operand", date.Options{}, []string{"+%F"}, "2026-10-12\n"},
		{"empty format", date.Options{}, []string{"+"}, "\n"},
		{"iso date", date.Options{ISO8601: "date"}, nil, "2026-10-12\n"},
		{"iso hours", date.Options{ISO8601: "hours"}, nil, "2026-10-12T09+00:00\n"},
		{"iso minutes", date.Options{ISO8601: "minutes"}, nil, "2026-10-12T09:05+00:00\n"},
		{"iso seconds", date.Options{ISO8601: "seconds"}, nil, "2026-10-12T09:05:03+00:00\n"},
		{"iso ns", date.Options{ISO8601: "ns"}, nil, "2026-10-12T09:05:03,123456789+00:00\n"},
		{"rfc 3339 date", date.Options{RFC3339: "date"}, nil, "2026-10-12\n"},
		{"rfc 3339 seconds", date.Options{RFC3339: "seconds"}, nil, "2026-10-12 09:05:03+00:00\n"},
		{"rfc 3339 ns", date.Options{RFC3339: "ns"}, nil, "2026-10-12 09:05:03.123456789+00:00\n"},
		{"rfc 5322", date.Options{RFC5322: true}, nil, "Mon, 12 Oct 2026 09:05:03 +0000\n"},
		{"date string", date.Options{Date: "2000-01-01 12:00", HasDate: true}, []string{"+%s"}, "946728000\n"},
		{"empty date string", date.Options{HasDate: true}, []string{"+%T"}, "00:00:00\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(t, tt.opts, "", tt.args...)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v (stderr %q), want %q", out, err, errb, tt.want)
			}
		})
	}
}

func TestZones(t *testing.T) {
	ist := time.FixedZone("IST", 5*3600+30*60)
	var out bytes.Buffer
	d := date.New(date.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
	d.Now = func() time.Time { return fixed }
	d.Location = ist
	if err := d.Run([]string{"+%H:%M %Z"}); err != nil || out.String() != "14:35 IST\n" {
		t.Fatalf("Location: out = %q, err = %v", out.String(), err)
	}

	out.Reset()
	d = date.New(date.Options{UTC: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
	d.Now = func() time.Time { return fixed.In(ist) }
	d.Location = ist
	if err := d.Run([]string{"+%H:%M %Z"}); err != nil || out.String() != "09:05 UTC\n" {
		t.Fatalf("-u: out = %q, err = %v", out.String(), err)
	}

	for tz, want := range map[string]string{"UTC": "09:05 UTC", "Not/AZone": "09:05 UTC", "UTC0": "09:05 UTC"} {
		out.Reset()
		d = date.New(date.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
		d.Now = func() time.Time { return fixed }
		d.Getenv = func(k string) string {
			if k == "TZ" {
				return tz
			}
			return ""
		}
		if err := d.Run([]string{"+%H:%M %Z"}); err != nil || out.String() != want+"\n" {
			t.Fatalf("TZ=%s: out = %q, err = %v", tz, out.String(), err)
		}
	}
}

func TestInvalidDate(t *testing.T) {
	out, errb, err := run(t, date.Options{Date: "not a date", HasDate: true}, "")
	if !errors.Is(err, core.ErrFailed) || out != "" || errb != "datex: invalid date 'not a date'\n" {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dates")
	if err := os.WriteFile(path, []byte("2026-01-01\nbogus\n@86400\nlast"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errb, err := run(t, date.Options{File: path}, "", "+%F")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "2026-01-01\n1970-01-02\n"; out != want {
		t.Fatalf("out = %q, want %q", out, want)
	}
	if want := "datex: invalid date 'bogus'\ndatex: invalid date 'last'\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}

	out, _, err = run(t, date.Options{File: "-"}, "2026-05-06\ntomorrow\n", "+%F")
	if err != nil || out != "2026-05-06\n2026-10-13\n" {
		t.Fatalf("stdin: out = %q, err = %v", out, err)
	}

	_, errb, err = run(t, date.Options{File: filepath.Join(dir, "missing")}, "")
	if !errors.Is(err, core.ErrFailed) || !strings.HasSuffix(errb, "missing: No such file or directory\n") {
		t.Fatalf("missing: stderr = %q, err = %v", errb, err)
	}
}

func TestReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(t, date.Options{Reference: path}, "", "+%F %T")
	if err != nil || out != "2001-02-03 04:05:06\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}

	_, errb, err := run(t, date.Options{Reference: "no-such-file"}, "")
	if !errors.Is(err, core.ErrFailed) || errb != "datex: no-such-file: No such file or directory\n" {
		t.Fatalf("missing: stderr = %q, err = %v", errb, err)
	}
}

func TestSetIsRefused(t *testing.T) {
	for _, tt := range []struct {
		opts date.Options
		args []string
	}{
		{date.Options{Set: "2026-01-01", HasSet: true}, nil},
		{date.Options{}, []string{"10121200"}},
	} {
		out, errb, err := run(t, tt.opts, "", tt.args...)
		if !errors.Is(err, core.ErrFailed) || out != "" || errb != "datex: cannot set date: Operation not supported\n" {
			t.Fatalf("%+v %v: out = %q, stderr = %q, err = %v", tt.opts, tt.args, out, errb, err)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		opts date.Options
		args []string
		want string
	}{
		{date.Options{}, []string{"+%F", "+%T"}, "extra operand '+%T'"},
		{date.Options{HasDate: true, File: "f"}, nil, "the options to specify dates for printing are mutually exclusive"},
		{date.Options{HasDate: true, Reference: "f"}, nil, "the options to specify dates for printing are mutually exclusive"},
		{date.Options{HasSet: true, Reference: "f"}, nil, "the options to print and set the time may not be used together"},
		{date.Options{ISO8601: "date", RFC5322: true}, nil, "multiple output formats specified"},
		{date.Options{RFC3339: "ns"}, []string{"+%F"}, "multiple output formats specified"},
		{date.Options{ISO8601: "weeks"}, nil, "invalid argument 'weeks' for '--iso-8601'"},
		{date.Options{RFC3339: "hours"}, nil, "invalid argument 'hours' for '--rfc-3339'"},
		{date.Options{HasDate: true}, []string{"%F"}, "the argument '%F' lacks a leading '+';\nwhen using an option to specify date(s), any non-option\nargument must be a format string beginning with '+'"},
	}
	for _, tt := range tests {
		_, _, err := run(t, tt.opts, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Errorf("%+v %v: err = %v, want usage error %q", tt.opts, tt.args, err, tt.want)
		}
	}
}
