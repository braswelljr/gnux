//go:build js

package utmp

import "time"

// DefaultFile is empty: the browser has no users.
const DefaultFile = ""

// Boot is not known in the browser.
func Boot() (time.Time, error) { return time.Time{}, ErrUnsupported }

// LoadAvg is not known in the browser.
func LoadAvg() ([]float64, error) { return nil, ErrUnsupported }

// ReadFile is not available in the browser.
func ReadFile(string) ([]Entry, error) { return nil, ErrUnsupported }

// Current reports no sessions in the browser.
func Current() ([]Entry, error) { return nil, ErrUnsupported }
