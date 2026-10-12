// Package datex wires the datex command (GNU date) to its engine.
package datex

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/date"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the datex tool. GNU date's -I takes an optional argument
// attached to it (-Iseconds), which pflag would read as a cluster of other
// options, so the arguments are parsed here, with -I rewritten first.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "date",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for datex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    date.Name,
		SeeAlso: []string{"touchx", "sleepx", "lsx", "uptimex"},
		Use:     date.Name + " [OPTION]... [+FORMAT]",
		Short:   "Print the date and time.",
		Long: "datex prints the current date and time, or the one -d STRING describes, in the " +
			"C locale's default format ('Mon Jan  2 15:04:05 UTC 2006') or in +FORMAT. FORMAT " +
			"takes GNU's conversions: %Y %m %d %H %M %S and friends, %a %A %b %B %p (in " +
			"English), %s (seconds since the epoch), %N (nanoseconds), %z %:z %::z %:::z and " +
			"%Z (zones), %j %U %W %V %G %u %w (day and week numbers), %F %T %D %R %r %c %x %X " +
			"%h %e %k %l %C %y %g %n %t %%, and the flags -, _, 0, ^ and # with a field width " +
			"(%-d, %_H, %10s). -I[FMT] prints ISO 8601 (date, hours, minutes, seconds or ns; " +
			"date by default), --rfc-3339=FMT RFC 3339 (date, seconds or ns) and -R RFC 5322; " +
			"only one output format may be given. -u uses UTC, and otherwise the TZ " +
			"environment variable or the local zone applies. -d STRING accepts the forms GNU " +
			"date accepts most often: 'now', 'today', 'yesterday', 'tomorrow', '@SECONDS', " +
			"ISO 8601 and RFC 3339 dates and times, MM/DD/YYYY, 'Jan 2 2006', '2 Jan 2006', " +
			"times of day (14:30, 2:30pm, noon, midnight) with UTC, Z or a numeric offset, " +
			"weekdays ('next friday', 'last monday') and relative items ('3 days ago', '+2 " +
			"hours', 'next month', 'fortnight'), combined as in '2026-01-31 +1 month' or " +
			"'tomorrow 09:00'. Anything else is reported as an invalid date, where GNU's " +
			"grammar is larger still. -f FILE prints one date per line of FILE (- for " +
			"standard input) and -r FILE the last modification time of FILE. gnux never " +
			"changes system settings, so -s and a MMDDhhmm[[CC]YY][.ss] operand fail with " +
			"'cannot set date: Operation not supported'. --debug and --resolution are not " +
			"implemented.",
		Permissions: "Printing needs no permissions; -f needs read permission on FILE and -r " +
			"search permission on the directories leading to it.",
		UseCases: []string{
			"Timestamping file names, log lines and backups.",
			"Converting between epoch seconds and readable dates.",
			"Computing dates relative to now or to another date, such as 'next friday'.",
			"Printing machine-readable ISO 8601 or RFC 3339 timestamps.",
			"Showing the current time in another zone with TZ or -u.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Check the date and time on a server:",
				Bash:        "datex",
				PowerShell:  "datex",
			},
			{
				Description: "Get today's date in a sortable form for a file name:",
				Bash:        "datex -I",
				PowerShell:  "datex -I",
			},
			{
				Description: "Stamp a log line with a precise, sortable time:",
				Bash:        "datex -Iseconds",
				PowerShell:  "datex -Iseconds",
			},
			{
				Description: "Check UTC when coordinating with people in other time zones:",
				Bash:        "datex -u",
				PowerShell:  "datex -u",
			},
			{
				Description: "Print the time the way your report needs it:",
				Bash:        "datex '+%Y-%m-%d %H:%M:%S'",
				PowerShell:  "datex '+%Y-%m-%d %H:%M:%S'",
			},
			{
				Description: "Name tonight's database backup after today's date:",
				Bash:        "cpx db.sqlite \"db-$(datex +%F).sqlite\"",
				PowerShell:  "cpx db.sqlite \"db-$(datex +%F).sqlite\"",
			},
			{
				Description: "Get a Unix timestamp to store or compare:",
				Bash:        "datex +%s",
				PowerShell:  "datex +%s",
			},
			{
				Description: "Read a Unix timestamp you found in a log or a database:",
				Bash:        "datex -u -d @1700000000",
				PowerShell:  "datex -u -d '@1700000000'",
			},
			{
				Description: "Find the date three days ago, to clean up older files:",
				Bash:        "datex -d '3 days ago' +%F",
				PowerShell:  "datex -d '3 days ago' +%F",
			},
			{
				Description: "Find the date of next Friday for a deadline:",
				Bash:        "datex -d 'next friday' +%A,%F",
				PowerShell:  "datex -d 'next friday' +%A,%F",
			},
			{
				Description: "See when a report was last saved:",
				Bash:        "datex -r report.txt",
				PowerShell:  "datex -r report.txt",
			},
			{
				Description: "See what time it is in Tokyo before calling a colleague:",
				Bash:        "TZ=Asia/Tokyo datex",
				PowerShell:  "envx TZ=Asia/Tokyo datex",
			},
			{
				Description: "Produce a timestamp an API expects in RFC 3339:",
				Bash:        "datex --rfc-3339=seconds",
				PowerShell:  "datex --rfc-3339=seconds",
			},
			{
				Description: "Convert a column of dates into timestamps for sorting:",
				Bash:        "datex -f dates.txt +%s",
				PowerShell:  "datex -f dates.txt +%s",
			},
			{
				Description: "Timestamp a log line in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'datex -Iseconds >> run.log'",
				PowerShell:  "gnux shell -c 'datex -Iseconds >> run.log'",
			},
		},
	}
}

// Register installs the datex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("date", "d", "", "display time described by `STRING`, not 'now'")
	fs.StringP("file", "f", "", "like --date; once for each line of `DATEFILE`")
	fs.StringP("iso-8601", "I", "", "output date/time in ISO 8601 format. `FMT`='date' for date only (the default), 'hours', 'minutes', 'seconds', or 'ns'")
	fs.Lookup("iso-8601").NoOptDefVal = "date"
	fs.BoolP("rfc-email", "R", false, "output date and time in RFC 5322 format")
	fs.String("rfc-3339", "", "output date/time in RFC 3339 format. `FMT`='date', 'seconds', or 'ns'")
	fs.StringP("reference", "r", "", "display the last modification time of `FILE`")
	fs.StringP("set", "s", "", "set time described by `STRING` (not supported)")
	fs.BoolP("utc", "u", false, "print or set Coordinated Universal Time (UTC)")
	fs.Bool("universal", false, "same as --utc")
}

// parsed is a command line split into its parts.
type parsed struct {
	opts     date.Options
	operands []string
	help     bool
	version  bool
}

// takesValue lists the short options whose value is the rest of their
// argument or the next one.
const takesValue = "dfrs"

// rewrite turns GNU's attached -I[FMT] into --iso-8601[=FMT], which pflag
// reads correctly, leaving every other argument as it is.
func rewrite(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return append(out, args[i:]...)
		}
		if strings.HasPrefix(a, "--") {
			out = append(out, a)
			name, _, hasValue := strings.Cut(a[2:], "=")
			if !hasValue && i+1 < len(args) {
				switch name {
				case "date", "file", "reference", "set", "rfc-3339":
					i++
					out = append(out, args[i])
				}
			}
			continue
		}
		if len(a) < 2 || a[0] != '-' {
			out = append(out, a)
			continue
		}
		rewritten := false
		for j := 1; j < len(a); j++ {
			c := a[j]
			if strings.IndexByte(takesValue, c) >= 0 {
				out = append(out, a)
				if j == len(a)-1 && i+1 < len(args) {
					i++
					out = append(out, args[i])
				}
				rewritten = true
				break
			}
			if c == 'I' {
				if j > 1 {
					out = append(out, a[:j])
				}
				iso := "--iso-8601"
				if j+1 < len(a) {
					iso += "=" + a[j+1:]
				}
				out = append(out, iso)
				rewritten = true
				break
			}
		}
		if !rewritten {
			out = append(out, a)
		}
	}
	return out
}

// parseArgs parses the flags on fs and folds them into date.Options.
func parseArgs(fs *pflag.FlagSet, args []string) (parsed, error) {
	var p parsed
	if err := fs.Parse(rewrite(args)); err != nil {
		return p, core.Usagef("%s", err)
	}
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	str := func(name string) string {
		s, _ := fs.GetString(name)
		return s
	}
	p.help, p.version = on("help"), on("version")
	p.opts = date.Options{
		Date:      str("date"),
		HasDate:   fs.Changed("date"),
		File:      str("file"),
		Reference: str("reference"),
		Set:       str("set"),
		HasSet:    fs.Changed("set"),
		UTC:       on("utc") || on("universal"),
		RFC3339:   str("rfc-3339"),
		RFC5322:   on("rfc-email"),
	}
	if fs.Changed("iso-8601") {
		p.opts.ISO8601 = str("iso-8601")
		if p.opts.ISO8601 == "" {
			p.opts.ISO8601 = "date"
		}
		if full, err := matchISO(p.opts.ISO8601); err == nil {
			p.opts.ISO8601 = full
		}
	}
	if fs.Changed("rfc-3339") {
		if full, err := matchRFC(p.opts.RFC3339); err == nil {
			p.opts.RFC3339 = full
		}
	}
	p.operands = fs.Args()
	return p, nil
}

// matchISO and matchRFC complete an unambiguous abbreviation of a precision
// ("sec" for "seconds"), as GNU's argmatch does; anything else is left for
// the engine to reject.
func matchISO(s string) (string, error) {
	return match(s, "date", "hours", "minutes", "seconds", "ns")
}

func matchRFC(s string) (string, error) {
	return match(s, "date", "seconds", "ns")
}

func match(s string, words ...string) (string, error) {
	var found []string
	for _, w := range words {
		if w == s {
			return w, nil
		}
		if strings.HasPrefix(w, s) {
			found = append(found, w)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return s, core.Usagef("no unique match for '%s'", s)
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	p, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	switch {
	case p.help:
		Spec().WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	case p.version:
		tool.PrintVersion(s.Out, date.Name)
		return nil
	}
	return date.New(p.opts, s).Run(p.operands)
}
