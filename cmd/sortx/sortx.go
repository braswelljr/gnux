// Package sortx wires the sortx command (GNU sort) to its engine.
package sortx

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sort"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// usageExit is GNU sort's status for every kind of trouble, a command-line
// mistake included; 1 means "not sorted" under --check.
const usageExit = 2

// Spec describes the sortx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "sort",
		Flags:     Register,
		Run:       run,
		UsageExit: usageExit,
	}
}

// Meta returns the documentation metadata for sortx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    sort.Name,
		SeeAlso: []string{"headx", "tailx", "wcx", "catx"},
		Use:     sort.Name + " [OPTION]... [FILE]...",
		Short:   "Sort lines of text files.",
		Long: "sortx writes the sorted concatenation of all FILEs to standard output. With no " +
			"FILE, or when FILE is -, it reads standard input. By default whole lines are " +
			"compared byte by byte. -n, -g, -h, -M and -V compare as numbers, floating-point " +
			"numbers, sizes with SI suffixes (2K, 1G), month names and version numbers; -b, " +
			"-d, -f and -i ignore leading blanks, keep only blanks and alphanumerics, fold case " +
			"and keep only printable characters; -r reverses the result. -k POS1[,POS2] sorts " +
			"on a key, where each POS is F[.C][OPTS]: field F, character C, and any of the " +
			"ordering letters bdfghiMnrV, which then apply to that key alone. A key with no " +
			"letters inherits the global options. Fields are separated by -t SEP, or else by " +
			"the change from a non-blank to a blank character, so a field keeps its leading " +
			"blanks. When all keys compare equal, the whole lines are compared as a last " +
			"resort, unless -s (stable) or -u (unique, which keeps the first of each run of " +
			"equal lines) is given. -c and -C check whether the input is already sorted, -m " +
			"merges inputs that already are, -o FILE writes the result to FILE (which may be " +
			"one of the inputs), and -z makes NUL the line delimiter. Ordering options that " +
			"contradict each other, such as -n with -g, are refused. Differences from GNU " +
			"sort: comparison always follows the C locale (byte order, ASCII case folding and " +
			"month names, '.' as the decimal point, no thousands separator) whatever LANG and " +
			"LC_ALL say; -R, --random-sort, --random-source and --debug are not implemented; " +
			"-S, -T, --parallel, --batch-size and --compress-program are accepted and ignored, " +
			"because sortx sorts in memory; when -c and -C are both given, the diagnosing -c " +
			"wins whatever their order, so disorder is never hidden.",
		Permissions: "Needs read permission on each FILE and search permission on the " +
			"directories leading to it. With -o, needs write permission on the output file, " +
			"or on its directory when it does not exist yet. Every input is checked to be " +
			"readable before the output file is opened, and the output is only emptied once " +
			"all input has been read, so sorting a file onto itself is safe.",
		UseCases: []string{
			"Putting a list of names, words or paths into a predictable order.",
			"Ordering numbers or sizes correctly with -n or -h instead of alphabetically.",
			"Sorting a CSV-like or colon-separated file on one column with -t and -k.",
			"Removing duplicate lines in one step with -u.",
			"Sorting release tags or file names that contain version numbers with -V.",
			"Finding the largest entries of a listing with -rn and headx.",
			"Merging several already sorted files cheaply with -m.",
			"Verifying in a script that a file is sorted with -c or -C.",
			"Sorting a file in place with -o.",
			"Handling NUL-separated file lists safely with -z.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "success; with -c or -C, the input is sorted"},
			{Code: "1", Meaning: "with -c or -C, the input is not sorted"},
			{Code: "2", Meaning: "trouble: an input could not be read, the output could not be written, or a command-line mistake"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Sort the lines of a file:",
				Bash:        "sortx names.txt",
				PowerShell:  "sortx names.txt",
			},
			{
				Description: "Sort in reverse order:",
				Bash:        "sortx -r names.txt",
				PowerShell:  "sortx -r names.txt",
			},
			{
				Description: "Sort numbers by value rather than as text:",
				Bash:        "sortx -n numbers.txt",
				PowerShell:  "sortx -n numbers.txt",
			},
			{
				Description: "Sort and drop duplicate lines:",
				Bash:        "sortx -u words.txt",
				PowerShell:  "sortx -u words.txt",
			},
			{
				Description: "Sort case-insensitively:",
				Bash:        "sortx -f names.txt",
				PowerShell:  "sortx -f names.txt",
			},
			{
				Description: "Sort a colon-separated file numerically on its third field:",
				Bash:        "sortx -t : -k 3,3n /etc/passwd",
				PowerShell:  "sortx -t : -k 3,3n users.txt",
			},
			{
				Description: "Sort on the second column, then by the first in reverse:",
				Bash:        "sortx -k 2,2 -k 1,1r scores.txt",
				PowerShell:  "sortx -k 2,2 -k 1,1r scores.txt",
			},
			{
				Description: "Sort human-readable sizes:",
				Bash:        "printf '2K\\n1G\\n10M\\n' | sortx -h",
				PowerShell:  "'2K', '1G', '10M' | sortx -h",
			},
			{
				Description: "Sort version numbers naturally:",
				Bash:        "printf 'v1.10\\nv1.2\\nv1.9\\n' | sortx -V",
				PowerShell:  "'v1.10', 'v1.2', 'v1.9' | sortx -V",
			},
			{
				Description: "Print the three largest numbers of a file:",
				Bash:        "sortx -rn numbers.txt | headx -n 3",
				PowerShell:  "sortx -rn numbers.txt | headx -n 3",
			},
			{
				Description: "Sort a file in place:",
				Bash:        "sortx -o names.txt names.txt",
				PowerShell:  "sortx -o names.txt names.txt",
			},
			{
				Description: "Merge files that are already sorted:",
				Bash:        "sortx -m jan.txt feb.txt mar.txt",
				PowerShell:  "sortx -m jan.txt feb.txt mar.txt",
			},
			{
				Description: "Fail a script when a file is not sorted (exit status 1):",
				Bash:        "sortx -C names.txt || echox 'names.txt is not sorted'",
				PowerShell:  "sortx -C names.txt; if ($LASTEXITCODE -ne 0) { 'names.txt is not sorted' }",
			},
			{
				Description: "Report the first line out of order:",
				Bash:        "sortx -c names.txt",
				PowerShell:  "sortx -c names.txt",
			},
			{
				Description: "Sort a NUL-separated list of file names:",
				Bash:        "find . -print0 | sortx -z",
				PowerShell:  "sortx -z list0.bin",
			},
			{
				Description: "Sort a directory listing in the portable gnux shell:",
				Bash:        "gnux shell -c 'lsx -1 | sortx -r'",
				PowerShell:  "gnux shell -c 'lsx -1 | sortx -r'",
			},
		},
	}
}

// Register installs the sortx flags on fs.
func Register(fs *pflag.FlagSet) {
	// Ordering options.
	fs.BoolP("ignore-leading-blanks", "b", false, "ignore leading blanks")
	fs.BoolP("dictionary-order", "d", false, "consider only blanks and alphanumeric characters")
	fs.BoolP("ignore-case", "f", false, "fold lower case to upper case characters")
	fs.BoolP("general-numeric-sort", "g", false, "compare according to general numerical value")
	fs.BoolP("ignore-nonprinting", "i", false, "consider only printable characters")
	fs.BoolP("month-sort", "M", false, "compare (unknown) < 'JAN' < ... < 'DEC'")
	fs.BoolP("human-numeric-sort", "h", false, "compare human readable numbers (e.g., 2K 1G)")
	fs.BoolP("numeric-sort", "n", false, "compare according to string numerical value")
	fs.BoolP("reverse", "r", false, "reverse the result of comparisons")
	fs.String("sort", "",
		"sort according to `WORD`: general-numeric -g, human-numeric -h, month -M, numeric -n, version -V")
	fs.BoolP("version-sort", "V", false, "natural sort of (version) numbers within text")

	// Other options.
	fs.StringP("check", "c", "",
		"check for sorted input; do not sort. `WHEN` (diagnose-first, quiet, silent) says whether to report the first disorder")
	fs.Lookup("check").NoOptDefVal = "diagnose-first"
	fs.BoolP("check-quiet", "C", false, "like -c, but do not report first bad line")
	fs.String("files0-from", "",
		"read input from the files specified by NUL-terminated names in file `F`; if F is - then read names from standard input")
	fs.StringArrayP("key", "k", nil, "sort via a key; `KEYDEF` gives location and type")
	fs.BoolP("merge", "m", false, "merge already sorted files; do not sort")
	fs.StringP("output", "o", "", "write result to `FILE` instead of standard output")
	fs.BoolP("stable", "s", false, "stabilize sort by disabling last-resort comparison")
	fs.StringP("field-separator", "t", "", "use `SEP` instead of non-blank to blank transition")
	fs.BoolP("unique", "u", false, "output only the first of an equal run")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")

	// Tuning options GNU sort has for sorting on disk; sortx sorts in
	// memory and accepts them so existing scripts keep working.
	fs.StringP("buffer-size", "S", "", "accepted for compatibility and ignored (`SIZE` of the main memory buffer)")
	fs.StringArrayP("temporary-directory", "T", nil, "accepted for compatibility and ignored (`DIR` for temporaries)")
	fs.Int("parallel", 0, "accepted for compatibility and ignored (`N` concurrent sorts)")
	fs.Int("batch-size", 0, "accepted for compatibility and ignored (merge at most `NMERGE` inputs at once)")
	fs.String("compress-program", "", "accepted for compatibility and ignored (`PROG` to compress temporaries)")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "check-quiet")
}

// checkModes maps the --check argument to its mode.
var checkModes = map[string]sort.CheckMode{
	"diagnose-first": sort.CheckDiagnose,
	"quiet":          sort.CheckQuiet,
	"silent":         sort.CheckQuiet,
}

// errEmptyTab is GNU's fatal "empty tab" diagnostic for -t ”.
var errEmptyTab = errors.New("empty tab")

// buildOptions folds the parsed flags into sort.Options.
func buildOptions(fs *pflag.FlagSet) (sort.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := sort.Options{
		IgnoreLeadingBlanks: on("ignore-leading-blanks"),
		DictionaryOrder:     on("dictionary-order"),
		IgnoreCase:          on("ignore-case"),
		IgnoreNonprinting:   on("ignore-nonprinting"),
		GeneralNumeric:      on("general-numeric-sort"),
		HumanNumeric:        on("human-numeric-sort"),
		Month:               on("month-sort"),
		Numeric:             on("numeric-sort"),
		Version:             on("version-sort"),
		Reverse:             on("reverse"),
		Merge:               on("merge"),
		Stable:              on("stable"),
		Unique:              on("unique"),
		ZeroTerminated:      on("zero-terminated"),
	}
	if keys, _ := fs.GetStringArray("key"); len(keys) > 0 {
		o.Keys = keys
	}
	o.Output, _ = fs.GetString("output")
	o.Files0From, _ = fs.GetString("files0-from")
	o.FieldSeparator, _ = fs.GetString("field-separator")
	if fs.Changed("field-separator") && o.FieldSeparator == "" {
		return sort.Options{}, errEmptyTab
	}

	if word, _ := fs.GetString("sort"); fs.Changed("sort") {
		switch word {
		case "general-numeric":
			o.GeneralNumeric = true
		case "human-numeric":
			o.HumanNumeric = true
		case "month":
			o.Month = true
		case "numeric":
			o.Numeric = true
		case "version":
			o.Version = true
		default:
			return sort.Options{}, core.Usagef("invalid argument '%s' for '--sort'", word)
		}
	}

	if on("check-quiet") {
		o.Check = sort.CheckQuiet
	}
	// -c after -C (or before it) still diagnoses: disorder is never hidden.
	if fs.Changed("check") {
		when, _ := fs.GetString("check")
		mode, ok := checkModes[when]
		if !ok {
			return sort.Options{}, core.Usagef("invalid argument '%s' for '--check'", when)
		}
		if o.Check == sort.CheckNone || mode == sort.CheckDiagnose {
			o.Check = mode
		}
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if errors.Is(err, errEmptyTab) {
		// Fatal in GNU sort, without the --help hint of a usage error.
		fmt.Fprintf(s.Err, "%s: %s\n", sort.Name, err)
		return &core.ExitError{Code: usageExit}
	}
	if err != nil {
		return err
	}
	return sort.New(opts, s).Run(args)
}
