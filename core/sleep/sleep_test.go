package sleep_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sleep"
)

// newEngine returns an engine whose pauses are recorded instead of taken.
func newEngine() (s *sleep.Sleep, slept *[]time.Duration, out, errb *bytes.Buffer) {
	out, errb = &bytes.Buffer{}, &bytes.Buffer{}
	s = sleep.New(core.Streams{In: strings.NewReader(""), Out: out, Err: errb})
	slept = &[]time.Duration{}
	s.Sleep = func(d time.Duration) { *slept = append(*slept, d) }
	return s, slept, out, errb
}

func TestRunSleepsForTheTotal(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want time.Duration
	}{
		{"seconds by default", []string{"5"}, 5 * time.Second},
		{"zero", []string{"0"}, 0},
		{"explicit seconds", []string{"5s"}, 5 * time.Second},
		{"minutes", []string{"2m"}, 2 * time.Minute},
		{"hours", []string{"3h"}, 3 * time.Hour},
		{"days", []string{"1d"}, 24 * time.Hour},
		{"fraction", []string{"0.5"}, 500 * time.Millisecond},
		{"fraction with suffix", []string{"1.5m"}, 90 * time.Second},
		{"leading dot", []string{".25"}, 250 * time.Millisecond},
		{"trailing dot", []string{"2."}, 2 * time.Second},
		{"exponent", []string{"1e3"}, 1000 * time.Second},
		{"negative exponent", []string{"15e-1s"}, 1500 * time.Millisecond},
		{"hexadecimal float", []string{"0x1p3"}, 8 * time.Second},
		{"explicit plus sign", []string{"+2"}, 2 * time.Second},
		{"operands are summed", []string{"1m", "30", "0.5s"}, 90*time.Second + 500*time.Millisecond},
		{"mixed units", []string{"1d", "1h", "1m", "1s"}, 25*time.Hour + time.Minute + time.Second},
		{"nanosecond precision", []string{"0.000000001"}, time.Nanosecond},
		{"decimal fractions do not drift", []string{"1.1"}, 1100 * time.Millisecond},
		{"infinity", []string{"infinity"}, sleep.Forever},
		{"inf", []string{"inf"}, sleep.Forever},
		{"infinity is case-insensitive", []string{"INF"}, sleep.Forever},
		{"infinity with a suffix", []string{"infd"}, sleep.Forever},
		{"infinity plus a finite interval", []string{"1", "inf"}, sleep.Forever},
		{"too large for a duration", []string{"1e30"}, sleep.Forever},
		{"overflowing float", []string{"1e999"}, sleep.Forever},
		{"sum overflows a duration", []string{"100000000000d", "100000000000d"}, sleep.Forever},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, slept, out, errb := newEngine()
			if err := s.Run(tt.args); err != nil {
				t.Fatalf("Run(%q): %v", tt.args, err)
			}
			if want := []time.Duration{tt.want}; !reflect.DeepEqual(*slept, want) {
				t.Errorf("slept %v, want %v", *slept, want)
			}
			if out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("stdout = %q, stderr = %q, want both empty", out.String(), errb.String())
			}
		})
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    string
		wantStderr string
	}{
		{"missing operand", nil, "missing operand", ""},
		{"not a number", []string{"abc"}, "invalid time interval 'abc'", ""},
		{"empty operand", []string{""}, "invalid time interval ''", ""},
		{"suffix only", []string{"s"}, "invalid time interval 's'", ""},
		{"unknown suffix", []string{"5x"}, "invalid time interval '5x'", ""},
		{"two suffixes", []string{"5ms"}, "invalid time interval '5ms'", ""},
		{"negative", []string{"-1"}, "invalid time interval '-1'", ""},
		{"negative infinity", []string{"-inf"}, "invalid time interval '-inf'", ""},
		{"not a number value", []string{"nan"}, "invalid time interval 'nan'", ""},
		{"underscore separators", []string{"0x1_0p0"}, "invalid time interval '0x1_0p0'", ""},
		{"one bad operand among good ones", []string{"1", "x", "2"}, "invalid time interval 'x'", ""},
		{
			"every bad operand is reported",
			[]string{"a", "1", "b", "c"},
			"invalid time interval 'c'",
			"sleepx: invalid time interval 'a'\nsleepx: invalid time interval 'b'\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, slept, out, errb := newEngine()
			err := s.Run(tt.args)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("Run(%q) = %v, want a usage error", tt.args, err)
			}
			if usage.Msg != tt.wantErr {
				t.Errorf("usage error = %q, want %q", usage.Msg, tt.wantErr)
			}
			if errb.String() != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", errb.String(), tt.wantStderr)
			}
			if out.Len() != 0 {
				t.Errorf("stdout = %q, want empty", out.String())
			}
			if len(*slept) != 0 {
				t.Errorf("slept %v despite the error", *slept)
			}
		})
	}
}

func TestDefaultSleepActuallyPauses(t *testing.T) {
	s := sleep.New(core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}})
	const d = 20 * time.Millisecond
	start := time.Now()
	if err := s.Run([]string{"0.02"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if elapsed := time.Since(start); elapsed < d {
		t.Fatalf("returned after %v, want at least %v", elapsed, d)
	}
}
