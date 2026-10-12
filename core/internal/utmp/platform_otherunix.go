//go:build unix && !linux && !darwin && !freebsd

package utmp

import "time"

// DefaultFile is the traditional utmp location.
const DefaultFile = "/var/run/utmp"

var decode = DecodeLinux

// Boot is not known here.
func Boot() (time.Time, error) { return time.Time{}, ErrUnsupported }

// LoadAvg is not known here.
func LoadAvg() ([]float64, error) { return nil, ErrUnsupported }
