// Package numfmtx wires the numfmtx command (GNU numfmt) to its engine.
package numfmtx

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/numfmt"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the numfmtx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "numfmt",
		Flags: Register,
		Run:   run,
	}
}

// Register installs the numfmtx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("debug", false, "print warnings about invalid input")
	fs.StringP("delimiter", "d", "", "use `X` instead of whitespace for field delimiter")
	fs.Var(&once{}, "field", "replace the numbers in these input `FIELDS` (default=1); see FIELDS below")
	fs.String("format", "", "use printf style floating-point `FORMAT`; see FORMAT below for details")
	fs.String("from", "", "auto-scale input numbers to `UNIT`s; default is 'none'; see UNIT below")
	fs.String("from-unit", "", "specify the input unit size (instead of the default 1)")
	fs.Bool("grouping", false, "use locale-defined grouping of digits, e.g. 1,000,000 (which means it has no effect in the C/POSIX locale)")
	fs.String("header", "", "print (without converting) the first `N` header lines; N defaults to 1 if not specified")
	fs.Lookup("header").NoOptDefVal = "1"
	fs.String("invalid", "", "failure mode for invalid numbers: `MODE` can be: abort (default), fail, warn, ignore")
	fs.String("padding", "", "pad the output to `N` characters; positive N will right-align; negative N will left-align; padding is ignored if the output is wider than N; the default is to automatically pad if a whitespace is found")
	fs.String("round", "", "use `METHOD` for rounding when scaling; METHOD can be: up, down, from-zero (default), towards-zero, nearest")
	fs.String("suffix", "", "add `SUFFIX` to output numbers, and accept optional SUFFIX in input numbers")
	fs.String("to", "", "auto-scale output numbers to `UNIT`s; see UNIT below")
	fs.String("to-unit", "", "the output unit size (instead of the default 1)")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
}

// once is a string flag that remembers being given twice, which GNU refuses
// for --field.
type once struct {
	val   string
	count int
}

func (o *once) Set(s string) error { o.val = s; o.count++; return nil }
func (o *once) String() string     { return o.val }
func (o *once) Type() string       { return "string" }

// fatal is a mistake GNU reports without the --help hint.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

func fatalf(format string, a ...any) error {
	return &fatal{fmt.Sprintf(format, a...)}
}

type choice[T any] struct {
	name  string
	value T
}

// match resolves an argument as GNU's XARGMATCH does: an exact name or an
// unambiguous prefix of one, and a list of the valid ones otherwise.
func match[T any](flag, arg string, choices []choice[T]) (T, error) {
	var found T
	n := 0
	for _, c := range choices {
		if c.name == arg {
			return c.value, nil
		}
		if arg != "" && strings.HasPrefix(c.name, arg) {
			found = c.value
			n++
		}
	}
	if n == 1 {
		return found, nil
	}
	what := "invalid"
	if n > 1 {
		what = "ambiguous"
	}
	var b strings.Builder
	b.WriteString(what + " argument '" + arg + "' for '--" + flag + "'\nValid arguments are:")
	for _, c := range choices {
		b.WriteString("\n  - '" + c.name + "'")
	}
	return found, &core.UsageError{Msg: b.String()}
}

var (
	fromScales = []choice[numfmt.Scale]{{"none", numfmt.ScaleNone}, {"auto", numfmt.ScaleAuto},
		{"si", numfmt.ScaleSI}, {"iec", numfmt.ScaleIEC}, {"iec-i", numfmt.ScaleIECI}}
	toScales = []choice[numfmt.Scale]{{"none", numfmt.ScaleNone}, {"si", numfmt.ScaleSI},
		{"iec", numfmt.ScaleIEC}, {"iec-i", numfmt.ScaleIECI}}
	rounds = []choice[numfmt.Round]{{"up", numfmt.RoundUp}, {"down", numfmt.RoundDown},
		{"from-zero", numfmt.RoundFromZero}, {"towards-zero", numfmt.RoundTowardsZero}, {"nearest", numfmt.RoundNearest}}
	invalids = []choice[numfmt.Invalid]{{"abort", numfmt.InvalidAbort}, {"fail", numfmt.InvalidFail},
		{"warn", numfmt.InvalidWarn}, {"ignore", numfmt.InvalidIgnore}}
)

// buildOptions folds the parsed flags into numfmt.Options.
func buildOptions(fs *pflag.FlagSet) (numfmt.Options, error) {
	var o numfmt.Options
	var err error
	str := func(name string) string { v, _ := fs.GetString(name); return v }

	if fs.Changed("from") {
		if o.From, err = match("from", str("from"), fromScales); err != nil {
			return o, err
		}
	}
	if fs.Changed("from-unit") {
		if o.FromUnit, err = unitSize(str("from-unit")); err != nil {
			return o, err
		}
	}
	if fs.Changed("to") {
		if o.To, err = match("to", str("to"), toScales); err != nil {
			return o, err
		}
	}
	if fs.Changed("to-unit") {
		if o.ToUnit, err = unitSize(str("to-unit")); err != nil {
			return o, err
		}
	}
	if fs.Changed("round") {
		if o.Round, err = match("round", str("round"), rounds); err != nil {
			return o, err
		}
	}
	o.Grouping, _ = fs.GetBool("grouping")
	if fs.Changed("padding") {
		v := str("padding")
		p, ok := padding(v)
		if !ok || p == 0 {
			return o, fatalf("invalid padding value '%s'", v)
		}
		o.Padding = p
	}
	if f := fs.Lookup("field").Value.(*once); f.count > 0 {
		if f.count > 1 {
			return o, fatalf("multiple field specifications")
		}
		o.Fields = f.val
		if o.Fields == "" {
			return o, core.Usagef("missing list of fields")
		}
	}
	if fs.Changed("delimiter") {
		d := str("delimiter")
		if len(d) > 1 {
			return o, fatalf("the delimiter must be a single character")
		}
		o.DelimiterSet = true
		if d != "" {
			o.Delimiter = d[0]
		}
	}
	o.ZeroTerminated, _ = fs.GetBool("zero-terminated")
	o.Suffix = str("suffix")
	o.Debug, _ = fs.GetBool("debug")
	if fs.Changed("header") {
		v := str("header")
		h, err := strconv.ParseUint(v, 10, 64)
		if err != nil || h == 0 {
			return o, fatalf("invalid header value '%s'", v)
		}
		o.Header = h
	}
	o.Format = str("format")
	if fs.Changed("invalid") {
		if o.Invalid, err = match("invalid", str("invalid"), invalids); err != nil {
			return o, err
		}
	}
	return o, nil
}

// padding reads a --padding value as xstrtoimax does: blanks, a sign and
// digits; a value out of range is clamped.
func padding(s string) (int64, bool) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	if t == "" {
		return 0, false
	}
	p, err := strconv.ParseInt(t, 10, 64)
	if errors.Is(err, strconv.ErrRange) {
		if strings.HasPrefix(t, "-") {
			return math.MinInt64, true
		}
		return math.MaxInt64, true
	}
	return p, err == nil
}

// unitSize reads a --from-unit or --to-unit size: a number, optionally
// followed by a unit as --from=auto reads them (K is 1000, Ki is 1024).
func unitSize(s string) (uint64, error) {
	bad := fatalf("invalid unit size: '%s'", s)
	digits := len(s) - len(strings.TrimLeft(s, "0123456789"))
	if digits == 0 {
		return 0, bad
	}
	n, err := strconv.ParseUint(s[:digits], 10, 64)
	if err != nil {
		return 0, bad
	}
	rest := s[digits:]
	base := uint64(1000)
	if strings.HasSuffix(rest, "i") && len(rest) == 2 {
		base, rest = 1024, rest[:1]
	}
	switch {
	case rest == "" && base == 1000:
	case len(rest) == 1 && strings.IndexByte("KkMGTPEZYRQ", rest[0]) >= 0:
		p := strings.IndexByte("KMGTPEZYRQ", rest[0]) + 1
		if rest == "k" {
			if base == 1024 {
				return 0, bad
			}
			p = 1
		}
		for range p {
			if n > math.MaxUint64/base {
				return 0, bad
			}
			n *= base
		}
	default:
		return 0, bad
	}
	if n == 0 {
		return 0, bad
	}
	return n, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var f *fatal
	if errors.As(err, &f) {
		d := core.NewDiag(numfmt.Name, s)
		d.Errorf("%s", f.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	return numfmt.New(opts, s).Run(args)
}

// Meta returns the documentation metadata for numfmtx.
func Meta() docmeta.Meta {
	n := numfmt.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"dux", "dfx", "lsx", "sortx"},
		Use:     n + " [OPTION]... [NUMBER]...",
		Short:   "Convert numbers to and from human-readable strings.",
		Long: n + " reformats each NUMBER, or the numbers in each line of standard input when " +
			"there are none. --to writes a number with a unit suffix: si in powers of 1000 " +
			"(1.5k, 2.0M), iec in powers of 1024 (1.5K), iec-i with an i (1.5Ki). --from reads " +
			"such suffixes: si and iec as those powers, iec-i requiring the i, and auto taking " +
			"1K as 1000 and 1Ki as 1024. A scaled value below 10 keeps one decimal; the rest " +
			"are rounded away from zero, or as --round says. --from-unit and --to-unit set the " +
			"size of an input and an output unit, --suffix adds text after each number (and " +
			"accepts it on input), and --format takes a printf-style %f directive with a width, " +
			"precision, zero padding, and text around it. In a line, only the first field is " +
			"converted unless --field lists others (cut's syntax, or - for all); fields are " +
			"separated by blanks, or by -d's character, and a field that started after blanks " +
			"keeps its width. --header copies the first N lines unchanged; -z reads and writes " +
			"NUL-terminated lines. A number that cannot be converted stops the run with status " +
			"2 (--invalid=abort, the default); --invalid=fail prints it unchanged and carries " +
			"on, still ending with 2, warn does so ending with 0, and ignore without a message. " +
			"gnux works in the C locale: the decimal point is '.', and --grouping (or ' in " +
			"--format) adds no separators, as in GNU numfmt under LC_ALL=C. Values are computed " +
			"with the 64-bit precision of GNU's long double on x86-64, so a plain number of " +
			"more than 18 digits needs --to, as there.",
		Permissions: "Reads standard input and writes standard output only; needs no permissions.",
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "success, or invalid input under --invalid=warn or ignore"},
			{Code: "1", Meaning: "a command-line mistake"},
			{Code: "2", Meaning: "a number could not be converted"},
		},
		UseCases: []string{
			"Turning byte counts from wcx, statx or dux -b into sizes people can read.",
			"Reading sizes such as 10G or 512Mi from a config file back into bytes.",
			"Formatting a column of a report with a fixed width and precision.",
			"Converting between units, such as kilobytes to bytes, in a pipeline.",
		},
		Examples: []docmeta.Example{
			{Description: "Show a byte count the way people read sizes:", Bash: n + " --to=iec 1048576", PowerShell: n + " --to=iec 1048576", Output: "1.0M"},
			{Description: "Use decimal units, as disk makers do:", Bash: n + " --to=si 1500000", PowerShell: n + " --to=si 1500000", Output: "1.5M"},
			{Description: "Turn a size from a config file back into bytes:", Bash: n + " --from=iec 4G", PowerShell: n + " --from=iec 4G", Output: "4294967296"},
			{Description: "Read sizes written either way, 1K as 1000 and 1Ki as 1024:", Bash: n + " --from=auto 1K 1Ki", PowerShell: n + " --from=auto 1K 1Ki", Output: "1000\n1024"},
			{Description: "Make the size column of a listing readable, keeping the header:",
				Bash:       "lsx -l | " + n + " --header --field=5 --to=iec",
				PowerShell: "lsx -l | " + n + " --header --field=5 --to=iec"},
			{Description: "Round sizes down, never overstating them:", Bash: n + " --to=si --round=down 1999", PowerShell: n + " --to=si --round=down 1999", Output: "1.9k"},
			{Description: "Add a unit and pad the numbers to a column:", Bash: n + " --to=iec --suffix=B --padding=7 1024 2048000", PowerShell: n + " --to=iec --suffix=B --padding=7 1024 2048000", Output: "  1.0KB\n  2.0MB"},
			{Description: "Print with three decimals inside some text:", Bash: n + " --format='took %.3fs' 1.23456", PowerShell: n + " --format='took %.3fs' 1.23456", Output: "took 1.235s"},
			{Description: "Convert kilobyte counts, as dfx prints them, to bytes:", Bash: "echox 512 | " + n + " --from-unit=1024", PowerShell: "echox 512 | " + n + " --from-unit=1024", Output: "524288"},
			{Description: "Convert the second column of a CSV file:", Bash: n + " -d, --field=2 --to=si < sizes.csv", PowerShell: "Get-Content sizes.csv | " + n + " -d, --field=2 --to=si"},
			{Description: "Keep going past values that are not numbers, but say which:", Bash: n + " --invalid=warn --to=si 1000 n/a 2000", PowerShell: n + " --invalid=warn --to=si 1000 n/a 2000"},
		},
	}
}
