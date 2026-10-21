// Package prx wires the prx command (GNU pr) to its engine.
//
// pr's command line does not fit a flag set: its options include digits
// (-3 is three columns), short options whose value must be attached (-e8,
// -n:3), a +FIRST_PAGE operand, and options whose meaning depends on the
// order and the company they keep (-s and -w next to -S, -W and the column
// options). So prx reads it itself, as GNU's getopt_long does, and Register
// declares the options for the help and the reference only.
package prx

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pr"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the prx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "pr",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Register declares the prx options, for the documentation.
func Register(fs *pflag.FlagSet) {
	fs.String("pages", "", "begin [stop] printing with page `FIRST_PAGE[:LAST_PAGE]`; also +FIRST_PAGE[:LAST_PAGE]")
	fs.String("columns", "", "output `COLUMN` columns and print columns down, unless -a is used; also -COLUMN")
	fs.BoolP("across", "a", false, "print columns across rather than down, used together with -COLUMN")
	fs.BoolP("show-control-chars", "c", false, "use hat notation (^G) and octal backslash notation")
	fs.BoolP("double-space", "d", false, "double space the output")
	fs.StringP("date-format", "D", "", "use `FORMAT` for the header date")
	fs.StringP("expand-tabs", "e", "", "expand input `CHAR`s (TABs) to tab `WIDTH` (8); written -e[CHAR[WIDTH]]")
	fs.BoolP("form-feed", "f", false, "use form feeds instead of newlines to separate pages (also -F)")
	fs.StringP("header", "h", "", "use a centered `HEADER` instead of filename in page header")
	fs.StringP("output-tabs", "i", "", "replace spaces with `CHAR`s (TABs) to tab `WIDTH` (8); written -i[CHAR[WIDTH]]")
	fs.BoolP("join-lines", "J", false, "merge full lines, turns off -W line truncation, no column alignment, --sep-string[=STRING] sets separators")
	fs.StringP("length", "l", "", "set the page length to `PAGE_LENGTH` (66) lines")
	fs.BoolP("merge", "m", false, "print all files in parallel, one in each column, truncate lines, but join lines of full length with -J")
	fs.StringP("number-lines", "n", "", "number lines, use `DIGITS` (5) digits, then `SEP` (TAB); written -n[SEP[DIGITS]]")
	fs.StringP("first-line-number", "N", "", "start counting with `NUMBER` at 1st line of first page printed")
	fs.StringP("indent", "o", "", "offset each line with `MARGIN` (zero) spaces")
	fs.BoolP("no-file-warnings", "r", false, "omit warning when a file cannot be opened")
	fs.StringP("separator", "s", "", "separate columns by a single character, default for `CHAR` is TAB without -w and 'no char' with -w; written -s[CHAR]")
	fs.StringP("sep-string", "S", "", "separate columns by `STRING`; without -S, a blank with -J and a space otherwise; written -S[STRING]")
	fs.BoolP("omit-header", "t", false, "omit page headers and trailers; implied if PAGE_LENGTH <= 10")
	fs.BoolP("omit-pagination", "T", false, "omit page headers and trailers, eliminate any pagination by form feeds set in input files")
	fs.BoolP("show-nonprinting", "v", false, "use octal backslash notation")
	fs.StringP("width", "w", "", "set page width to `PAGE_WIDTH` (72) characters for multiple text-column output only, -s[char] turns off (72)")
	fs.StringP("page-width", "W", "", "set page width to `PAGE_WIDTH` (72) characters always, truncate lines, except -J option is set, no interference with -S or -s")
}

// argKind is whether an option takes a value.
type argKind int

const (
	noArg argKind = iota
	required
	optional
)

type longOpt struct {
	name  string
	kind  argKind
	short byte // the short option it stands for; 0 for --pages and --columns
}

var longOpts = []longOpt{
	{"pages", required, 0}, {"columns", required, 0},
	{"across", noArg, 'a'}, {"show-control-chars", noArg, 'c'}, {"double-space", noArg, 'd'},
	{"date-format", required, 'D'}, {"expand-tabs", optional, 'e'}, {"form-feed", noArg, 'f'},
	{"header", required, 'h'}, {"output-tabs", optional, 'i'}, {"join-lines", noArg, 'J'},
	{"length", required, 'l'}, {"merge", noArg, 'm'}, {"number-lines", optional, 'n'},
	{"first-line-number", required, 'N'}, {"indent", required, 'o'}, {"no-file-warnings", noArg, 'r'},
	{"separator", optional, 's'}, {"sep-string", optional, 'S'}, {"omit-header", noArg, 't'},
	{"omit-pagination", noArg, 'T'}, {"show-nonprinting", noArg, 'v'}, {"width", required, 'w'},
	{"page-width", required, 'W'},
}

// shortOpts is GNU pr's option string, without its digits and leading '-'.
var shortOpts = map[byte]argKind{
	'D': required, 'F': noArg, 'J': noArg, 'N': required, 'S': optional, 'T': noArg, 'W': required,
	'a': noArg, 'b': noArg, 'c': noArg, 'd': noArg, 'e': optional, 'f': noArg, 'h': required,
	'i': optional, 'l': required, 'm': noArg, 'n': optional, 'o': required, 'r': noArg,
	's': optional, 't': noArg, 'v': noArg, 'w': required,
}

// fatal is a mistake GNU reports without the --help hint.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

func quote(s string) string { return "'" + s + "'" }

// parser holds the state of GNU pr's option loop.
type parser struct {
	o                      pr.Options
	files                  []string
	oldOptions, oldW, oldS bool
	// digits are the column count given as -COLUMN. A later digit after
	// another option starts it again (-2 -a -3 is 3), but the option alone
	// leaves it be (-12 -a is 12), as GNU's accumulation does.
	digits string
	fresh  bool
}

// parseArgs reads a pr command line into pr.Options and the files.
func parseArgs(args []string) (pr.Options, []string, error) {
	var p parser
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			p.files = append(p.files, args[i+1:]...)
			return p.finish()
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			opt, err := findLong(name)
			if err != nil {
				return pr.Options{}, nil, err
			}
			switch {
			case opt.kind == noArg && hasValue:
				return pr.Options{}, nil, core.Usagef("option '--%s' doesn't allow an argument", opt.name)
			case opt.kind == required && !hasValue:
				if i+1 >= len(args) {
					return pr.Options{}, nil, core.Usagef("option '--%s' requires an argument", opt.name)
				}
				i++
				value, hasValue = args[i], true
			}
			p.fresh = true
			if err := p.option(opt.short, opt.name, value, hasValue); err != nil {
				return pr.Options{}, nil, err
			}
		case strings.HasPrefix(a, "-") && a != "-":
			for j := 1; j < len(a); j++ {
				ch := a[j]
				if ch >= '0' && ch <= '9' {
					if p.fresh {
						p.digits, p.fresh = "", false
					}
					p.digits += string(ch)
					continue
				}
				kind, ok := shortOpts[ch]
				if !ok {
					return pr.Options{}, nil, core.Usagef("invalid option -- '%c'", ch)
				}
				p.fresh = true
				value, hasValue := "", false
				switch {
				case kind == optional && j+1 < len(a):
					value, hasValue = a[j+1:], true
					j = len(a)
				case kind == required && j+1 < len(a):
					value, hasValue = a[j+1:], true
					j = len(a)
				case kind == required:
					if i+1 >= len(args) {
						return pr.Options{}, nil, core.Usagef("option requires an argument -- '%c'", ch)
					}
					i++
					value, hasValue = args[i], true
				}
				if err := p.option(ch, "", value, hasValue); err != nil {
					return pr.Options{}, nil, err
				}
			}
		default:
			// An operand: +FIRST_PAGE[:LAST_PAGE] unless --pages came first.
			p.fresh = true
			if p.o.FirstPage == 0 && strings.HasPrefix(a, "+") {
				if first, last, ok, err := pages(a[1:]); err != nil {
					return pr.Options{}, nil, err
				} else if ok {
					p.o.FirstPage, p.o.LastPage = first, last
					continue
				}
			}
			p.files = append(p.files, a)
		}
	}
	return p.finish()
}

func findLong(name string) (longOpt, error) {
	var matches []longOpt
	for _, o := range longOpts {
		if o.name == name {
			return o, nil
		}
		if strings.HasPrefix(o.name, name) {
			matches = append(matches, o)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return longOpt{}, core.Usagef("unrecognized option '--%s'", name)
	}
	var list strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&list, " '--%s'", m.name)
	}
	return longOpt{}, core.Usagef("option '--%s' is ambiguous; possibilities:%s", name, list.String())
}

// option applies one option, as GNU pr's main does.
func (p *parser) option(ch byte, long, value string, hasValue bool) error {
	o := &p.o
	var err error
	switch long {
	case "pages":
		first, last, ok, perr := pages(value)
		if perr != nil {
			return perr
		}
		if !ok {
			return &fatal{"invalid page range " + quote(value)}
		}
		o.FirstPage, o.LastPage = first, last
	case "columns":
		if o.Columns, err = number(value, 1, "invalid number of columns"); err != nil {
			return err
		}
		p.digits = ""
	}
	if long == "pages" || long == "columns" {
		return nil
	}
	switch ch {
	case 'a':
		o.Across = true
	case 'b':
		o.Balance = true
	case 'c':
		o.ShowControlChars = true
	case 'd':
		o.DoubleSpace = true
	case 'D':
		o.DateFormat = value
	case 'e':
		if hasValue {
			if err := charWidth(value, 'e', &o.InputTab, &o.InputTabWidth); err != nil {
				return err
			}
		}
		o.ExpandTabs = true
	case 'f', 'F':
		o.FormFeed = true
	case 'h':
		o.Header, o.HeaderSet = value, true
	case 'i':
		if hasValue {
			if err := charWidth(value, 'i', &o.OutputTab, &o.OutputTabWidth); err != nil {
				return err
			}
		}
		o.OutputTabs = true
	case 'J':
		o.JoinLines = true
	case 'l':
		if o.Length, err = number(value, 1, "'-l PAGE_LENGTH' invalid number of lines"); err != nil {
			return err
		}
	case 'm':
		o.Merge = true
	case 'n':
		o.NumberLines = true
		if hasValue {
			if err := charWidth(value, 'n', &o.NumberSeparator, &o.NumberWidth); err != nil {
				return err
			}
		}
	case 'N':
		if o.FirstLineNumber, err = number(value, math.MinInt32, "'-N NUMBER' invalid starting line number"); err != nil {
			return err
		}
		o.FirstLineNumberSet = true
	case 'o':
		if o.Indent, err = number(value, 0, "'-o MARGIN' invalid line offset"); err != nil {
			return err
		}
	case 'r':
		o.NoFileWarnings = true
	case 's':
		p.oldOptions, p.oldS = true, true
		if !o.UseSeparator && hasValue {
			o.Separator = value
		}
	case 'S':
		p.oldS = false
		o.Separator, o.UseSeparator = "", true
		if hasValue {
			o.Separator = value
		}
	case 't':
		o.OmitHeader = true
		o.OmitPagination = false
	case 'T':
		o.OmitPagination = true
		o.OmitHeader = false
	case 'v':
		o.ShowNonprinting = true
	case 'w':
		p.oldOptions, p.oldW = true, true
		width, err := number(value, 1, "'-w PAGE_WIDTH' invalid number of characters")
		if err != nil {
			return err
		}
		if !o.Truncate {
			o.Width = width
		}
	case 'W':
		p.oldW = false
		o.Truncate = true
		if o.Width, err = number(value, 1, "'-W PAGE_WIDTH' invalid number of characters"); err != nil {
			return err
		}
	}
	return nil
}

// finish resolves what -s and -w mean, as GNU pr does after reading its
// options, for compatibility with older pr programs.
func (p *parser) finish() (pr.Options, []string, error) {
	o := &p.o
	if p.digits != "" {
		n, err := number(p.digits, 1, "invalid number of columns")
		if err != nil {
			return pr.Options{}, nil, err
		}
		o.Columns = n
	}
	explicit := o.Columns > 0
	if o.Merge && explicit {
		return pr.Options{}, nil, &fatal{"cannot specify number of columns when printing in parallel"}
	}
	if o.Merge && o.Across {
		return pr.Options{}, nil, &fatal{"cannot specify both printing across and printing in parallel"}
	}
	if p.oldOptions {
		switch {
		case p.oldW:
			if o.Merge || explicit {
				o.Truncate = true
				if p.oldS {
					o.UseSeparator = true
				}
			} else {
				o.JoinLines = true
			}
		case !o.UseSeparator && p.oldS && (o.Merge || explicit):
			if !o.Truncate {
				o.JoinLines = true
				if o.Separator != "" {
					o.UseSeparator = true
				}
			} else {
				o.UseSeparator = true
			}
		}
	}
	return *o, p.files, nil
}

// pages reads FIRST_PAGE[:LAST_PAGE]. ok is false for a range that is not
// one, which as +... makes the operand a file name.
func pages(s string) (first, last uint64, ok bool, err error) {
	digits := len(s) - len(strings.TrimLeft(s, "0123456789"))
	if digits == 0 {
		return 0, 0, false, nil
	}
	first, perr := strconv.ParseUint(s[:digits], 10, 64)
	if perr != nil {
		return 0, 0, false, &fatal{fmt.Sprintf("--pages argument %s too large", quote(s))}
	}
	if first == 0 {
		return 0, 0, false, nil
	}
	rest := s[digits:]
	last = math.MaxUint64
	if strings.HasPrefix(rest, ":") {
		rest = rest[1:]
		d := len(rest) - len(strings.TrimLeft(rest, "0123456789"))
		if d == 0 {
			return 0, 0, false, nil
		}
		if last, perr = strconv.ParseUint(rest[:d], 10, 64); perr != nil {
			return 0, 0, false, &fatal{fmt.Sprintf("--pages argument %s too large", quote(s))}
		}
		if last < first {
			return 0, 0, false, nil
		}
		rest = rest[d:]
	}
	if rest != "" {
		return 0, 0, false, nil
	}
	return first, last, true, nil
}

// number reads a decimal option value of at least min, as GNU's getoptnum.
func number(s string, minimum int, what string) (int, error) {
	t := strings.TrimLeft(s, " \t\n\v\f\r")
	n, err := strconv.ParseInt(t, 10, 64)
	switch {
	case errors.Is(err, strconv.ErrRange) || (err == nil && n > math.MaxInt32):
		return 0, &fatal{fmt.Sprintf("%s: %s: Value too large for defined data type", what, quote(s))}
	case err != nil:
		return 0, &fatal{fmt.Sprintf("%s: %s", what, quote(s))}
	case n < int64(minimum) && minimum > 0:
		return 0, &fatal{fmt.Sprintf("%s: %s: Numerical result out of range", what, quote(s))}
	case n < int64(minimum):
		return 0, &fatal{fmt.Sprintf("%s: %s", what, quote(s))}
	}
	return int(n), nil
}

// charWidth reads the [CHAR[WIDTH]] value of -e, -i and -n.
func charWidth(s string, opt byte, char *byte, width *int) error {
	if s == "" {
		return core.Usagef("'-%c': Invalid argument: %s", opt, quote(s))
	}
	if s[0] < '0' || s[0] > '9' {
		*char = s[0]
		s = s[1:]
	}
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	switch {
	case errors.Is(err, strconv.ErrRange) || (err == nil && n > math.MaxInt32):
		return core.Usagef("'-%c' extra characters or invalid number in the argument: %s: Value too large for defined data type", opt, quote(s))
	case err != nil || n <= 0:
		return core.Usagef("'-%c' extra characters or invalid number in the argument: %s", opt, quote(s))
	}
	*width = int(n)
	return nil
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	opts, files, err := parseArgs(args)
	var f *fatal
	if errors.As(err, &f) {
		d := core.NewDiag(pr.Name, s)
		d.Errorf("%s", f.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	return pr.New(opts, s).Run(files)
}

// Meta returns the documentation metadata for prx.
func Meta() docmeta.Meta {
	n := pr.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"fmtx", "foldx", "nlx", "pastex"},
		Use:     n + " [OPTION]... [FILE]...",
		Short:   "Paginate or columnate files for printing.",
		Long: n + " breaks each FILE into pages of 66 lines (-l) with a five-line header, the " +
			"date, the file name and the page number, and a five-line footer; -t drops the " +
			"header and footer and -T also the page breaks that form feeds in the input make. " +
			"-COLUMN (or --columns) prints the text in that many columns down the page, " +
			"balancing the last page, -a across instead, and -m prints the FILEs side by side, " +
			"one per column; columns are cut to fit the page width (72, -w or -W) unless -J " +
			"joins whole lines. -n numbers the lines, -d double-spaces, -o indents, -h replaces " +
			"the file name in the header and -D the date's format, -e and -i expand and make tabs, " +
			"-c and -v show control characters, -s and -S choose the column separator, and " +
			"+FIRST[:LAST] (or --pages) prints only those pages. With no FILE, or when FILE is " +
			"-, it reads standard input. The layout is GNU pr's, page break for page break. " +
			"gnux works in the C locale, so the header's date is in the C locale's words.",
		Permissions: "Needs read permission on each FILE. Nothing is modified.",
		UseCases: []string{
			"Printing source code or logs with page headers and numbers.",
			"Putting a long list into several columns to fit a screen or a page.",
			"Comparing files side by side with -m.",
			"Numbering the lines of a listing for review.",
		},
		Examples: []docmeta.Example{
			{Description: "Add page headers and page breaks to a file before printing it:", Bash: n + " report.txt | lpr", PowerShell: n + " report.txt | Out-Printer"},
			{Description: "Show a long list of names in three columns, without headers:", Bash: "lsx -1 | " + n + " -3 -t", PowerShell: "lsx -1 | " + n + " -3 -t"},
			{Description: "Fill the columns across, row by row:", Bash: "seqx 10 | " + n + " -5 -a -t", PowerShell: "seqx 10 | " + n + " -5 -a -t"},
			{Description: "Put two files side by side to compare them:", Bash: n + " -m -t old.conf new.conf", PowerShell: n + " -m -t old.conf new.conf"},
			{Description: "Number the lines of a script, with a custom title:", Bash: n + " -n -h 'deploy script' deploy.sh", PowerShell: n + " -n -h 'deploy script' deploy.ps1"},
			{Description: "Print only pages 3 to 5:", Bash: n + " +3:5 manual.txt", PowerShell: n + " +3:5 manual.txt"},
			{Description: "Join CSV columns with a comma instead of aligning them:", Bash: n + " -m -t -s, names.txt emails.txt", PowerShell: n + " -m -t -s, names.txt emails.txt"},
			{Description: "Use short pages and a form feed between them:", Bash: n + " -l 40 -f notes.txt", PowerShell: n + " -l 40 -f notes.txt"},
		},
	}
}
