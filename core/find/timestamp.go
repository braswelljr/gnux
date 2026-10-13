package find

import (
	"strconv"
	"strings"
	"time"
)

// timestampLayouts are the spellings -newerXt accepts, a practical subset of
// what GNU's date parser reads. A layout without a zone is in the engine's
// Location.
var timestampLayouts = []string{
	"2006-01-02",
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05 Z07:00",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"20060102",
	"Jan 2 2006",
	"2 Jan 2006",
	"Jan 2 2006 15:04",
	"Jan 2 2006 15:04:05",
	"2 Jan 2006 15:04",
	"2 Jan 2006 15:04:05",
	time.RFC1123,
	time.RFC1123Z,
	time.ANSIC,
	time.UnixDate,
}

// parseTimestamp reads the reference time of -newerXt: "@SECONDS" since the
// epoch (with an optional fraction) or one of timestampLayouts.
func parseTimestamp(s string, loc *time.Location) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "@"); ok {
		secs, frac, _ := strings.Cut(rest, ".")
		n, err := strconv.ParseInt(secs, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		var nanos int64
		if frac != "" {
			if len(frac) > 9 {
				frac = frac[:9]
			}
			v, err := strconv.ParseUint(frac, 10, 64)
			if err != nil {
				return time.Time{}, false
			}
			for range 9 - len(frac) {
				v *= 10
			}
			nanos = int64(v)
		}
		return time.Unix(n, nanos), true
	}
	for _, layout := range timestampLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
