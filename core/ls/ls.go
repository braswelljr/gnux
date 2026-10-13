// Package ls implements the engine behind lsx, a GNU-compatible implementation of the
// GNU/UNIX ls command: list information about files and the contents of
// directories. It mirrors ls's flags, layouts, diagnostics and exit statuses.
package ls

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
)

// Exit-status severities, as documented by GNU ls.
const (
	minor   = 1 // e.g. cannot access an entry found while listing a directory
	serious = 2 // e.g. cannot access a command-line argument
)

// entry is one name to be listed, with everything known about it.
type entry struct {
	name string // as displayed: the operand as typed, or the base name in a directory
	path string // as accessed

	info os.FileInfo // nil when the entry could not be examined
	mode os.FileMode // type bits survive a failed stat when the directory supplied them
	sys  sysInfo

	link   string      // symlink target text (long format only)
	target os.FileInfo // what a symlink resolves to; nil when dangling or not needed

	shown string // name after quoting
	width int    // display width of shown
}

// isDir reports whether the entry is a directory.
func (e *entry) isDir() bool {
	return e.info != nil && e.info.IsDir()
}

// isDirLike also counts symbolic links that resolve to a directory, which is
// what --group-directories-first groups on.
func (e *entry) isDirLike() bool {
	return e.isDir() || (e.target != nil && e.target.IsDir())
}

// Run lists each operand, or the current directory when there are none. It
// returns nil, core.ErrFailed for minor problems, or a *core.ExitError with
// status 2 for serious trouble; diagnostics have already been written.
func (l *Ls) Run(args []string) error {
	if err := l.prepare(); err != nil {
		return err
	}
	l.out = core.NewWriter(l.streams.Out)
	defer func() { _ = l.out.Flush() }()

	if len(args) == 0 {
		args = []string{"."}
	}

	operands := make([]*entry, 0, len(args))
	for _, arg := range args {
		if e := l.statOperand(arg); e != nil {
			operands = append(operands, e)
		}
	}
	l.sortEntries(operands)

	var files, dirs []*entry
	for _, e := range operands {
		if e.isDir() && !l.Directory {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}

	if len(files) > 0 {
		l.printEntries(files)
		l.printed = true
	}

	// A lone directory operand is listed without its "dir:" header.
	header := l.Recursive || len(args) > 1 || len(files) > 0
	for _, d := range dirs {
		l.listDir(d.path, d.info, header, serious, nil)
	}

	switch l.status {
	case 0:
		return nil
	case minor:
		return core.ErrFailed
	default:
		return &core.ExitError{Code: l.status}
	}
}

// prepare resolves the settings that depend on the environment: the default
// format, the line width, color and the time style.
func (l *Ls) prepare() error {
	l.format = l.Format
	if l.format == FormatAuto {
		l.format = FormatSingle
		if l.IsTerminal {
			l.format = FormatColumns
		}
	}

	l.sortKey = l.Sort
	if l.sortKey == SortAuto {
		l.sortKey = SortName
		// Without -l, asking for another timestamp means "sort by it".
		if l.format != FormatLong && l.Time != TimeModified {
			l.sortKey = SortTime
		}
	}

	l.hideCtl = l.HideControlChars || (l.IsTerminal && l.Quoting == QuoteLiteral)

	// As in GNU ls, TIME_STYLE supplies the style when --time-style does not.
	name := l.TimeStyle
	if name == "" {
		name = l.Getenv("TIME_STYLE")
	}
	style, err := newTimeStyle(name)
	if err != nil {
		return err
	}
	l.style = style

	l.width = l.lineWidth()

	l.palette = nil
	if l.Color == WhenAlways || (l.Color == WhenAuto && l.IsTerminal && l.Getenv("TERM") != "dumb") {
		l.palette = newPalette(l.Getenv("LS_COLORS"))
	}
	return nil
}

// lineWidth picks the output width: -w, then the terminal, then COLUMNS, then 80.
func (l *Ls) lineWidth() int {
	switch {
	case l.Width < 0:
		return 0
	case l.Width > 0:
		return l.Width
	}
	if l.IsTerminal && l.TermWidth != nil {
		if w := l.TermWidth(); w > 0 {
			return w
		}
	}
	if w, err := strconv.Atoi(strings.TrimSpace(l.Getenv("COLUMNS"))); err == nil && w >= 0 {
		return w
	}
	return 80
}

// fail writes a diagnostic and raises the exit status to at least severity.
func (l *Ls) fail(severity int, format string, a ...any) {
	// Keep stdout and stderr in the order the user would see from GNU ls on a
	// terminal: everything listed so far precedes the complaint.
	if l.out != nil {
		_ = l.out.Flush()
	}
	l.diag.Errorf(format, a...)
	l.status = max(l.status, severity)
}

// statOperand examines a command-line argument, reporting failure as serious
// trouble. Symbolic links are followed with -L and -H, and - unless a flag asks
// about the link itself (-l, -d, -F) - when they point at a directory.
func (l *Ls) statOperand(arg string) *entry {
	var (
		info os.FileInfo
		err  error
	)
	switch {
	case l.Dereference || l.DereferenceArgs:
		info, err = os.Stat(arg)
	case l.Directory || l.format == FormatLong || l.Indicator == IndicatorClassify:
		info, err = os.Lstat(arg)
	default:
		info, err = os.Stat(arg)
		if err != nil || !info.IsDir() {
			info, err = os.Lstat(arg)
		}
	}
	if err != nil {
		l.fail(serious, "cannot access %s: %s", quoteArg(arg), core.ErrText(err))
		return nil
	}

	e := l.named(arg, arg)
	l.fill(e, info)
	return e
}

// named starts an entry, quoting its name once for every later use.
func (l *Ls) named(name, path string) *entry {
	e := &entry{name: name, path: path}
	e.shown = l.quoteName(name)
	e.width = displayWidth(e.shown)
	return e
}

// put writes s to stdout. Write errors are ignored: there is nowhere useful to
// report a broken stdout.
func (l *Ls) put(s string) {
	_, _ = l.out.WriteString(s)
}

// newEntry examines one name found in a directory. A failure is a minor
// problem: the entry is still listed, with "?" for everything unknown.
func (l *Ls) newEntry(dir, name string, de os.DirEntry) *entry {
	e := l.named(name, joinPath(dir, name))

	var (
		info os.FileInfo
		err  error
	)
	switch {
	case l.Dereference:
		info, err = os.Stat(e.path)
	case de != nil:
		info, err = de.Info()
	default:
		info, err = os.Lstat(e.path)
	}
	if err != nil {
		l.fail(minor, "cannot access %s: %s", quoteArg(e.path), core.ErrText(err))
		if de != nil {
			e.mode = de.Type()
		}
		return e
	}
	l.fill(e, info)
	return e
}

// fill records info on e and, for a symbolic link, resolves what the chosen
// options need to know about its target.
func (l *Ls) fill(e *entry, info os.FileInfo) {
	e.info = info
	e.mode = info.Mode()
	e.sys = statSys(info)

	if e.mode&os.ModeSymlink == 0 {
		return
	}
	if l.format == FormatLong {
		e.link, _ = os.Readlink(e.path)
	}
	if l.format == FormatLong || l.palette != nil || l.GroupDirectories {
		if target, err := os.Stat(e.path); err == nil {
			e.target = target
		}
	}
}

// listDir prints the contents of dir, then recurses with -R. severity is how
// bad a failure to open it is: serious for an operand, minor for a directory
// reached by recursion.
func (l *Ls) listDir(dir string, info os.FileInfo, header bool, severity int, ancestors []os.FileInfo) {
	// Only following links can lead back into a directory already being listed.
	if l.Dereference {
		for _, a := range ancestors {
			if os.SameFile(a, info) {
				l.fail(serious, "%s: not listing already-listed directory", quoteArg(dir))
				return
			}
		}
	}

	f, err := os.Open(dir)
	if err != nil {
		l.fail(severity, "cannot open directory %s: %s", quoteArg(dir), core.ErrText(err))
		return
	}
	names, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		// Whatever was read before the error is still listed.
		l.fail(severity, "reading directory %s: %s", quoteArg(dir), core.ErrText(err))
	}

	entries := make([]*entry, 0, len(names)+2)
	if l.All {
		entries = append(entries, l.newEntry(dir, ".", nil), l.newEntry(dir, "..", nil))
	}
	for _, de := range names {
		if l.ignored(de.Name()) {
			continue
		}
		entries = append(entries, l.newEntry(dir, de.Name(), de))
	}
	l.sortEntries(entries)

	if header {
		if l.printed {
			l.put("\n")
		}
		l.put(l.quoteName(dir) + ":\n")
	}
	if l.format == FormatLong || l.ShowBlocks {
		var blocks int64
		for _, e := range entries {
			blocks += e.sys.blocks
		}
		l.put("total " + l.blockString(blocks) + "\n")
	}
	l.printEntries(entries)
	l.printed = true

	if !l.Recursive {
		return
	}
	ancestors = append(ancestors, info)
	for _, e := range entries {
		if e.name == "." || e.name == ".." || !e.isDir() {
			continue
		}
		l.listDir(e.path, e.info, true, minor, ancestors)
	}
}

// ignored reports whether a directory entry is hidden by the dot-file rule,
// -B, -I or --hide.
func (l *Ls) ignored(name string) bool {
	if !l.All && !l.AlmostAll {
		if strings.HasPrefix(name, ".") {
			return true
		}
		for _, p := range l.Hide {
			if matchPattern(p, name) {
				return true
			}
		}
	}
	if l.IgnoreBackups && strings.HasSuffix(name, "~") {
		return true
	}
	for _, p := range l.Ignore {
		if matchPattern(p, name) {
			return true
		}
	}
	return false
}

// matchPattern matches name against a shell pattern the way fnmatch does with
// FNM_PERIOD: a leading dot must be matched literally. A malformed pattern
// matches nothing.
func matchPattern(pattern, name string) bool {
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(pattern, ".") {
		return false
	}
	// path.Match rather than filepath.Match: the pattern syntax (backslash
	// escapes included) must not change with the host OS.
	ok, err := path.Match(shellClasses(pattern), name)
	return err == nil && ok
}

// shellClasses rewrites the shell's "[!...]" negated character class into the
// "[^...]" spelling, the only one path.Match understands.
func shellClasses(pattern string) string {
	if !strings.Contains(pattern, "[!") {
		return pattern
	}
	b := []byte(pattern)
	for i := 0; i < len(b)-1; i++ {
		switch b[i] {
		case '\\':
			i++ // the escaped character is literal
		case '[':
			if b[i+1] == '!' {
				b[i+1] = '^'
			}
			// Skip the class so a "[" inside it is not taken for another one.
			for i++; i < len(b) && b[i] != ']'; i++ {
				if b[i] == '\\' {
					i++
				}
			}
		}
	}
	return string(b)
}

// joinPath appends name to dir with a forward slash, as GNU ls prints it. Every
// supported platform accepts "/" as a separator, so the result is also usable
// for file access.
func joinPath(dir, name string) string {
	if dir == "" || os.IsPathSeparator(dir[len(dir)-1]) {
		return dir + name
	}
	return dir + "/" + name
}

// quoteArg quotes a name for a diagnostic: single quotes, switching to double
// quotes when the name itself contains one.
func quoteArg(s string) string {
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	return "'" + s + "'"
}

// printEntries lays out one group of entries in the selected format.
func (l *Ls) printEntries(entries []*entry) {
	if len(entries) == 0 {
		return
	}
	switch l.format {
	case FormatLong:
		l.printLong(entries)
	case FormatColumns:
		l.printColumns(entries, true)
	case FormatAcross:
		l.printColumns(entries, false)
	case FormatCommas:
		l.printCommas(entries)
	default:
		f := l.newFrills(entries, true)
		for _, e := range entries {
			l.writeCell(e, f)
			l.put("\n")
		}
	}
}
