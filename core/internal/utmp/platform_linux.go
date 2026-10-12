//go:build linux

package utmp

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultFile is where the system records who is logged in.
const DefaultFile = "/var/run/utmp"

var decode = DecodeLinux

// Boot returns the time the system started, from /proc/uptime.
func Boot() (time.Time, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return time.Time{}, ErrUnsupported
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(-time.Duration(secs * float64(time.Second))).Truncate(time.Second), nil
}

// LoadAvg returns the 1, 5 and 15 minute load averages from /proc/loadavg.
func LoadAvg() ([]float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))
	var out []float64
	for _, f := range fields[:min(3, len(fields))] {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
