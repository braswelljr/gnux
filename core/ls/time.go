package ls

import (
	"strconv"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
)

// recentWindow is half an average Gregorian year. A timestamp within this
// window before now (and not in the future) is "recent" and shows the time of
// day; anything else shows the year instead.
const recentWindow = 31556952 / 2 * time.Second

// timeStyle formats timestamps for the long format.
type timeStyle struct {
	recent, old string
	// strftime marks recent/old as strftime formats (--time-style=+FORMAT)
	// rather than Go layouts.
	strftime bool
}

// timeStyles lists the named --time-style values with their Go layouts.
var timeStyles = []word[timeStyle]{
	{"full-iso", timeStyle{recent: "2006-01-02 15:04:05.000000000 -0700", old: "2006-01-02 15:04:05.000000000 -0700"}},
	{"long-iso", timeStyle{recent: "2006-01-02 15:04", old: "2006-01-02 15:04"}},
	{"iso", timeStyle{recent: "01-02 15:04", old: "2006-01-02 "}},
	{"locale", timeStyle{recent: "Jan _2 15:04", old: "Jan _2  2006"}},
}

// newTimeStyle resolves a --time-style value. "" selects "locale"; a leading
// "+" introduces a strftime format, optionally two separated by a newline (the
// first for older files, the second for recent ones). The "posix-" prefix is
// accepted and ignored: lsx always formats as the POSIX locale would.
func newTimeStyle(style string) (timeStyle, error) {
	if style == "" {
		style = "locale"
	}
	if format, ok := strings.CutPrefix(style, "+"); ok {
		old, recent, two := strings.Cut(format, "\n")
		if !two {
			recent = old
		}
		return timeStyle{recent: recent, old: old, strftime: true}, nil
	}
	return matchWord("time style", strings.TrimPrefix(style, "posix-"), timeStyles)
}

// ValidateTimeStyle reports whether style is an acceptable --time-style value,
// returning a usage error when it is not.
func ValidateTimeStyle(style string) error {
	_, err := newTimeStyle(style)
	return err
}

// format renders t, choosing the recent or old form relative to now.
func (s timeStyle) format(t, now time.Time, loc *time.Location) string {
	layout := s.old
	if t.After(now.Add(-recentWindow)) && !t.After(now) {
		layout = s.recent
	}
	if loc != nil {
		t = t.In(loc)
	}
	if s.strftime {
		return strftime(layout, t)
	}
	return t.Format(layout)
}

// strftimeLayouts maps strftime conversions onto Go layouts.
var strftimeLayouts = map[byte]string{
	'a': "Mon", 'A': "Monday", 'b': "Jan", 'B': "January", 'h': "Jan",
	'c': "Mon Jan _2 15:04:05 2006", 'd': "02", 'D': "01/02/06", 'e': "_2",
	'F': "2006-01-02", 'H': "15", 'I': "03", 'j': "002", 'm': "01", 'M': "04",
	'p': "PM", 'R': "15:04", 'S': "05", 'T': "15:04:05", 'x': "01/02/06",
	'X': "15:04:05", 'y': "06", 'Y': "2006", 'z': "-0700", 'Z': "MST",
}

// strftime formats t with the common strftime conversions. Unknown conversions
// are copied through unchanged.
func strftime(format string, t time.Time) string {
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != '%' || i+1 == len(format) {
			b.WriteByte(c)
			continue
		}
		i++
		switch c = format[i]; c {
		case '%':
			b.WriteByte('%')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 's':
			b.WriteString(strconv.FormatInt(t.Unix(), 10))
		case 'N':
			b.WriteString(t.Format("000000000"))
		case 'C':
			b.WriteString(pad2(t.Year()/100, '0'))
		case 'k':
			b.WriteString(pad2(t.Hour(), ' '))
		case 'l':
			b.WriteString(pad2((t.Hour()+11)%12+1, ' '))
		case 'u':
			b.WriteString(strconv.Itoa((int(t.Weekday())+6)%7 + 1))
		case 'w':
			b.WriteString(strconv.Itoa(int(t.Weekday())))
		default:
			if layout, ok := strftimeLayouts[c]; ok {
				b.WriteString(t.Format(layout))
			} else {
				b.WriteByte('%')
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

func pad2(n int, fill byte) string {
	s := strconv.Itoa(n)
	if len(s) < 2 {
		return string(fill) + s
	}
	return s
}

// word pairs an option argument with the value it selects.
type word[T comparable] struct {
	name  string
	value T
}

// matchWord resolves an option argument against its table of valid words. As
// in GNU tools, any prefix is accepted as long as every word it could stand for
// selects the same value. context names the option in the error ("--sort").
func matchWord[T comparable](context, arg string, table []word[T]) (T, error) {
	var (
		found     T
		matched   bool
		ambiguous bool
	)
	for _, w := range table {
		if w.name == arg {
			return w.value, nil
		}
		if arg == "" || !strings.HasPrefix(w.name, arg) {
			continue
		}
		if matched && w.value != found {
			ambiguous = true
		}
		found, matched = w.value, true
	}
	if matched && !ambiguous {
		return found, nil
	}

	problem := "invalid"
	if ambiguous {
		problem = "ambiguous"
	}
	var b strings.Builder
	b.WriteString(problem)
	b.WriteString(" argument '")
	b.WriteString(arg)
	b.WriteString("' for '")
	b.WriteString(context)
	b.WriteString("'\nValid arguments are:")
	for _, w := range table {
		b.WriteString("\n  - '")
		b.WriteString(w.name)
		b.WriteString("'")
	}
	var zero T
	return zero, core.Usagef("%s", b.String())
}

// ParseSort resolves a --sort argument.
func ParseSort(arg string) (SortKey, error) {
	return matchWord("--sort", arg, []word[SortKey]{
		{"none", SortNone},
		{"name", SortName},
		{"time", SortTime},
		{"size", SortSize},
		{"extension", SortExtension},
		{"version", SortVersion},
		{"width", SortWidth},
	})
}

// ParseTime resolves a --time argument.
func ParseTime(arg string) (TimeKind, error) {
	return matchWord("--time", arg, []word[TimeKind]{
		{"atime", TimeAccessed},
		{"access", TimeAccessed},
		{"use", TimeAccessed},
		{"ctime", TimeChanged},
		{"status", TimeChanged},
		{"mtime", TimeModified},
		{"modification", TimeModified},
		{"birth", TimeBirth},
		{"creation", TimeBirth},
	})
}

// ParseWhen resolves a WHEN argument such as --color takes. context names the
// option in the error ("--color").
func ParseWhen(context, arg string) (When, error) {
	return matchWord(context, arg, []word[When]{
		{"always", WhenAlways},
		{"yes", WhenAlways},
		{"force", WhenAlways},
		{"never", WhenNever},
		{"no", WhenNever},
		{"none", WhenNever},
		{"auto", WhenAuto},
		{"tty", WhenAuto},
		{"if-tty", WhenAuto},
	})
}
