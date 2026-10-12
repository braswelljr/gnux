package date

import (
	"strconv"
	"strings"
	"time"
)

// Format renders t through a GNU date FORMAT: strftime conversions in the C
// locale, GNU's flags (- _ 0 ^ #), a field width, the E and O modifiers
// (accepted and ignored) and GNU's extensions (%N, %q, %s, %:z, %::z, %:::z,
// %P, %k, %l). An unknown conversion is copied through literally, as GNU does.
func Format(format string, t time.Time) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		start := i
		var sp spec
		i++
	flags:
		for ; i < len(format); i++ {
			switch format[i] {
			case '-', '_', '0':
				sp.pad = format[i]
			case '^':
				sp.upper = true
			case '#':
				sp.swap = true
			default:
				break flags
			}
		}
		for ; i < len(format) && format[i] >= '0' && format[i] <= '9'; i++ {
			sp.width = sp.width*10 + int(format[i]-'0')
			sp.widthSet = true
		}
		for i < len(format) && (format[i] == 'E' || format[i] == 'O') {
			i++
		}
		colons := 0
		for i < len(format) && format[i] == ':' {
			colons++
			i++
		}
		if i >= len(format) {
			b.WriteString(format[start:])
			break
		}
		if colons > 0 && (format[i] != 'z' || colons > 3) {
			b.WriteString(format[start : i+1])
			continue
		}
		if !sp.conv(&b, format[i], colons, t) {
			b.WriteString(format[start : i+1])
		}
	}
	return b.String()
}

// spec is one conversion's flags and width.
type spec struct {
	pad      byte // 0 (the conversion's default), '-', '_' or '0'
	upper    bool
	swap     bool
	width    int
	widthSet bool
}

var (
	dayNames   = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	monthNames = []string{"January", "February", "March", "April", "May", "June", "July",
		"August", "September", "October", "November", "December"}
)

// conv writes one conversion and reports whether c was a known conversion.
func (sp spec) conv(b *strings.Builder, c byte, colons int, t time.Time) bool {
	year, month, day := t.Date()
	hour, minute, sec := t.Clock()
	yday := t.YearDay() - 1
	wday := int(t.Weekday())
	hour12 := hour % 12
	if hour12 == 0 {
		hour12 = 12
	}

	switch c {
	case '%':
		sp.text(b, "%", false)
	case 'n':
		sp.text(b, "\n", false)
	case 't':
		sp.text(b, "\t", false)
	case 'a':
		sp.text(b, dayNames[wday][:3], sp.swap)
	case 'A':
		sp.text(b, dayNames[wday], sp.swap)
	case 'b', 'h':
		sp.text(b, monthNames[month-1][:3], sp.swap)
	case 'B':
		sp.text(b, monthNames[month-1], sp.swap)
	case 'p':
		ampm := "AM"
		if hour >= 12 {
			ampm = "PM"
		}
		if sp.swap {
			ampm = strings.ToLower(ampm)
		}
		sp.text(b, ampm, false)
	case 'P':
		ampm := "am"
		if hour >= 12 {
			ampm = "pm"
		}
		sp.text(b, ampm, false)
	case 'Z':
		name, _ := t.Zone()
		if sp.swap {
			name = strings.ToLower(name)
		}
		sp.text(b, name, false)
	case 'c':
		sp.text(b, Format("%a %b %e %H:%M:%S %Y", t), false)
	case 'D', 'x':
		sp.text(b, Format("%m/%d/%y", t), false)
	case 'F':
		sp.text(b, Format("%Y-%m-%d", t), false)
	case 'r':
		sp.text(b, Format("%I:%M:%S %p", t), false)
	case 'R':
		sp.text(b, Format("%H:%M", t), false)
	case 'T', 'X':
		sp.text(b, Format("%H:%M:%S", t), false)
	case 'C':
		sp.number(b, int64(floorDiv(year, 100)), 2, '0')
	case 'y':
		sp.number(b, int64(((year%100)+100)%100), 2, '0')
	case 'Y':
		sp.number(b, int64(year), 4, '0')
	case 'G':
		isoYear, _ := t.ISOWeek()
		sp.number(b, int64(isoYear), 4, '0')
	case 'g':
		isoYear, _ := t.ISOWeek()
		sp.number(b, int64(((isoYear%100)+100)%100), 2, '0')
	case 'V':
		_, week := t.ISOWeek()
		sp.number(b, int64(week), 2, '0')
	case 'U':
		sp.number(b, int64((yday+7-wday)/7), 2, '0')
	case 'W':
		sp.number(b, int64((yday+7-(wday+6)%7)/7), 2, '0')
	case 'm':
		sp.number(b, int64(month), 2, '0')
	case 'd':
		sp.number(b, int64(day), 2, '0')
	case 'e':
		sp.number(b, int64(day), 2, ' ')
	case 'j':
		sp.number(b, int64(yday+1), 3, '0')
	case 'H':
		sp.number(b, int64(hour), 2, '0')
	case 'k':
		sp.number(b, int64(hour), 2, ' ')
	case 'I':
		sp.number(b, int64(hour12), 2, '0')
	case 'l':
		sp.number(b, int64(hour12), 2, ' ')
	case 'M':
		sp.number(b, int64(minute), 2, '0')
	case 'S':
		sp.number(b, int64(sec), 2, '0')
	case 's':
		sp.number(b, t.Unix(), 1, '0')
	case 'u':
		u := wday
		if u == 0 {
			u = 7
		}
		sp.number(b, int64(u), 1, '0')
	case 'w':
		sp.number(b, int64(wday), 1, '0')
	case 'q':
		sp.number(b, int64((int(month)-1)/3+1), 1, '0')
	case 'N':
		digits := 9
		if sp.widthSet && sp.width > 0 {
			digits = sp.width
		}
		ns := strconv.Itoa(t.Nanosecond())
		ns = strings.Repeat("0", 9-len(ns)) + ns
		if digits <= 9 {
			ns = ns[:digits]
		} else {
			ns += strings.Repeat("0", digits-9)
		}
		b.WriteString(ns)
	case 'z':
		sp.text(b, zoneOffset(t, colons), false)
	default:
		return false
	}
	return true
}

// number writes v padded to the conversion's default width with its default
// pad character, both of which the flags and an explicit width override.
func (sp spec) number(b *strings.Builder, v int64, width int, pad byte) {
	if sp.widthSet {
		width = sp.width
	}
	switch sp.pad {
	case '-':
		width = 0
	case '_':
		pad = ' '
	case '0':
		pad = '0'
	}
	neg := v < 0
	digits := strconv.FormatInt(v, 10)
	if neg {
		digits = digits[1:]
	}
	n := len(digits)
	if neg {
		n++
	}
	fill := ""
	if width > n {
		fill = strings.Repeat(string(pad), width-n)
	}
	if pad == '0' {
		if neg {
			b.WriteByte('-')
		}
		b.WriteString(fill)
	} else {
		b.WriteString(fill)
		if neg {
			b.WriteByte('-')
		}
	}
	b.WriteString(digits)
}

// text writes s, upper-cased by ^ (or by # when upperOnSwap), and padded on
// the left to an explicit width with spaces, or zeros under the 0 flag.
func (sp spec) text(b *strings.Builder, s string, upperOnSwap bool) {
	if sp.upper || upperOnSwap {
		s = strings.ToUpper(s)
	}
	if sp.widthSet && sp.pad != '-' && sp.width > len(s) {
		pad := " "
		if sp.pad == '0' {
			pad = "0"
		}
		b.WriteString(strings.Repeat(pad, sp.width-len(s)))
	}
	b.WriteString(s)
}

// zoneOffset renders t's UTC offset as %z (+hhmm), %:z (+hh:mm), %::z
// (+hh:mm:ss) or %:::z (only as precise as needed).
func zoneOffset(t time.Time, colons int) string {
	_, off := t.Zone()
	sign := byte('+')
	if off < 0 {
		sign = '-'
		off = -off
	}
	h, m, s := off/3600, off%3600/60, off%60
	two := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	out := string(sign) + two(h)
	switch colons {
	case 0:
		return out + two(m)
	case 1:
		return out + ":" + two(m)
	case 2:
		return out + ":" + two(m) + ":" + two(s)
	default:
		if s != 0 {
			return out + ":" + two(m) + ":" + two(s)
		}
		if m != 0 {
			return out + ":" + two(m)
		}
		return out
	}
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
