// Package grep implements the engine behind grepx, a GNU-compatible
// implementation of the grep command: print the lines that match patterns.
package grep

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/visual"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "grepx"

// Syntax is the language the patterns are written in.
type Syntax int

const (
	// SyntaxBasic is POSIX basic regular expressions (-G), the default.
	SyntaxBasic Syntax = iota
	// SyntaxExtended is POSIX extended regular expressions (-E).
	SyntaxExtended
	// SyntaxFixed takes the patterns as fixed strings (-F).
	SyntaxFixed
)

// Color says when output is colored (--color=WHEN).
type Color int

const (
	// ColorNever writes plain text, the default.
	ColorNever Color = iota
	// ColorAlways always colors.
	ColorAlways
	// ColorAuto colors when standard output is a terminal and TERM is set
	// (which Windows does not require) and is not "dumb".
	ColorAuto
)

// Filename says whether output lines are prefixed with the file name.
type Filename int

const (
	// FilenameDefault prefixes when more than one file is searched.
	FilenameDefault Filename = iota
	// FilenameWith always prefixes (-H).
	FilenameWith
	// FilenameWithout never prefixes (-h).
	FilenameWithout
)

// Directories is what to do with an operand that is a directory (-d ACTION).
type Directories int

const (
	// DirectoriesRead reads it as a file, which fails with "Is a directory".
	DirectoriesRead Directories = iota
	// DirectoriesSkip skips it silently.
	DirectoriesSkip
	// DirectoriesRecurse searches the files under it (-r).
	DirectoriesRecurse
)

// Devices is what to do with devices, FIFOs and sockets (-D ACTION).
type Devices int

const (
	// DevicesDefault reads the ones named as operands and skips the ones met
	// while descending a directory.
	DevicesDefault Devices = iota
	// DevicesRead reads them all.
	DevicesRead
	// DevicesSkip skips them all.
	DevicesSkip
)

// BinaryFiles is how a file holding a NUL byte is treated (--binary-files).
type BinaryFiles int

const (
	// BinaryFilesBinary reports that such a file matches instead of printing
	// its lines.
	BinaryFilesBinary BinaryFiles = iota
	// BinaryFilesText treats it as text (-a).
	BinaryFilesText
	// BinaryFilesWithoutMatch takes it not to match (-I).
	BinaryFilesWithoutMatch
)

// GlobKind says what a FileGlob does.
type GlobKind int

const (
	// GlobExclude skips the files whose name matches (--exclude=GLOB).
	GlobExclude GlobKind = iota
	// GlobInclude searches only the files whose name matches (--include=GLOB).
	GlobInclude
	// GlobExcludeFrom skips the files matching any glob listed, one to a
	// line, in the file Pattern names (--exclude-from=FILE).
	GlobExcludeFrom
)

// FileGlob is one file-selection option. They are kept in one ordered list
// because GNU lets the last matching one decide.
type FileGlob struct {
	Kind    GlobKind
	Pattern string
}

// Options mirrors the GNU grep flag surface. The zero value is plain grep:
// basic regular expressions, matching lines printed, binary files reported
// rather than printed, directories not searched.
type Options struct {
	// Syntax is the pattern language (-G, -E, -F).
	Syntax Syntax
	// Patterns are the -e arguments. Each may hold several patterns,
	// separated by newlines. When Patterns and PatternFiles are both empty
	// the first operand of Run is the pattern list.
	Patterns []string
	// PatternFiles are the -f arguments: files of patterns, one to a line.
	// "-" is standard input. An empty file contributes no pattern.
	PatternFiles []string
	// IgnoreCase (-i) ignores case distinctions in patterns and data.
	IgnoreCase bool
	// Invert (-v) selects the lines that do not match.
	Invert bool
	// WordRegexp (-w) only accepts matches that form whole words.
	WordRegexp bool
	// LineRegexp (-x) only accepts matches of a whole line.
	LineRegexp bool

	// Count (-c) prints a count of selected lines per file instead of them.
	Count bool
	// FilesWithMatches (-l) prints only the names of files with a selected
	// line. It wins over FilesWithoutMatch, and both win over Count.
	FilesWithMatches bool
	// FilesWithoutMatch (-L) prints only the names of files with none.
	FilesWithoutMatch bool
	// MaxCount (-m NUM) stops reading a file after NUM selected lines. It
	// applies only when MaxCountSet is true, so the zero value has no limit.
	MaxCount int64
	// MaxCountSet records that -m was given.
	MaxCountSet bool
	// OnlyMatching (-o) prints only the matched parts of a line, one to a
	// line.
	OnlyMatching bool
	// Quiet (-q) writes nothing and stops at the first selected line.
	Quiet bool
	// NoMessages (-s) suppresses the diagnostics about files that do not
	// exist or cannot be read.
	NoMessages bool
	// Color is --color=WHEN.
	Color Color

	// ByteOffset (-b) prefixes each line with its offset in the file.
	ByteOffset bool
	// Filename is -H or -h.
	Filename Filename
	// Label (--label) is the name shown for standard input, "(standard
	// input)" when empty.
	Label string
	// LineNumber (-n) prefixes each line with its number in the file.
	LineNumber bool
	// InitialTab (-T) puts a tab between the prefix and the line.
	InitialTab bool
	// Null (-Z) follows each file name with a NUL instead of the usual
	// separator.
	Null bool

	// After (-A NUM) prints NUM lines of context after each selected line.
	After int
	// Before (-B NUM) prints NUM lines of context before each selected line.
	Before int
	// Context records that a context option was given, even with a count of
	// zero: groups of lines are then set apart by the group separator.
	Context bool
	// GroupSeparator (--group-separator=SEP) replaces "--" between groups. It
	// applies only when GroupSeparatorSet is true, so that an empty separator
	// can be asked for.
	GroupSeparator string
	// GroupSeparatorSet records that --group-separator was given.
	GroupSeparatorSet bool
	// NoGroupSeparator (--no-group-separator) prints no separator at all.
	NoGroupSeparator bool

	// Directories is -d ACTION; -r and -R are DirectoriesRecurse.
	Directories Directories
	// Dereference (-R) follows every symbolic link while descending. Without
	// it only the links named as operands are followed.
	Dereference bool
	// Devices is -D ACTION.
	Devices Devices
	// FileGlobs are the --include, --exclude and --exclude-from options, in
	// the order they were given.
	FileGlobs []FileGlob
	// ExcludeDir are the --exclude-dir globs.
	ExcludeDir []string
	// BinaryFiles is --binary-files=TYPE, -a or -I.
	BinaryFiles BinaryFiles

	// NullData (-z) makes NUL, not newline, the line terminator of the input
	// and the output.
	NullData bool
	// LineBuffered (--line-buffered) flushes the output after every line.
	LineBuffered bool

	// Pretty is gnux's terminal view, not a GNU option: when lines carry a
	// file name, the name heads each file's lines once instead (as ripgrep
	// prints them), with a blank line between files, in the color of the
	// Theme's file heading when color is on (Theme is a built-in theme; ""
	// takes GNUX_THEME's). It changes nothing with -c, -l, -L, -q, -Z or -z.
	Pretty bool
	Theme  string
}

// Grep is the engine. Construct it with New.
type Grep struct {
	Options

	// Getenv is how the engine reads the environment (TERM and GREP_COLORS,
	// for --color); it defaults to os.Getenv.
	Getenv func(string) string
	// IsTerminal reports whether standard output is a terminal, which
	// --color=auto and the flushing of each line depend on. New sets it from
	// the stream it is given.
	IsTerminal bool

	streams core.Streams
	diag    *core.Diag
	out     *bufio.Writer
	in      *bufio.Reader

	m          matcher
	pal        palette
	delim      byte
	separator  []byte // the group separator line; nil when there is none
	globs      []fileGlob
	dirGlobs   []fileGlob
	limit      int64 // -m; negative when there is none
	quiet      bool  // no line is printed: -q, -l, -L or -c
	stopAtHit  bool  // one selected line settles a file: -q, -l or -L
	list       int   // +1 lists matching files, -1 the others, 0 neither
	counting   bool
	flushLines bool

	withName bool // the current operand's lines carry the file name
	heading  bool // Pretty applies: names head their lines
	headed   bool // the current file's heading has been written
	theme    *visual.Theme
	used     bool // a group has been printed, so the next needs a separator
	matched  bool // a line was selected somewhere
	failed   bool // an error occurred
	stop     bool // nothing more to do: -q found its line, or output is gone
	werr     error

	buf      []byte // the output line being assembled
	long     []byte // a line longer than the read buffer
	spanBuf  []span
	ring     []contextLine // -B: the lines not printed yet
	ringHead int
	ringLen  int
}

// contextLine is a line kept in case it becomes leading context.
type contextLine struct {
	data   []byte
	offset int64
}

// errIsDirectory is what reading a directory fails with; it is named here so
// that the text is the same on every platform.
var errIsDirectory = errors.New("is a directory")

// New builds a grep engine reading from and writing to s.
func New(opts Options, s core.Streams) *Grep {
	return &Grep{
		Options:    opts,
		Getenv:     os.Getenv,
		IsTerminal: core.IsTerminal(s.Out),
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// Run searches the operands. Unless the options supplied patterns (Patterns
// or PatternFiles), the first operand is the pattern list; the rest are the
// files, "-" being standard input. With no file, standard input is searched,
// or the working directory when searching recursively.
//
// The returned error is nil when a line was selected, a *core.ExitError with
// code 1 when none was, and one with code 2 when an error occurred (unless -q
// found a line). A missing pattern is a *core.UsageError.
func (g *Grep) Run(args []string) error {
	patterns, files, err := g.patternList(args)
	if err != nil {
		return err
	}
	if g.m, err = g.compile(patterns); err != nil {
		g.diag.Errorf("%s", err)
		return &core.ExitError{Code: 2}
	}
	if err := g.setup(); err != nil {
		return err
	}
	if g.limit == 0 && g.list >= 0 {
		// Nothing can be selected; GNU does not even open the files.
		return &core.ExitError{Code: 1}
	}

	g.out = core.NewWriter(g.streams.Out)
	g.in = core.NewReader(strings.NewReader(""))

	switch {
	case len(files) > 0:
		for _, name := range files {
			if g.stop {
				break
			}
			g.operand(name, len(files) > 1)
		}
	case g.Directories == DirectoriesRecurse:
		g.tree()
	default:
		g.operand("-", false)
	}

	g.flush()
	if g.werr != nil && !core.IsClosedPipe(g.werr) {
		g.diag.Errorf("write error: %s", core.ErrText(g.werr))
		g.failed = true
	}
	switch {
	case g.Quiet && g.matched:
		return nil
	case g.failed:
		return &core.ExitError{Code: 2}
	case g.matched:
		return nil
	}
	return &core.ExitError{Code: 1}
}

// patternList gathers the patterns from -e, -f or the first operand, and
// returns the operands that are left.
func (g *Grep) patternList(args []string) (patterns, files []string, err error) {
	if len(g.Patterns) == 0 && len(g.PatternFiles) == 0 {
		if len(args) == 0 {
			return nil, nil, core.Usagef("missing PATTERNS operand")
		}
		return strings.Split(args[0], "\n"), args[1:], nil
	}
	for _, p := range g.Patterns {
		patterns = append(patterns, strings.Split(p, "\n")...)
	}
	for _, name := range g.PatternFiles {
		data, err := g.readAll(name)
		if err != nil {
			// GNU reports this one even under -s: without its patterns the
			// run means nothing.
			g.diag.Errorf("%s: %s", name, core.ErrText(err))
			return nil, nil, &core.ExitError{Code: 2}
		}
		text := strings.TrimSuffix(string(data), "\n")
		if len(data) > 0 {
			patterns = append(patterns, strings.Split(text, "\n")...)
		}
	}
	return patterns, args, nil
}

// readAll reads a whole pattern or glob file; "-" is standard input.
func (g *Grep) readAll(name string) ([]byte, error) {
	if name == "-" {
		if g.streams.In == nil {
			return nil, nil
		}
		return io.ReadAll(g.streams.In)
	}
	return fsys.ReadFile(name)
}

// setup derives the run's settings from the options.
func (g *Grep) setup() error {
	g.delim = '\n'
	if g.NullData {
		g.delim = 0
	}

	// -q overrides -l and -L, which override -c, as in GNU grep.
	switch {
	case g.Quiet:
	case g.FilesWithMatches:
		g.list = 1
	case g.FilesWithoutMatch:
		g.list = -1
	default:
		g.counting = g.Count
	}
	g.stopAtHit = g.Quiet || g.list != 0
	g.quiet = g.stopAtHit || g.counting

	g.limit = -1
	if g.MaxCountSet && g.MaxCount >= 0 {
		g.limit = g.MaxCount
	}
	if g.After < 0 {
		g.After = 0
	}
	if g.Before < 0 {
		g.Before = 0
	}

	if (g.Context || g.After > 0 || g.Before > 0) && !g.NoGroupSeparator {
		sep := "--"
		if g.GroupSeparatorSet {
			sep = g.GroupSeparator
		}
		g.separator = []byte(sep)
	}

	g.pal = defaultPalette()
	switch g.Color {
	case ColorAlways:
		g.pal.on = true
	case ColorAuto:
		term := g.Getenv("TERM")
		g.pal.on = g.IsTerminal && term != "dumb" && (term != "" || !needTerm)
	}
	if g.pal.on {
		g.pal.parse(g.Getenv("GREP_COLORS"))
	}
	g.heading = g.Pretty && !g.quiet && !g.Null && !g.NullData
	g.theme = visual.Use(g.heading && g.pal.on, g.Theme, g.Getenv, g.diag.Warnf)
	if g.separator != nil {
		line := g.pal.paint(nil, g.pal.se, string(g.separator))
		line = append(line, '\n')
		g.separator = line
	}

	// A terminal is shown each line as it is found, as stdio does for GNU.
	g.flushLines = g.LineBuffered || g.IsTerminal

	for _, fg := range g.FileGlobs {
		switch fg.Kind {
		case GlobInclude:
			g.globs = append(g.globs, fileGlob{compileGlob(fg.Pattern), true})
		case GlobExclude:
			g.globs = append(g.globs, fileGlob{compileGlob(fg.Pattern), false})
		case GlobExcludeFrom:
			data, err := g.readAll(fg.Pattern)
			if err != nil {
				g.diag.Errorf("%s: %s", fg.Pattern, core.ErrText(err))
				return &core.ExitError{Code: 2}
			}
			for _, line := range strings.Split(string(data), "\n") {
				if line = strings.TrimSuffix(line, "\r"); line != "" {
					g.globs = append(g.globs, fileGlob{compileGlob(line), false})
				}
			}
		}
	}
	for _, glob := range g.ExcludeDir {
		g.dirGlobs = append(g.dirGlobs, fileGlob{compileGlob(trimSeps(glob)), false})
	}
	return nil
}

// operand searches one file named on the command line.
func (g *Grep) operand(name string, several bool) {
	if name == "-" {
		g.withName = g.showName(false, several)
		label := g.Label
		if label == "" {
			label = "(standard input)"
		}
		in := g.streams.In
		if in == nil {
			in = strings.NewReader("")
		}
		g.search(in, label)
		return
	}

	// Stat, not Lstat: a symbolic link named as an operand is followed.
	info, err := fsys.Stat(name)
	if err != nil {
		g.fileError(name, err)
		return
	}
	if info.IsDir() {
		switch {
		case g.Directories == DirectoriesSkip:
		case excluded(g.dirGlobs, trimSeps(name), true):
		case g.Directories == DirectoriesRecurse:
			g.withName = g.showName(true, several)
			prefix := name
			if !isSep(prefix[len(prefix)-1]) {
				prefix += string(os.PathSeparator)
			}
			g.walk(name, prefix, []os.FileInfo{info})
		default:
			g.fileError(name, errIsDirectory)
		}
		return
	}
	if g.Devices == DevicesSkip && isDevice(info.Mode()) {
		return
	}
	if excluded(g.globs, name, true) {
		return
	}
	g.withName = g.showName(false, several)
	g.searchFile(name)
}

// tree searches the working directory, which is what -r means with no file.
// Names are printed without a leading "./".
func (g *Grep) tree() {
	info, err := fsys.Stat(".")
	if err != nil {
		g.fileError(".", err)
		return
	}
	g.withName = g.showName(true, false)
	g.walk(".", "", []os.FileInfo{info})
}

// showName reports whether the lines of an operand carry its name: always
// with -H, never with -h, otherwise when there are several operands or the
// operand is a directory being searched.
func (g *Grep) showName(directory, several bool) bool {
	switch g.Filename {
	case FilenameWith:
		return true
	case FilenameWithout:
		return false
	}
	return several || (directory && g.Directories == DirectoriesRecurse)
}

// walk searches the files under dir, in name order so that the output does
// not depend on the filesystem. prefix is what a child's name is appended to;
// ancestors are the directories being descended, for loop detection.
func (g *Grep) walk(dir, prefix string, ancestors []os.FileInfo) {
	entries, err := fsys.ReadDir(dir) // sorted by name
	if err != nil {
		g.fileError(dir, err)
		return
	}
	for _, entry := range entries {
		if g.stop {
			return
		}
		name := entry.Name()
		path := prefix + name
		mode := entry.Type()

		var info os.FileInfo
		if mode&os.ModeSymlink != 0 {
			if !g.Dereference {
				continue
			}
			if info, err = fsys.Stat(path); err != nil {
				g.fileError(path, err)
				continue
			}
			mode = info.Mode()
		}

		switch {
		case mode.IsDir():
			if excluded(g.dirGlobs, name, false) {
				continue
			}
			below := ancestors
			if g.Dereference {
				// Only a followed link can lead back up the tree.
				if info == nil {
					if info, err = entry.Info(); err != nil {
						g.fileError(path, err)
						continue
					}
				}
				if isAncestor(ancestors, info) {
					g.warnf("%s: warning: recursive directory loop", path)
					continue
				}
				below = make([]os.FileInfo, 0, len(ancestors)+1)
				below = append(below, ancestors...)
				below = append(below, info)
			}
			g.walk(path, path+string(os.PathSeparator), below)
		case isDevice(mode) && g.Devices != DevicesRead:
		case excluded(g.globs, name, false):
		default:
			g.searchFile(path)
		}
	}
}

func isAncestor(ancestors []os.FileInfo, info os.FileInfo) bool {
	for _, a := range ancestors {
		if os.SameFile(a, info) {
			return true
		}
	}
	return false
}

// isDevice reports whether mode is what -D calls a device: a character or
// block device, a FIFO or a socket.
func isDevice(mode os.FileMode) bool {
	return mode&(os.ModeDevice|os.ModeCharDevice|os.ModeNamedPipe|os.ModeSocket) != 0
}

// searchFile opens and searches one file.
func (g *Grep) searchFile(name string) {
	f, err := fsys.Open(name)
	if err != nil {
		g.fileError(name, err)
		return
	}
	defer f.Close()
	g.search(f, name)
}

// fileError reports a file that could not be searched. -s silences the
// message, never the exit status.
func (g *Grep) fileError(name string, err error) {
	g.failed = true
	if g.NoMessages {
		return
	}
	g.flush()
	g.diag.Errorf("%s: %s", name, core.ErrText(err))
}

// warnf writes a diagnostic that does not change the exit status. What is
// already buffered for standard output goes first, so that the two streams
// read in order on a terminal.
func (g *Grep) warnf(format string, a ...any) {
	g.flush()
	g.diag.Warnf(format, a...)
}

// write sends assembled output to the buffered writer. A failure is kept for
// the end of the run and stops the search: nobody is reading.
func (g *Grep) write(b []byte) {
	if g.werr != nil {
		return
	}
	if _, err := g.out.Write(b); err != nil {
		g.werr = err
		g.stop = true
	}
}

func (g *Grep) flush() {
	if g.out == nil || g.werr != nil {
		return
	}
	if err := g.out.Flush(); err != nil {
		g.werr = err
		g.stop = true
	}
}
