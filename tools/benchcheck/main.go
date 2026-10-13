// Command benchcheck compares two files of `go test -bench` output and fails
// when a benchmark got slower or started allocating more: the pass/fail
// check that `make bench/check` and the Bench workflow run.
//
// Usage:
//
//	go run ./tools/benchcheck [-time PERCENT] [-allocs PERCENT] BASELINE LATEST
//
// Timings depend on the machine and on what else it is doing, so a benchmark
// only counts as slower when every one of its new runs is slower than every
// one of its old runs, and by more than the -time tolerance: the two ranges
// must not overlap. Run each benchmark several times (make bench COUNT=5) to
// give that rule something to work with. Allocations per operation hardly
// vary from run to run, so they are compared by their medians against the
// tighter -allocs tolerance.
//
// It prints one line per benchmark that both files hold, and exits 1 when
// any regressed. A benchmark in only one file is listed and never fails the
// check: the suite is allowed to grow.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
)

func main() {
	timeTol := flag.Float64("time", 25, "percent a benchmark may slow down, beyond run-to-run noise")
	allocTol := flag.Float64("allocs", 10, "percent a benchmark's allocations per operation may grow")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: benchcheck [-time PERCENT] [-allocs PERCENT] BASELINE LATEST")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}

	var runs [2]map[string]*Samples
	for i, name := range flag.Args() {
		f, err := os.Open(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcheck: %v\n", err)
			os.Exit(2)
		}
		runs[i], err = Parse(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcheck: %s: %v\n", name, err)
			os.Exit(2)
		}
	}
	if !Report(os.Stdout, runs[0], runs[1], *timeTol, *allocTol) {
		os.Exit(1)
	}
}

// Samples are the measurements of one benchmark across its runs.
type Samples struct {
	Time   []float64 // ns/op
	Allocs []float64 // allocs/op
}

// Parse reads `go test -bench` output: every line that starts with
// "Benchmark" and holds value-unit pairs. The -N suffix that names
// GOMAXPROCS is dropped, so results from machines with different core counts
// still line up.
func Parse(r io.Reader) (map[string]*Samples, error) {
	out := map[string]*Samples{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		name := fields[0]
		if i := strings.LastIndexByte(name, '-'); i > 0 {
			if _, err := strconv.Atoi(name[i+1:]); err == nil {
				name = name[:i]
			}
		}
		s := out[name]
		if s == nil {
			s = &Samples{}
			out[name] = s
		}
		// fields[1] is the iteration count; value-unit pairs follow.
		for i := 2; i+1 < len(fields); i += 2 {
			v, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				continue
			}
			switch fields[i+1] {
			case "ns/op":
				s.Time = append(s.Time, v)
			case "allocs/op":
				s.Allocs = append(s.Allocs, v)
			}
		}
	}
	return out, sc.Err()
}

// Report writes the comparison of latest against baseline and reports
// whether every benchmark is within its tolerances.
func Report(w io.Writer, baseline, latest map[string]*Samples, timeTol, allocTol float64) (ok bool) {
	var names, added, removed []string
	for name := range latest {
		if baseline[name] != nil {
			names = append(names, name)
		} else {
			added = append(added, name)
		}
	}
	for name := range baseline {
		if latest[name] == nil {
			removed = append(removed, name)
		}
	}
	slices.Sort(names)
	slices.Sort(added)
	slices.Sort(removed)

	ok = true
	width := 0
	for _, name := range names {
		width = max(width, len(name))
	}
	for _, name := range names {
		old, cur := baseline[name], latest[name]
		verdict := "ok"
		if Slower(old.Time, cur.Time, timeTol) {
			verdict, ok = "SLOWER", false
		}
		if MoreAllocs(old.Allocs, cur.Allocs, allocTol) {
			if verdict == "ok" {
				verdict = "MORE ALLOCATIONS"
			} else {
				verdict += ", MORE ALLOCATIONS"
			}
			ok = false
		}
		fmt.Fprintf(w, "%-*s  time %s  allocs %s  %s\n", width, name,
			change(old.Time, cur.Time), change(old.Allocs, cur.Allocs), verdict)
	}
	for _, name := range added {
		fmt.Fprintf(w, "%s  new: not in the baseline\n", name)
	}
	for _, name := range removed {
		fmt.Fprintf(w, "%s  gone: only in the baseline\n", name)
	}
	switch {
	case len(names) == 0:
		fmt.Fprintln(w, "benchcheck: the two files share no benchmark; nothing was compared")
	case ok:
		fmt.Fprintf(w, "benchcheck: %d benchmarks within %g%% time and %g%% allocations\n", len(names), timeTol, allocTol)
	default:
		fmt.Fprintf(w, "benchcheck: regression beyond %g%% time or %g%% allocations\n", timeTol, allocTol)
	}
	return ok
}

// Slower reports whether the new timings are slower than the old ones beyond
// noise: the fastest new run is slower than the slowest old run by more than
// tol percent.
func Slower(old, cur []float64, tol float64) bool {
	if len(old) == 0 || len(cur) == 0 {
		return false
	}
	return slices.Min(cur) > slices.Max(old)*(1+tol/100)
}

// MoreAllocs reports whether the median allocation count grew by more than
// tol percent, and by more than a handful: going from 2 allocations to 3 is
// a 50% rise that means nothing.
func MoreAllocs(old, cur []float64, tol float64) bool {
	if len(old) == 0 || len(cur) == 0 {
		return false
	}
	before, after := Median(old), Median(cur)
	return after > before*(1+tol/100) && after-before > 8
}

// Median returns the middle value of v, which must not be empty.
func Median(v []float64) float64 {
	s := slices.Sorted(slices.Values(v))
	if n := len(s); n%2 == 0 {
		return (s[n/2-1] + s[n/2]) / 2
	}
	return s[len(s)/2]
}

// change renders the move between two medians as a signed percentage.
func change(old, cur []float64) string {
	if len(old) == 0 || len(cur) == 0 {
		return "     n/a"
	}
	before, after := Median(old), Median(cur)
	if before == 0 {
		if after == 0 {
			return "   +0.0%"
		}
		return "    +inf"
	}
	return fmt.Sprintf("%+7.1f%%", (after-before)/before*100)
}
