package date

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// errInvalid reports a date string Parse does not understand.
var errInvalid = errors.New("invalid date")

// isoLayouts are the absolute forms tried on the whole string first, most
// specific first; Go reads fractional seconds after any seconds field.
var isoLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

// Parse reads a date string the way GNU date -d does, for the forms it
// supports: an empty string (the start of today), "now", "today", "@SECONDS",
// ISO 8601 / RFC 3339 dates and times, MM/DD[/YYYY], "Month D [YYYY]" and
// "D Month [YYYY]", times of day (HH:MM[:SS[.frac]], 10am, 3:30pm, midnight,
// noon), the zones UTC/GMT/Z and numeric offsets after a time, weekdays
// ("monday", "next friday", "last sunday"), "yesterday", "tomorrow", and
// relative items ("1 day ago", "+2 hours", "-3 weeks", "next month",
// "fortnight"). Items combine: "2026-01-31 +1 month", "tomorrow 09:00".
// Anything else is an error.
func Parse(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	loc := now.Location()
	if s == "" {
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc), nil
	}
	if epoch, ok := strings.CutPrefix(s, "@"); ok {
		t, err := parseEpoch(epoch)
		if err != nil {
			return time.Time{}, err
		}
		return t.In(loc), nil
	}
	for _, layout := range isoLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t.In(loc), nil
		}
	}

	var p parser
	if err := p.run(tokenize(strings.ToLower(s))); err != nil {
		return time.Time{}, err
	}
	return p.resolve(now)
}

// parseEpoch parses seconds since the Unix epoch with an optional fraction.
func parseEpoch(s string) (time.Time, error) {
	whole, frac, _ := strings.Cut(s, ".")
	sec, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return time.Time{}, errInvalid
	}
	var nsec int64
	if frac != "" {
		if len(frac) > 9 || strings.Trim(frac, "0123456789") != "" {
			return time.Time{}, errInvalid
		}
		nsec, _ = strconv.ParseInt(frac+strings.Repeat("0", 9-len(frac)), 10, 64)
		if strings.HasPrefix(whole, "-") {
			nsec = -nsec
		}
	}
	return time.Unix(sec, nsec), nil
}

var (
	numUnitRe = regexp.MustCompile(`^([+-]?\d+)([a-z]+)$`)
	isoDateRe = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	isoTRe    = regexp.MustCompile(`^(\d{4}-\d{1,2}-\d{1,2})t(.+)$`)
	slashRe   = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2}|\d{4}))?$`)
	timeRe    = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?::(\d{2})(?:[.,](\d{1,9}))?)?(am|pm|a\.m\.|p\.m\.)?$`)
	hourAmRe  = regexp.MustCompile(`^(\d{1,2})(am|pm|a\.m\.|p\.m\.)$`)
	offsetRe  = regexp.MustCompile(`^([+-])(\d{2}):?(\d{2})$`)
)

// tokenize splits a lower-cased date string into words, separating a number
// glued to a unit ("2days") and a date glued to its time by 't'.
func tokenize(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '\t' || r == ',' }) {
		if m := isoTRe.FindStringSubmatch(f); m != nil {
			out = append(out, m[1], m[2])
			continue
		}
		if m := numUnitRe.FindStringSubmatch(f); m != nil && m[2] != "am" && m[2] != "pm" {
			out = append(out, m[1], m[2])
			continue
		}
		out = append(out, f)
	}
	return out
}

// relItem is one relative item, such as "2 hours".
type relItem struct {
	unit string
	n    int
}

// parser accumulates the items of a date string.
type parser struct {
	hasDate        bool
	year           int
	yearSet        bool
	month, day     int
	hasTime        bool
	hour, min, sec int
	nsec           int
	zone           *time.Location
	rel            []relItem
	hasWday        bool
	wday           time.Weekday
	ordinal        int
}

var units = map[string]string{
	"year": "year", "years": "year",
	"month": "month", "months": "month",
	"fortnight": "fortnight", "fortnights": "fortnight",
	"week": "week", "weeks": "week",
	"day": "day", "days": "day",
	"hour": "hour", "hours": "hour",
	"minute": "minute", "minutes": "minute", "min": "minute", "mins": "minute",
	"second": "second", "seconds": "second", "sec": "second", "secs": "second",
}

func weekday(s string) (time.Weekday, bool) {
	s = strings.TrimSuffix(s, ".")
	for i, name := range dayNames {
		l := strings.ToLower(name)
		if s == l || s == l[:3] || (len(s) >= 3 && strings.HasPrefix(l, s)) {
			return time.Weekday(i), true
		}
	}
	return 0, false
}

func monthName(s string) (int, bool) {
	s = strings.TrimSuffix(s, ".")
	if len(s) < 3 {
		return 0, false
	}
	for i, name := range monthNames {
		l := strings.ToLower(name)
		if s == l[:3] || strings.HasPrefix(l, s) || (s == "sept" && i == 8) {
			return i + 1, true
		}
	}
	return 0, false
}

func (p *parser) run(toks []string) error {
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		next := ""
		if i+1 < len(toks) {
			next = toks[i+1]
		}
		switch {
		case t == "now" || t == "today":
		case t == "yesterday":
			p.rel = append(p.rel, relItem{"day", -1})
		case t == "tomorrow":
			p.rel = append(p.rel, relItem{"day", 1})
		case t == "midnight":
			if err := p.setTime(0, 0, 0, 0); err != nil {
				return err
			}
		case t == "noon":
			if err := p.setTime(12, 0, 0, 0); err != nil {
				return err
			}
		case t == "ago":
			if len(p.rel) == 0 {
				return errInvalid
			}
			p.rel[len(p.rel)-1].n = -p.rel[len(p.rel)-1].n
		case t == "utc" || t == "gmt" || t == "z" || t == "ut":
			if p.zone != nil {
				return errInvalid
			}
			p.zone = time.UTC
		case t == "next" || t == "last" || t == "this":
			ord := map[string]int{"next": 1, "last": -1, "this": 0}[t]
			if u, ok := units[next]; ok {
				p.rel = append(p.rel, relItem{u, ord})
			} else if w, ok := weekday(next); ok {
				if err := p.setWday(w, ord); err != nil {
					return err
				}
			} else {
				return errInvalid
			}
			i++
		case units[t] != "":
			p.rel = append(p.rel, relItem{units[t], 1})
		case isWeekday(t):
			w, _ := weekday(t)
			if err := p.setWday(w, 0); err != nil {
				return err
			}
		case isMonth(t):
			m, _ := monthName(t)
			d, err := strconv.Atoi(next)
			if err != nil {
				return errInvalid
			}
			i++
			if err := p.setDate(0, m, d, false); err != nil {
				return err
			}
			if i+1 < len(toks) && len(toks[i+1]) == 4 && isDigits(toks[i+1]) {
				p.year, p.yearSet = atoi(toks[i+1]), true
				i++
			}
		default:
			consumed, err := p.number(t, next, toks, i)
			if err != nil {
				return err
			}
			i += consumed
		}
	}
	return nil
}

// number handles a token that starts with a digit or a sign: dates, times,
// zone offsets, relative amounts and "D Month [YYYY]". It returns how many
// following tokens it consumed.
func (p *parser) number(t, next string, toks []string, i int) (int, error) {
	if m := isoDateRe.FindStringSubmatch(t); m != nil {
		return 0, p.setDate(atoi(m[1]), atoi(m[2]), atoi(m[3]), true)
	}
	if m := slashRe.FindStringSubmatch(t); m != nil {
		year, yearSet := 0, m[3] != ""
		if yearSet {
			year = atoi(m[3])
			if len(m[3]) == 2 {
				year += 1900
				if year < 1969 {
					year += 100
				}
			}
		}
		return 0, p.setDate(year, atoi(m[1]), atoi(m[2]), yearSet)
	}
	if m := timeRe.FindStringSubmatch(t); m != nil {
		ampm := m[5]
		consumed := 0
		if ampm == "" && (next == "am" || next == "pm") {
			ampm, consumed = next, 1
		}
		h, err := hour12(atoi(m[1]), ampm)
		if err != nil {
			return 0, err
		}
		ns := 0
		if m[4] != "" {
			ns = atoi(m[4] + strings.Repeat("0", 9-len(m[4])))
		}
		return consumed, p.setTime(h, atoi(m[2]), atoi(m[3]), ns)
	}
	if m := hourAmRe.FindStringSubmatch(t); m != nil {
		h, err := hour12(atoi(m[1]), m[2])
		if err != nil {
			return 0, err
		}
		return 0, p.setTime(h, 0, 0, 0)
	}
	if m := offsetRe.FindStringSubmatch(t); m != nil && p.hasTime && p.zone == nil {
		if _, isUnit := units[next]; !isUnit {
			off := atoi(m[2])*3600 + atoi(m[3])*60
			if m[1] == "-" {
				off = -off
			}
			p.zone = time.FixedZone("", off)
			return 0, nil
		}
	}

	n, err := strconv.Atoi(t)
	if err != nil {
		return 0, errInvalid
	}
	if u, ok := units[next]; ok {
		p.rel = append(p.rel, relItem{u, n})
		return 1, nil
	}
	if mo, ok := monthName(next); ok && !strings.ContainsAny(t, "+-") {
		if err := p.setDate(0, mo, n, false); err != nil {
			return 0, err
		}
		if i+2 < len(toks) && len(toks[i+2]) == 4 && isDigits(toks[i+2]) {
			p.year, p.yearSet = atoi(toks[i+2]), true
			return 2, nil
		}
		return 1, nil
	}
	if p.hasDate && !p.yearSet && len(t) == 4 && isDigits(t) {
		p.year, p.yearSet = n, true
		return 0, nil
	}
	return 0, errInvalid
}

func (p *parser) setDate(year, month, day int, yearSet bool) error {
	if p.hasDate {
		return errInvalid
	}
	p.hasDate, p.year, p.yearSet, p.month, p.day = true, year, yearSet, month, day
	return nil
}

func (p *parser) setTime(h, m, s, ns int) error {
	if p.hasTime || h > 23 || m > 59 || s > 60 {
		return errInvalid
	}
	p.hasTime, p.hour, p.min, p.sec, p.nsec = true, h, m, s, ns
	return nil
}

func (p *parser) setWday(w time.Weekday, ord int) error {
	if p.hasWday {
		return errInvalid
	}
	p.hasWday, p.wday, p.ordinal = true, w, ord
	return nil
}

// resolve turns the accumulated items into a time, relative to now: the
// absolute fields first (in the given zone, if any), then the weekday, then
// the relative items, as GNU does.
func (p *parser) resolve(now time.Time) (time.Time, error) {
	out := now.Location()
	loc := out
	if p.zone != nil {
		loc = p.zone
	}
	base := now.In(loc)
	year, month, day := base.Date()
	hour, minute, sec := base.Clock()
	nsec := base.Nanosecond()

	if p.hasDate {
		month, day = time.Month(p.month), p.day
		if p.yearSet {
			year = p.year
		}
		if !p.hasTime {
			hour, minute, sec, nsec = 0, 0, 0, 0
		}
	}
	if p.hasTime {
		hour, minute, sec, nsec = p.hour, p.min, p.sec, p.nsec
	} else if p.hasWday {
		hour, minute, sec, nsec = 0, 0, 0, 0
	}

	t := time.Date(year, month, day, hour, minute, sec, nsec, loc)
	if p.hasDate && (t.Month() != month || t.Day() != day) {
		return time.Time{}, errInvalid
	}

	if p.hasWday {
		cur := t.Weekday()
		diff := (int(p.wday) - int(cur) + 7) % 7
		bump := p.ordinal
		if p.ordinal > 0 && cur != p.wday {
			bump--
		}
		t = t.AddDate(0, 0, diff+7*bump)
	}

	var years, months, days int
	var dur time.Duration
	for _, r := range p.rel {
		switch r.unit {
		case "year":
			years += r.n
		case "month":
			months += r.n
		case "fortnight":
			days += 14 * r.n
		case "week":
			days += 7 * r.n
		case "day":
			days += r.n
		case "hour":
			dur += time.Duration(r.n) * time.Hour
		case "minute":
			dur += time.Duration(r.n) * time.Minute
		case "second":
			dur += time.Duration(r.n) * time.Second
		}
	}
	t = t.AddDate(years, months, days).Add(dur)
	return t.In(out), nil
}

func hour12(h int, ampm string) (int, error) {
	switch ampm {
	case "":
		return h, nil
	case "am", "a.m.":
		if h < 1 || h > 12 {
			return 0, errInvalid
		}
		return h % 12, nil
	default:
		if h < 1 || h > 12 {
			return 0, errInvalid
		}
		return h%12 + 12, nil
	}
}

func isWeekday(s string) bool {
	_, ok := weekday(s)
	return ok && !isDigits(s)
}

func isMonth(s string) bool {
	_, ok := monthName(s)
	return ok
}

func isDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
