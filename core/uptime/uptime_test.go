package uptime

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

var now = time.Date(2026, 10, 12, 14, 3, 22, 0, time.UTC)

func run(t *testing.T, entries []utmp.Entry, avg []float64, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	u := New(Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	u.Now = func() time.Time { return now }
	u.Location = time.UTC
	u.current = func() ([]utmp.Entry, error) { return entries, nil }
	u.readFile = func(string) ([]utmp.Entry, error) { return nil, errors.New("no such file or directory") }
	u.loadAvg = func() ([]float64, error) {
		if avg == nil {
			return nil, utmp.ErrUnsupported
		}
		return avg, nil
	}
	err := u.Run(args)
	return out.String(), errb.String(), err
}

func boot(ago time.Duration) utmp.Entry { return utmp.Entry{Type: utmp.BootTime, Time: now.Add(-ago)} }

func user(name string) utmp.Entry { return utmp.Entry{Type: utmp.UserProcess, User: name} }

func TestUptime(t *testing.T) {
	tests := []struct {
		name    string
		entries []utmp.Entry
		avg     []float64
		want    string
	}{
		{"minutes", []utmp.Entry{boot(42 * time.Minute), user("a")}, []float64{0.5, 0.25, 0.125},
			" 14:03:22  up  0:42,  1 user,  load average: 0.50, 0.25, 0.12\n"},
		{"one day", []utmp.Entry{boot(25*time.Hour + 3*time.Minute), user("a"), user("b")}, []float64{1, 2, 3},
			" 14:03:22  up 1 day  1:03,  2 users,  load average: 1.00, 2.00, 3.00\n"},
		{"days", []utmp.Entry{boot(10*24*time.Hour + 13*time.Hour)}, []float64{0, 0, 0},
			" 14:03:22  up 10 days 13:00,  0 users,  load average: 0.00, 0.00, 0.00\n"},
		{"no load average", []utmp.Entry{boot(time.Hour)}, nil,
			" 14:03:22  up  1:00,  0 users\n"},
		{"dead and empty records are not users", []utmp.Entry{boot(time.Hour), {Type: utmp.DeadProcess, User: "x"}, {Type: utmp.UserProcess}}, nil,
			" 14:03:22  up  1:00,  0 users\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(t, tt.entries, tt.avg)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v (%q), want %q", out, err, errb, tt.want)
			}
		})
	}
}

func TestNoBootTime(t *testing.T) {
	out, errb, err := run(t, []utmp.Entry{user("a")}, nil)
	if !errors.Is(err, core.ErrFailed) || errb != "uptimex: couldn't get boot time\n" ||
		out != " 14:03:22  up ???? days ??:??,  1 user\n" {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

func TestFileAndOperands(t *testing.T) {
	_, errb, err := run(t, nil, nil, "/var/log/wtmp")
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errb, "uptimex: /var/log/wtmp: No such file or directory\n") {
		t.Fatalf("missing file: %v %q", err, errb)
	}
	_, _, err = run(t, nil, nil, "a", "b")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'b'" {
		t.Fatalf("extra operand: %v", err)
	}
}

func TestLive(t *testing.T) {
	var out, errb bytes.Buffer
	err := New(Options{}, core.Streams{Out: &out, Err: &errb}).Run(nil)
	if strings.Contains(errb.String(), "not supported") {
		t.Skip("no records here")
	}
	if err != nil || !strings.Contains(out.String(), "  up ") {
		t.Fatalf("%v %q %q", err, out.String(), errb.String())
	}
}
