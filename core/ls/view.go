package ls

import (
	"errors"
	"os"
	"slices"
	"strconv"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/gitstat"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// This file holds gnux's own terminal view: the tree, icons, the themed
// long format, the git column and the filters. None of it runs unless an
// option of gnux's own asks for it, so GNU's output is untouched by default.

// ParseScale resolves a --color-scale argument.
func ParseScale(arg string) (Scale, error) {
	return matchWord("--color-scale", arg, []word[Scale]{
		{"all", ScaleAll},
		{"size", ScaleSize},
		{"age", ScaleAge},
		{"none", ScaleNone},
	})
}

// ThemeNames lists the built-in themes --theme and GNUX_THEME take.
func ThemeNames() []string {
	return visual.ThemeNames()
}

// ParseTheme resolves a --theme argument to a built-in theme's name.
func ParseTheme(arg string) (string, error) {
	names := visual.ThemeNames()
	table := make([]word[string], len(names))
	for i, name := range names {
		table[i] = word[string]{name, name}
	}
	return matchWord("--theme", arg, table)
}

// prepareView settles the view options once color, the format and the
// terminal are known.
func (l *Ls) prepareView() error {
	if l.Tree {
		// A tree is one entry per line, in the long format or not, and its
		// rows carry no dired offsets.
		if l.format != FormatLong {
			l.format = FormatSingle
		}
		l.dired = false
	}
	l.icons = l.Icons == WhenAlways || (l.Icons == WhenAuto && l.IsTerminal)
	l.branches = visual.BranchesFor(l.Getenv)

	l.theme = nil
	if l.palette != nil && (l.Pretty || l.ColorScale != ScaleNone) {
		theme, err := visual.LoadTheme(l.Theme, l.Getenv)
		switch {
		case err != nil && l.Theme != "":
			l.diag.Errorf("invalid theme %s", quoteArg(l.Theme))
			return &core.ExitError{Code: serious}
		case err != nil:
			l.diag.Warnf("ignoring the theme in GNUX_THEME: %v", err)
		}
		l.theme = theme
	}

	l.selfUID = os.Getuid()
	l.selfGIDs = map[uint32]bool{}
	if gid := os.Getgid(); gid >= 0 {
		l.selfGIDs[uint32(gid)] = true
	}
	if gids, err := os.Getgroups(); err == nil {
		for _, g := range gids {
			l.selfGIDs[uint32(g)] = true
		}
	}
	return nil
}

// closeRepos releases the git repositories opened for --git and
// --git-ignore.
func (l *Ls) closeRepos() {
	for _, r := range l.repos {
		r.Close()
	}
	l.repos, l.dirRoots = nil, nil
}

// paint writes s in the theme's color for key; uncolored without a theme.
// The escape sequences are not counted for --dired.
func (l *Ls) paint(key, s string) {
	l.paintCode(l.theme.Code(key), s)
}

func (l *Ls) paintCode(code, s string) {
	if code == "" || s == "" {
		l.put(s)
		return
	}
	if !l.usedColor {
		l.usedColor = true
		l.putRaw("\x1b[0m")
	}
	l.putRaw("\x1b[" + code + "m")
	l.put(s)
	l.putRaw("\x1b[0m")
}

// leadWidth is the width of what comes before a name: its tree branch and
// its icon.
func (l *Ls) leadWidth(e *entry) int {
	w := displayWidth(e.branch)
	if l.icons {
		w += visual.IconWidth
	}
	return w
}

// putLead writes an entry's tree branch and icon. The icon takes the
// color of the name.
func (l *Ls) putLead(e *entry) {
	l.paint(visual.Branch, e.branch)
	if !l.icons {
		return
	}
	code := ""
	if l.palette != nil {
		code = l.palette.entryColor(e)
	}
	l.paintCode(code, visual.Icon(e.name, iconKind(e)))
	l.put(" ")
}

// iconKind tells visual.Icon what an entry is.
func iconKind(e *entry) visual.Kind {
	mode := e.mode
	switch {
	case mode.IsDir():
		return visual.KindDir
	case mode&os.ModeSymlink != 0:
		if e.target != nil && e.target.IsDir() {
			return visual.KindLinkDir
		}
		return visual.KindLink
	case mode&os.ModeNamedPipe != 0:
		return visual.KindPipe
	case mode&os.ModeSocket != 0:
		return visual.KindSocket
	case mode&os.ModeDevice != 0:
		return visual.KindDevice
	case mode.IsRegular() && mode&0o111 != 0:
		return visual.KindExec
	}
	return visual.KindFile
}

// putMode writes the mode column, each letter in its theme color.
func (l *Ls) putMode(e *entry, mode string) {
	if l.theme == nil || len(mode) != 10 {
		l.put(mode)
		return
	}
	typeKey := visual.Punctuation
	switch mode[0] {
	case 'd':
		typeKey = visual.TypeDir
	case 'l':
		typeKey = visual.TypeLink
	case 'p':
		typeKey = visual.TypePipe
	case 's':
		typeKey = visual.TypeSocket
	case 'b', 'c':
		typeKey = visual.TypeDevice
	case '?':
		typeKey = visual.TypeUnknown
	}
	l.paint(typeKey, mode[:1])
	keys := [9]string{
		visual.UserRead, visual.UserWrite, visual.UserExec,
		visual.GroupRead, visual.GroupWrite, visual.GroupExec,
		visual.OtherRead, visual.OtherWrite, visual.OtherExec,
	}
	if !e.mode.IsRegular() {
		keys[2] = visual.UserExecDir
	}
	for i := 1; i < 10; i++ {
		c := mode[i]
		key := keys[i-1]
		switch c {
		case '-':
			key = visual.Punctuation
		case 's', 'S', 't', 'T':
			key = visual.Special
		case '?':
			key = ""
		}
		l.paintCode(l.theme.Code(key), mode[i:i+1])
	}
}

// octalOf is the --octal-permissions column: the permission and special
// bits as four octal digits.
func octalOf(e *entry) string {
	if e.info == nil {
		return "????"
	}
	bits := uint32(e.mode.Perm())
	if e.mode&os.ModeSetuid != 0 {
		bits |= 0o4000
	}
	if e.mode&os.ModeSetgid != 0 {
		bits |= 0o2000
	}
	if e.mode&os.ModeSticky != 0 {
		bits |= 0o1000
	}
	s := strconv.FormatUint(uint64(bits), 8)
	return padLeftZero(s, 4)
}

func padLeftZero(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// idKey picks the theme key for an owner or group: the caller's own, or
// someone else's.
func (l *Ls) idKey(e *entry, group bool) string {
	if e.info == nil {
		return ""
	}
	if group {
		if l.selfGIDs[e.sys.gid] {
			return visual.You
		}
		return visual.Others
	}
	if l.selfUID >= 0 && uint32(l.selfUID) == e.sys.uid {
		return visual.You
	}
	return visual.Others
}

// sizeKey picks the theme key for an entry's size.
func (l *Ls) sizeKey(e *entry) string {
	if l.ColorScale == ScaleSize || l.ColorScale == ScaleAll {
		return visual.SizeKey(sizeOf(e))
	}
	return visual.Size
}

// timeKey picks the theme key for an entry's timestamp.
func (l *Ls) timeKey(e *entry) string {
	if e.info != nil && (l.ColorScale == ScaleAge || l.ColorScale == ScaleAll) {
		return visual.AgeKey(l.timeOf(e), l.Now())
	}
	return visual.Date
}

// repoFor returns the git repository whose working tree holds dir, opening
// each one once; nil when there is none or it cannot be read (which is
// reported once, as a minor problem).
func (l *Ls) repoFor(dir string) *gitstat.Repo {
	if l.repos == nil {
		l.repos, l.dirRoots = map[string]*gitstat.Repo{}, map[string]string{}
	}
	root, seen := l.dirRoots[dir]
	if !seen {
		var ok bool
		var err error
		root, ok, err = gitstat.FindRoot(dir)
		if err != nil || !ok {
			root = ""
		}
		l.dirRoots[dir] = root
	}
	if root == "" {
		return nil
	}
	repo, opened := l.repos[root]
	if !opened {
		var err error
		if repo, err = gitstat.Open(root, l.Getenv); err != nil {
			l.fail(minor, "cannot read git repository %s: %s", quoteArg(root), gitErrText(err))
			repo = nil
		}
		l.repos[root] = repo
	}
	return repo
}

func gitErrText(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return core.ErrText(err)
	}
	return err.Error()
}

// parentOf is the directory an entry's path is in.
func parentOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if os.IsPathSeparator(p[i]) {
			if i == 0 {
				return p[:1]
			}
			return p[:i]
		}
	}
	return "."
}

// gitStatus fills in an entry's git status for --git.
func (l *Ls) gitStatus(e *entry) string {
	if e.info == nil {
		return "  "
	}
	dir := parentOf(e.path)
	if e.isDir() {
		dir = e.path
	}
	repo := l.repoFor(dir)
	if repo == nil {
		return "  "
	}
	s, ok := repo.Status(e.path, e.isDir())
	if !ok {
		return "  "
	}
	return string([]byte{byte(s.Staged), byte(s.Unstaged)})
}

// putGit writes a status, each letter in its color.
func (l *Ls) putGit(status string) {
	for i := range len(status) {
		key := ""
		switch gitstat.Code(status[i]) {
		case gitstat.New:
			key = visual.GitNew
		case gitstat.Modified:
			key = visual.GitMod
		case gitstat.Deleted:
			key = visual.GitDel
		case gitstat.TypeChange:
			key = visual.GitType
		case gitstat.Ignored:
			key = visual.GitIgnore
		case gitstat.Conflicted:
			key = visual.GitClash
		case gitstat.Clean:
			key = visual.GitClean
		}
		l.paint(key, status[i:i+1])
	}
}

// gitIgnored reports whether --git-ignore leaves out the entry name found
// in dir.
func (l *Ls) gitIgnored(dir, name string, isDir bool) bool {
	repo := l.repoFor(dir)
	return repo != nil && repo.IsIgnored(joinPath(dir, name), isDir)
}

// keep applies --only-dirs and --only-files.
func (l *Ls) keep(e *entry) bool {
	switch l.Only {
	case OnlyDirs:
		return e.isDirLike()
	case OnlyFiles:
		return !e.isDirLike()
	}
	return true
}

// shown filters entries for display, leaving the list itself as it is.
func (l *Ls) shown(entries []*entry) []*entry {
	if l.Only == OnlyAll {
		return entries
	}
	return slices.DeleteFunc(slices.Clone(entries), func(e *entry) bool { return !l.keep(e) })
}

// totalSize is --total-size's size of a directory: the apparent size of
// everything under it, symbolic links counted as themselves and never
// followed. A directory that cannot be read is reported, and counts what
// could be.
func (l *Ls) totalSize(dir string) int64 {
	if l.totals == nil {
		l.totals = map[string]int64{}
	}
	if n, ok := l.totals[dir]; ok {
		return n
	}
	l.totals[dir] = 0 // a loop through a bind mount ends here
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		l.fail(minor, "cannot read directory %s: %s", quoteArg(dir), core.ErrText(err))
	}
	var n int64
	for _, de := range entries {
		info, err := de.Info()
		if err != nil {
			continue
		}
		if info.IsDir() {
			n += l.totalSize(joinPath(dir, de.Name()))
			continue
		}
		n += info.Size()
	}
	l.totals[dir] = n
	return n
}

// runTree lists the operands as the roots of one tree, each directory's
// contents drawn beneath it, in one table so that the long format's
// columns line up across the whole tree.
func (l *Ls) runTree(operands []*entry) {
	var rows []*entry
	for _, e := range operands {
		rows = append(rows, e)
		if e.isDir() && !l.Directory {
			rows = l.treeChildren(e, nil, 1, []os.FileInfo{e.info}, serious, rows)
		}
	}
	l.printEntries(rows)
	l.printed = true
}

// treeChildren appends the contents of the directory parent, and theirs,
// to rows, each with its branches drawn. last is the path of "last
// sibling" flags down to parent; depth the level of parent's children.
func (l *Ls) treeChildren(parent *entry, last []bool, depth int, ancestors []os.FileInfo, severity int, rows []*entry) []*entry {
	if l.Level > 0 && depth > l.Level {
		return rows
	}
	entries, ok := l.readDir(parent.path, severity)
	if !ok {
		return rows
	}
	entries = slices.DeleteFunc(entries, func(e *entry) bool {
		return e.name == "." || e.name == ".." || !l.keep(e)
	})
	for i, e := range entries {
		here := append(slices.Clone(last), i == len(entries)-1)
		e.branch = l.branches.Prefix(here)
		rows = append(rows, e)
		if !e.isDir() {
			continue
		}
		if l.Dereference && slices.ContainsFunc(ancestors, func(a os.FileInfo) bool { return os.SameFile(a, e.info) }) {
			l.fail(serious, "%s: not listing already-listed directory", quoteArg(e.path))
			continue
		}
		rows = l.treeChildren(e, here, depth+1, append(ancestors, e.info), minor, rows)
	}
	return rows
}
