// Package fsys is how the engines reach the file system. Its functions have
// the signatures of the os and path/filepath functions they stand for
// (fsys.Open is os.Open, fsys.Abs is filepath.Abs), and by default they are
// exactly those functions. A sandbox can put a Rooted file system in their
// place, which resolves every path, absolute or relative to its own working
// directory, against a set of os.Root handles and refuses anything outside
// them: that is how gnux confines itself under FreeBSD's Capsicum, where a
// process may open files only relative to directories it already holds.
//
// The engines under core/ call these functions, never os.Open and the like
// directly, which the linter enforces (forbidigo in .golangci.yml).
package fsys

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// FS is a file system the engines can be pointed at.
type FS interface {
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	Create(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	ReadDir(name string) ([]os.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Mkdir(name string, perm os.FileMode) error
	MkdirAll(name string, perm os.FileMode) error
	MkdirTemp(dir, pattern string) (string, error)
	CreateTemp(dir, pattern string) (*os.File, error)
	Remove(name string) error
	RemoveAll(name string) error
	Rename(oldname, newname string) error
	Link(oldname, newname string) error
	Symlink(oldname, newname string) error
	Readlink(name string) (string, error)
	Chmod(name string, mode os.FileMode) error
	Chown(name string, uid, gid int) error
	Lchown(name string, uid, gid int) error
	Chtimes(name string, atime, mtime time.Time) error
	Truncate(name string, size int64) error
	Getwd() (string, error)
	Chdir(dir string) error
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
}

// OS is the file system the process sees: the os and filepath functions.
type OS struct{}

func (OS) Open(name string) (*os.File, error) { return os.Open(name) }
func (OS) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}
func (OS) Create(name string) (*os.File, error)       { return os.Create(name) }
func (OS) Stat(name string) (os.FileInfo, error)      { return os.Stat(name) }
func (OS) Lstat(name string) (os.FileInfo, error)     { return os.Lstat(name) }
func (OS) ReadDir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
func (OS) ReadFile(name string) ([]byte, error)       { return os.ReadFile(name) }
func (OS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (OS) Mkdir(name string, perm os.FileMode) error         { return os.Mkdir(name, perm) }
func (OS) MkdirAll(name string, perm os.FileMode) error      { return os.MkdirAll(name, perm) }
func (OS) MkdirTemp(dir, pattern string) (string, error)     { return os.MkdirTemp(dir, pattern) }
func (OS) CreateTemp(dir, pattern string) (*os.File, error)  { return os.CreateTemp(dir, pattern) }
func (OS) Remove(name string) error                          { return os.Remove(name) }
func (OS) RemoveAll(name string) error                       { return os.RemoveAll(name) }
func (OS) Rename(oldname, newname string) error              { return os.Rename(oldname, newname) }
func (OS) Link(oldname, newname string) error                { return os.Link(oldname, newname) }
func (OS) Symlink(oldname, newname string) error             { return os.Symlink(oldname, newname) }
func (OS) Readlink(name string) (string, error)              { return os.Readlink(name) }
func (OS) Chmod(name string, mode os.FileMode) error         { return os.Chmod(name, mode) }
func (OS) Chown(name string, uid, gid int) error             { return os.Chown(name, uid, gid) }
func (OS) Lchown(name string, uid, gid int) error            { return os.Lchown(name, uid, gid) }
func (OS) Chtimes(name string, atime, mtime time.Time) error { return os.Chtimes(name, atime, mtime) }
func (OS) Truncate(name string, size int64) error            { return os.Truncate(name, size) }
func (OS) Getwd() (string, error)                            { return os.Getwd() }
func (OS) Chdir(dir string) error                            { return os.Chdir(dir) }
func (OS) Abs(path string) (string, error)                   { return filepath.Abs(path) }
func (OS) EvalSymlinks(path string) (string, error)          { return filepath.EvalSymlinks(path) }

// current is the file system the functions below use.
var current atomic.Pointer[FS]

func init() {
	var fs FS = OS{}
	current.Store(&fs)
}

// Use makes fs the file system of the whole process. A sandbox calls it once,
// before any tool runs.
func Use(fs FS) { current.Store(&fs) }

// Current is the file system in use.
func Current() FS { return *current.Load() }

// Sandboxed reports whether the file system in use is not the process's own.
func Sandboxed() bool {
	_, own := Current().(OS)
	return !own
}

func Open(name string) (*os.File, error) { return Current().Open(name) }
func OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return Current().OpenFile(name, flag, perm)
}
func Create(name string) (*os.File, error)       { return Current().Create(name) }
func Stat(name string) (os.FileInfo, error)      { return Current().Stat(name) }
func Lstat(name string) (os.FileInfo, error)     { return Current().Lstat(name) }
func ReadDir(name string) ([]os.DirEntry, error) { return Current().ReadDir(name) }
func ReadFile(name string) ([]byte, error)       { return Current().ReadFile(name) }
func WriteFile(name string, data []byte, perm os.FileMode) error {
	return Current().WriteFile(name, data, perm)
}
func Mkdir(name string, perm os.FileMode) error        { return Current().Mkdir(name, perm) }
func MkdirAll(name string, perm os.FileMode) error     { return Current().MkdirAll(name, perm) }
func MkdirTemp(dir, pattern string) (string, error)    { return Current().MkdirTemp(dir, pattern) }
func CreateTemp(dir, pattern string) (*os.File, error) { return Current().CreateTemp(dir, pattern) }
func Remove(name string) error                         { return Current().Remove(name) }
func RemoveAll(name string) error                      { return Current().RemoveAll(name) }
func Rename(oldname, newname string) error             { return Current().Rename(oldname, newname) }
func Link(oldname, newname string) error               { return Current().Link(oldname, newname) }
func Symlink(oldname, newname string) error            { return Current().Symlink(oldname, newname) }
func Readlink(name string) (string, error)             { return Current().Readlink(name) }
func Chmod(name string, mode os.FileMode) error        { return Current().Chmod(name, mode) }
func Chown(name string, uid, gid int) error            { return Current().Chown(name, uid, gid) }
func Lchown(name string, uid, gid int) error           { return Current().Lchown(name, uid, gid) }
func Chtimes(name string, atime, mtime time.Time) error {
	return Current().Chtimes(name, atime, mtime)
}
func Truncate(name string, size int64) error   { return Current().Truncate(name, size) }
func Getwd() (string, error)                   { return Current().Getwd() }
func Chdir(dir string) error                   { return Current().Chdir(dir) }
func Abs(path string) (string, error)          { return Current().Abs(path) }
func EvalSymlinks(path string) (string, error) { return Current().EvalSymlinks(path) }
