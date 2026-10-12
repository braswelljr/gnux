// Package uniqx wires the uniqx command (GNU uniq) to its engine.
package uniqx

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uniq"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the uniqx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "uniq",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for uniqx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    uniq.Name,
		SeeAlso: []string{"cutx", "wcx", "catx", "headx"},
		Use:     uniq.Name + " [OPTION]... [INPUT [OUTPUT]]",
		Short:   "Report or omit repeated lines.",
		Long: "uniqx filters adjacent matching lines from INPUT (or standard input), writing to " +
			"OUTPUT (or standard output). With no options, matching lines are merged to the " +
			"first occurrence. Only adjacent lines are compared, so a file is usually sorted " +
			"first. -c prefixes each line with its number of occurrences, -d prints one copy of " +
			"each repeated line, -u only the lines that are not repeated, and -D every line of " +
			"each repeated group, with --all-repeated=METHOD (none, prepend, separate) putting an " +
			"empty line before or between the groups. --group[=METHOD] (separate, prepend, " +
			"append, both) prints every line with the groups delimited by an empty line, and " +
			"cannot be combined with -c, -d, -D or -u; -D cannot be combined with -c either. " +
			"The comparison skips the first N fields with -f (a field is a run of blanks then " +
			"non-blanks), then N characters with -s, and compares at most N characters with -w. " +
			"-z makes NUL the line delimiter, and the output always ends each line with it. " +
			"Deviations from GNU: characters for -s and -w are bytes, as in GNU's C locale, and " +
			"-i folds ASCII letters only. The obsolete -N and +N forms of -f and -s are not " +
			"accepted. METHOD arguments may be abbreviated, as in GNU.",
		Permissions: "Needs read permission on INPUT and search permission on the directories " +
			"leading to it. When OUTPUT is given it is created or truncated, which needs write " +
			"permission on it if it exists, or on its directory if it does not. INPUT itself is " +
			"never modified.",
		UseCases: []string{
			"Removing duplicate lines from sorted output.",
			"Counting how often each value occurs with sort and -c, a frequency table.",
			"Listing only the values that appear more than once with -d.",
			"Finding the values that appear exactly once with -u.",
			"Grouping related lines visually with --group.",
			"Comparing lines while ignoring a leading timestamp or ID with -f or -s.",
			"Treating lines as equal when they share a prefix with -w.",
			"Deduplicating NUL-separated file lists safely with -z.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove adjacent duplicate lines:",
				Bash:        "uniqx names.txt",
				PowerShell:  "uniqx names.txt",
			},
			{
				Description: "Write the result to another file:",
				Bash:        "uniqx names.txt unique-names.txt",
				PowerShell:  "uniqx names.txt unique-names.txt",
			},
			{
				Description: "Count the occurrences of each line:",
				Bash:        "uniqx -c sorted.txt",
				PowerShell:  "uniqx -c sorted.txt",
			},
			{
				Description: "Print only the lines that are repeated:",
				Bash:        "uniqx -d sorted.txt",
				PowerShell:  "uniqx -d sorted.txt",
			},
			{
				Description: "Print only the lines that are not repeated:",
				Bash:        "uniqx -u sorted.txt",
				PowerShell:  "uniqx -u sorted.txt",
			},
			{
				Description: "Print every repeated line, the groups separated by an empty line:",
				Bash:        "uniqx --all-repeated=separate sorted.txt",
				PowerShell:  "uniqx --all-repeated=separate sorted.txt",
			},
			{
				Description: "Print all lines with each group of equal lines set apart:",
				Bash:        "uniqx --group sorted.txt",
				PowerShell:  "uniqx --group sorted.txt",
			},
			{
				Description: "Compare without regard to case:",
				Bash:        "uniqx -i words.txt",
				PowerShell:  "uniqx -i words.txt",
			},
			{
				Description: "Ignore the first field (a timestamp, say) when comparing:",
				Bash:        "uniqx -f 1 events.log",
				PowerShell:  "uniqx -f 1 events.log",
			},
			{
				Description: "Ignore the first 4 characters when comparing:",
				Bash:        "uniqx -s 4 ids.txt",
				PowerShell:  "uniqx -s 4 ids.txt",
			},
			{
				Description: "Treat lines that share their first 8 characters as equal:",
				Bash:        "uniqx -w 8 commits.txt",
				PowerShell:  "uniqx -w 8 commits.txt",
			},
			{
				Description: "Deduplicate a NUL-separated list:",
				Bash:        "uniqx -z files.lst",
				PowerShell:  "uniqx -z files.lst",
			},
			{
				Description: "Remove duplicates from a pipeline's output:",
				Bash:        "catx a.txt b.txt | uniqx",
				PowerShell:  "catx a.txt b.txt | uniqx",
			},
			{
				Description: "Count the distinct lines of a file:",
				Bash:        "uniqx sorted.txt | wcx -l",
				PowerShell:  "uniqx sorted.txt | wcx -l",
			},
			{
				Description: "Deduplicate in the portable gnux shell:",
				Bash:        "gnux shell -c 'catx sorted.txt | uniqx -c'",
				PowerShell:  "gnux shell -c 'catx sorted.txt | uniqx -c'",
			},
		},
	}
}

// Register installs the uniqx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("count", "c", false, "prefix lines by the number of occurrences")
	fs.BoolP("repeated", "d", false, "only print duplicate lines, one for each group")
	fs.StringP("all-repeated", "D", "",
		"print all duplicate lines; groups can be delimited with an empty line; `METHOD` is none (the default), prepend or separate")
	fs.Lookup("all-repeated").NoOptDefVal = "none"
	fs.StringP("skip-fields", "f", "", "avoid comparing the first `N` fields")
	fs.String("group", "",
		"show all items, separating groups with an empty line; `METHOD` is separate (the default), prepend, append or both")
	fs.Lookup("group").NoOptDefVal = "separate"
	fs.BoolP("ignore-case", "i", false, "ignore differences in case when comparing")
	fs.StringP("skip-chars", "s", "", "avoid comparing the first `N` characters")
	fs.BoolP("unique", "u", false, "only print unique lines")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
	fs.StringP("check-chars", "w", "", "compare no more than `N` characters in lines")
}

var delimits = []choice[uniq.Delimit]{
	{"none", uniq.DelimitNone},
	{"prepend", uniq.DelimitPrepend},
	{"separate", uniq.DelimitSeparate},
}

var groups = []choice[uniq.Group]{
	{"separate", uniq.GroupSeparate},
	{"prepend", uniq.GroupPrepend},
	{"append", uniq.GroupAppend},
	{"both", uniq.GroupBoth},
}

type choice[T any] struct {
	name  string
	value T
}

// match resolves a METHOD argument as GNU's argmatch does: an exact name, or
// an unambiguous prefix of one.
func match[T any](flag, arg string, choices []choice[T]) (T, error) {
	var (
		found T
		n     int
	)
	for _, c := range choices {
		if c.name == arg {
			return c.value, nil
		}
		if arg != "" && strings.HasPrefix(c.name, arg) {
			found = c.value
			n++
		}
	}
	switch n {
	case 1:
		return found, nil
	case 0:
		return found, core.Usagef("invalid argument '%s' for '--%s'", arg, flag)
	default:
		return found, core.Usagef("ambiguous argument '%s' for '--%s'", arg, flag)
	}
}

// count parses a -f, -s or -w value. Values too large to represent are
// clamped, as GNU does: no line is that long anyway.
func count(fs *pflag.FlagSet, name, what string) (int64, error) {
	text, _ := fs.GetString(name)
	n, err := strconv.ParseUint(text, 10, 63)
	if errors.Is(err, strconv.ErrRange) {
		return math.MaxInt64, nil
	}
	if err != nil {
		return 0, core.Usagef("invalid number of %s: '%s'", what, text)
	}
	return int64(n), nil
}

// buildOptions folds the parsed flags into uniq.Options.
func buildOptions(fs *pflag.FlagSet) (uniq.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := uniq.Options{
		Count:          on("count"),
		Repeated:       on("repeated"),
		IgnoreCase:     on("ignore-case"),
		Unique:         on("unique"),
		ZeroTerminated: on("zero-terminated"),
	}

	var err error
	if fs.Changed("all-repeated") {
		o.AllRepeated = true
		method, _ := fs.GetString("all-repeated")
		if o.Delimit, err = match("all-repeated", method, delimits); err != nil {
			return uniq.Options{}, err
		}
	}
	if fs.Changed("group") {
		method, _ := fs.GetString("group")
		if o.Group, err = match("group", method, groups); err != nil {
			return uniq.Options{}, err
		}
	}
	if fs.Changed("skip-fields") {
		if o.SkipFields, err = count(fs, "skip-fields", "fields to skip"); err != nil {
			return uniq.Options{}, err
		}
	}
	if fs.Changed("skip-chars") {
		if o.SkipChars, err = count(fs, "skip-chars", "bytes to skip"); err != nil {
			return uniq.Options{}, err
		}
	}
	if fs.Changed("check-chars") {
		if o.CheckChars, err = count(fs, "check-chars", "bytes to compare"); err != nil {
			return uniq.Options{}, err
		}
		o.CheckCharsSet = true
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return uniq.New(opts, s).Run(args)
}
