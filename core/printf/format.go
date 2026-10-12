package printf

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The conversions are produced with Go's fmt where its output matches C's,
// and by hand where it does not: Go adds a 0x prefix to a zero with %#x, a
// plus sign to unsigned values, spells infinities "+Inf", picks the shortest
// representation for %g without a precision, and writes hexadecimal floats
// with a two-digit exponent.

// goVerb builds the Go format for sp, with the flags and conversion given.
func goVerb(sp spec, flags string, verb byte) string {
	var b strings.Builder
	b.WriteByte('%')
	b.WriteString(flags)
	if sp.hasWid {
		b.WriteString(strconv.Itoa(sp.width))
	}
	if sp.hasPrec {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(sp.prec))
	}
	b.WriteByte(verb)
	return b.String()
}

func formatSigned(sp spec, v int64) string {
	return fmt.Sprintf(goVerb(sp, sp.flags, 'd'), v)
}

func formatUnsigned(sp spec, v uint64) string {
	// '+' and ' ' apply to signed conversions only.
	flags := strings.NewReplacer("+", "", " ", "").Replace(sp.flags)
	verb := sp.verb
	if verb == 'u' {
		verb = 'd'
	}
	if v == 0 && (verb == 'x' || verb == 'X') {
		flags = strings.ReplaceAll(flags, "#", "")
	}
	return fmt.Sprintf(goVerb(sp, flags, verb), v)
}

func formatFloat(sp spec, v float64) string {
	upper := sp.verb >= 'A' && sp.verb <= 'Z'
	if math.IsInf(v, 0) || math.IsNaN(v) {
		s := "inf"
		if math.IsNaN(v) {
			s = "nan"
		}
		if upper {
			s = strings.ToUpper(s)
		}
		switch {
		case math.Signbit(v):
			s = "-" + s
		case strings.Contains(sp.flags, "+"):
			s = "+" + s
		case strings.Contains(sp.flags, " "):
			s = " " + s
		}
		return pad(sp, s)
	}

	switch sp.verb {
	case 'a', 'A':
		return hexFloat(sp, v, upper)
	case 'g', 'G':
		// C's default precision is 6, and a precision of 0 means 1.
		if !sp.hasPrec {
			sp.prec, sp.hasPrec = 6, true
		} else if sp.prec == 0 {
			sp.prec = 1
		}
	}
	return fmt.Sprintf(goVerb(sp, sp.flags, sp.verb), v)
}

// hexFloat renders %a: 0x1.8p+1 for 3, with the exponent in as few digits as
// it needs and zero padding placed after the 0x.
func hexFloat(sp spec, v float64, upper bool) string {
	prec := -1
	if sp.hasPrec {
		prec = sp.prec
	}
	s := strconv.FormatFloat(math.Abs(v), 'x', prec, 64)
	mant, exp, _ := strings.Cut(s, "p")
	sign := exp[:1]
	digits := strings.TrimLeft(exp[1:], "0")
	if digits == "" {
		digits = "0"
	}
	if strings.Contains(sp.flags, "#") && !strings.Contains(mant, ".") {
		mant += "."
	}
	body := mant[2:] + "p" + sign + digits // without the 0x

	lead := ""
	switch {
	case math.Signbit(v):
		lead = "-"
	case strings.Contains(sp.flags, "+"):
		lead = "+"
	case strings.Contains(sp.flags, " "):
		lead = " "
	}
	lead += "0x"
	if n := sp.width - len(lead) - len(body); sp.hasWid && n > 0 &&
		strings.Contains(sp.flags, "0") && !strings.Contains(sp.flags, "-") {
		body = strings.Repeat("0", n) + body
	}
	out := lead + body
	if upper {
		out = strings.ToUpper(out)
	}
	return pad(sp, out)
}

// FormatFloat formats v under a single C floating-point directive such as
// "%.3f", "%-10e" or "%a": flags from "-+ #0'", an optional width and
// precision (no '*'), and one of the conversions e, E, f, F, g, G, a, A. It
// reports false when directive is not of that form. It is what seqx -f uses.
func FormatFloat(directive string, v float64) (string, bool) {
	if !strings.HasPrefix(directive, "%") || len(directive) < 2 {
		return "", false
	}
	var sp spec
	i := 1
	for i < len(directive) && strings.IndexByte("-+ #0'", directive[i]) >= 0 {
		if directive[i] != '\'' {
			sp.flags += string(directive[i])
		}
		i++
	}
	j := i
	for i < len(directive) && isDigit(directive[i]) {
		i++
	}
	if i > j {
		sp.width, _ = strconv.Atoi(directive[j:i])
		sp.hasWid = true
	}
	if i < len(directive) && directive[i] == '.' {
		i++
		j = i
		for i < len(directive) && isDigit(directive[i]) {
			i++
		}
		sp.prec, _ = strconv.Atoi(directive[j:i])
		sp.hasPrec = true
	}
	if i != len(directive)-1 || !strings.ContainsRune("eEfFgGaA", rune(directive[i])) {
		return "", false
	}
	sp.verb = directive[i]
	return formatFloat(sp, v), true
}

// ShellQuote quotes s so that a POSIX shell reads it back as the same single
// word, as %q does: unchanged when it is made only of characters that need
// no quoting, otherwise in single quotes (double quotes when s holds a single
// quote and nothing the shell would expand), with control characters written
// as $'\n'-style escapes.
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if safe(s) {
		return s
	}
	hasCtl := false
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			hasCtl = true
			break
		}
	}
	if !hasCtl {
		if strings.Contains(s, "'") && !strings.ContainsAny(s, "\"$`\\!") {
			return `"` + s + `"`
		}
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}

	// Quote printable runs in single quotes and control bytes as $'...'.
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7f {
			if inQuote {
				b.WriteByte('\'')
				inQuote = false
			}
			b.WriteString("$'" + ctlEscape(c) + "'")
			continue
		}
		if c == '\'' {
			if inQuote {
				b.WriteByte('\'')
				inQuote = false
			}
			b.WriteString(`\'`)
			continue
		}
		if !inQuote {
			b.WriteByte('\'')
			inQuote = true
		}
		b.WriteByte(c)
	}
	if inQuote {
		b.WriteByte('\'')
	}
	return b.String()
}

// safe reports whether s needs no quoting for a shell.
func safe(s string) bool {
	if s[0] == '~' || s[0] == '#' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("%+,-./:=@_^", c) >= 0:
		case c >= 0x80:
		default:
			return false
		}
	}
	return true
}

// ctlEscape spells a control byte the way $'...' does.
func ctlEscape(c byte) string {
	switch c {
	case '\a':
		return `\a`
	case '\b':
		return `\b`
	case '\t':
		return `\t`
	case '\n':
		return `\n`
	case '\v':
		return `\v`
	case '\f':
		return `\f`
	case '\r':
		return `\r`
	case 0x1b:
		return `\E`
	}
	return fmt.Sprintf(`\%03o`, c)
}
