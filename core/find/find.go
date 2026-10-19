// Package find implements the engine behind findx, a GNU-compatible
// implementation of the find command: walk directory trees and evaluate an
// expression of tests and actions against every file met.
//
// find's command line is an expression, not a set of options, so the engine
// parses it: Run receives the raw arguments.
package find

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/sysstat"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "findx"

// Options is what a library caller sets directly. Everything else (the
// symbolic link mode, the starting points and the expression) is given to Run
// as arguments, exactly as on the command line, because in find the order of
// the words is the meaning. The zero value is plain find.
type Options struct {
	// Warn turns the warnings on from the start, as a leading -warn does.
	// GNU turns them on by itself when standard input is a terminal; so does
	// findx, and -nowarn in the expression turns them off again.
	Warn bool
}

// symlink modes: which symbolic links are followed.
const (
	followNever = iota // -P
	followArgs         // -H
	followAll          // -L
)

// Find is the engine. Construct it with New.
type Find struct {
	Options

	// Now is the clock the time tests (-mtime, -mmin, -daystart, -ls) are
	// measured against; it is read once per Run. It defaults to time.Now.
	Now func() time.Time
	// Location is the time zone -daystart, -printf, -ls and -newerXt use. It
	// defaults to time.Local.
	Location *time.Location
	// Exec runs the gnux tool called name, if there is one (the shape of
	// cmd.RunTool). -exec and its relatives ask it first, so a command named
	// after a tool runs in-process; any other command, or every command when
	// Exec is nil, is a program found on PATH.
	Exec func(name string, args []string, s core.Streams) (status int, found bool)
	// PromptFn answers the questions -ok and -okdir ask. When nil the
	// question is written to the error stream and a line is read from the
	// input stream.
	PromptFn func(question string) bool
	// IsTerminal reports whether a stream is a terminal; warnings are on by
	// default when the input stream is one. It defaults to core.IsTerminal.
	IsTerminal func(v any) bool

	streams core.Streams
	diag    *core.Diag

	// Per-run state.
	follow     int
	maxDepth   int // -1: no limit
	minDepth   int
	depthFirst bool
	xdev       bool
	ignoreRace bool
	warn       bool
	regexType  string
	now        time.Time
	dayStart   time.Time // GNU's cur_day_start: 24 hours ago, or today's midnight
	root       pred
	stdout     *output
	outputs    map[string]*output
	execs      []*execAction
	stdin      *bufio.Reader
	quit       bool
	failed     bool // a failure that has no diagnostic of its own (-exec ... +)
	buf        []byte
	names      idNames
	ls         lsWidths
}

// New builds a find engine reading and writing s.
func New(opts Options, s core.Streams) *Find {
	return &Find{
		Options:    opts,
		Now:        time.Now,
		Location:   time.Local,
		IsTerminal: core.IsTerminal,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

// output is one destination of the printing actions: standard output, or a
// file named by -fprint, -fprint0, -fprintf or -fls. Actions naming the same
// file share one output, so their lines interleave in order.
type output struct {
	w      *bufio.Writer
	file   *os.File
	name   string
	failed bool
}

// entry is one file met by the walk. Metadata is loaded when a test first
// asks for it, so a walk that only looks at names never calls stat.
type entry struct {
	path  string // as printed: the starting point plus the names below it
	start string // the starting point this file was reached from
	depth int

	de     fs.DirEntry // nil for a starting point
	follow bool        // info describes the target of a symbolic link

	info     os.FileInfo
	infoErr  error
	infoDone bool
	sys      sysstat.Stat
	sysDone  bool

	pruned bool
}

// expressionError reports a mistake in the options or the expression the way
// GNU find does: the message alone and status 1, with no pointer to --help.
func (f *Find) expressionError(err error) error {
	var usage *core.UsageError
	if !errors.As(err, &usage) {
		return err
	}
	f.diag.Errorf("%s", usage.Msg)
	return core.ErrFailed
}

// Run parses args ([-H] [-L] [-P] [starting-point...] [expression]) and walks
// every starting point. The error is core.ErrFailed, after a diagnostic, for a
// command line find cannot parse (GNU find adds no --help hint there) and when
// any file could not be processed.
func (f *Find) Run(args []string) (err error) {
	f.reset()

	rest, err := f.parseLeading(args)
	if err != nil {
		return f.expressionError(err)
	}
	split := 0
	for split < len(rest) && !startsExpression(rest[split]) {
		split++
	}
	starts := rest[:split]
	if len(starts) == 0 {
		starts = []string{"."}
	}

	defer f.closeOutputs()
	if err := f.parseExpression(rest[split:]); err != nil {
		return f.expressionError(err)
	}

	for _, start := range starts {
		if f.quit {
			break
		}
		f.walkStart(start)
	}
	for _, x := range f.execs {
		f.flushBatch(x)
	}
	f.finishOutputs()
	if f.failed {
		return core.ErrFailed
	}
	return f.diag.Result()
}

// reset clears the state one Run leaves behind, so an engine can run twice.
func (f *Find) reset() {
	f.follow = followNever
	f.maxDepth = -1
	f.minDepth = 0
	f.depthFirst = false
	f.xdev = false
	f.ignoreRace = false
	f.regexType = "findutils-default"
	f.quit = false
	f.failed = false
	f.execs = nil
	f.stdin = nil
	f.names = idNames{}
	f.ls = defaultLsWidths
	f.outputs = map[string]*output{}
	f.stdout = &output{w: core.NewWriter(f.streams.Out), name: "standard output"}
	f.warn = f.Warn || (f.IsTerminal != nil && f.IsTerminal(f.streams.In))

	now := time.Now
	if f.Now != nil {
		now = f.Now
	}
	if f.Location == nil {
		f.Location = time.Local
	}
	f.now = now()
	f.dayStart = f.now.Add(-24 * time.Hour)
}

// parseLeading consumes the real options, which come before the starting
// points: -H, -L and -P (the last one given wins), -O LEVEL (accepted and
// ignored: the expression is never reordered) and -D (refused).
func (f *Find) parseLeading(args []string) ([]string, error) {
	for len(args) > 0 {
		arg := args[0]
		switch {
		case arg == "-P":
			f.follow = followNever
		case arg == "-H":
			f.follow = followArgs
		case arg == "-L":
			f.follow = followAll
		case arg == "--":
			return args[1:], nil
		case arg == "-D":
			return nil, core.Usagef("the -D debug options are not supported")
		case strings.HasPrefix(arg, "-O"):
			level := arg[2:]
			if level == "" || strings.Trim(level, "0123456789") != "" {
				return nil, core.Usagef("The -O option must be immediately followed by a decimal integer")
			}
		default:
			return args, nil
		}
		args = args[1:]
	}
	return args, nil
}

// startsExpression reports whether arg ends the starting points: a word that
// begins with '-' (but not "-" alone, which is a file name), or one of the
// operators ( ) , !.
func startsExpression(arg string) bool {
	switch arg {
	case "(", ")", ",", "!":
		return true
	}
	return len(arg) > 1 && arg[0] == '-'
}

// walkStart examines one starting point and the tree below it.
func (f *Find) walkStart(start string) {
	info, err := fsys.Lstat(start)
	if err != nil {
		f.diag.Errorf("'%s': %s", start, errText(err))
		return
	}
	e := &entry{path: start, start: start, info: info, infoDone: true}
	if f.follow != followNever && info.Mode()&fs.ModeSymlink != 0 {
		target, err := fsys.Stat(start)
		switch {
		case err == nil:
			e.info, e.follow = target, true
		case !danglingLink(err):
			// Neither followed nor simply broken: say why, then carry on
			// with the link itself, as for a broken one.
			f.diag.Errorf("'%s': %s", start, errText(err))
		}
	}

	var dev uint64
	if f.xdev {
		dev = e.sysstat().Dev
	}
	f.walk(e, dev, nil)
}

// walk visits e and, when it is a directory, everything below it: before its
// contents by default, after them with -depth. ancestors holds the
// directories being walked above e, for the loop check under -L.
func (f *Find) walk(e *entry, dev uint64, ancestors []*entry) {
	isDir := f.isDir(e)

	if isDir && f.follow == followAll {
		// Only a followed symbolic link can lead back to a directory that is
		// still being walked, so only -L pays for the stat this needs.
		if info, ok := f.stat(e); ok {
			for _, a := range ancestors {
				if a.info != nil && os.SameFile(a.info, info) {
					f.diag.Errorf("File system loop detected; '%s' is part of the same file system loop as '%s'.", e.path, a.path)
					return
				}
			}
		}
	}

	if !f.depthFirst {
		f.visit(e)
		if f.quit {
			return
		}
	}

	if isDir && !e.pruned && (f.maxDepth < 0 || e.depth < f.maxDepth) && f.sameDevice(e, dev) {
		f.descend(e, dev, ancestors)
		if f.quit {
			return
		}
	}

	if f.depthFirst {
		f.visit(e)
	}
}

// vanished reports whether err is a file disappearing during the walk that
// -ignore_readdir_race asked not to hear about.
func (f *Find) vanished(err error) bool {
	return f.ignoreRace && errors.Is(err, fs.ErrNotExist)
}

// sameDevice answers -xdev: a directory on another file system is visited
// but not entered.
func (f *Find) sameDevice(e *entry, dev uint64) bool {
	if !f.xdev {
		return true
	}
	sys, ok := f.sysOf(e)
	return ok && sys.Dev == dev
}

// descend walks the entries of directory e in name order.
func (f *Find) descend(e *entry, dev uint64, ancestors []*entry) {
	// fsys.ReadDir returns the entries sorted by name, and whatever it read
	// before an error.
	entries, err := fsys.ReadDir(e.path)
	if err != nil && !f.vanished(err) {
		f.diag.Errorf("'%s': %s", e.path, errText(err))
	}
	if len(entries) == 0 {
		return
	}
	ancestors = append(ancestors, e)
	for _, de := range entries {
		child := &entry{path: joinPath(e.path, de.Name()), start: e.start, depth: e.depth + 1, de: de}
		if f.follow == followAll && de.Type()&fs.ModeSymlink != 0 {
			info, err := fsys.Stat(child.path)
			switch {
			case err == nil:
				child.info, child.infoDone, child.follow = info, true, true
			case !danglingLink(err):
				f.diag.Errorf("'%s': %s", child.path, errText(err))
			}
		}
		f.walk(child, dev, ancestors)
		if f.quit {
			return
		}
	}
}

// visit evaluates the expression against e.
func (f *Find) visit(e *entry) {
	if e.depth < f.minDepth {
		return
	}
	f.root(e)
}

// isDir reports whether the walk descends into e. The directory entry
// answers for a file below a starting point, so nothing is stat'ed for it.
func (f *Find) isDir(e *entry) bool {
	if e.infoDone {
		return e.info != nil && e.info.IsDir()
	}
	return e.de.IsDir()
}

// typeOf returns the file type bits of e as the walk sees it.
func (f *Find) typeOf(e *entry) fs.FileMode {
	if e.infoDone {
		if e.info == nil {
			return fs.ModeIrregular
		}
		return e.info.Mode().Type()
	}
	return e.de.Type()
}

// stat returns the metadata of e, loading it on first use. A file that
// cannot be examined is diagnosed once and makes the asking test false.
func (f *Find) stat(e *entry) (os.FileInfo, bool) {
	if !e.infoDone {
		e.infoDone = true
		// The directory entry's Info is an lstat of the entry (and free on
		// platforms whose directory listing already carries the metadata).
		e.info, e.infoErr = e.de.Info()
		if e.infoErr != nil {
			e.info = nil
			if !f.vanished(e.infoErr) {
				f.diag.Errorf("'%s': %s", e.path, errText(e.infoErr))
			}
		}
	}
	return e.info, e.info != nil
}

// sysstat returns the metadata os.FileInfo does not carry. It must only be
// called once stat has succeeded.
func (e *entry) sysstat() sysstat.Stat {
	if !e.sysDone {
		e.sysDone = true
		e.sys = sysstat.Of(e.info)
	}
	return e.sys
}

// sysOf is stat followed by sysstat.
func (f *Find) sysOf(e *entry) (sysstat.Stat, bool) {
	if _, ok := f.stat(e); !ok {
		return sysstat.Stat{}, false
	}
	return e.sysstat(), true
}

// refStat examines a file named in the expression (-newer FILE, -samefile
// FILE) the way the walk examines a starting point: through a symbolic link
// under -H and -L, falling back to the link itself when it is broken.
func (f *Find) refStat(path string) (os.FileInfo, error) {
	if f.follow != followNever {
		if info, err := fsys.Stat(path); err == nil {
			return info, nil
		}
	}
	return fsys.Lstat(path)
}

// danglingLink reports whether a failed stat means "a symbolic link whose
// target is missing", which find answers by examining the link itself.
func danglingLink(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// errText is core.ErrText with the symbolic link loop error spelled as GNU
// spells it on every platform.
func errText(err error) string {
	if errors.Is(err, syscall.ELOOP) {
		return "Too many levels of symbolic links"
	}
	return core.ErrText(err)
}

func isSep(c byte) bool { return c == '/' || c == os.PathSeparator }

// joinPath appends name to dir with one separator between them. The
// separator is always '/', which every platform gnux runs on accepts, so
// that the paths printed, and the patterns of -path and -regex that match
// them, are the same everywhere.
func joinPath(dir, name string) string {
	if dir != "" && isSep(dir[len(dir)-1]) {
		return dir + name
	}
	return dir + "/" + name
}

// baseName returns the last element of path with trailing separators
// removed, as GNU's base_name does: "a/b/" is "b", "/" is "/", "." is ".".
func baseName(path string) string {
	end := len(path)
	for end > 1 && isSep(path[end-1]) {
		end--
	}
	i := end
	for i > 0 && !isSep(path[i-1]) {
		i--
	}
	if i == end {
		return path[:end]
	}
	return path[i:end]
}

// dirName returns what %h prints: everything before the last separator, or
// "." when there is none.
func dirName(path string) string {
	end := len(path)
	for end > 1 && isSep(path[end-1]) {
		end--
	}
	i := end
	for i > 0 && !isSep(path[i-1]) {
		i--
	}
	if i == 0 {
		return "."
	}
	for i > 1 && isSep(path[i-1]) {
		i--
	}
	return path[:i]
}

// --- output ---

// outputFor returns the shared output for a file named by an -f* action,
// creating (and truncating) the file the first time it is named.
func (f *Find) outputFor(path string) (*output, error) {
	switch path {
	case "/dev/stdout":
		return f.stdout, nil
	case "/dev/stderr":
		if o, ok := f.outputs[path]; ok {
			return o, nil
		}
		o := &output{w: core.NewWriter(f.streams.Err), name: path}
		f.outputs[path] = o
		return o, nil
	}
	if o, ok := f.outputs[path]; ok {
		return o, nil
	}
	file, err := fsys.Create(path)
	if err != nil {
		return nil, err
	}
	o := &output{w: core.NewWriter(file), file: file, name: path}
	f.outputs[path] = o
	return o, nil
}

// write sends b to o. A reader that has gone away ends the walk quietly; any
// other failure is reported once and ends it too.
func (f *Find) write(o *output, b []byte) {
	if o.failed {
		return
	}
	if _, err := o.w.Write(b); err != nil {
		f.outputFailed(o, err)
	}
}

func (f *Find) outputFailed(o *output, err error) {
	o.failed = true
	if !core.IsClosedPipe(err) {
		f.diag.Errorf("write error: %s", core.ErrText(err))
	}
	if o == f.stdout {
		f.quit = true
	}
}

// flushOutputs writes out everything buffered, so that what a command run by
// -exec prints, and what -ok asks, appear after what find printed before.
func (f *Find) flushOutputs() {
	f.flush(f.stdout)
	for _, o := range f.outputs {
		f.flush(o)
	}
}

func (f *Find) flush(o *output) {
	if o.failed {
		return
	}
	if err := o.w.Flush(); err != nil {
		f.outputFailed(o, err)
	}
}

// finishOutputs flushes every output and closes the files, reporting a
// failed close as a write error.
func (f *Find) finishOutputs() {
	f.flushOutputs()
	for _, o := range f.outputs {
		if o.file == nil {
			continue
		}
		if err := o.file.Close(); err != nil && !o.failed {
			f.diag.Errorf("write error: %s", core.ErrText(err))
		}
		o.file = nil
	}
}

// closeOutputs releases the files when Run returns early, on a command line
// that could not be parsed.
func (f *Find) closeOutputs() {
	for _, o := range f.outputs {
		if o.file != nil {
			_ = o.file.Close()
			o.file = nil
		}
	}
}

// ask puts a question for -ok. The default writes it to the error stream and
// reads one line of the input stream; an answer starting with y or Y is yes.
func (f *Find) ask(question string) bool {
	if f.PromptFn != nil {
		return f.PromptFn(question)
	}
	_, _ = io.WriteString(f.streams.Err, question)
	if f.stdin == nil {
		f.stdin = core.NewReader(f.streams.In)
	}
	line, err := f.stdin.ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	line = strings.TrimSpace(line)
	return line != "" && (line[0] == 'y' || line[0] == 'Y')
}
