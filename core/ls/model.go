package ls

import (
	"bufio"
	"os"
	"time"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "lsx"

// Format selects how entries are laid out.
type Format int

const (
	// FormatAuto picks columns on a terminal and one entry per line otherwise,
	// as GNU ls does when no format flag is given.
	FormatAuto Format = iota
	// FormatLong is -l: one entry per line with mode, owner, size and time.
	FormatLong
	// FormatSingle is -1: one name per line.
	FormatSingle
	// FormatColumns is -C: columns, sorted down each column.
	FormatColumns
	// FormatAcross is -x: columns, sorted across each row.
	FormatAcross
	// FormatCommas is -m: a comma-separated stream filling each line.
	FormatCommas
)

// SortKey selects the ordering of entries.
type SortKey int

const (
	// SortAuto sorts by name, or by time when -c/-u is given without -l.
	SortAuto SortKey = iota
	// SortName sorts by name, byte-wise (the C locale).
	SortName
	// SortNone is -U: directory order.
	SortNone
	// SortTime is -t: newest first.
	SortTime
	// SortSize is -S: largest first.
	SortSize
	// SortExtension is -X: by the text after the last dot.
	SortExtension
	// SortVersion is -v: natural ordering of embedded numbers.
	SortVersion
	// SortWidth is --sort=width: shortest displayed name first.
	SortWidth
)

// TimeKind selects which timestamp is shown and sorted on.
type TimeKind int

const (
	// TimeModified is the modification time (the default).
	TimeModified TimeKind = iota
	// TimeAccessed is -u: the last access time.
	TimeAccessed
	// TimeChanged is -c: the last status change time.
	TimeChanged
	// TimeBirth is --time=birth: the creation time, where the platform has one.
	TimeBirth
)

// Indicator selects the suffix appended to names to show their type.
type Indicator int

const (
	// IndicatorNone appends nothing.
	IndicatorNone Indicator = iota
	// IndicatorSlash is -p: "/" after directories.
	IndicatorSlash
	// IndicatorFileType is --file-type: like classify but without "*".
	IndicatorFileType
	// IndicatorClassify is -F: one of */=@| by file type.
	IndicatorClassify
)

// When is the tri-state taken by --color.
type When int

const (
	// WhenNever disables the feature.
	WhenNever When = iota
	// WhenAuto enables the feature only when stdout is a terminal.
	WhenAuto
	// WhenAlways enables the feature unconditionally.
	WhenAlways
)

// Quoting selects how names are rendered.
type Quoting int

const (
	// QuoteLiteral prints names as they are.
	QuoteLiteral Quoting = iota
	// QuoteEscape is -b: C-style backslash escapes for nongraphic characters.
	QuoteEscape
	// QuoteC is -Q: escapes as with -b, wrapped in double quotes.
	QuoteC
)

// Options mirrors the GNU ls flag surface.
type Options struct {
	All       bool // -a: do not ignore entries starting with "."
	AlmostAll bool // -A: like -a, but without "." and ".."

	Format    Format
	Directory bool // -d: list directories themselves, not their contents
	Recursive bool // -R: list subdirectories recursively

	Sort             SortKey
	Reverse          bool // -r
	GroupDirectories bool // --group-directories-first
	Time             TimeKind
	TimeStyle        string // --time-style; "" means "locale"
	HumanReadable    bool   // -h: sizes as 1K, 234M, 2G (powers of 1024)
	SI               bool   // --si: likewise, but powers of 1000
	ShowBlocks       bool   // -s: print the allocated size of each file
	Inode            bool   // -i: print the index number of each file
	NumericIDs       bool   // -n: numeric user and group IDs
	NoOwner          bool   // -g: long format without the owner column
	NoGroup          bool   // -o / -G: long format without the group column
	Indicator        Indicator
	Dereference      bool // -L: show what symbolic links point to
	DereferenceArgs  bool // -H: follow symbolic links named on the command line
	Color            When
	Quoting          Quoting
	HideControlChars bool     // -q: print ? instead of nongraphic characters
	IgnoreBackups    bool     // -B: skip entries ending in "~"
	Ignore           []string // -I: skip entries matching these shell patterns
	Hide             []string // --hide: likewise, unless -a or -A is given
	Width            int      // -w: line width; 0 detects it, negative means no limit
	TabSize          int      // -T: tab stops for column padding; 0 pads with spaces only
}

// Ls is the engine. Construct it with New.
type Ls struct {
	Options

	// IsTerminal says whether stdout is a terminal. It selects the default
	// format (columns versus one per line), hides control characters and gates
	// --color=auto. It defaults to probing the output stream.
	IsTerminal bool

	// TermWidth returns the terminal width in columns, or 0 when unknown. It is
	// consulted only when IsTerminal is true and -w was not given.
	TermWidth func() int

	// Getenv reads the environment (COLUMNS, LS_COLORS, TERM). It defaults to
	// os.Getenv.
	Getenv func(string) string

	// Now is the clock used to decide whether a timestamp is recent. It
	// defaults to time.Now.
	Now func() time.Time

	// Location is the time zone timestamps are shown in. It defaults to
	// time.Local.
	Location *time.Location

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer

	status    int  // 0 ok, 1 minor problems, 2 serious trouble
	printed   bool // something has been written, so the next header needs a blank line
	usedColor bool // the initial reset sequence has been emitted
	ids       idCache

	// Settings resolved from Options and the environment by prepare.
	format  Format
	sortKey SortKey
	hideCtl bool
	palette *palette // nil when color is off
	style   timeStyle
	width   int // line width; 0 means no limit
}

// New builds a listing engine writing to s.
func New(opts Options, s core.Streams) *Ls {
	l := &Ls{
		Options:    opts,
		IsTerminal: core.IsTerminal(s.Out),
		Getenv:     os.Getenv,
		Now:        time.Now,
		Location:   time.Local,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
	l.TermWidth = func() int {
		if f, ok := s.Out.(*os.File); ok {
			return terminalWidth(f)
		}
		return 0
	}
	return l
}
