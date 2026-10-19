// Package lsx wires the lsx command (GNU ls) to its listing engine. It only
// declares flags and folds them into ls.Options; every listing decision lives
// in core/ls.
package lsx

import (
	"slices"
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ls"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// defaultTabSize is the tab stop GNU ls assumes when padding columns.
const defaultTabSize = 8

// Spec describes the lsx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "ls",
		Flags: Register,
		Run:   run,
		// ls reserves 1 for minor problems; a bad command line is "serious".
		UsageExit: 2,
	}
}

// Meta returns the documentation metadata for lsx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    ls.Name,
		SeeAlso: []string{"realpathx", "wcx", "touchx"},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "success"},
			{Code: "1", Meaning: "minor trouble: an entry found while listing could not be examined"},
			{Code: "2", Meaning: "serious trouble: a command-line operand could not be accessed, a directory could not be read, or a command-line mistake"},
		},
		Use:   ls.Name + " [OPTION]... [FILE]...",
		Short: "List information about files and directory contents.",
		Long: "lsx lists each FILE, or the current directory when none is given. Operands that " +
			"are not directories are listed first, then the contents of each directory under a " +
			"'dir:' header. Entries whose names start with '.' are hidden unless -a or -A is " +
			"given. On a terminal the names are arranged in columns sized to the window; when " +
			"the output is piped or redirected there is one name per line. -l selects the long " +
			"format (mode, link count, owner, group, size, timestamp and name), where timestamps " +
			"from the last six months show the time of day and older ones show the year. " +
			"Entries are sorted by name using plain byte comparison - the C locale's collation, " +
			"so uppercase sorts before lowercase regardless of the environment's locale - or by " +
			"time (-t), size (-S), extension (-X) or version (-v), optionally reversed with -r. " +
			"When several flags of one family are given (-l/-1/-C/-x/-m, the sort keys, -a/-A, " +
			"-c/-u/--time, …) the last one on the command line wins, as in GNU ls. The exit " +
			"status is 0 on success, 1 for minor problems such as an entry that cannot be " +
			"examined, and 2 for serious trouble such as an operand that cannot be accessed.",
		Permissions: "Listing a directory needs read permission on it; examining its entries " +
			"(anything beyond bare names: -l, -s, -i, -F, --color, sorting by time or size) also " +
			"needs search permission. Entries that cannot be examined are still listed, with '?' " +
			"in place of the unknown fields. On Windows, which has no POSIX owner, group, inode " +
			"or block count, those columns hold placeholders, and with no POSIX permission bits " +
			"every directory is colored as a plain directory, never as other-writable or sticky.",
		UseCases: []string{
			"Seeing what is in a directory at a glance.",
			"Inspecting permissions, ownership, sizes and timestamps with -l.",
			"Finding the newest or largest files with -lt or -lS.",
			"Walking a tree with -R, or feeding one name per line to a script with -1.",
			"Showing hidden files with -A, or telling file types apart with -F or --color.",
			"Checking that a build or download produced the expected files before the next step runs.",
			"Comparing versioned release directories in natural order with -v.",
			"Auditing which files changed recently by sorting on modification, access or change time.",
			"Producing stable, script-friendly listings (one per line, no colors) that behave the same on every platform.",
			"Spotting oddly named files (spaces, control characters) with -Q, -b or -q.",
		},
		Examples: []docmeta.Example{
			{
				Description: "List the current directory:",
				Bash:        "lsx",
				PowerShell:  "lsx",
			},
			{
				Description: "Long listing including hidden files, with human-readable sizes:",
				Bash:        "lsx -lah",
				PowerShell:  "lsx -lah",
			},
			{
				Description: "Newest files first, in the long format:",
				Bash:        "lsx -lt /var/log",
				PowerShell:  "lsx -lt $env:TEMP",
			},
			{
				Description: "Largest files first, with sizes in powers of 1000:",
				Bash:        "lsx -lS --si downloads/",
				PowerShell:  "lsx -lS --si downloads",
			},
			{
				Description: "Smallest files first (reverse the size sort):",
				Bash:        "lsx -lShr downloads/",
				PowerShell:  "lsx -lShr downloads",
			},
			{
				Description: "Recursively list a tree, marking directories and executables:",
				Bash:        "lsx -RF src/",
				PowerShell:  "lsx -RF src",
			},
			{
				Description: "One name per line, directories first, skipping object files:",
				Bash:        "lsx -1 --group-directories-first -I '*.o' build/",
				PowerShell:  "lsx -1 --group-directories-first -I '*.o' build",
			},
			{
				Description: "List a directory itself rather than its contents, with ISO timestamps:",
				Bash:        "lsx -ld --time-style=long-iso ~/projects",
				PowerShell:  "lsx -ld --time-style=long-iso $HOME\\projects",
			},
			{
				Description: "Sort release directories by version number, so v1.10 comes after v1.9:",
				Bash:        "lsx -v releases/",
				PowerShell:  "lsx -v releases",
			},
			{
				Description: "Group files by extension, as a comma-separated list:",
				Bash:        "lsx -X -m assets/",
				PowerShell:  "lsx -X -m assets",
			},
			{
				Description: "Fill rows across rather than down columns, in at most 60 columns:",
				Bash:        "lsx -x -w 60 /usr/bin",
				PowerShell:  "lsx -x -w 60 C:\\Windows\\System32",
			},
			{
				Description: "Show inode numbers and allocated blocks, with directories marked by a slash:",
				Bash:        "lsx -isp",
				PowerShell:  "lsx -isp",
			},
			{
				Description: "Long format with numeric user and group IDs, or without the owner or group columns:",
				Bash:        "lsx -n data/ && lsx -g data/ && lsx -o data/ && lsx -lG data/",
				PowerShell:  "lsx -n data; lsx -g data; lsx -o data; lsx -lG data",
			},
			{
				Description: "Sort by last access time (-u) or status change time (-c), showing that time:",
				Bash:        "lsx -ltu cache/ && lsx -lt --time=ctime cache/",
				PowerShell:  "lsx -ltu cache; lsx -lt --time=ctime cache",
			},
			{
				Description: "Show complete timestamps, or a custom format:",
				Bash:        "lsx --full-time notes.txt && lsx -l --time-style=+%Y-%m-%d notes.txt",
				PowerShell:  "lsx --full-time notes.txt; lsx -l --time-style=+%Y-%m-%d notes.txt",
			},
			{
				Description: "Hide temporary and backup files, unless -a or -A is also given:",
				Bash:        "lsx -B --hide='*.tmp' work/",
				PowerShell:  "lsx -B --hide='*.tmp' work",
			},
			{
				Description: "Quote names, or escape control characters, to spot odd file names:",
				Bash:        "lsx -Q && lsx -b && lsx -q",
				PowerShell:  "lsx -Q; lsx -b; lsx -q",
			},
			{
				Description: "Follow symbolic links: those named on the command line (-H), or all of them (-L):",
				Bash:        "lsx -lH current && lsx -lL lib/",
				PowerShell:  "lsx -lH current; lsx -lL lib",
			},
			{
				Description: "List in directory order without sorting, or sort with a word instead of a letter:",
				Bash:        "lsx -U && lsx --sort=size -l",
				PowerShell:  "lsx -U; lsx --sort=size -l",
			},
			{
				Description: "Color by file type and mark types (but not executables) when writing to a terminal:",
				Bash:        "lsx --color=auto --file-type -C -T 4",
				PowerShell:  "lsx --color=auto --file-type -C -T 4",
			},
			{
				Description: "Count the entries in a directory, hidden ones included:",
				Bash:        "lsx -A | wcx -l",
				PowerShell:  "lsx -A | wcx -l",
			},
			{
				Description: "Print the name of the most recently modified file:",
				Bash:        "lsx -t | headx -n 1",
				PowerShell:  "lsx -t | headx -n 1",
			},
			{
				Description: "Fail a script when an expected file is missing (exit status 2):",
				Bash:        "lsx dist/app.js > /dev/null || echox 'build output missing'",
				PowerShell:  "lsx dist/app.js > $null; if ($LASTEXITCODE -ne 0) { 'build output missing' }",
			},
			{
				Description: "Run the same listing through the portable gnux shell:",
				Bash:        "gnux shell -c 'lsx -1 *.log | wcx -l'",
				PowerShell:  "gnux shell -c 'lsx -1 *.log | wcx -l'",
			},
		},
	}
}

// Flag families in which the last flag given wins. Each list names the long
// form of every member; see latest.
var (
	hiddenFlags    = []string{"all", "almost-all"}
	formatFlags    = []string{"long", "one-per-line", "columns", "across", "commas", "long-no-owner", "long-no-group", "numeric-uid-gid", "full-time"}
	sortFlags      = []string{"sort-time", "sort-size", "sort-extension", "unsorted", "sort-version", "sort"}
	timeFlags      = []string{"ctime", "atime", "time"}
	sizeFlags      = []string{"human-readable", "si"}
	indicatorFlags = []string{"classify", "slash", "file-type"}
	derefFlags     = []string{"dereference", "dereference-command-line"}
	quotingFlags   = []string{"quote-name", "escape"}
	styleFlags     = []string{"time-style", "full-time"}
)

// Register installs every lsx flag onto fs.
func Register(fs *pflag.FlagSet) {
	// GNU ls resolves conflicting flags by position (the last one wins), which
	// pflag does not expose. Flags that belong to such a family are therefore
	// backed by ordered values sharing this counter.
	seq := new(int)
	flag := func(name, short, usage string) {
		fs.VarPF(&ordered{seq: seq, boolean: true}, name, short, usage).NoOptDefVal = "true"
	}
	word := func(name, usage string) {
		fs.VarPF(&ordered{seq: seq}, name, "", usage)
	}

	flag("all", "a", "do not ignore entries starting with .")
	flag("almost-all", "A", "do not list implied . and ..")
	flag("escape", "b", "print C-style escapes for nongraphic characters")
	fs.BoolP("ignore-backups", "B", false, "do not list implied entries ending with ~")
	flag("ctime", "c", "with -lt: sort by, and show, ctime (time of last change of file status "+
		"information); with -l: show ctime and sort by name; otherwise: sort by ctime, newest first")
	flag("columns", "C", "list entries by columns")
	fs.String("color", "never", "color the output `WHEN`: 'always' (the default when WHEN is omitted), 'auto', or 'never'")
	fs.Lookup("color").NoOptDefVal = "always"
	fs.BoolP("directory", "d", false, "list directories themselves, not their contents")
	flag("classify", "F", "append indicator (one of */=@|) to entries")
	flag("file-type", "", "likewise, except do not append '*'")
	flag("full-time", "", "like -l --time-style=full-iso")
	flag("long-no-owner", "g", "like -l, but do not list owner")
	fs.Bool("group-directories-first", false, "group directories before files; any use of --sort=none (-U) disables grouping")
	fs.BoolP("no-group", "G", false, "in a long listing, don't print group names")
	flag("human-readable", "h", "with -l and -s, print sizes like 1K 234M 2G etc.")
	flag("si", "", "likewise, but use powers of 1000 not 1024")
	flag("dereference-command-line", "H", "follow symbolic links listed on the command line")
	fs.Var(&patterns{}, "hide", "do not list implied entries matching shell `PATTERN` (overridden by -a or -A)")
	fs.BoolP("inode", "i", false, "print the index number of each file")
	fs.VarP(&patterns{}, "ignore", "I", "do not list implied entries matching shell `PATTERN`")
	flag("long", "l", "use a long listing format")
	flag("dereference", "L", "when showing file information for a symbolic link, show information "+
		"for the file the link references rather than for the link itself")
	flag("commas", "m", "fill width with a comma separated list of entries")
	flag("numeric-uid-gid", "n", "like -l, but list numeric user and group IDs")
	flag("long-no-group", "o", "like -l, but do not list group information")
	flag("slash", "p", "append / indicator to directories")
	fs.BoolP("hide-control-chars", "q", false, "print ? instead of nongraphic characters")
	flag("quote-name", "Q", "enclose entry names in double quotes")
	fs.BoolP("reverse", "r", false, "reverse order while sorting")
	fs.BoolP("recursive", "R", false, "list subdirectories recursively")
	fs.BoolP("size", "s", false, "print the allocated size of each file, in blocks")
	flag("sort-size", "S", "sort by file size, largest first")
	word("sort", "sort by `WORD` instead of name: none (-U), size (-S), time (-t), version (-v), extension (-X), name, width")
	word("time", "select which timestamp is shown or sorted on, `WORD`: access time (-u): atime, access, use; "+
		"metadata change time (-c): ctime, status; modified time (default): mtime, modification; "+
		"birth time: birth, creation")
	word("time-style", "time/date format with -l, `TIME_STYLE`: full-iso, long-iso, iso, locale, or +FORMAT")
	flag("sort-time", "t", "sort by time, newest first; see --time")
	fs.StringP("tabsize", "T", "", "assume tab stops at each `COLS` instead of 8")
	flag("atime", "u", "with -lt: sort by, and show, access time; with -l: show access time and "+
		"sort by name; otherwise: sort by access time, newest first")
	flag("unsorted", "U", "do not sort; list entries in directory order")
	flag("sort-version", "v", "natural sort of (version) numbers within text")
	fs.StringP("width", "w", "", "set output width to `COLS`; 0 means no limit")
	flag("across", "x", "list entries by lines instead of by columns")
	flag("sort-extension", "X", "sort alphabetically by entry extension")
	flag("one-per-line", "1", "list one file per line")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "across", "atime", "columns", "commas", "ctime", "long", "long-no-group", "long-no-owner", "one-per-line", "slash", "sort-extension", "sort-size", "sort-time", "sort-version", "unsorted")
}

// run builds the listing options from the parsed flags and runs the engine
// against args.
func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return ls.New(opts, s).Run(args)
}

// RunAs returns the Run of one of ls's variants, which GNU builds from the
// same source as ls: dirx (columns) and vdirx (the long format). The variant
// takes every lsx option; its format applies when no option picks one, and it
// escapes unprintable characters in names (as -b does) unless an option picks
// the quoting. Its diagnostics carry its own name.
func RunAs(name string, format ls.Format) func(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return func(fs *pflag.FlagSet, args []string, s core.Streams) error {
		opts, err := buildOptions(fs)
		if err != nil {
			return err
		}
		if opts.Format == ls.FormatAuto {
			opts.Format = format
		}
		if latest(fs, quotingFlags) == "" {
			opts.Quoting = ls.QuoteEscape
		}
		return ls.NewAs(name, opts, s).Run(args)
	}
}

// buildOptions folds the parsed flags into a single ls.Options, resolving each
// flag family to the member given last. It returns a usage error for a
// malformed option argument.
func buildOptions(fs *pflag.FlagSet) (ls.Options, error) {
	o := ls.Options{TabSize: defaultTabSize}

	o.Directory, _ = fs.GetBool("directory")
	o.Recursive, _ = fs.GetBool("recursive")
	o.Reverse, _ = fs.GetBool("reverse")
	o.ShowBlocks, _ = fs.GetBool("size")
	o.Inode, _ = fs.GetBool("inode")
	o.GroupDirectories, _ = fs.GetBool("group-directories-first")
	o.HideControlChars, _ = fs.GetBool("hide-control-chars")
	o.IgnoreBackups, _ = fs.GetBool("ignore-backups")
	o.Ignore = patternsOf(fs, "ignore")
	o.Hide = patternsOf(fs, "hide")

	switch latest(fs, hiddenFlags) {
	case "all":
		o.All = true
	case "almost-all":
		o.AlmostAll = true
	}

	// Several of the format flags also change what the long format shows, so
	// those effects apply whenever the flag was given, whoever wins the format.
	o.NoOwner = given(fs, "long-no-owner")
	o.NoGroup = given(fs, "long-no-group")
	if g, _ := fs.GetBool("no-group"); g {
		o.NoGroup = true
	}
	o.NumericIDs = given(fs, "numeric-uid-gid")
	for _, name := range inOrder(fs, formatFlags) {
		switch name {
		case "one-per-line":
			// -1 does not cancel an earlier long format; GNU ls prints the
			// long listing either way.
			if o.Format != ls.FormatLong {
				o.Format = ls.FormatSingle
			}
		case "columns":
			o.Format = ls.FormatColumns
		case "across":
			o.Format = ls.FormatAcross
		case "commas":
			o.Format = ls.FormatCommas
		default:
			o.Format = ls.FormatLong
		}
	}

	var err error
	switch name := latest(fs, sortFlags); name {
	case "sort-time":
		o.Sort = ls.SortTime
	case "sort-size":
		o.Sort = ls.SortSize
	case "sort-extension":
		o.Sort = ls.SortExtension
	case "unsorted":
		o.Sort = ls.SortNone
	case "sort-version":
		o.Sort = ls.SortVersion
	case "sort":
		if o.Sort, err = ls.ParseSort(value(fs, name)); err != nil {
			return ls.Options{}, err
		}
	}

	switch name := latest(fs, timeFlags); name {
	case "ctime":
		o.Time = ls.TimeChanged
	case "atime":
		o.Time = ls.TimeAccessed
	case "time":
		if o.Time, err = ls.ParseTime(value(fs, name)); err != nil {
			return ls.Options{}, err
		}
	}

	switch latest(fs, styleFlags) {
	case "full-time":
		o.TimeStyle = "full-iso"
	case "time-style":
		o.TimeStyle = value(fs, "time-style")
		if err := ls.ValidateTimeStyle(o.TimeStyle); err != nil {
			return ls.Options{}, err
		}
	}

	switch latest(fs, sizeFlags) {
	case "human-readable":
		o.HumanReadable = true
	case "si":
		o.SI = true
	}

	switch latest(fs, indicatorFlags) {
	case "classify":
		o.Indicator = ls.IndicatorClassify
	case "slash":
		o.Indicator = ls.IndicatorSlash
	case "file-type":
		o.Indicator = ls.IndicatorFileType
	}

	switch latest(fs, derefFlags) {
	case "dereference":
		o.Dereference = true
	case "dereference-command-line":
		o.DereferenceArgs = true
	}

	switch latest(fs, quotingFlags) {
	case "quote-name":
		o.Quoting = ls.QuoteC
	case "escape":
		o.Quoting = ls.QuoteEscape
	}

	color, _ := fs.GetString("color")
	if o.Color, err = ls.ParseWhen("--color", color); err != nil {
		return ls.Options{}, err
	}

	if w, _ := fs.GetString("width"); w != "" {
		n, err := strconv.Atoi(w)
		if err != nil || n < 0 {
			return ls.Options{}, core.Usagef("invalid line width: '%s'", w)
		}
		// The engine keeps 0 for "detect the width"; GNU's "-w 0" means no limit.
		o.Width = n
		if n == 0 {
			o.Width = -1
		}
	}
	if t, _ := fs.GetString("tabsize"); t != "" {
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 {
			return ls.Options{}, core.Usagef("invalid tab size: '%s'", t)
		}
		o.TabSize = n
	}

	return o, nil
}

// ordered is a flag value that remembers when it was last set, relative to the
// other ordered flags of the same flag set. It backs both switches (boolean)
// and flags taking a word.
type ordered struct {
	seq     *int // shared counter, bumped on every Set
	at      int  // value of the counter when this flag was last set; 0 if never
	val     string
	boolean bool
}

// Set records the value and its position on the command line.
func (o *ordered) Set(s string) error {
	if o.boolean {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		s = strconv.FormatBool(b)
	}
	*o.seq++
	o.at = *o.seq
	o.val = s
	return nil
}

// String returns the current value.
func (o *ordered) String() string {
	if o.boolean && o.val == "" {
		return "false"
	}
	return o.val
}

// Type names the value kind for help output.
// IsBoolFlag marks a switch as taking no argument, as pflag's own booleans
// are marked (lsx --long=x is refused, as in GNU).
func (o *ordered) IsBoolFlag() bool { return o.boolean }

func (o *ordered) Type() string {
	if o.boolean {
		return "bool"
	}
	return "string"
}

// position returns when the named flag was last given, or 0 when it was not
// (an explicit "--flag=false" counts as not given).
func position(fs *pflag.FlagSet, name string) int {
	f := fs.Lookup(name)
	if f == nil {
		return 0
	}
	o, ok := f.Value.(*ordered)
	if !ok || (o.boolean && o.val != "true") {
		return 0
	}
	return o.at
}

// given reports whether the named ordered flag was set.
func given(fs *pflag.FlagSet, name string) bool {
	return position(fs, name) > 0
}

// value returns the argument of the named ordered flag.
func value(fs *pflag.FlagSet, name string) string {
	return fs.Lookup(name).Value.String()
}

// inOrder returns the members of a flag family that were given, earliest first.
func inOrder(fs *pflag.FlagSet, family []string) []string {
	var set []string
	for _, name := range family {
		if given(fs, name) {
			set = append(set, name)
		}
	}
	slices.SortFunc(set, func(a, b string) int { return position(fs, a) - position(fs, b) })
	return set
}

// latest returns the member of a flag family given last, or "" when none was.
func latest(fs *pflag.FlagSet, family []string) string {
	if set := inOrder(fs, family); len(set) > 0 {
		return set[len(set)-1]
	}
	return ""
}

// patterns is a repeatable flag value collecting shell patterns. Unlike
// pflag's string slices it never splits its argument on commas, which are
// ordinary characters in a file name pattern.
type patterns struct {
	list []string
}

// Set appends one pattern.
func (p *patterns) Set(s string) error {
	p.list = append(p.list, s)
	return nil
}

// String returns the most recent pattern, for help output.
func (p *patterns) String() string {
	if len(p.list) == 0 {
		return ""
	}
	return p.list[len(p.list)-1]
}

// Type names the value kind for help output.
func (p *patterns) Type() string {
	return "string"
}

// patternsOf returns every pattern given to the named flag.
func patternsOf(fs *pflag.FlagSet, name string) []string {
	if p, ok := fs.Lookup(name).Value.(*patterns); ok {
		return p.list
	}
	return nil
}
