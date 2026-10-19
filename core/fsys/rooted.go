package fsys

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Rooted is a file system made of directories: each path, absolute or
// relative to its own working directory, is resolved within the directory
// that holds it, through an os.Root, so that neither .. nor a symbolic link
// leads out of it; a path in none of them is refused (permission denied),
// as is a change to one that is read-only. It is safe for concurrent use.
type Rooted struct {
	mu    sync.Mutex
	roots []*root // longest path first, so the innermost directory wins
	cwd   string
	// processCwd makes the working directory the process's own (os.Getwd,
	// os.Chdir), for the gnux_rooted test build; a sandbox keeps its own.
	processCwd bool
}

// root is one directory of a Rooted file system.
type root struct {
	path     string // absolute, without symbolic links
	alias    string // absolute, as given, which may hold symbolic links
	r        *os.Root
	readOnly bool
}

// NewRooted opens a Rooted file system of the directories write (readable
// and writable) and read (readable only), working in cwd, which may lie
// outside them (a relative path then reaches nothing).
func NewRooted(write, read []string, cwd string) (*Rooted, error) {
	// The working directory as the kernel knows it, as the roots are; once
	// confined, nothing could find that out.
	if real, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = real
	}
	fsys := &Rooted{cwd: filepath.Clean(cwd)}
	add := func(dir string, readOnly bool) error {
		alias, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		real := alias
		if r, err := filepath.EvalSymlinks(alias); err == nil {
			real = r
		}
		r, err := os.OpenRoot(real)
		if err != nil {
			return err
		}
		fsys.roots = append(fsys.roots, &root{path: real, alias: alias, r: r, readOnly: readOnly})
		return nil
	}
	for _, d := range write {
		if err := add(d, false); err != nil {
			fsys.Close()
			return nil, err
		}
	}
	for _, d := range read {
		if err := add(d, true); err != nil {
			fsys.Close()
			return nil, err
		}
	}
	sort.SliceStable(fsys.roots, func(i, j int) bool { return len(fsys.roots[i].path) > len(fsys.roots[j].path) })
	return fsys, nil
}

// Close closes every root.
func (f *Rooted) Close() {
	for _, r := range f.roots {
		_ = r.r.Close()
	}
}

// abs is name made absolute against the working directory.
func (f *Rooted) abs(name string) string {
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	cwd, _ := f.Getwd()
	return filepath.Join(cwd, name)
}

// resolve finds the root holding name and name's path within it. A name
// that ends in a separator keeps it, which says the name must be a
// directory.
func (f *Rooted) resolve(op, name string, write bool) (*root, string, error) {
	if name == "" {
		return nil, "", &os.PathError{Op: op, Path: name, Err: syscall.ENOENT}
	}
	r, rel, ok := f.locate(f.abs(name))
	if ok {
		if write && r.readOnly {
			return nil, "", &os.PathError{Op: op, Path: name, Err: syscall.EROFS}
		}
		if rel != "." && (strings.HasSuffix(name, "/") || strings.HasSuffix(name, string(filepath.Separator))) {
			rel += string(filepath.Separator)
		}
		return r, rel, nil
	}
	return nil, "", &os.PathError{Op: op, Path: name, Err: syscall.EACCES}
}

// locate finds the root holding the absolute path abs, under the path it
// really has or the one it was given, and abs within it.
func (f *Rooted) locate(abs string) (*root, string, bool) {
	for _, r := range f.roots {
		for _, base := range []string{r.path, r.alias} {
			rel, err := filepath.Rel(base, abs)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return r, rel, true
			}
		}
	}
	return nil, "", false
}

// fix gives an error from a root the path the caller used, and reports a
// path that escapes the root (through a symbolic link) as permission denied.
func fix(err error, op, name string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "path escapes from parent") {
		return &os.PathError{Op: op, Path: name, Err: syscall.EACCES}
	}
	var pe *os.PathError
	if errors.As(err, &pe) {
		return &os.PathError{Op: pe.Op, Path: name, Err: pe.Err}
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le
	}
	return err
}

// writes reports whether flag opens a file to change it.
func writes(flag int) bool {
	return flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0
}

// escapes reports whether err is os.Root refusing a path that leaves it,
// which it says also of every absolute symbolic link, even one that leads
// back inside.
func escapes(err error) bool {
	return err != nil && strings.Contains(err.Error(), "path escapes from parent")
}

// do runs op on name in the root that holds it. Should the root refuse to
// follow a symbolic link (an absolute one, or one into another root), name
// is resolved here, through every root, and op runs once more on where it
// leads; follow says whether a link in name's last component is followed.
func (f *Rooted) do(op, name string, write, follow bool, run func(r *root, rel string) error) error {
	r, rel, err := f.resolve(op, name, write)
	if err != nil {
		return err
	}
	err = run(r, rel)
	if !escapes(err) {
		return fix(err, op, name)
	}
	real, rerr := f.realPath(name, follow)
	if rerr != nil {
		return &os.PathError{Op: op, Path: name, Err: unwrap(rerr)}
	}
	r, rel, rerr = f.resolve(op, real, write)
	if rerr != nil {
		return &os.PathError{Op: op, Path: name, Err: unwrap(rerr)}
	}
	return fix(run(r, rel), op, name)
}

// realPath is name with its symbolic links resolved through the roots: all
// of them with follow, all but the last component's without.
func (f *Rooted) realPath(name string, follow bool) (string, error) {
	if follow {
		// A file yet to be made, or a link's missing target: where it would
		// be, so that the operation finds it missing, or makes it.
		return f.evalSymlinks(name, true)
	}
	abs := f.abs(name)
	dir, err := f.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(abs)), nil
}

func (f *Rooted) Open(name string) (*os.File, error) { return f.OpenFile(name, os.O_RDONLY, 0) }

func (f *Rooted) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	var file *os.File
	err := f.do("open", name, writes(flag), true, func(r *root, rel string) (err error) {
		file, err = r.r.OpenFile(rel, flag, perm)
		return err
	})
	return file, err
}

func (f *Rooted) Create(name string) (*os.File, error) {
	return f.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

func (f *Rooted) Stat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := f.do("stat", name, false, true, func(r *root, rel string) (err error) {
		info, err = r.r.Stat(rel)
		return err
	})
	return info, err
}

func (f *Rooted) Lstat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := f.do("lstat", name, false, false, func(r *root, rel string) (err error) {
		info, err = r.r.Lstat(rel)
		return err
	})
	return info, err
}

func (f *Rooted) ReadDir(name string) ([]os.DirEntry, error) {
	dir, err := f.Open(name)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	// os.ReadDir sorts by name; so does this.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	// An entry's Info examines the file when it is asked for, by its path,
	// which must go through the roots too (and, under Capsicum, could not go
	// any other way).
	for i, e := range entries {
		entries[i] = rootedEntry{DirEntry: e, fsys: f, path: filepath.Join(name, e.Name())}
	}
	return entries, fix(err, "readdirent", name)
}

// rootedEntry is a directory entry whose Info is looked up in its Rooted
// file system.
type rootedEntry struct {
	os.DirEntry
	fsys *Rooted
	path string
}

func (e rootedEntry) Info() (os.FileInfo, error) { return e.fsys.Lstat(e.path) }

func (f *Rooted) ReadFile(name string) ([]byte, error) {
	var data []byte
	err := f.do("open", name, false, true, func(r *root, rel string) (err error) {
		data, err = r.r.ReadFile(rel)
		return err
	})
	return data, err
}

func (f *Rooted) WriteFile(name string, data []byte, perm os.FileMode) error {
	return f.do("open", name, true, true, func(r *root, rel string) error { return r.r.WriteFile(rel, data, perm) })
}

func (f *Rooted) Mkdir(name string, perm os.FileMode) error {
	return f.do("mkdir", name, true, false, func(r *root, rel string) error { return r.r.Mkdir(rel, perm) })
}

func (f *Rooted) MkdirAll(name string, perm os.FileMode) error {
	return f.do("mkdir", name, true, true, func(r *root, rel string) error { return r.r.MkdirAll(rel, perm) })
}

// tempName is os's temporary name: the pattern's prefix, a random number and
// its suffix.
func tempName(dir, pattern string) (string, error) {
	if strings.ContainsRune(pattern, filepath.Separator) {
		return "", errors.New("pattern contains path separator")
	}
	prefix, suffix := pattern, ""
	if i := strings.LastIndexByte(pattern, '*'); i >= 0 {
		prefix, suffix = pattern[:i], pattern[i+1:]
	}
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, prefix+strconv.FormatUint(uint64(rand.Uint32()), 10)+suffix), nil
}

func (f *Rooted) MkdirTemp(dir, pattern string) (string, error) {
	for range 10000 {
		name, err := tempName(dir, pattern)
		if err != nil {
			return "", &os.PathError{Op: "mkdirtemp", Path: dir + string(filepath.Separator) + pattern, Err: err}
		}
		err = f.Mkdir(name, 0o700)
		if err == nil {
			return name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", &os.PathError{Op: "mkdirtemp", Path: dir, Err: fs.ErrExist}
}

func (f *Rooted) CreateTemp(dir, pattern string) (*os.File, error) {
	for range 10000 {
		name, err := tempName(dir, pattern)
		if err != nil {
			return nil, &os.PathError{Op: "createtemp", Path: dir + string(filepath.Separator) + pattern, Err: err}
		}
		file, err := f.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
	}
	return nil, &os.PathError{Op: "createtemp", Path: dir, Err: fs.ErrExist}
}

func (f *Rooted) Remove(name string) error {
	return f.do("remove", name, true, false, func(r *root, rel string) error { return r.r.Remove(rel) })
}

func (f *Rooted) RemoveAll(name string) error {
	r, rel, err := f.resolve("unlinkat", name, true)
	if err != nil {
		return err
	}
	if rel == "." {
		// os.Root cannot remove itself; empty it, then remove it through
		// its parent, should that be a root too.
		entries, err := f.ReadDir(name)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := r.r.RemoveAll(e.Name()); err != nil {
				return fix(err, "unlinkat", name)
			}
		}
		return f.Remove(name)
	}
	return f.do("unlinkat", name, true, false, func(r *root, rel string) error { return r.r.RemoveAll(rel) })
}

// pair resolves both names of a rename or link, which must share a root.
func (f *Rooted) pair(op, oldname, newname string) (*root, string, string, error) {
	or, orel, err := f.resolve(op, oldname, true)
	if err != nil {
		return nil, "", "", &os.LinkError{Op: op, Old: oldname, New: newname, Err: unwrap(err)}
	}
	nr, nrel, err := f.resolve(op, newname, true)
	if err != nil {
		return nil, "", "", &os.LinkError{Op: op, Old: oldname, New: newname, Err: unwrap(err)}
	}
	if or != nr {
		// Different directories are different devices as far as a rename
		// can tell; mv then copies, as it does across file systems.
		return nil, "", "", &os.LinkError{Op: op, Old: oldname, New: newname, Err: syscall.EXDEV}
	}
	return or, orel, nrel, nil
}

func unwrap(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// twoNames runs a rename or link of oldname to newname in their root. Should
// the root refuse a symbolic link in either one's directory, both are
// resolved here and it runs once more; neither last component is followed.
func (f *Rooted) twoNames(op, oldname, newname string, run func(r *root, orel, nrel string) error) error {
	r, orel, nrel, err := f.pair(op, oldname, newname)
	if err != nil {
		return err
	}
	err = run(r, orel, nrel)
	if escapes(err) {
		oreal, oerr := f.realPath(oldname, false)
		nreal, nerr := f.realPath(newname, false)
		if oerr == nil && nerr == nil {
			if r, orel, nrel, err = f.pair(op, oreal, nreal); err != nil {
				return &os.LinkError{Op: op, Old: oldname, New: newname, Err: unwrap(err)}
			}
			err = run(r, orel, nrel)
		}
	}
	if err != nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: unwrap(fix(err, op, oldname))}
	}
	return nil
}

func (f *Rooted) Rename(oldname, newname string) error {
	return f.twoNames("rename", oldname, newname, func(r *root, orel, nrel string) error { return r.r.Rename(orel, nrel) })
}

func (f *Rooted) Link(oldname, newname string) error {
	return f.twoNames("link", oldname, newname, func(r *root, orel, nrel string) error { return r.r.Link(orel, nrel) })
}

// Symlink makes newname a link holding the text oldname, which is not
// examined: following the link later is what the roots constrain.
func (f *Rooted) Symlink(oldname, newname string) error {
	err := f.do("symlink", newname, true, false, func(r *root, rel string) error { return r.r.Symlink(oldname, rel) })
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: unwrap(err)}
	}
	return nil
}

func (f *Rooted) Readlink(name string) (string, error) {
	var target string
	err := f.do("readlink", name, false, false, func(r *root, rel string) (err error) {
		target, err = r.r.Readlink(rel)
		return err
	})
	return target, err
}

func (f *Rooted) Chmod(name string, mode os.FileMode) error {
	return f.do("chmod", name, true, true, func(r *root, rel string) error { return r.r.Chmod(rel, mode) })
}

func (f *Rooted) Chown(name string, uid, gid int) error {
	return f.do("chown", name, true, true, func(r *root, rel string) error { return r.r.Chown(rel, uid, gid) })
}

func (f *Rooted) Lchown(name string, uid, gid int) error {
	return f.do("lchown", name, true, false, func(r *root, rel string) error { return r.r.Lchown(rel, uid, gid) })
}

func (f *Rooted) Chtimes(name string, atime, mtime time.Time) error {
	return f.do("chtimes", name, true, true, func(r *root, rel string) error { return r.r.Chtimes(rel, atime, mtime) })
}

func (f *Rooted) Truncate(name string, size int64) error {
	file, err := f.OpenFile(name, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(size); err != nil {
		return &os.PathError{Op: "truncate", Path: name, Err: unwrap(err)}
	}
	return nil
}

func (f *Rooted) Getwd() (string, error) {
	if f.processCwd {
		return os.Getwd() //nolint:forbidigo // the test build follows the process
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cwd, nil
}

func (f *Rooted) Chdir(dir string) error {
	info, err := f.Stat(dir)
	if err != nil {
		return &os.PathError{Op: "chdir", Path: dir, Err: unwrap(err)}
	}
	if !info.IsDir() {
		return &os.PathError{Op: "chdir", Path: dir, Err: syscall.ENOTDIR}
	}
	abs := f.abs(dir)
	if f.processCwd {
		return os.Chdir(abs) //nolint:forbidigo // the test build follows the process
	}
	f.mu.Lock()
	f.cwd = abs
	f.mu.Unlock()
	return nil
}

func (f *Rooted) Abs(path string) (string, error) { return f.abs(path), nil }

// EvalSymlinks resolves path's symbolic links within the roots, as
// filepath.EvalSymlinks does in the whole file system. A directory outside
// every root that the path passes through is taken by its name (it cannot
// be examined), but the path must end inside a root.
func (f *Rooted) EvalSymlinks(path string) (string, error) {
	return f.evalSymlinks(path, false)
}

// evalSymlinks is EvalSymlinks; with missingLast, a last component that does
// not exist is no error, and the path it would have is returned.
func (f *Rooted) evalSymlinks(path string, missingLast bool) (string, error) {
	abs := f.abs(path)
	// Start from the root that holds the path, as its real path, so that
	// nothing above it need be examined.
	var resolved, rest string
	if r, rel, ok := f.locate(abs); ok {
		resolved, rest = r.path, rel
	} else {
		vol := filepath.VolumeName(abs)
		resolved, rest = vol+string(filepath.Separator), abs[len(vol):]
	}
	parts := strings.Split(rest, string(filepath.Separator))
	for hops := 0; len(parts) > 0; {
		part := parts[0]
		parts = parts[1:]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			resolved = filepath.Dir(resolved)
			continue
		}
		next := filepath.Join(resolved, part)
		if _, _, inside := f.locate(next); !inside {
			if len(parts) == 0 {
				return "", &os.PathError{Op: "lstat", Path: path, Err: syscall.EACCES}
			}
			resolved = next
			continue
		}
		info, err := f.Lstat(next)
		if err != nil {
			if missingLast && len(parts) == 0 && errors.Is(err, fs.ErrNotExist) {
				return next, nil
			}
			return "", &os.PathError{Op: "lstat", Path: path, Err: unwrap(err)}
		}
		if info.Mode()&os.ModeSymlink == 0 {
			resolved = next
			continue
		}
		if hops++; hops > 255 {
			return "", &os.PathError{Op: "lstat", Path: path, Err: syscall.ELOOP}
		}
		target, err := f.Readlink(next)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(target) {
			vol := filepath.VolumeName(target)
			resolved = vol + string(filepath.Separator)
			target = target[len(vol):]
		}
		parts = append(strings.Split(target, string(filepath.Separator)), parts...)
	}
	return filepath.Clean(resolved), nil
}

// String names the file system's directories, for diagnostics.
func (f *Rooted) String() string {
	var parts []string
	for _, r := range f.roots {
		mode := "rw"
		if r.readOnly {
			mode = "ro"
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", r.path, mode))
	}
	return strings.Join(parts, ", ")
}
