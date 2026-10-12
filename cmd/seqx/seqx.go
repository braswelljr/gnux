// Package seqx wires the seqx command (GNU seq) to its engine.
package seqx

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/seq"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the seqx tool. Its operands are often negative numbers
// (seqx -5 5, seqx 10 -2 1), which a flag parser would take for options, so
// the arguments are parsed here the way GNU seq parses them.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "seq",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for seqx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    seq.Name,
		SeeAlso: []string{"printfx", "headx", "yesx"},
		Use:     seq.Name + " [OPTION]... [FIRST [INCREMENT]] LAST",
		Short:   "Print a sequence of numbers.",
		Long: "seqx prints the numbers from FIRST to LAST, in steps of INCREMENT, one per " +
			"line: 'seqx LAST', 'seqx FIRST LAST' or 'seqx FIRST INCREMENT LAST'. FIRST and " +
			"INCREMENT default to 1, and INCREMENT may be negative to count down; nothing is " +
			"printed when FIRST is already past LAST. Negative operands need no quoting or " +
			"'--': as in GNU seq, an argument that starts with '-' followed by a digit or '.' " +
			"is a number, and options must come before the operands. Without -f, numbers are " +
			"printed with as many decimal places as FIRST and INCREMENT have, and the " +
			"arithmetic is exact, so 'seqx 0.1 0.1 0.3' ends at 0.3. -f FORMAT prints each " +
			"number through one printf floating-point directive (%e, %f, %g or %a, with " +
			"flags, width and precision) and literal text around it; -s STRING separates the " +
			"numbers with STRING instead of a newline; -w pads them with leading zeros to " +
			"equal width, and cannot be combined with -f. LAST may be 'inf' to count " +
			"forever, until the reader of the output goes away. Differences from GNU seq: " +
			"-f values are formatted from double precision rather than long double, and an " +
			"infinite FIRST or INCREMENT is refused as an invalid argument.",
		Permissions: "Needs no filesystem permissions; it only writes to standard output.",
		UseCases: []string{
			"Driving a loop over numbered items in a script, portably on every platform.",
			"Generating test data: a known number of lines to feed into another tool.",
			"Producing zero-padded numbers with -w for file names that sort correctly.",
			"Building comma- or space-separated lists of numbers with -s.",
			"Counting down with a negative increment.",
			"Generating evenly spaced decimal values, exactly, for tables or plots.",
			"Formatting numbers with a prefix, suffix or fixed precision through -f.",
			"Producing an endless numbered stream with 'inf' that a later stage cuts short.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the numbers 1 to 5:",
				Bash:        "seqx 5",
				PowerShell:  "seqx 5",
			},
			{
				Description: "Print the numbers from 3 to 7:",
				Bash:        "seqx 3 7",
				PowerShell:  "seqx 3 7",
			},
			{
				Description: "Count in steps of 5:",
				Bash:        "seqx 0 5 25",
				PowerShell:  "seqx 0 5 25",
			},
			{
				Description: "Count down with a negative increment:",
				Bash:        "seqx 10 -2 0",
				PowerShell:  "seqx 10 -2 0",
			},
			{
				Description: "Start from a negative number, with no quoting needed:",
				Bash:        "seqx -3 3",
				PowerShell:  "seqx -3 3",
			},
			{
				Description: "Step through decimals exactly:",
				Bash:        "seqx 0 0.25 1",
				PowerShell:  "seqx 0 0.25 1",
			},
			{
				Description: "Pad the numbers with zeros to equal width:",
				Bash:        "seqx -w 8 12",
				PowerShell:  "seqx -w 8 12",
			},
			{
				Description: "Separate the numbers with commas on one line:",
				Bash:        "seqx -s , 1 10",
				PowerShell:  "seqx -s , 1 10",
			},
			{
				Description: "Format each number with a prefix and fixed precision:",
				Bash:        "seqx -f 'item-%03.0f' 1 3",
				PowerShell:  "seqx -f 'item-%03.0f' 1 3",
			},
			{
				Description: "Print percentages, with %% for a literal percent sign:",
				Bash:        "seqx -f '%.0f%%' 0 25 100",
				PowerShell:  "seqx -f '%.0f%%' 0 25 100",
			},
			{
				Description: "Number lines endlessly, stopping when the reader has enough:",
				Bash:        "seqx 1 inf | headx -n 3",
				PowerShell:  "seqx 1 inf | headx -n 3",
			},
			{
				Description: "Create ten numbered, zero-padded files:",
				Bash:        "touchx $(seqx -f 'log-%02.0f.txt' 1 10)",
				PowerShell:  "touchx (seqx -f 'log-%02.0f.txt' 1 10)",
			},
			{
				Description: "Sum the numbers 1 to 100 with another program:",
				Bash:        "seqx -s + 1 100 | bc",
				PowerShell:  "(seqx 1 100 | Measure-Object -Sum).Sum",
			},
			{
				Description: "Count the numbers a sequence produces:",
				Bash:        "seqx 0 3 100 | wcx -l",
				PowerShell:  "seqx 0 3 100 | wcx -l",
			},
			{
				Description: "Generate numbered lines in the portable gnux shell:",
				Bash:        "gnux shell -c 'seqx -f \"line %g\" 3 > lines.txt; catx lines.txt'",
				PowerShell:  "gnux shell -c 'seqx -f \"line %g\" 3 > lines.txt; catx lines.txt'",
			},
		},
	}
}

// Register declares the seqx flags. They document the tool and are listed by
// --help; parsing itself is done by parseArgs, because operands may be
// negative numbers.
func Register(fs *pflag.FlagSet) {
	fs.StringP("format", "f", "", "use printf style floating-point `FORMAT`")
	fs.StringP("separator", "s", "\\n", "use `STRING` to separate numbers")
	fs.BoolP("equal-width", "w", false, "equalize width by padding with leading zeroes")
}

// longOptions are the long options, in the order getopt would try them.
var longOptions = []string{"equal-width", "format", "help", "separator", "version"}

// parsed is the outcome of parseArgs.
type parsed struct {
	opts     seq.Options
	operands []string
	help     bool
	version  bool
}

// parseArgs parses a seqx command line the way GNU seq does: options come
// first, and an argument that starts with '-' followed by a digit or '.' is
// a negative number that ends option parsing.
func parseArgs(args []string) (parsed, error) {
	var p parsed
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if len(a) < 2 || a[0] != '-' || a[1] == '.' || (a[1] >= '0' && a[1] <= '9') {
			break
		}

		if strings.HasPrefix(a, "--") {
			name, value, hasValue := strings.Cut(a[2:], "=")
			long, err := matchLong(name)
			if err != nil {
				return p, err
			}
			switch long {
			case "format", "separator":
				if !hasValue {
					if i+1 >= len(args) {
						return p, core.Usagef("option '--%s' requires an argument", long)
					}
					i++
					value = args[i]
				}
				p.set(long, value)
			default:
				if hasValue {
					return p, core.Usagef("option '--%s' doesn't allow an argument", long)
				}
				switch long {
				case "equal-width":
					p.opts.EqualWidth = true
				case "help":
					p.help = true
				case "version":
					p.version = true
				}
			}
			continue
		}

		// A cluster of short options: -w, -wf FORMAT, -s, (value attached or next).
		for j := 1; j < len(a); j++ {
			switch c := a[j]; c {
			case 'w':
				p.opts.EqualWidth = true
			case 'f', 's':
				value := a[j+1:]
				if value == "" {
					if i+1 >= len(args) {
						return p, core.Usagef("option requires an argument -- '%c'", c)
					}
					i++
					value = args[i]
				}
				if c == 'f' {
					p.set("format", value)
				} else {
					p.set("separator", value)
				}
				j = len(a)
			default:
				return p, core.Usagef("invalid option -- '%c'", c)
			}
		}
	}
	p.operands = args[i:]
	return p, nil
}

func (p *parsed) set(name, value string) {
	if name == "format" {
		p.opts.Format = value
		return
	}
	p.opts.Separator, p.opts.SeparatorSet = value, true
}

// matchLong resolves a long option name, accepting unambiguous prefixes.
func matchLong(name string) (string, error) {
	var match string
	for _, l := range longOptions {
		if l == name {
			return l, nil
		}
		if name != "" && strings.HasPrefix(l, name) {
			if match != "" {
				return "", core.Usagef("option '--%s' is ambiguous", name)
			}
			match = l
		}
	}
	if match == "" {
		return "", core.Usagef("unrecognized option '--%s'", name)
	}
	return match, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	p, err := parseArgs(args)
	if err != nil {
		return err
	}
	switch {
	case p.help:
		spec := Spec()
		spec.WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	case p.version:
		tool.PrintVersion(s.Out, seq.Name)
		return nil
	}
	return seq.New(p.opts, s).Run(p.operands)
}
