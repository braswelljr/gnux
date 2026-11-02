package uptime

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

func TestPanel(t *testing.T) {
	var out, errb bytes.Buffer
	u := New(Options{Pretty: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	u.Now = func() time.Time { return now }
	u.Location = time.UTC
	u.CPUs = func() int { return 4 }
	u.Getenv = func(k string) string { return map[string]string{"LANG": "C"}[k] }
	u.current = func() ([]utmp.Entry, error) {
		return []utmp.Entry{boot(3*24*time.Hour + 4*time.Hour + 12*time.Minute), user("ada"), user("lin")}, nil
	}
	u.loadAvg = func() ([]float64, error) { return []float64{1, 3.2, 4.8}, nil }
	if err := u.Run(nil); err != nil {
		t.Fatalf("Run: %v (%s)", err, errb.String())
	}
	// A quarter, 80% and more than all of the 4 CPUs.
	want := "+- up 3 days, 4:12 ----------------------+\n" +
		"| Time    14:03:22                       |\n" +
		"| Users   2                              |\n" +
		"| Load     1 min  ####............  1.00 |\n" +
		"|          5 min  #############...  3.20 |\n" +
		"|         15 min  ################  4.80 |\n" +
		"| CPUs    4                              |\n" +
		"+----------------------------------------+\n"
	if out.String() != want {
		t.Fatalf("got\n%s\nwant\n%s", out.String(), want)
	}
}

func TestPanelWithoutBootTime(t *testing.T) {
	var out, errb bytes.Buffer
	u := New(Options{Pretty: true}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	u.Now = func() time.Time { return now }
	u.Getenv = func(string) string { return "" }
	u.current = func() ([]utmp.Entry, error) { return nil, nil }
	u.loadAvg = func() ([]float64, error) { return nil, utmp.ErrUnsupported }
	if err := u.Run(nil); err == nil || errb.String() != "uptimex: couldn't get boot time\n" {
		t.Fatalf("err %v, stderr %q", err, errb.String())
	}
	if !strings.Contains(out.String(), "up ????") || strings.Contains(out.String(), "Load") {
		t.Fatalf("panel: %q", out.String())
	}
}
