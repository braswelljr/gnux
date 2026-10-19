package fsys

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WalkDir is filepath.WalkDir over the file system in use.
func WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(root, fs.FileInfoToDirEntry(info), fn)
	}
	if err == filepath.SkipDir || err == filepath.SkipAll {
		return nil
	}
	return err
}

func walkDir(path string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			err = nil
		}
		return err
	}
	entries, err := ReadDir(path)
	if err != nil {
		if err = fn(path, d, err); err != nil {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
	}
	for _, e := range entries {
		if err := walkDir(filepath.Join(path, e.Name()), e, fn); err != nil {
			if err == filepath.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// Walk is filepath.Walk over the file system in use.
func Walk(root string, fn filepath.WalkFunc) error {
	info, err := Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walk(root, info, fn)
	}
	if err == filepath.SkipDir || err == filepath.SkipAll {
		return nil
	}
	return err
}

func walk(path string, info os.FileInfo, fn filepath.WalkFunc) error {
	if !info.IsDir() {
		return fn(path, info, nil)
	}
	entries, err := ReadDir(path)
	err1 := fn(path, info, err)
	if err != nil || err1 != nil {
		return err1
	}
	for _, e := range entries {
		name := filepath.Join(path, e.Name())
		fi, err := Lstat(name)
		if err != nil {
			if err := fn(name, fi, err); err != nil && err != filepath.SkipDir {
				return err
			}
			continue
		}
		if err := walk(name, fi, fn); err != nil {
			if !fi.IsDir() || err != filepath.SkipDir {
				return err
			}
		}
	}
	return nil
}

// Glob is filepath.Glob over the file system in use: the names matching
// pattern, sorted, or nil when none does. Only a malformed pattern is an
// error.
func Glob(pattern string) ([]string, error) {
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, err
	}
	if !hasMeta(pattern) {
		if _, err := Lstat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	dir, file := filepath.Split(pattern)
	dir = cleanGlobPath(dir)
	if !hasMeta(dir[len(filepath.VolumeName(dir)):]) {
		return glob(dir, file, nil)
	}
	if dir == pattern {
		return nil, filepath.ErrBadPattern
	}
	dirs, err := Glob(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, d := range dirs {
		if matches, err = glob(d, file, matches); err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(filepath.Separator):
		return path
	}
	return path[:len(path)-1]
}

func glob(dir, pattern string, matches []string) ([]string, error) {
	info, err := Stat(dir)
	if err != nil || !info.IsDir() {
		return matches, nil
	}
	entries, err := ReadDir(dir)
	if err != nil {
		return matches, nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		if ok, err := filepath.Match(pattern, n); err != nil {
			return matches, err
		} else if ok {
			matches = append(matches, filepath.Join(dir, n))
		}
	}
	return matches, nil
}

func hasMeta(path string) bool {
	magic := `*?[`
	if filepath.Separator != '\\' {
		magic = `*?[\`
	}
	return strings.ContainsAny(path, magic)
}
