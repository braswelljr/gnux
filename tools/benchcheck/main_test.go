package main

import (
	"strings"
	"testing"
)

const sample = `goos: linux
pkg: github.com/braswelljr/gnux/cmd
BenchmarkStream/grep_fox-8      	     100	   3000000 ns/op	2500.00 MB/s	  160000 B/op	     100 allocs/op
BenchmarkStream/grep_fox-8      	     100	   3200000 ns/op	2400.00 MB/s	  160000 B/op	     100 allocs/op
BenchmarkDispatch/truex-8       	  800000	      1400 ns/op	    6600 B/op	      52 allocs/op
PASS
ok  	github.com/braswelljr/gnux/cmd	5.2s
`

func TestParse(t *testing.T) {
	got, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	grep := got["BenchmarkStream/grep_fox"]
	if grep == nil || len(grep.Time) != 2 || grep.Time[1] != 3200000 || len(grep.Allocs) != 2 || grep.Allocs[0] != 100 {
		t.Fatalf("grep = %+v", grep)
	}
	if d := got["BenchmarkDispatch/truex"]; d == nil || d.Time[0] != 1400 || d.Allocs[0] != 52 {
		t.Fatalf("dispatch = %+v", d)
	}
	if len(got) != 2 {
		t.Fatalf("parsed %d benchmarks, want 2", len(got))
	}
}

func TestSlower(t *testing.T) {
	tests := []struct {
		name     string
		old, cur []float64
		want     bool
	}{
		{"same", []float64{100, 110}, []float64{105, 108}, false},
		{"overlapping ranges are noise", []float64{100, 140}, []float64{130, 200}, false},
		{"slower but within tolerance", []float64{100, 110}, []float64{120, 130}, false},
		{"every run slower, beyond tolerance", []float64{100, 110}, []float64{140, 150}, true},
		{"faster", []float64{100, 110}, []float64{50, 60}, false},
		{"one run each", []float64{100}, []float64{126}, true},
		{"nothing to compare", nil, []float64{100}, false},
	}
	for _, tt := range tests {
		if got := Slower(tt.old, tt.cur, 25); got != tt.want {
			t.Errorf("%s: Slower = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMoreAllocs(t *testing.T) {
	tests := []struct {
		name     string
		old, cur []float64
		want     bool
	}{
		{"same", []float64{52, 52}, []float64{52, 52}, false},
		{"a handful more means nothing", []float64{2}, []float64{3}, false},
		{"within tolerance", []float64{1000}, []float64{1090}, false},
		{"beyond tolerance", []float64{1000}, []float64{1200}, true},
		{"fewer", []float64{1000}, []float64{10}, false},
		{"from none to many", []float64{0}, []float64{500}, true},
	}
	for _, tt := range tests {
		if got := MoreAllocs(tt.old, tt.cur, 10); got != tt.want {
			t.Errorf("%s: MoreAllocs = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestReport(t *testing.T) {
	baseline := map[string]*Samples{
		"BenchmarkA":    {Time: []float64{100, 104}, Allocs: []float64{10, 10}},
		"BenchmarkB":    {Time: []float64{100, 104}, Allocs: []float64{100, 100}},
		"BenchmarkGone": {Time: []float64{1}},
	}
	latest := map[string]*Samples{
		"BenchmarkA":   {Time: []float64{101, 99}, Allocs: []float64{10, 10}},
		"BenchmarkB":   {Time: []float64{200, 210}, Allocs: []float64{400, 400}},
		"BenchmarkNew": {Time: []float64{1}},
	}
	var out strings.Builder
	if Report(&out, baseline, latest, 25, 10) {
		t.Error("Report = ok, want a regression")
	}
	for _, want := range []string{
		"BenchmarkA  time    -2.0%  allocs    +0.0%  ok\n",
		"BenchmarkB  time  +101.0%  allocs  +300.0%  SLOWER, MORE ALLOCATIONS\n",
		"BenchmarkNew  new: not in the baseline\n",
		"BenchmarkGone  gone: only in the baseline\n",
		"benchcheck: regression beyond 25% time or 10% allocations\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	delete(latest, "BenchmarkB")
	if !Report(&out, baseline, latest, 25, 10) {
		t.Errorf("Report found a regression where there is none:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "benchcheck: 1 benchmarks within 25% time and 10% allocations\n") {
		t.Errorf("unexpected summary:\n%s", out.String())
	}
}
