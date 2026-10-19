//go:build wasip1

package utmp

import "time"

// DefaultFile is empty: a WASI sandbox has no users.
const DefaultFile = ""

// Boot is not known in a WASI sandbox.
func Boot() (time.Time, error) { return time.Time{}, ErrUnsupported }

// LoadAvg is not known in a WASI sandbox.
func LoadAvg() ([]float64, error) { return nil, ErrUnsupported }

// ReadFile is not available in a WASI sandbox.
func ReadFile(string) ([]Entry, error) { return nil, ErrUnsupported }

// Current reports no sessions in a WASI sandbox.
func Current() ([]Entry, error) { return nil, ErrUnsupported }
