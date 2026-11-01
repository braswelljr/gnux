// Package gitstat reads what git knows about the files of a working tree,
// without running git: each file's status (staged and unstaged, as eza's
// --git column shows it) and whether .gitignore rules exclude it. It reads
// the repository directly (the index, HEAD's tree from loose objects and
// pack files, the ignore files), through core/fsys, so it behaves the same
// on every platform and inside a sandbox.
//
// What it does not do, compared with git status: rename detection (a rename
// shows as a deletion and a new file), line-ending or filter conversion (a
// file whose content only differs by such a conversion shows as modified),
// reftable repositories (the staged column is then always clean) and
// alternates (objects borrowed from another repository are not found).
package gitstat

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core/fsys"
)

// Code is one column of a file's status.
type Code byte

// The status codes, as eza prints them.
const (
	Clean      Code = '-'
	New        Code = 'N'
	Modified   Code = 'M'
	Deleted    Code = 'D'
	TypeChange Code = 'T'
	Ignored    Code = 'I'
	Conflicted Code = 'U'
)

// rank orders codes for a directory, which shows the most significant
// status among its files; an ignored file does not make its directory
// anything.
func (c Code) rank() int {
	switch c {
	case New:
		return 1
	case Deleted:
		return 2
	case TypeChange:
		return 3
	case Modified:
		return 4
	case Conflicted:
		return 5
	}
	return 0
}

// Status is a file's status: Staged compares the index with HEAD, Unstaged
// the working tree with the index.
type Status struct {
	Staged, Unstaged Code
}

func merge(a, b Code) Code {
	if b.rank() > a.rank() {
		return b
	}
	return a
}

// Repo is one repository's working tree. Its methods are not safe for
// concurrent use. Close it to release the pack files it holds open.
type Repo struct {
	// Root is the working tree's top directory, absolute.
	Root string

	gitDir, commonDir string
	hashLen           int
	fileMode          bool
	excludesFile      string
	reftable          bool
	getenv            func(string) string

	index      map[string]*indexEntry // stage 0 entries
	paths      []string               // every index path, sorted
	conflicts  map[string]bool
	indexMtime time.Time

	headRead  bool
	head      map[string]treeEntry
	headOnly  []string // paths in HEAD and not in the index, sorted
	objects   *objectStore
	files     map[string]Status
	dirs      map[string]Status
	untracked map[string]bool
	ign       *ignorer
}

// Find returns the repository whose working tree holds dir, or nil when
// there is none. getenv reads HOME and XDG_CONFIG_HOME, for the global
// ignore file.
func Find(dir string, getenv func(string) string) (*Repo, error) {
	root, ok, err := FindRoot(dir)
	if err != nil || !ok {
		return nil, err
	}
	return Open(root, getenv)
}

// FindRoot returns the top of the working tree that holds dir, absolute,
// and false when dir is in none. It only looks for a .git entry, so a
// caller listing many directories can open each repository once.
func FindRoot(dir string) (string, bool, error) {
	cur, err := fsys.Abs(dir)
	if err != nil {
		return "", false, err
	}
	for {
		if _, ok, err := gitDirAt(cur); err != nil || ok {
			return cur, ok, err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false, nil
		}
		cur = parent
	}
}

// Open reads the repository whose working tree's top is root.
func Open(root string, getenv func(string) string) (*Repo, error) {
	gitDir, ok, err := gitDirAt(root)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%s: not a git working tree", root)
	}
	return open(root, gitDir, getenv)
}

// gitDirAt reports the git directory of a working tree whose top is dir:
// dir/.git, or the directory a .git file names (a linked worktree or a
// submodule).
func gitDirAt(dir string) (string, bool, error) {
	dotGit := filepath.Join(dir, ".git")
	info, err := fsys.Lstat(dotGit)
	switch {
	case err != nil:
		return "", false, nil
	case info.IsDir():
		return dotGit, true, nil
	case !info.Mode().IsRegular():
		return "", false, nil
	}
	data, err := fsys.ReadFile(dotGit)
	if err != nil {
		return "", false, err
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return "", false, nil
	}
	target = filepath.FromSlash(strings.TrimSpace(target))
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}
	return target, true, nil
}

func open(root, gitDir string, getenv func(string) string) (*Repo, error) {
	r := &Repo{
		Root: root, gitDir: gitDir, commonDir: gitDir, hashLen: 20, fileMode: true, getenv: getenv,
		files: map[string]Status{}, dirs: map[string]Status{}, untracked: map[string]bool{},
	}
	if data, err := fsys.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		common := filepath.FromSlash(strings.TrimSpace(string(data)))
		if !filepath.IsAbs(common) {
			common = filepath.Join(gitDir, common)
		}
		r.commonDir = common
	}
	r.readConfig()
	if err := r.readIndex(); err != nil {
		return nil, err
	}
	r.objects = &objectStore{dir: filepath.Join(r.commonDir, "objects"), hashLen: r.hashLen, cache: map[string]object{}}
	r.ign = &ignorer{repo: r, sets: map[string][]rule{}, memo: map[string]bool{}}
	return r, nil
}

// Close releases the pack files the repository holds open.
func (r *Repo) Close() {
	if r != nil && r.objects != nil {
		r.objects.close()
	}
}

// readConfig picks out the few settings this package honors from the
// repository's config: the object format, core.fileMode,
// core.excludesFile and the reference storage.
func (r *Repo) readConfig() {
	data, err := fsys.ReadFile(filepath.Join(r.commonDir, "config"))
	if err != nil {
		return
	}
	section := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = strings.ToLower(strings.Trim(strings.Fields(strings.Trim(line, "[]"))[0], "\""))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.ToLower(strings.TrimSpace(key)), strings.Trim(strings.TrimSpace(value), "\"")
		switch section + "." + key {
		case "extensions.objectformat":
			if strings.EqualFold(value, "sha256") {
				r.hashLen = 32
			}
		case "extensions.refstorage":
			r.reftable = strings.EqualFold(value, "reftable")
		case "core.filemode":
			r.fileMode = !isFalse(value)
		case "core.excludesfile":
			r.excludesFile = r.expandHome(value)
		}
	}
}

func isFalse(v string) bool {
	switch strings.ToLower(v) {
	case "false", "no", "off", "0":
		return true
	}
	return false
}

// expandHome expands a leading "~/" as git does.
func (r *Repo) expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(r.home(), filepath.FromSlash(rest))
	}
	return filepath.FromSlash(p)
}

func (r *Repo) home() string {
	if h := r.getenv("HOME"); h != "" {
		return h
	}
	return r.getenv("USERPROFILE")
}

// rel returns path relative to the working tree's top, slash-separated
// ("" for the top itself), or false when it is outside the tree.
func (r *Repo) rel(p string) (string, bool) {
	abs, err := fsys.Abs(p)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(r.Root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "", true
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// inGitDir reports whether rel is the .git directory or inside it.
func inGitDir(rel string) bool {
	return rel == ".git" || strings.HasPrefix(rel, ".git/")
}

// Status returns the status of the file or directory at p (as the caller
// reaches it: relative to the working directory, or absolute), and false
// when p is outside the working tree. A directory has the most significant
// status of the files under it.
func (r *Repo) Status(p string, isDir bool) (Status, bool) {
	rel, ok := r.rel(p)
	if !ok {
		return Status{}, false
	}
	if inGitDir(rel) {
		return Status{Clean, Clean}, true
	}
	if isDir {
		return r.dirStatus(rel), true
	}
	return r.fileStatus(rel), true
}

// IsIgnored reports whether the ignore rules exclude the file or directory
// at p. Anything outside the working tree, and the .git directory itself,
// is not ignored.
func (r *Repo) IsIgnored(p string, isDir bool) bool {
	rel, ok := r.rel(p)
	if !ok || rel == "" || inGitDir(rel) {
		return false
	}
	if _, tracked := r.index[rel]; tracked {
		return false
	}
	return r.ign.ignored(rel, isDir)
}

func (r *Repo) fileStatus(rel string) Status {
	if s, ok := r.files[rel]; ok {
		return s
	}
	var s Status
	e, tracked := r.index[rel]
	switch {
	case r.conflicts[rel]:
		s = Status{Conflicted, Conflicted}
	case !tracked && r.ign.ignored(rel, false):
		s = Status{Clean, Ignored}
	case !tracked:
		s = Status{Clean, New}
	default:
		s = Status{r.staged(rel, e), r.worktree(rel, e)}
	}
	r.files[rel] = s
	return s
}

// staged compares an index entry with HEAD.
func (r *Repo) staged(rel string, e *indexEntry) Code {
	r.readHead()
	h, ok := r.head[rel]
	switch {
	case r.head == nil:
		return Clean
	case !ok:
		return New
	case h.mode>>12 != e.mode>>12:
		return TypeChange
	case h.hash != e.hash || h.mode != e.mode:
		return Modified
	}
	return Clean
}

// worktree compares the file in the working tree with its index entry,
// trusting the index's stat data when it matches, as git does, and
// otherwise hashing the content.
func (r *Repo) worktree(rel string, e *indexEntry) Code {
	if e.mode == modeGitlink {
		return Clean
	}
	full := filepath.Join(r.Root, filepath.FromSlash(rel))
	info, err := fsys.Lstat(full)
	if err != nil {
		return Deleted
	}
	isLink := info.Mode()&os.ModeSymlink != 0
	switch {
	case info.IsDir(), isLink != (e.mode>>12 == modeSymlink>>12):
		return TypeChange
	case r.fileMode && !isLink && (info.Mode()&0o100 != 0) != (e.mode&0o100 != 0):
		return Modified
	}
	if uint32(info.Size()) == e.size && e.sameTime(info.ModTime()) && info.ModTime().Before(r.indexMtime) {
		return Clean
	}
	var content []byte
	if isLink {
		target, err := fsys.Readlink(full)
		if err != nil {
			return Modified
		}
		content = []byte(filepath.ToSlash(target))
	} else if content, err = fsys.ReadFile(full); err != nil {
		return Modified
	}
	if blobHash(r.hashLen, content) != e.hash {
		return Modified
	}
	return Clean
}

func (r *Repo) dirStatus(rel string) Status {
	if s, ok := r.dirs[rel]; ok {
		return s
	}
	if rel != "" && r.ign.ignored(rel, true) {
		s := Status{Clean, Ignored}
		r.dirs[rel] = s
		return s
	}
	prefix := ""
	if rel != "" {
		prefix = rel + "/"
	}
	s := Status{Clean, Clean}
	for _, p := range withPrefix(r.paths, prefix) {
		f := r.fileStatus(p)
		s.Staged, s.Unstaged = merge(s.Staged, f.Staged), merge(s.Unstaged, f.Unstaged)
	}
	r.readHead()
	if len(withPrefix(r.headOnly, prefix)) > 0 {
		s.Staged = merge(s.Staged, Deleted)
	}
	if r.hasUntracked(rel) {
		s.Unstaged = merge(s.Unstaged, New)
	}
	r.dirs[rel] = s
	return s
}

// withPrefix returns the run of sorted paths that start with prefix.
func withPrefix(sorted []string, prefix string) []string {
	if prefix == "" {
		return sorted
	}
	lo, _ := slices.BinarySearch(sorted, prefix)
	hi := lo
	for hi < len(sorted) && strings.HasPrefix(sorted[hi], prefix) {
		hi++
	}
	return sorted[lo:hi]
}

// hasUntracked reports whether the directory rel holds a file that is
// neither tracked nor ignored, looking past ignored directories, the .git
// directory and nested repositories.
func (r *Repo) hasUntracked(rel string) bool {
	if v, ok := r.untracked[rel]; ok {
		return v
	}
	r.untracked[rel] = false // a directory reached again through a link
	dir := filepath.Join(r.Root, filepath.FromSlash(rel))
	entries, _ := fsys.ReadDir(dir)
	found := false
	for _, de := range entries {
		child := path.Join(rel, de.Name())
		if inGitDir(child) {
			continue
		}
		if de.IsDir() {
			if _, nested, _ := gitDirAt(filepath.Join(dir, de.Name())); nested {
				continue
			}
			if !r.ign.ignored(child, true) && r.hasUntracked(child) {
				found = true
				break
			}
			continue
		}
		if _, tracked := r.index[child]; !tracked && !r.conflicts[child] && !r.ign.ignored(child, false) {
			found = true
			break
		}
	}
	r.untracked[rel] = found
	return found
}

// errCorrupt reports a repository file this package cannot read.
var errCorrupt = errors.New("unsupported or corrupt git data")

// trimNUL returns b up to its first NUL.
func trimNUL(b []byte) []byte {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return b[:i]
	}
	return b
}
