package touch

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var errInvalidDate = errors.New("invalid date format")

// dateLayouts are the absolute forms -d accepts, most specific first. Go's
// parser takes fractional seconds after any seconds field, so they need no
// layouts of their own.
var dateLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

// parseDate parses a -d string: "now", "@EPOCH", or an ISO 8601 / RFC 3339
// date with an optional time and zone. A date without a zone is local to now's
// location.
func parseDate(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if isNow(s) {
		return now, nil
	}
	if epoch, ok := strings.CutPrefix(s, "@"); ok {
		return parseEpoch(epoch)
	}
	for _, layout := range dateLayouts {
		if when, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return when, nil
		}
	}
	return time.Time{}, errInvalidDate
}

// parseEpoch parses seconds since the Unix epoch, with an optional fraction.
func parseEpoch(s string) (time.Time, error) {
	whole, frac, _ := strings.Cut(s, ".")
	sec, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return time.Time{}, errInvalidDate
	}

	var nsec int64
	if frac != "" {
		if len(frac) > 9 || strings.Trim(frac, "0123456789") != "" {
			return time.Time{}, errInvalidDate
		}
		nsec, _ = strconv.ParseInt(frac+strings.Repeat("0", 9-len(frac)), 10, 64)
		if strings.HasPrefix(whole, "-") {
			nsec = -nsec
		}
	}
	return time.Unix(sec, nsec), nil
}

// parseStamp parses a -t stamp, [[CC]YY]MMDDhhmm[.ss], in now's location. A
// missing year is the current one; a two-digit year follows POSIX, with 69-99
// in the 1900s and 00-68 in the 2000s.
func parseStamp(s string, now time.Time) (time.Time, error) {
	main, secs, hasSecs := strings.Cut(s, ".")
	if strings.Trim(main, "0123456789") != "" || strings.Trim(secs, "0123456789") != "" {
		return time.Time{}, errInvalidDate
	}
	if hasSecs && len(secs) != 2 {
		return time.Time{}, errInvalidDate
	}

	year := now.Year()
	switch len(main) {
	case 8:
	case 10:
		year = 1900 + num(main[:2])
		if year < 1969 {
			year += 100
		}
		main = main[2:]
	case 12:
		year = num(main[:4])
		main = main[4:]
	default:
		return time.Time{}, errInvalidDate
	}

	month, day := time.Month(num(main[:2])), num(main[2:4])
	hour, minute, sec := num(main[4:6]), num(main[6:8]), num(secs)

	// time.Date normalizes out-of-range fields (month 13, February 30), so a
	// stamp is valid only if it survives the round trip unchanged.
	utc := time.Date(year, month, day, hour, minute, sec, 0, time.UTC)
	if utc.Month() != month || utc.Day() != day ||
		utc.Hour() != hour || utc.Minute() != minute || utc.Second() != sec {
		return time.Time{}, errInvalidDate
	}
	return time.Date(year, month, day, hour, minute, sec, 0, now.Location()), nil
}

// num converts a string already known to hold only digits; "" is 0.
func num(digits string) int {
	n, _ := strconv.Atoi(digits)
	return n
}
