//go:build darwin || freebsd

package utmp

import (
	"encoding/binary"
	"time"

	"golang.org/x/sys/unix"
)

// Boot returns the time the system started, from kern.boottime.
func Boot() (time.Time, error) {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000), nil //nolint:unconvert
}

// LoadAvg returns the 1, 5 and 15 minute load averages from vm.loadavg: a
// struct loadavg of three fixed-point numbers and their scale.
func LoadAvg() ([]float64, error) {
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		return nil, err
	}
	if len(raw) < 24 {
		return nil, ErrUnsupported
	}
	le := binary.LittleEndian
	scale := float64(le.Uint64(raw[16:]))
	if scale == 0 {
		return nil, ErrUnsupported
	}
	return []float64{
		float64(le.Uint32(raw[0:])) / scale,
		float64(le.Uint32(raw[4:])) / scale,
		float64(le.Uint32(raw[8:])) / scale,
	}, nil
}
