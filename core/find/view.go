package find

import (
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view of what -print writes (BR-13): each name in its
// type's color as lsx shows it, an icon before it, or the matches drawn as
// a tree under their starting point. Only -print to standard output takes
// part; -print0, -printf, -ls and the -f actions write GNU's bytes.

// treeNode is one file in the tree of matches: a starting point, a match,
// or a directory only on the way to one.
type treeNode struct {
	name     string // the starting point as given, else a base name
	kind     visual.Kind
	code     string // the name's color
	matched  bool
	children []*treeNode
	index    map[string]*treeNode
}

// prepareView sets up what the view needs for one Run.
func (f *Find) prepareView() {
	f.viewing = f.Tree || f.Color || f.Icons
	f.trees = nil
	if !f.viewing {
		return
	}
	getenv := f.getenv()
	f.theme = visual.Use(f.Color, f.Theme, getenv, f.diag.Warnf)
	f.nameColors = nil
	if f.Color {
		f.nameColors = visual.NewNames(getenv("LS_COLORS"))
	}
	f.branches = visual.BranchesFor(getenv)
}

func (f *Find) getenv() func(string) string {
	if f.Getenv != nil {
		return f.Getenv
	}
	return os.Getenv
}

// viewPrint is -print to standard output under the view.
func (f *Find) viewPrint(e *entry) {
	mode := f.modeOf(e)
	if f.Tree {
		f.addToTree(e, mode)
		return
	}
	base := baseName(e.path)
	dir := e.path[:len(e.path)-len(base)]
	f.buf = f.buf[:0]
	if f.Icons {
		f.buf = append(f.buf, visual.Paint(f.nameColor(mode, base), visual.Icon(base, kindOf(mode)))...)
		f.buf = append(f.buf, ' ')
	}
	f.buf = append(f.buf, f.theme.Paint(visual.Punctuation, dir)...)
	f.buf = append(f.buf, visual.Paint(f.nameColor(mode, base), base)...)
	f.buf = append(f.buf, '\n')
	f.write(f.stdout, f.buf)
}

// modeOf is e's mode for its color and icon: what the walk already knows,
// else the directory entry's lstat, read quietly (the color is not worth a
// diagnostic GNU would not print).
func (f *Find) modeOf(e *entry) fs.FileMode {
	if e.infoDone {
		if e.info == nil {
			return 0
		}
		return e.info.Mode()
	}
	if info, err := e.de.Info(); err == nil {
		return info.Mode()
	}
	return e.de.Type()
}

func (f *Find) nameColor(mode fs.FileMode, name string) string {
	if f.nameColors == nil {
		return ""
	}
	return f.nameColors.ModeColor(mode, name)
}

// kindOf tells visual.Icon what a mode is.
func kindOf(mode fs.FileMode) visual.Kind {
	switch {
	case mode.IsDir():
		return visual.KindDir
	case mode&fs.ModeSymlink != 0:
		return visual.KindLink
	case mode&fs.ModeNamedPipe != 0:
		return visual.KindPipe
	case mode&fs.ModeSocket != 0:
		return visual.KindSocket
	case mode&fs.ModeDevice != 0:
		return visual.KindDevice
	case mode.IsRegular() && mode&0o111 != 0:
		return visual.KindExec
	}
	return visual.KindFile
}

// addToTree records a match under the tree of its starting point, with
// the directories between them.
func (f *Find) addToTree(e *entry, mode fs.FileMode) {
	if len(f.trees) == 0 || f.trees[len(f.trees)-1].name != e.start {
		f.trees = append(f.trees, &treeNode{name: e.start, kind: visual.KindDir})
	}
	n := f.trees[len(f.trees)-1]
	rest := e.path[len(e.start):]
	for rest != "" {
		for rest != "" && isSep(rest[0]) {
			rest = rest[1:]
		}
		if rest == "" {
			break
		}
		name := rest
		if i := strings.IndexFunc(rest, func(r rune) bool { return r < 0x80 && isSep(byte(r)) }); i >= 0 {
			name, rest = rest[:i], rest[i:]
		} else {
			rest = ""
		}
		child := n.index[name]
		if child == nil {
			child = &treeNode{name: name, kind: visual.KindDir}
			if n.index == nil {
				n.index = map[string]*treeNode{}
			}
			n.index[name] = child
			n.children = append(n.children, child)
		}
		n = child
	}
	n.matched = true
	n.kind = kindOf(mode)
	n.code = f.nameColor(mode, n.name)
}

// drawTrees writes the trees of matches, once the walk has ended, and with
// the view on a count of them as tree(1) gives one.
func (f *Find) drawTrees() {
	if !f.Tree {
		return
	}
	dirs, files := 0, 0
	for _, root := range f.trees {
		f.drawNode(root, nil)
		d, fl := root.count()
		dirs, files = dirs+d, files+fl
	}
	f.trees = nil
	if !f.Pretty || dirs+files == 0 {
		return
	}
	line := plural(dirs, "directory", "directories") + ", " + plural(files, "file", "files")
	f.write(f.stdout, []byte("\n"+f.theme.Paint(visual.Punctuation, line)+"\n"))
}

func (f *Find) drawNode(n *treeNode, last []bool) {
	f.buf = append(f.buf[:0], f.theme.Paint(visual.Branch, f.branches.Prefix(last))...)
	code := n.code
	if !n.matched {
		// A directory on the way to a match is context, not a result.
		code = f.theme.Code(visual.Punctuation)
	}
	if f.Icons {
		f.buf = append(f.buf, visual.Paint(code, visual.Icon(n.name, n.kind))...)
		f.buf = append(f.buf, ' ')
	}
	f.buf = append(f.buf, visual.Paint(code, n.name)...)
	f.buf = append(f.buf, '\n')
	f.write(f.stdout, f.buf)
	for i, c := range n.children {
		f.drawNode(c, append(last, i == len(n.children)-1))
	}
}

// count returns the matched directories and other files below n.
func (n *treeNode) count() (dirs, files int) {
	for _, c := range n.children {
		if c.matched {
			if c.kind == visual.KindDir {
				dirs++
			} else {
				files++
			}
		}
		d, fl := c.count()
		dirs, files = dirs+d, files+fl
	}
	return dirs, files
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
